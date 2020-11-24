package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/session"
	"github.com/gocolly/colly/v2"
)

const (
	newCityMicrocreameryHostname     = "newcitymicrocreamery.com"
	newCityOriginalsSectionTitleText = "New City Originals"
	headerMessage                    = `Here are today's New City flavors :icecream:`
)

// nolint: gochecknoglobals
var (
	newCityCambridgeMenuAddress = fmt.Sprintf("https://%s/cambridge-menu", newCityMicrocreameryHostname)
	detailsEmojiMap             = map[string]string{
		"E": ":egg:",
		"G": ":ear_of_rice:", // okay rice is generally gluten-free but this gets the idea across
		"S": ":seedling:",    // this one is also a stretch – it looks kind of like a soy bean?
		"A": ":tumbler_glass:",
		"N": ":peanuts:",
	}

	botTokenFlag              = flag.String("token", "", "Bot token for the Discord API")
	channelIDFlag             = flag.Int64("channel_id", 0, "Channel ID the bot should post to")
	onlyOriginalsFlag         = flag.Bool("only_originals", false, "Filter only New City Originals ice cream flavors")
	useCacheFlag              = flag.Bool("cache", false, "Use cache to determine changed flavors")
	outputUnchangedResultFlag = flag.Bool("print_unchanged", false, "Will output even if the flavors have not changed (does nothing if -cache is not set)")
	cachePathFlag             = flag.String("cache_path", "newcity.json", "Path to write cache to (does nothing if -cache is not set)")
	dryRunFlag                = flag.Bool("dry_run", false, "Dry-run")
)

var (
	// ErrNoDiscordBotToken happens when a token is not provided via the -bot_token flag or the BOT_TOKEN environment variable.
	ErrNoDiscordBotToken = errors.New("no Discord bot token provided")
	// ErrNoDiscordChannelID happens when a channel ID is not provided via the -channel_id flag, or it is not a valid int64.
	ErrNoDiscordChannelID = errors.New("no Discord channel ID provided")
)

// IceCream is a flavor type.
type IceCream struct {
	Name        string
	Description string
	RawDetails  []string
}

// IceCreams is a map of New City section titles to flavor lists.
type IceCreams map[string][]IceCream

func main() {
	flag.Parse()

	useCache := *useCacheFlag
	cachePath := *cachePathFlag

	var (
		cachedIceCreams IceCreams
		err             error
	)

	if useCache {
		cachedIceCreams, err = NewIceCreamsFromFile(cachePath)
		if err != nil {
			log.Fatal(err)
		}
	}

	// fetch ice creams
	iceCreams, err := NewIceCreams()
	if err != nil {
		log.Fatal(err)
	}

	messages := iceCreams.Messages(useCache, cachedIceCreams)

	dryRun := *dryRunFlag

	if dryRun {
		for _, message := range messages {
			fmt.Println(message)
		}
	} else if err := PostToDiscord(messages...); err != nil {
		log.Fatal(err)
	}

	if useCache {
		if err := iceCreams.Write(cachePath); err != nil {
			log.Fatal(err)
		}
	}
}

// NewIceCreams creates a new iceCreams instance using data scraped from the New City website.
func NewIceCreams() (IceCreams, error) {
	c := colly.NewCollector(
		colly.AllowedDomains(newCityMicrocreameryHostname),
	)

	iceCreams := IceCreams{}

	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL.String())
	})

	c.OnHTML(".menu-section", func(section *colly.HTMLElement) {
		category := section.ChildText(".menu-section-title")

		iceCreamsInCategory := []IceCream{}

		section.ForEach(".menu-item", func(i int, item *colly.HTMLElement) {
			iceCream := IceCream{}
			iceCream.Name = item.ChildText(".menu-item-title")
			iceCream.Description = item.ChildText(".menu-item-description")

			details := item.ChildText("span.menu-item-price-top")
			if details != "" {
				details = strings.Trim(details, "()")
				iceCream.RawDetails = strings.Split(details, ",")
			}

			iceCreamsInCategory = append(iceCreamsInCategory, iceCream)
		})

		if len(iceCreamsInCategory) == 0 {
			return
		}

		iceCreams[category] = iceCreamsInCategory
	})

	err := c.Visit(newCityCambridgeMenuAddress)
	if err != nil {
		return nil, err
	}

	c.Wait()

	return iceCreams, nil
}

// NewIceCreamsFromFile creates a new IceCreams instance using data from a given file path.
func NewIceCreamsFromFile(filename string) (IceCreams, error) {
	data, err := ioutil.ReadFile(filepath.Clean(filename))
	if err != nil {
		if os.IsNotExist(err) {
			log.Println(warn("no cache file exists"))
			return IceCreams{}, nil
		}

		return nil, err
	}

	var iceCreams IceCreams

	if err := json.Unmarshal(data, &iceCreams); err != nil {
		return nil, err
	}

	return iceCreams, nil
}

