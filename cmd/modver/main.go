// Command modver compares two versions of the same Go packages
// and tells whether a Major, Minor, or Patchlevel version bump
// (or None)
// is needed to go from one to the other.
//
// Usage:
//
//	modver -pr URL [-token GITHUB_TOKEN]
//	modver -git REPO [-gitcmd GIT_COMMAND] [-q | -pretty] [-v1 OLDERVERSION -v2 NEWERVERSION | -versions] OLDERREV NEWERREV
//	modver [-q | -pretty] [-v1 OLDERVERSION -v2 NEWERVERSION] OLDERDIR NEWERDIR
//
// With `-pr URL`,
// the URL must be that of a github.com pull request
// (having the form https://HOST/OWNER/REPO/pull/NUMBER).
// The environment variable GITHUB_TOKEN must contain a valid GitHub access token,
// or else one must be supplied on the command line with -token.
// In this mode,
// modver compares the base of the pull-request branch with the head
// and produces a report that it adds as a comment to the pull request.
//
// With `-git REPO`,
// where REPO is the path to a Git repository,
// OLDER and NEWER are two revisions in the repository
// (e.g. hexadecimal SHA strings or "HEAD", etc)
// containing the older and newer versions of a Go module.
// Without the -git flag,
// OLDER and NEWER are two directories containing the older and newer versions of a Go module.
//
// With `-gitcmd GIT_COMMAND`,
// modver uses the given command for Git operations.
// This is "git" by default.
// If the command does not exist or is not found in your PATH,
// modver falls back to using the go-git library.
//
// With -v1 and -v2,
// modver checks whether the change from OLDERVERSION to NEWERVERSION
// (two version strings)
// is adequate for the differences detected between OLDER and NEWER.
// Output is either "OK" or "ERR"
// (followed by a description)
// and the exit code is 0 for OK and 1 for ERR.
// In quiet mode (-q),
// there is no output.
// With -git REPO and -versions instead of -v1 and -v2,
// the values for -v1 and -v2 are determined by querying the repo at the given revisions.
//
// Without -v1 and -v2
// (or -versions),
// output is a string describing the minimum version-number change required.
// In quiet mode (-q),
// there is no output,
// and the exit status is 0, 1, 2, 3, or 4
// for None, Patchlevel, Minor, Major, and error.
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/bobg/subcmd/v2"
	"golang.org/x/mod/semver"

	"github.com/bobg/modver/v3"
)

const errorStatus = 4

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		ctx = context.Background()

		c maincmd
	)

	return subcmd.Run(ctx, c, os.Args[1:])
}

type maincmd struct {
}

func (maincmd) Subcmds() subcmd.Map {
	return subcmd.Commands(
		"action", doAction, "run as a GitHub Action", nil,
		"dirs", doDirs, "compare the Go modules in two directory trees", subcmd.Params(
			"older", subcmd.String, "", `path to the "older" directory`,
			"newer", subcmd.String, "", `path to the "newer" directory`,
		),
		"git", doGit, "compare the Go modules in two versions of a Git repository", subcmd.Params(),
		"pr", doPR, "compare the Go modules in the base and head of a GitHub pull request", subcmd.Params(),
	)
}

// 	opts, err := parseArgs()
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "Error parsing args: %s\n", err)
// 		os.Exit(errorStatus)
// 	}

// 	ctx := context.Background()
// 	if opts.gitCmd != "" {
// 		ctx = modver.WithGit(ctx, opts.gitCmd)
// 	}

// 	res, err := doCompare(ctx, opts)
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "Error in comparing: %s\n", err)
// 		os.Exit(errorStatus)
// 	}

// 	exitCode := doShowResult(os.Stdout, res, opts)
// 	os.Exit(exitCode)
// }

func doShowResult(out io.Writer, res modver.Result, opts options) int {
	if opts.v1 != "" && opts.v2 != "" {
		var ok bool

		cmp := semver.Compare(opts.v1, opts.v2)
		switch res.Code() {
		case modver.None:
			ok = cmp <= 0 // v1 <= v2

		case modver.Patchlevel:
			ok = cmp < 0 // v1 < v2

		case modver.Minor:
			var (
				min1 = semver.MajorMinor(opts.v1)
				min2 = semver.MajorMinor(opts.v2)
			)
			ok = semver.Compare(min1, min2) < 0 // min1 < min2

		case modver.Major:
			var (
				maj1 = semver.Major(opts.v1)
				maj2 = semver.Major(opts.v2)
			)
			ok = semver.Compare(maj1, maj2) < 0 // maj1 < maj2
		}

		if ok {
			if !opts.quiet {
				if opts.versions {
					fmt.Fprintf(out, "OK using versions %s and %s: %s\n", opts.v1, opts.v2, res)
				} else {
					fmt.Fprintf(out, "OK %s\n", res)
				}
			}
			return 0
		}
		if !opts.quiet {
			if opts.versions {
				fmt.Fprintf(out, "ERR using versions %s and %s: %s\n", opts.v1, opts.v2, res)
			} else {
				fmt.Fprintf(out, "ERR %s\n", res)
			}
		}
		return 1
	}

	if opts.quiet {
		return int(res.Code())
	}

	if opts.pretty {
		modver.Pretty(out, res)
	} else {
		fmt.Fprintln(out, res)
	}

	return 0
}
