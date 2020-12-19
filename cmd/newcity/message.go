package main

import (
	"fmt"
	"io"
	"log"
	"strings"
)

const (
	newCityOriginalsSectionTitleText = "New City Originals"
	headerMessage                    = `Here are today's New City flavors :icecream:`
)

// nolint: gochecknoglobals
var (
	detailsEmojiMap = map[string]string{
		"E": ":egg:",
		"G": ":ear_of_rice:", // okay rice is generally gluten-free but this gets the idea across
		"S": ":seedling:",    // this one is also a stretch – it looks kind of like a soy bean?
		"A": ":tumbler_glass:",
		"N": ":peanuts:",
	}
)

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
		if _, err := s.WriteString(fmt.Sprintf("Flavors %s since the last run:\n", diffSide)); err != nil {
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
