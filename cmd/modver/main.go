// Command modver compares two versions of the same Go packages
// and tells whether a Major, Minor, or Patchlevel version bump
// (or None)
// is needed to go from one to the other.
//
// Usage:
//
//		modver -git REPO [-gitcmd GIT_COMMAND] [-q | -pretty] [-v1 OLDERVERSION -v2 NEWERVERSION | -versions] OLDERREV NEWERREV
//		modver [-q | -pretty] [-v1 OLDERVERSION -v2 NEWERVERSION] OLDERDIR NEWERDIR
//	 modver -pr URL [-q | -pretty]
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
	"os"

	"golang.org/x/mod/semver"

	"github.com/bobg/modver/v2"
)

const errorStatus = 4

func main() {
	opts, err := parseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing args: %s\n", err)
		os.Exit(errorStatus)
	}

	ctx := context.Background()
	if opts.gitCmd != "" {
		ctx = modver.WithGit(ctx, opts.gitCmd)
	}

	res, err := doCompare(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in comparing: %s\n", err)
		os.Exit(errorStatus)
	}

	doShowResultExit(res, opts)
}

func doShowResultExit(res modver.Result, opts options) {
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
					fmt.Printf("OK using versions %s and %s: %s\n", opts.v1, opts.v2, res)
				} else {
					fmt.Printf("OK %s\n", res)
				}
			}
			os.Exit(0)
		}
		if !opts.quiet {
			if opts.versions {
				fmt.Printf("ERR using versions %s and %s: %s\n", opts.v1, opts.v2, res)
			} else {
				fmt.Printf("ERR %s\n", res)
			}
		}
		os.Exit(1)
	}

	if opts.quiet {
		os.Exit(int(res.Code()))
	}

	if opts.pretty {
		modver.Pretty(os.Stdout, res)
	} else {
		fmt.Println(res)
	}
}
