package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/bobg/modver/v2"
	"github.com/bobg/modver/v2/internal"
)

func doCompare(ctx context.Context, opts options) (modver.Result, error) {
	if opts.pr != "" {
		host, owner, reponame, prnum, err := internal.ParsePR(opts.pr)
		if err != nil {
			return modver.None, errors.Wrap(err, "parsing pull-request URL")
		}
		if opts.ghtoken == "" {
			return modver.None, fmt.Errorf("usage: %s -pr URL [-token TOKEN]", os.Args[0])
		}
		gh, err := internal.NewClient(ctx, host, opts.ghtoken)
		if err != nil {
			return modver.None, errors.Wrap(err, "creating GitHub client")
		}
		return internal.PR(ctx, gh, owner, reponame, prnum)
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
