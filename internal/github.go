package internal

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/bobg/errors"
	"github.com/google/go-github/v50/github"
	"golang.org/x/oauth2"
)

// ParsePR parses a GitHub pull-request URL,
// which should have the form http(s)://HOST/OWNER/REPO/pull/NUMBER.
func ParsePR(pr string) (host, owner, reponame string, prnum int, err error) {
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
	host = u.Host
	owner, reponame = parts[0], parts[1]
	prnum, err = strconv.Atoi(parts[3])
	err = errors.Wrap(err, "parsing number from GitHub pull-request URL")
	return
}

// NewClient creates a new GitHub client talking to the given host and authenticated with the given token.
func NewClient(ctx context.Context, host, token string) (*github.Client, error) {
	if strings.ToLower(host) == "github.com" {
		return github.NewTokenClient(ctx, token), nil
	}
	oClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
	u := "https://" + host
	return github.NewEnterpriseClient(u, u, oClient)
}
