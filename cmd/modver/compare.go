package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v50/github"
	"github.com/pkg/errors"

	"github.com/bobg/modver/v2"
)

func doCompare(ctx context.Context, opts options) (modver.Result, error) {
	if opts.pr != "" {
		owner, reponame, prnum, err := parsePR(opts.pr)
		if err != nil {
			return modver.None, errors.Wrap(err, "parsing pull-request URL")
		}
		if opts.ghtoken == "" {
			return modver.None, fmt.Errorf("usage: %s -pr URL [-token TOKEN]", os.Args[0])
		}
		gh := github.NewTokenClient(ctx, opts.ghtoken)
		return doPR(ctx, gh, owner, reponame, prnum)
	}

	if opts.gitRepo != "" {
		if len(opts.args) != 2 {
			return nil, fmt.Errorf("usage: %s -git REPO [-gitcmd GIT_COMMAND] [-q | -pretty] [-v1 OLDERVERSION -v2 NEWERVERSION | -versions] OLDERREV NEWERREV", os.Args[0])
		}

		callback := modver.CompareDirs
		if opts.versions {
			callback = getTags(&opts.v1, &opts.v2, opts.args[0], opts.args[1])
		}

		return modver.CompareGitWith(ctx, opts.gitRepo, opts.args[0], opts.args[1], callback)
	}
	if len(opts.args) != 2 {
		return nil, fmt.Errorf("usage: %s [-q | -pretty] [-v1 OLDERVERSION -v2 NEWERVERSION] OLDERDIR NEWERDIR", os.Args[0])
	}
	return modver.CompareDirs(opts.args[0], opts.args[1])
}
