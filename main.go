package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/session"
	"github.com/gocolly/colly/v2"
)

const (
	newCityMicrocreameryHostname     = "newcitymicrocreamery.com"
	newCityOriginalsSectionTitleText = "New City Originals"
	headerMessage                    = `**Here are today's New City flavors :icecream:**`
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

	botTokenFlag  = flag.String("token", "", "Bot token for the Discord API")
	channelIDFlag = flag.Int64("channel_id", 0, "Channel ID the bot should post to")
	onlyNewFlag   = flag.Bool("only_new", false, "Filter only new ice creams")
	dryRunFlag    = flag.Bool("dry_run", false, "Dry-run")
)

var (
	// ErrNoDiscordBotToken happens when a token is not provided via the -bot_token flag or the BOT_TOKEN environment variable.
	ErrNoDiscordBotToken = errors.New("no Discord bot token provided")
	// ErrNoDiscordChannelID happens when a channel ID is not provided via the -channel_id flag, or it is not a valid int64.
	ErrNoDiscordChannelID = errors.New("no Discord channel ID provided")
)

type iceCream struct {
	Name        string
	Description string
	RawDetails  []string
}

// IceCreams is a map of New City section titles to flavor lists.
type IceCreams map[string][]iceCream

func main() {
	flag.Parse()

	dryRun := *dryRunFlag

	// fetch ice creams
	iceCreams, err := NewIceCreams()
	if err != nil {
		log.Fatal(err)
	}

	messages := iceCreams.Messages()

	// auth with Discord
	if dryRun {
		for _, message := range messages {
			fmt.Println(message)
		}

		return
	}

	// If not in dry-run mode, post to Discord
	if err := PostToDiscord(messages...); err != nil {
		log.Fatal(err)
	}
}

// NewIceCreams creates a new iceCreams instance using data scraped from the New City website.
func NewIceCreams() (IceCreams, error) {
	c := colly.NewCollector(
		colly.AllowedDomains(newCityMicrocreameryHostname),
	)

	iceCreams := IceCreams{}
	onlyNew := *onlyNewFlag

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	c.OnHTML(".menu-section", func(section *colly.HTMLElement) {
		category := section.ChildText(".menu-section-title")
		if onlyNew && category != newCityOriginalsSectionTitleText {
			return
		}

		iceCreamsInCategory := []iceCream{}

		section.ForEach(".menu-item", func(i int, item *colly.HTMLElement) {
			iceCream := iceCream{}
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

// PostToDiscord posts the given messages to Discord.
func PostToDiscord(messages ...string) error {
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

		fmt.Println("SENT:", m.ID)
	}

	return nil
}

// Messages splits an IceCreams instance into a list of messages.
func (c IceCreams) Messages() []string {
	messages := []string{}

	messages = append(messages, headerMessage)

	for category, creams := range c {
		s := strings.Builder{}

		if len(c) > 1 {
			s.WriteString(fmt.Sprintf("**%s**\n", category))
		}

		for _, cream := range creams {
			s.WriteString(fmt.Sprintf("• %s: *%s* %s\n", cream.Name, cream.Description, cream.Details()))
		}

		messages = append(messages, s.String())
	}

	return messages
}

func (c iceCream) Details() string {
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

// Close closes an io.Closer and handles the possible Close error.
func Close(c io.Closer) {
	err := c.Close()
	if err != nil {
		log.Fatal(err)
	}
}
