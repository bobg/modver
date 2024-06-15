package internal

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"html/template"
	"io"
	"regexp"
	"strings"

	"github.com/bobg/errors"
	"github.com/bobg/prcomment"
	"github.com/google/go-github/v62/github"

	"github.com/bobg/modver/v2"
)

func PR(ctx context.Context, gh *github.Client, owner, reponame string, prnum int) (modver.Result, error) {
	var modverResult modver.Result

	commenter := prcomment.NewCommenter(gh, func(ctx context.Context, pr *github.PullRequest) (string, error) {
		result, err := modver.CompareGit2(ctx, *pr.Base.Repo.CloneURL, *pr.Base.SHA, *pr.Head.Repo.CloneURL, *pr.Head.SHA)
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
		err = commentTpl.Execute(out, s)
		return out.String(), err
	})
	commenter.IsComment = func(c *github.IssueComment) bool {
		var r io.Reader = strings.NewReader(*c.Body)
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
		return modver.None, errors.Wrap(err, "adding or updating PR comment")
	}

	return modverResult, nil
}

var modverCommentRegex = regexp.MustCompile(`^# Modver result$`)

//go:embed comment.md.tmpl
var commentTplStr string

var commentTpl = template.Must(template.New("").Parse(commentTplStr))
