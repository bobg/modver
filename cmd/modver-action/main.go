package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/bobg/errors"
	"github.com/bobg/prcomment"
	"github.com/google/go-github/v62/github"

	"github.com/bobg/modver/v2"
	"github.com/bobg/modver/v2/internal"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %s", err)
	}
}

func run() error {
	os.Setenv("GOROOT", "/usr/local/go") // Work around some Docker weirdness.

	prURL := os.Getenv("INPUT_PULL_REQUEST_URL")
	host, owner, reponame, prnum, err := prcomment.ParsePR(prURL)
	if err != nil {
		return errors.Wrapf(err, "parsing pull-request URL %s", prURL)
	}
	token := os.Getenv("INPUT_GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("no GitHub token in the environment variable INPUT_GITHUB_TOKEN")
	}

	hc := new(http.Client)
	gh := github.NewClient(hc).WithAuthToken(token)
	if !strings.EqualFold(host, "github.com") {
		var (
			u   = "https://" + host
			err error
		)
		gh, err = gh.WithEnterpriseURLs(u, u)
		if err != nil {
			return errors.Wrap(err, "setting enterprise URLs")
		}
	}

	modverResult, err := internal.PR(context.Background(), gh, owner, reponame, prnum)
	if err != nil {
		return errors.Wrap(err, "running modver analysis")
	}

	modver.Pretty(os.Stdout, modverResult)
	return nil
}
