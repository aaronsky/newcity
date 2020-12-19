package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gocolly/colly/v2"
)

// nolint: gochecknoglobals
var (
	newCityCambridgeMenuAddress = fmt.Sprintf("https://%s/cambridge-menu", newCityMicrocreameryHostname)
)

// IceCream is a flavor type.
type IceCream struct {
	Name        string
	Description string
	RawDetails  []string
}

// IceCreams is a map of New City section titles to flavor lists.
type IceCreams map[string][]IceCream

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

func (c *IceCreams) Write(filename string) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0600)
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

// Close closes an io.Closer and handles the possible Close error.
func Close(c io.Closer) {
	err := c.Close()
	if err != nil {
		log.Fatal(err)
	}
}
