package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/google/go-github/v50/github"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"

	"github.com/bobg/modver/v2"
)

func doCompare(ctx context.Context, opts options) (modver.Result, error) {
	if opts.pr != "" {
		owner, reponame, prnum, err := parsePR(opts.pr)
		if err != nil {
			return modver.None, errors.Wrap(err, "parsing pull-request URL")
		}
		if opts.ghtoken == "" {
			return modver.None, fmt.Errorf("usage: %s -pr URL -token TOKEN [-q | -pretty]", os.Args[0])
		}
		gh := github.NewTokenClient(ctx, opts.ghtoken)
		return doPR(ctx, gh, owner, reponame, prnum)
	}

	if opts.gitRepo != "" {
		if flag.NArg() != 2 {
			return nil, fmt.Errorf("usage: %s -git REPO [-gitcmd GIT_COMMAND] [-q | -pretty] [-v1 OLDERVERSION -v2 NEWERVERSION | -versions] OLDERREV NEWERREV", os.Args[0])
		}

		callback := modver.CompareDirs
		if opts.versions {
			callback = getTags(&opts.v1, &opts.v2, flag.Arg(0), flag.Arg(1))
		}

		return modver.CompareGitWith(ctx, opts.gitRepo, flag.Arg(0), flag.Arg(1), callback)
	}
	if flag.NArg() != 2 {
		return nil, fmt.Errorf("usage: %s [-q | -pretty] [-v1 OLDERVERSION -v2 NEWERVERSION] OLDERDIR NEWERDIR", os.Args[0])
	}
	return modver.CompareDirs(flag.Arg(0), flag.Arg(1))
}

func getTags(v1, v2 *string, olderRev, newerRev string) func(older, newer string) (modver.Result, error) {
	return func(older, newer string) (modver.Result, error) {
		tag, err := getTag(older, olderRev)
		if err != nil {
			return modver.None, fmt.Errorf("getting tag from %s: %w", older, err)
		}
		*v1 = tag

		tag, err = getTag(newer, newerRev)
		if err != nil {
			return modver.None, fmt.Errorf("getting tag from %s: %w", newer, err)
		}
		*v2 = tag

		return modver.CompareDirs(older, newer)
	}
}

func getTag(dir, rev string) (string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", fmt.Errorf("opening %s: %w", dir, err)
	}
	tags, err := repo.Tags()
	if err != nil {
		return "", fmt.Errorf("getting tags in %s: %w", dir, err)
	}
	hash, err := repo.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		return "", fmt.Errorf(`resolving revision "%s" in %s: %w`, rev, dir, err)
	}
	repoCommit, err := object.GetCommit(repo.Storer, *hash)
	if err != nil {
		return "", fmt.Errorf("getting commit at %s: %w", rev, err)
	}

	return getTagHelper(dir, rev, repo.Storer, tags, hash, repoCommit)
}

func getTagHelper(dir, rev string, s storer.EncodedObjectStorer, tags storer.ReferenceIter, hash *plumbing.Hash, repoCommit *object.Commit) (string, error) {
	var result string

OUTER:
	for {
		tref, err := tags.Next()
		if errors.Is(err, io.EOF) {
			return result, nil
		}
		if err != nil {
			return "", fmt.Errorf("iterating over tags in %s: %w", dir, err)
		}
		tag := strings.TrimPrefix(string(tref.Name()), "refs/tags/")
		if !semver.IsValid(tag) {
			continue
		}
		tagCommit, err := object.GetCommit(s, tref.Hash())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: getting commit for tag %s: %s", tref.Name(), err)
			continue
		}
		if tagCommit.Hash != *hash {
			bases, err := repoCommit.MergeBase(tagCommit)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: getting merge base of %s and %s: %s", rev, tag, err)
				continue
			}
		INNER:
			for _, base := range bases {
				switch base.Hash {
				case *hash:
					// This tag comes later than the checked-out commit.
					continue OUTER
				case tagCommit.Hash:
					// The checked-out commit comes later than the tag.
					break INNER
				}
			}
		}
		if result == "" || semver.Compare(result, tag) < 0 { // result < tag
			result = tag
		}
	}
}
