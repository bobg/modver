package main

import (
	"context"
	"fmt"
	"os"

	"github.com/bobg/errors"

	"github.com/bobg/modver/v3"
	"github.com/bobg/modver/v3/internal"
)

func doAction(ctx context.Context, _ []string) error {
	os.Setenv("GOROOT", "/usr/local/go") // Work around some Docker weirdness.

	prURL := os.Getenv("INPUT_PULL_REQUEST_URL")

	host, owner, reponame, prnum, err := internal.ParsePR(prURL)
	if err != nil {
		return errors.Wrapf(err, "parsing pull request URL %s", prURL)
	}

	token := os.Getenv("INPUT_GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("no GitHub token in the environment variable INPUT_GITHUB_TOKEN")
	}

	gh, err := internal.NewClient(ctx, host, token)
	if err != nil {
		return errors.Wrap(err, "creating GitHub client")
	}

	result, err := internal.PR(ctx, gh, owner, reponame, prnum)
	if err != nil {
		return errors.Wrap(err, "running comparison")
	}

	modver.Pretty(os.Stdout, result)

	return nil
}
