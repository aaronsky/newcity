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
	errNoPath  = errors.New("no path provided using the -path flag")
	errNoToken = errors.New("no value provided for the GITHUB_TOKEN environment variable")
)

// nolint: gochecknoglobals
var (
	key         = flag.String("key", "", "Cache key to inspect")
	path        = flag.String("path", "", "Path to write cached file to")
	owner       = flag.String("owner", "aaronsky", "Github user or org name that repo belongs to")
	repo        = flag.String("repo", "newcity", "Repository name to check for artifacts")
	githubToken = os.Getenv("GITHUB_TOKEN")
)

func main() {
	flag.Parse()

	if err := validateInputs(); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	client := NewGithub(ctx, githubToken)

	artifact, err := client.ArtifactMatching(ctx, *owner, *repo, *key)
	if err != nil {
		log.Fatal(err)
	}

	if err = client.WriteArtifactToPath(ctx, *owner, *repo, artifact, *path); err != nil {
		log.Fatal(err)
	}
}

func validateInputs() error {
	if *key == "" {
		return errNoKey
	}

	if *path == "" {
		return errNoPath
	}

	if githubToken == "" {
		return errNoToken
	}

	return nil
}