// PostToDiscord posts the given messages to Discord.
func PostToDiscord(messages ...string) error {
	if len(messages) == 0 {
		log.Println(warn("no messages to send to Discord"))
		return nil
	}

	token := *botTokenFlag
	if token == "" {
		token = os.Getenv("BOT_TOKEN")
		if token == "" {
			return ErrNoDiscordBotToken
		}
	}

	channelID := *channelIDFlag
	if channelID == 0 {
		return ErrNoDiscordChannelID
	}

	// auth with Discord
	s, err := session.New("Bot " + token)
	if err != nil {
		return err
	}

	// get channel to send to
	if err := s.Open(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	defer Close(s)

	// create message
	for _, message := range messages {
		m, err := s.SendText(discord.ChannelID(channelID), message)
		if err != nil {
			return err
		}

		log.Println("SENT:", m.ID)
	}

	return nil
}

func (c *IceCreams) Write(filename string) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0600)
}

// Messages splits an IceCreams instance into a list of messages.
func (c IceCreams) Messages(useCache bool, cache IceCreams) []string {
	messages := []string{bold(headerMessage)}
	onlyOriginals := *onlyOriginalsFlag
	outputUnchangedResult := *outputUnchangedResultFlag

	if cache == nil {
		cache = IceCreams{}
	}

	for category, creams := range c {
		if onlyOriginals && category != newCityOriginalsSectionTitleText {
			continue
		}

		addedCreams, removedCreams := Difference(creams, cache[category])

		s := strings.Builder{}
		shouldWriteCategoryTitle := !onlyOriginals && len(c) > 1 && (len(addedCreams) > 0 || len(removedCreams) > 0)

		writeCategory(&s, addedCreams, removedCreams, category, shouldWriteCategoryTitle, useCache, outputUnchangedResult)

		if s.Len() == 0 {
			continue
		}

		messages = append(messages, s.String())
	}

	if len(messages) == 1 && !outputUnchangedResult {
		return []string{}
	}

	return messages
}

// Details returns a formatted emojified string describing the flavor's allergens.
func (c IceCream) Details() string {
	if len(c.RawDetails) == 0 {
		return ""
	}

	e := make([]string, len(c.RawDetails))

	for i, d := range c.RawDetails {
		if emoji, ok := detailsEmojiMap[strings.ToUpper(d)]; ok {
			e[i] = emoji
		} else {
			e[i] = d
		}
	}

	return fmt.Sprintf("(%s)", strings.Join(e, ", "))
}

// Difference returns the difference between two slices – what was added and removed.
func Difference(latest []IceCream, last []IceCream) (added []IceCream, removed []IceCream) {
	if len(latest) == 0 || len(last) == 0 {
		if len(latest) == 0 {
			removed = last
		}

		if len(last) == 0 {
			added = latest
		}

		return added, removed
	}

	added = []IceCream{}
	removed = []IceCream{}

	lastMapped := map[string]IceCream{}
	for _, lastEl := range last {
		lastMapped[lastEl.Name] = lastEl
	}

	for _, latestEl := range latest {
		if _, found := lastMapped[latestEl.Name]; !found {
			added = append(added, latestEl)
		} else {
			delete(lastMapped, latestEl.Name)
		}
	}

	for _, value := range lastMapped {
		removed = append(removed, value)
	}

	return added, removed
}

func warn(s string) string {
	return fmt.Sprintf("⚠️: %s", s)
}

func bold(s string) string {
	return fmt.Sprintf("**%s**", s)
}

func italic(s string) string {
	return fmt.Sprintf("*%s*", s)
}

func writeCategory(s io.StringWriter, addedCreams []IceCream, removedCreams []IceCream, category string, shouldWriteCategoryTitle bool, useCache bool, outputUnchangedResult bool) {
	// Print category header if there are multiple headers to print.
	if shouldWriteCategoryTitle {
		if _, err := s.WriteString(bold(category) + "\n"); err != nil {
			log.Println(err)
		}
	}

	writeCategorySubtitle(s, addedCreams, category, "added", useCache, outputUnchangedResult)

	for _, cream := range addedCreams {
		creamMessage := fmt.Sprintf("• %s: %s %s\n", cream.Name, italic(cream.Description), cream.Details())
		if _, err := s.WriteString(creamMessage); err != nil {
			log.Println(err)
		}
	}

	writeCategorySubtitle(s, removedCreams, category, "removed", useCache, outputUnchangedResult)

	if useCache {
		for _, cream := range removedCreams {
			creamMessage := fmt.Sprintf("• %s\n", cream.Name)
			if _, err := s.WriteString(creamMessage); err != nil {
				log.Println(err)
			}
		}
	}
}

func writeCategorySubtitle(s io.StringWriter, creams []IceCream, category string, diffSide string, useCache bool, outputUnchangedResult bool) {
	if !useCache {
		return
	}

	if len(creams) > 0 {
		if _, err := s.WriteString("Flavors added since the last run:\n"); err != nil {
			log.Println(err)
		}

		return
	}

	noChangesMessage := fmt.Sprintf("No new flavors were %s since the last run.", diffSide)

	if outputUnchangedResult {
		if _, err := s.WriteString(italic(noChangesMessage) + "\n"); err != nil {
			log.Println(err)
		}
	} else {
		log.Println(category + ": " + noChangesMessage)
	}
}

// Close closes an io.Closer and handles the possible Close error.
func Close(c io.Closer) {
	err := c.Close()
	if err != nil {
		log.Fatal(err)
	}
}
