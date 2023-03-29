package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/mod/semver"
)

type options struct {
	gitRepo, gitCmd, ghtoken, v1, v2, pr string
	quiet, pretty, versions              bool
	args                                 []string
}

func parseArgs() (options, error) {
	return parseArgsHelper(os.Args[1:])
}

func parseArgsHelper(args []string) (opts options, err error) {
	var fs flag.FlagSet

	fs.BoolVar(&opts.pretty, "pretty", false, "result is shown in a pretty format with (possibly) multiple lines and indentation")
	fs.BoolVar(&opts.quiet, "q", false, "quiet mode: prints no output, exits with status 0, 1, 2, 3, or 4 to mean None, Patchlevel, Minor, Major, or error")
	fs.BoolVar(&opts.versions, "versions", false, "with -git, compute values for -v1 and -v2 from the Git repository")
	fs.StringVar(&opts.ghtoken, "token", os.Getenv("GITHUB_TOKEN"), "GitHub access token")
	fs.StringVar(&opts.gitCmd, "gitcmd", "git", "use this command for git operations, if found; otherwise use the go-git library")
	fs.StringVar(&opts.gitRepo, "git", "", "Git repo URL")
	fs.StringVar(&opts.pr, "pr", "", "URL of GitHub pull request")
	fs.StringVar(&opts.v1, "v1", "", "version string of older version; with -v2 changes output to OK (exit status 0) for adequate version-number change, ERR (exit status 1) for inadequate")
	fs.StringVar(&opts.v2, "v2", "", "version string of newer version")
	if err := fs.Parse(args); err != nil {
		return opts, errors.Wrap(err, "parsing args")
	}
	opts.args = fs.Args()

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

func parsePR(pr string) (owner, reponame string, prnum int, err error) {
	u, err := url.Parse(pr)
	if err != nil {
		err = errors.Wrap(err, "parsing GitHub pull-request URL")
		return
	}
	path := strings.TrimLeft(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		err = fmt.Errorf("too few path elements in pull-request URL (got %d, want 4)", len(parts))
		return
	}
	if parts[2] != "pull" {
		err = fmt.Errorf("pull-request URL not in expected format")
		return
	}
	owner, reponame = parts[0], parts[1]
	prnum, err = strconv.Atoi(parts[3])
	err = errors.Wrap(err, "parsing number from GitHub pull-request URL")
	return
}
