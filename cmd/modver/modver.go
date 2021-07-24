// Command modver compares two versions of the same Go packages
// and tells whether a Major, Minor, or Patchlevel version bump
// (or None)
// is needed to go from one to the other.
//
// Usage:
//   modver [-git REPO] [-q] [-v1 OLDERVERSION -v2 NEWERVERSION] OLDER NEWER
//
// With `-git REPO`,
// where REPO is the path to a Git repository,
// OLDER and NEWER are two commits in the repository
// (e.g. hexadecimal SHA strings or "HEAD", etc)
// containing the older and newer versions of a Go module.
// Without the -git flag,
// OLDER and NEWER are two directories containing the older and newer versions of a Go module.
//
// With -v1 and -v2,
// modver checks whether the change from OLDERVERSION to NEWERVERSION
// (two version strings) is adequate for the differences detected between OLDER and NEWER.
// Output is either "OK" or "ERR"
// (followed by a description).
// In quiet mode (-q),
// there is no output,
// and the exit status is either 0 (OK) or 1 (error).
//
// Without -v1 and -v2,
// output is a string describing the minimum version-number change required.
// In quiet mode (-q),
// there is no output,
// and the exit status is 0, 1, 2, 3, or 4
// for None, Patchlevel, Minor, Major, and error.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/bobg/modver"
)

const errorStatus = 4

func main() {
	var (
		gitRepo = flag.String("git", "", "Git repo URL")
		quiet   = flag.Bool("q", false, "quiet mode: prints no output, exits with status 0, 1, 2, 3, or 4 to mean None, Patchlevel, Minor, Major, or error")
		v1      = flag.String("v1", "", "version string of older version; with -v2 changes output to OK (exit status 0) for adequate version-number change, ERR (exit status 1) for inadequate")
		v2      = flag.String("v2", "", "version string of newer version")
	)
	flag.Parse()

	if *v1 != "" && *v2 != "" {
		if !strings.HasPrefix(*v1, "v") {
			*v1 = "v" + *v1
		}
		if !strings.HasPrefix(*v2, "v") {
			*v2 = "v" + *v2
		}
		if !semver.IsValid(*v1) {
			fmt.Printf("Not a valid version string: %s\n", *v1)
			os.Exit(errorStatus)
		}
		if !semver.IsValid(*v2) {
			fmt.Printf("Not a valid version string: %s\n", *v2)
			os.Exit(errorStatus)
		}
	}

	var (
		res modver.Result
		err error
	)

	if *gitRepo != "" {
		if flag.NArg() != 2 {
			fmt.Printf("Usage: %s -git [-q] [-v1 OLDERVERSION -v2 NEWERVERSION] OLDERSHA NEWERSHA\n", os.Args[0])
			os.Exit(errorStatus)
		}
		res, err = modver.CompareGit(context.Background(), *gitRepo, flag.Arg(0), flag.Arg(1))
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(errorStatus)
		}
	} else {
		if flag.NArg() != 2 {
			fmt.Printf("Usage: %s [-q] [-v1 OLDERVERSION -v2 NEWERVERSION] OLDERDIR NEWERDIR\n", os.Args[0])
			os.Exit(errorStatus)
		}
		res, err = modver.CompareDirs(flag.Arg(0), flag.Arg(1))
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(errorStatus)
		}
	}

	if *v1 != "" && *v2 != "" {
		var ok bool

		cmp := semver.Compare(*v1, *v2)
		switch res.Code() {
		case modver.None:
			ok = cmp <= 0 // v1 <= v2

		case modver.Patchlevel:
			ok = cmp < 0 // v1 < v2

		case modver.Minor:
			var (
				min1 = semver.MajorMinor(*v1)
				min2 = semver.MajorMinor(*v2)
			)
			ok = semver.Compare(min1, min2) < 0 // min1 < min2

		case modver.Major:
			var (
				maj1 = semver.Major(*v1)
				maj2 = semver.Major(*v2)
			)
			ok = semver.Compare(maj1, maj2) < 0 // maj1 < maj2
		}

		if *quiet && ok {
			os.Exit(0)
		}
		if *quiet {
			os.Exit(1)
		}
		if ok {
			fmt.Println("OK")
		} else {
			fmt.Printf("ERR %s\n", res)
		}
	} else if *quiet {
		os.Exit(int(res.Code()))
	} else {
		fmt.Println(res)
	}
}
