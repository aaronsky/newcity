package main

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
)

const (
	twoMegabytes = 2 * 1024 * 1024
)

var (
	errNoMatchingArtifact = errors.New("could not find a matching artifact")
	errZipSlip            = errors.New("illegal file path")
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

// WriteArtifactsWithName downloads the file contents for the artfifact model to the given file path.
func (c *Client) WriteArtifactsWithName(ctx context.Context, owner string, repo string, artifact *github.Artifact, name string) error {
	u, _, err := c.client.Actions.DownloadArtifact(ctx, owner, repo, artifact.GetID(), true)
	if err != nil {
		return err
	}

	// outName := filepath.Join(os.TempDir(), owner, repo, fmt.Sprintf("%d.zip", artifact.GetID()))
	outName := filepath.Join(fmt.Sprintf("artifact-%d.zip", artifact.GetID()))

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

	files, err := unzip(outName, name)
	if err != nil {
		return err
	}

	fmt.Println("Extracted:\n" + strings.Join(files, "\n"))

	return nil
}

func unzip(source string, name string) ([]string, error) {
	filenames := []string{}

	destination, err := os.Getwd()
	if err != nil {
		return filenames, err
	}

	r, err := zip.OpenReader(source)
	if err != nil {
		return filenames, err
	}

	defer Close(r)

	for _, f := range r.File {
		if f.Name != name {
			continue
		}

		// I think this is protected from ZipSlip.
		fpath := filepath.Join(destination, f.Name) // #nosec
		if !strings.HasPrefix(fpath, filepath.Clean(destination)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: %w", fpath, errZipSlip)
		}

		fmt.Println(fpath, filepath.Clean(destination)+string(os.PathSeparator))

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			continue
		}

		out, err := os.OpenFile(filepath.Clean(fpath), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		limited := io.LimitReader(rc, twoMegabytes)
		_, err = io.Copy(out, limited)

		// Close the file without defer to close before next iteration of loop
		Close(out)
		Close(rc)

		if err != nil {
			return filenames, err
		}
	}

	return filenames, nil
}

// Close closes an io.Closer and handles the possible Close error.
func Close(c io.Closer) {
	err := c.Close()
	if err != nil {
		log.Fatal(err)
	}
}
