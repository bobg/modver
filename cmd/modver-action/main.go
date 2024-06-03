package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/bobg/prcomment"
	"github.com/google/go-github/v62/github"
	"github.com/pkg/errors"

	"github.com/bobg/modver/v2"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %s", err)
	}
}

func run() error {
	os.Setenv("GOROOT", "/usr/local/go") // Work around some Docker weirdness.

	prURL := os.Getenv("INPUT_PULL_REQUEST_URL")
	host, owner, reponame, prnum, err := prcomment.ParsePR(prURL)
	if err != nil {
		return errors.Wrapf(err, "parsing pull-request URL %s", prURL)
	}
	token := os.Getenv("INPUT_GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("no GitHub token in the environment variable INPUT_GITHUB_TOKEN")
	}

	hc := new(http.Client)
	gh := github.NewClient(hc).WithAuthToken(token)
	if !strings.EqualFold(host, "github.com") {
		u := "https://" + host
		gh = gh.WithEnterpriseURLs(u, u)
	}

	var modverResult modver.Result

	commenter := prcomment.NewCommenter(gh, func(ctx context.Context, pr *github.PullRequest) (string, error) {
		result, err := comparer(ctx, *pr.Base.Repo.CloneURL, *pr.Base.SHA, *pr.Head.Repo.CloneURL, *pr.Head.SHA)
		if err != nil {
			return "", err
		}

		modverResult = result

		report := new(bytes.Buffer)
		modver.Pretty(report, result)

		s := struct {
			Code   string
			Report string
		}{
			Code:   result.Code().String(),
			Report: report.String(),
		}

		out := new(bytes.Buffer)
		err := commentTpl.Execute(out, s)
		return out.String(), err
	})
	commenter.IsComment = func(c *github.IssueComment) bool {
		var r io.Reader = strings.NewReader(*comment.Body)
		r = &io.LimitedReader{R: r, N: 1024}
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			if modverCommentRegex.MatchString(sc.Text()) {
				return true
			}
		}
		return false
	}

	if err := commenter.AddOrUpdate(ctx, owner, reponame, prnum); err != nil {
		return errors.Wrap(err, "adding or updating PR comment")
	}

	modver.Pretty(os.Stdout, result)
}

var modverCommentRegex = regexp.MustCompile(`^# Modver result$`)

//go:embed comment.md.tmpl
var commentTplStr string

var commentTpl = template.Must(template.New("").Parse(commentTplStr))
