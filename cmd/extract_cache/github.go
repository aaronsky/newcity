package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
)

var (
	errNoMatchingArtifact = errors.New("could not find a matching artifact")
)

// Client wraps a GitHub client.
type Client struct {
	client *github.Client
}

// NewGithub creates a new GitHub client using the given PAT.
func NewGithub(ctx context.Context, token string) Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	return Client{client: github.NewClient(tc)}
}

// ArtifactMatching returns the first artifact model that matches the key, or an error if either a request failed or none was found.
func (c *Client) ArtifactMatching(ctx context.Context, owner string, repo string, key string) (*github.Artifact, error) {
	artifacts, _, err := c.client.Actions.ListArtifacts(ctx, owner, repo, nil)
	if err != nil {
		return nil, err
	}

	if artifacts.GetTotalCount() == 0 {
		return nil, fmt.Errorf("%s: %w", key, errNoMatchingArtifact)
	}

	for _, art := range artifacts.Artifacts {
		if art.GetName() == key {
			return art, nil
		}
	}

	return nil, fmt.Errorf("%s: %w", key, errNoMatchingArtifact)
}

// WriteArtifactToPath downloads the file contents for the artfifact model to the given file path.
func (c *Client) WriteArtifactToPath(ctx context.Context, owner string, repo string, artifact *github.Artifact, file string) error {
	u, _, err := c.client.Actions.DownloadArtifact(ctx, owner, repo, artifact.GetID(), true)
	if err != nil {
		return err
	}

	outName := filepath.Join(os.TempDir(), owner, repo, strconv.FormatInt(artifact.GetID(), 10))

	out, err := os.Create(outName)
	if err != nil {
		return err
	}

	defer Close(out)

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer Close(resp.Body)

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	fmt.Println(outName)

	return nil
}

// Close closes an io.Closer and handles the possible Close error.
func Close(c io.Closer) {
	err := c.Close()
	if err != nil {
		log.Fatal(err)
	}
}
