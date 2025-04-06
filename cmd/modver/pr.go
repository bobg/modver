package main

import (
	"context"
	"os"

	"github.com/bobg/errors"

	"github.com/bobg/modver/v3/internal"
)

func doPR(ctx context.Context, host, token, owner, reponame string, prnum int, _ []string) error {
	gh, err := internal.NewClient(ctx, host, token)
	if err != nil {
		return errors.Wrap(err, "creating GitHub client")
	}

	res, err := internal.PR(ctx, gh, owner, reponame, prnum)
	if err != nil {
		return err
	}

	var opts options // xxx
	doShowResult(os.Stdout, res, opts)

	return nil
}
