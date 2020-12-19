package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
)

var (
	errNoKey   = errors.New("no key provided using the -key flag")
	errNoToken = errors.New("no value provided for the GITHUB_TOKEN environment variable")
)

// nolint: gochecknoglobals
var (
	key         = flag.String("key", "", "Cache key to inspect")
	path        = flag.String("path", "newcity.json", "File to extract from cache into current directory")
	owner       = flag.String("owner", "aaronsky", "Github user or org name that repo belongs to")
	repo        = flag.String("repo", "newcity", "Repository name to check for artifacts")
	strict      = flag.Bool("strict", false, "Fail if an artifact is not found")
	githubToken = os.Getenv("GITHUB_TOKEN")
)

func main() {
	flag.Parse()

	if err := validateInputs(); err != nil {
		log.Fatal(err)
	}

	var strictLog func(...interface{})
	if *strict {
		strictLog = log.Fatal
	} else {
		strictLog = func(v ...interface{}) {
			log.Println(v...)
			os.Exit(0)
		}
	}

	ctx := context.Background()
	client := NewGithub(ctx, githubToken)

	artifact, err := client.ArtifactMatching(ctx, *owner, *repo, *key)
	if err != nil {
		strictLog(err)
	}

	err = client.WriteArtifactsWithName(ctx, *owner, *repo, artifact, *path)
	if err != nil {
		strictLog(err)
	}
}

func validateInputs() error {
	if *key == "" {
		return errNoKey
	}

	if githubToken == "" {
		return errNoToken
	}

	return nil
}
