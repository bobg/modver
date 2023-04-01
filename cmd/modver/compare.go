package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v50/github"
	"github.com/pkg/errors"

	"github.com/bobg/modver/v2"
	"github.com/bobg/modver/v2/internal"
)

func doCompare(ctx context.Context, opts options) (modver.Result, error) {
	return doCompareHelper(ctx, opts, internal.NewClient, internal.PR, modver.CompareGitWith, modver.CompareDirs)
}

type (
	newClientType      = func(ctx context.Context, host, token string) (*github.Client, error)
	prType             = func(ctx context.Context, gh *github.Client, owner, reponame string, prnum int) (modver.Result, error)
	compareGitWithType = func(ctx context.Context, repoURL, olderRev, newerRev string, f func(older, newer string) (modver.Result, error)) (modver.Result, error)
	compareDirsType    = func(older, newer string) (modver.Result, error)
)

func doCompareHelper(ctx context.Context, opts options, newClient newClientType, pr prType, compareGitWith compareGitWithType, compareDirs compareDirsType) (modver.Result, error) {
	if opts.pr != "" {
		host, owner, reponame, prnum, err := internal.ParsePR(opts.pr)
		if err != nil {
			return modver.None, errors.Wrap(err, "parsing pull-request URL")
		}
		if opts.ghtoken == "" {
			return modver.None, fmt.Errorf("usage: %s -pr URL [-token TOKEN]", os.Args[0])
		}
		gh, err := newClient(ctx, host, opts.ghtoken)
		if err != nil {
			return modver.None, errors.Wrap(err, "creating GitHub client")
		}
		return pr(ctx, gh, owner, reponame, prnum)
	}

	if opts.gitRepo != "" {
		if len(opts.args) != 2 {
			return nil, fmt.Errorf("usage: %s -git REPO [-gitcmd GIT_COMMAND] [-q | -pretty] [-v1 OLDERVERSION -v2 NEWERVERSION | -versions] OLDERREV NEWERREV", os.Args[0])
		}

		callback := modver.CompareDirs
		if opts.versions {
			callback = getTags(&opts.v1, &opts.v2, opts.args[0], opts.args[1])
		}

		return compareGitWith(ctx, opts.gitRepo, opts.args[0], opts.args[1], callback)
	}
	if len(opts.args) != 2 {
		return nil, fmt.Errorf("usage: %s [-q | -pretty] [-v1 OLDERVERSION -v2 NEWERVERSION] OLDERDIR NEWERDIR", os.Args[0])
	}
	return compareDirs(opts.args[0], opts.args[1])
}
