package internal

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"io"
	"regexp"
	"strings"
	"text/template"

	"github.com/bobg/errors"
	"github.com/google/go-github/v67/github"

	"github.com/bobg/modver/v3"
)

// PR performs modver analysis on a GitHub pull request.
func PR(ctx context.Context, gh *github.Client, owner, reponame string, prnum int) (modver.Result, error) {
	return prHelper(ctx, gh.Repositories, gh.PullRequests, gh.Issues, modver.CompareGit2, owner, reponame, prnum)
}

type reposIntf interface {
	Get(ctx context.Context, owner, reponame string) (*github.Repository, *github.Response, error)
}

type prsIntf interface {
	Get(ctx context.Context, owner, reponame string, number int) (*github.PullRequest, *github.Response, error)
}

type issuesIntf interface {
	createCommenter
	editCommenter
	ListComments(ctx context.Context, owner, reponame string, number int, opts *github.IssueListCommentsOptions) ([]*github.IssueComment, *github.Response, error)
}

func prHelper(ctx context.Context, repos reposIntf, prs prsIntf, issues issuesIntf, comparer func(ctx context.Context, baseURL, baseSHA, headURL, headSHA string) (modver.Result, error), owner, reponame string, prnum int) (modver.Result, error) {
	repo, _, err := repos.Get(ctx, owner, reponame)
	if err != nil {
		return modver.None, errors.Wrap(err, "getting repository")
	}
	pr, _, err := prs.Get(ctx, owner, reponame, prnum)
	if err != nil {
		return modver.None, errors.Wrap(err, "getting pull request")
	}
	result, err := comparer(ctx, *pr.Base.Repo.CloneURL, *pr.Base.SHA, *pr.Head.Repo.CloneURL, *pr.Head.SHA)
	if err != nil {
		return modver.None, errors.Wrap(err, "comparing versions")
	}
	comments, _, err := issues.ListComments(ctx, owner, reponame, prnum, nil)
	if err != nil {
		return modver.None, errors.Wrap(err, "listing PR comments")
	}

	for _, c := range comments {
		if isModverComment(c) {
			err = updateComment(ctx, issues, repo, c, result)
			return result, errors.Wrap(err, "updating PR comment")
		}
	}

	err = createComment(ctx, issues, repo, pr, result)
	return result, errors.Wrap(err, "creating PR comment")
}

var modverCommentRegex = regexp.MustCompile(`^# Modver result$`)

func isModverComment(comment *github.IssueComment) bool {
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

type createCommenter interface {
	CreateComment(ctx context.Context, owner, reponame string, num int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error)
}

func createComment(ctx context.Context, issues createCommenter, repo *github.Repository, pr *github.PullRequest, result modver.Result) error {
	body, err := commentBody(result)
	if err != nil {
		return errors.Wrap(err, "rendering comment body")
	}
	comment := &github.IssueComment{
		Body: &body,
	}
	_, _, err = issues.CreateComment(ctx, *repo.Owner.Login, *repo.Name, *pr.Number, comment)
	return errors.Wrap(err, "creating GitHub comment")
}

type editCommenter interface {
	EditComment(ctx context.Context, owner, reponame string, commentID int64, newComment *github.IssueComment) (*github.IssueComment, *github.Response, error)
}

func updateComment(ctx context.Context, issues editCommenter, repo *github.Repository, comment *github.IssueComment, result modver.Result) error {
	body, err := commentBody(result)
	if err != nil {
		return errors.Wrap(err, "rendering comment body")
	}
	newComment := &github.IssueComment{
		Body: &body,
	}
	_, _, err = issues.EditComment(ctx, *repo.Owner.Login, *repo.Name, *comment.ID, newComment)
	return errors.Wrap(err, "editing GitHub comment")
}

//go:embed comment.md.tmpl
var commentTplStr string

var commentTpl = template.Must(template.New("").Parse(commentTplStr))

func commentBody(result modver.Result) (string, error) {
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
}
