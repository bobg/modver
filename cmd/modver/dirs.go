package main

import (
	"context"
	"os"

	"github.com/bobg/modver/v3"
)

func doDirs(ctx context.Context, older, newer string, _ []string) error {
	res, err := modver.CompareDirs(older, newer)
	if err != nil {
		return err
	}

	var opts options // xxx
	doShowResult(os.Stdout, res, opts)

	return nil
}
