package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/mod/semver"
)

type options struct {
	gitRepo, gitCmd, ghtoken, v1, v2, pr string
	quiet, pretty, versions              bool
}

func parseArgs() (opts options, err error) {
	flag.BoolVar(&opts.pretty, "pretty", false, "result is shown in a pretty format with (possibly) multiple lines and indentation")
	flag.BoolVar(&opts.quiet, "q", false, "quiet mode: prints no output, exits with status 0, 1, 2, 3, or 4 to mean None, Patchlevel, Minor, Major, or error")
	flag.BoolVar(&opts.versions, "versions", false, "with -git, compute values for -v1 and -v2 from the Git repository")
	flag.StringVar(&opts.ghtoken, "token", os.Getenv("GITHUB_TOKEN"), "GitHub access token")
	flag.StringVar(&opts.gitCmd, "gitcmd", "git", "use this command for git operations, if found; otherwise use the go-git library")
	flag.StringVar(&opts.gitRepo, "git", "", "Git repo URL")
	flag.StringVar(&opts.pr, "pr", "", "URL of GitHub pull request")
	flag.StringVar(&opts.v1, "v1", "", "version string of older version; with -v2 changes output to OK (exit status 0) for adequate version-number change, ERR (exit status 1) for inadequate")
	flag.StringVar(&opts.v2, "v2", "", "version string of newer version")
	flag.Parse()

	if opts.pr != "" {
		if opts.gitRepo != "" {
			return opts, fmt.Errorf("do not specify -git with -pr")
		}
		if opts.v1 != "" || opts.v2 != "" || opts.versions {
			return opts, fmt.Errorf("do not specify -v1, -v2, or -versions with -pr")
		}
	}

	if opts.v1 != "" && opts.v2 != "" {
		if !strings.HasPrefix(opts.v1, "v") {
			opts.v1 = "v" + opts.v1
		}
		if !strings.HasPrefix(opts.v2, "v") {
			opts.v2 = "v" + opts.v2
		}
		if !semver.IsValid(opts.v1) {
			return opts, fmt.Errorf("not a valid version string: %s", opts.v1)
		}
		if !semver.IsValid(opts.v2) {
			return opts, fmt.Errorf("not a valid version string: %s", opts.v2)
		}
	}

	return opts, nil
}
