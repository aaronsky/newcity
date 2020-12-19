package main

import (
	"flag"
	"fmt"
	"log"
)

const (
	newCityMicrocreameryHostname = "newcitymicrocreamery.com"
)

// nolint: gochecknoglobals
var (
	botTokenFlag              = flag.String("token", "", "Bot token for the Discord API")
	channelIDFlag             = flag.Int64("channel_id", 0, "Channel ID the bot should post to")
	onlyOriginalsFlag         = flag.Bool("only_originals", false, "Filter only New City Originals ice cream flavors")
	useCacheFlag              = flag.Bool("cache", false, "Use cache to determine changed flavors")
	outputUnchangedResultFlag = flag.Bool("print_unchanged", false, "Will output even if the flavors have not changed (does nothing if -cache is not set)")
	readCachePathFlag         = flag.String("read_cache_path", "newcity.json", "Path to read cache from (does nothing if -cache is not set)")
	writeCachePathFlag        = flag.String("write_cache_path", "newcity.json", "Path to write cache to (does nothing if -cache is not set)")
	dryRunFlag                = flag.Bool("dry_run", false, "Dry-run")
)

func main() {
	flag.Parse()

	useCache := *useCacheFlag

	var (
		cachedIceCreams IceCreams
		err             error
	)

	if useCache {
		readCachePath := *readCachePathFlag

		cachedIceCreams, err = NewIceCreamsFromFile(readCachePath)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("Discovered valid cache")
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
		writeCachePath := *writeCachePathFlag
		if err := iceCreams.Write(writeCachePath); err != nil {
			log.Fatal(err)
		}
	}
}
