package main

import (
	"context"
	"os"

	"github.com/bobg/modver/v3"
)

func doGit(ctx context.Context, olderURL, olderRev, newerURL, newerRev string, _ []string) error {
	res, err := modver.CompareGit2(ctx, olderURL, olderRev, newerURL, newerRev)
	if err != nil {
		return err
	}

	var opts options // xxx
	doShowResult(os.Stdout, res, opts)

	return nil
}
