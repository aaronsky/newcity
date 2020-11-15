package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/gocolly/colly/v2"
)

const (
	newCityMicrocreameryHostname = "newcitymicrocreamery.com"
)

var (
	newCityCambridgeMenuAddress = fmt.Sprintf("https://%s/cambridge-menu", newCityMicrocreameryHostname)
)

type iceCream struct {
	Name        string
	Description string
	Details     []string
}

func main() {
	c := colly.NewCollector(
		colly.AllowedDomains(newCityMicrocreameryHostname),
	)

	iceCreams := map[string][]iceCream{}

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	c.OnHTML(".menu-section", func(section *colly.HTMLElement) {
		category := section.ChildText(".menu-section-title")
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
		log.Fatal(err)
	}

	c.Wait()

	fmt.Println(iceCreams)
}
