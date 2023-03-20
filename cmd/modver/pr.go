package main

import (
	"bytes"
	"context"
	_ "embed"
	"regexp"
	"text/template"

	"github.com/google/go-github/v50/github"
	"github.com/pkg/errors"

	"github.com/bobg/modver/v2"
)

func doPR(ctx context.Context, gh *github.Client, owner, reponame string, prnum int) (modver.Result, error) {
	repo, _, err := gh.Repositories.Get(ctx, owner, reponame)
	if err != nil {
		return modver.None, errors.Wrap(err, "getting repository")
	}
	pr, _, err := gh.PullRequests.Get(ctx, owner, reponame, prnum)
	if err != nil {
		return modver.None, errors.Wrap(err, "getting pull request")
	}
	result, err := modver.CompareGit(ctx, *repo.CloneURL, *pr.Base.SHA, *pr.Head.SHA)
	if err != nil {
		return modver.None, errors.Wrap(err, "comparing versions")
	}
	comments, _, err := gh.Issues.ListComments(ctx, owner, reponame, prnum, nil)
	if err != nil {
		return modver.None, errors.Wrap(err, "listing PR comments")
	}

	var comment *github.IssueComment
	for _, c := range comments {
		if isModverComment(c) {
			comment = c
			break
		}
	}

	if comment != nil {
		err = updateComment(ctx, gh, repo, comment, result)
		if err != nil {
			return modver.None, errors.Wrap(err, "updating PR comment")
		}
	} else {
		err = createComment(ctx, gh, repo, pr, result)
		if err != nil {
			return modver.None, errors.Wrap(err, "creating PR comment")
		}
	}

	return result, nil
}

var modverCommentRegex = regexp.MustCompile(`^# Modver result$`)

func isModverComment(comment *github.IssueComment) bool {
	return modverCommentRegex.MatchString(*comment.Body)
}

func createComment(ctx context.Context, gh *github.Client, repo *github.Repository, pr *github.PullRequest, result modver.Result) error {
	body, err := commentBody(result)
	if err != nil {
		return errors.Wrap(err, "rendering comment body")
	}
	comment := &github.IssueComment{
		Body: &body,
	}
	_, _, err = gh.Issues.CreateComment(ctx, *repo.Owner.Login, *repo.Name, *pr.Number, comment)
	return errors.Wrap(err, "creating GitHub comment")
}

func updateComment(ctx context.Context, gh *github.Client, repo *github.Repository, comment *github.IssueComment, result modver.Result) error {
	body, err := commentBody(result)
	if err != nil {
		return errors.Wrap(err, "rendering comment body")
	}
	newComment := &github.IssueComment{
		Body: &body,
	}
	_, _, err = gh.Issues.EditComment(ctx, *repo.Owner.Login, *repo.Name, *comment.ID, newComment)
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
