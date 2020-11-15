package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/session"
	"github.com/gocolly/colly/v2"
)

const (
	newCityMicrocreameryHostname = "newcitymicrocreamery.com"
	headerMessage                = `**Here are today's New City flavors :icecream:**`
	footerMessage                = `(e) contains egg     (g) contains gluten    (s) contains soy     (a) contains alcohol     (n) contains nuts`
)

var (
	newCityCambridgeMenuAddress = fmt.Sprintf("https://%s/cambridge-menu", newCityMicrocreameryHostname)

	botTokenFlag  = flag.String("token", "", "Bot token for the Discord API")
	channelIDFlag = flag.Int64("channel_id", 0, "Channel ID the bot should post to")
	onlyNewFlag   = flag.Bool("only_new", false, "Filter only new ice creams")
	dryRunFlag    = flag.Bool("dry_run", false, "Dry-run")
)

type iceCream struct {
	Name        string
	Description string
	Details     []string
}

type iceCreams map[string][]iceCream

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

	token := *botTokenFlag
	if token == "" {
		token = os.Getenv("BOT_TOKEN")
		if token == "" {
			log.Fatal("no Discord bot token provided")
		}
	}

	channelID := *channelIDFlag
	if channelID == 0 {
		log.Fatal("no Discord channel ID provided")
	}

	s, err := session.New("Bot " + token)
	if err != nil {
		log.Fatal(err)
	}

	// get channel to send to
	if err := s.Open(); err != nil {
		log.Fatalln("Failed to connect:", err)
	}
	defer s.Close()

	// create message
	for _, message := range messages {
		m, err := s.SendText(discord.ChannelID(channelID), message)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("SENT:", m.ID)
	}
}

func NewIceCreams() (iceCreams, error) {
	c := colly.NewCollector(
		colly.AllowedDomains(newCityMicrocreameryHostname),
	)

	iceCreams := iceCreams{}
	onlyNew := *onlyNewFlag

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	c.OnHTML(".menu-section", func(section *colly.HTMLElement) {
		category := section.ChildText(".menu-section-title")
		if onlyNew && category != "New City Originals" {
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
				iceCream.Details = strings.Split(details, ",")
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

func (c iceCreams) Messages() []string {
	messages := []string{}

	messages = append(messages, headerMessage)

	for category, creams := range c {
		s := strings.Builder{}
		s.WriteString(fmt.Sprintf("**%s**\n\n", category))
		for _, cream := range creams {
			s.WriteString(fmt.Sprintf("- %s: *%s* (%s)\n", cream.Name, cream.Description, strings.Join(cream.Details, ",")))
		}
		messages = append(messages, s.String())
	}

	if len(messages) > 1 {
		messages = append(messages, footerMessage)
	}

	return messages
}
