package main

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-github/v50/github"

	"github.com/bobg/modver/v2"
)

func TestPRHelper(t *testing.T) {
	var (
		ctx   = context.Background()
		repos mockReposService
		prs   mockPRsService
	)

	t.Run("new-comment", func(t *testing.T) {
		var issues mockIssuesService

		result, err := prHelper(ctx, repos, prs, &issues, mockComparer(modver.Minor), "owner", "repo", 17)
		if err != nil {
			t.Fatal(err)
		}
		if result.Code() != modver.Minor {
			t.Fatalf("got result %s, want %s", result, modver.Minor)
		}
		if !strings.HasPrefix(issues.body, "# Modver result") {
			t.Error("issues.body does not start with # Modver result")
		}
		if issues.commentID != 0 {
			t.Errorf("issues.commentID is %d, want 0", issues.commentID)
		}
	})

	t.Run("new-comment", func(t *testing.T) {
		issues := mockIssuesService{update: true}

		result, err := prHelper(ctx, repos, prs, &issues, mockComparer(modver.Minor), "owner", "repo", 17)
		if err != nil {
			t.Fatal(err)
		}
		if result.Code() != modver.Minor {
			t.Fatalf("got result %s, want %s", result, modver.Minor)
		}
		if !strings.HasPrefix(issues.body, "# Modver result") {
			t.Error("issues.body does not start with # Modver result")
		}
		if issues.commentID != 2 {
			t.Errorf("issues.commentID is %d, want 0", issues.commentID)
		}
	})
}

type mockReposService struct{}

func (mockReposService) Get(ctx context.Context, owner, reponame string) (*github.Repository, *github.Response, error) {
	return &github.Repository{
		Owner: &github.User{
			Login: &owner,
		},
		Name:     &reponame,
		CloneURL: ptr("cloneURL"),
	}, nil, nil
}

type mockPRsService struct{}

func (mockPRsService) Get(ctx context.Context, owner, reponame string, number int) (*github.PullRequest, *github.Response, error) {
	return &github.PullRequest{
		Base:   &github.PullRequestBranch{SHA: ptr("baseSHA")},
		Head:   &github.PullRequestBranch{SHA: ptr("headSHA")},
		Number: ptr(17),
	}, nil, nil
}

type mockIssuesService struct {
	update      bool
	owner, repo string
	commentID   int64
	body        string
}

func (m *mockIssuesService) CreateComment(ctx context.Context, owner, reponame string, num int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error) {
	m.owner = owner
	m.repo = reponame
	m.commentID = 0
	m.body = *comment.Body
	return nil, nil, nil
}

func (m *mockIssuesService) EditComment(ctx context.Context, owner, reponame string, commentID int64, newComment *github.IssueComment) (*github.IssueComment, *github.Response, error) {
	m.owner = owner
	m.repo = reponame
	m.commentID = commentID
	m.body = *newComment.Body
	return nil, nil, nil
}

func (m *mockIssuesService) ListComments(ctx context.Context, owner, reponame string, number int, opts *github.IssueListCommentsOptions) ([]*github.IssueComment, *github.Response, error) {
	result := []*github.IssueComment{{
		ID:   ptr(int64(1)),
		Body: ptr("not a modver comment"),
	}}
	if m.update {
		result = append(result, &github.IssueComment{
			ID:   ptr(int64(2)),
			Body: ptr("# Modver result\n\nwoop"),
		})
	}
	return result, nil, nil
}

func mockComparer(result modver.Result) func(ctx context.Context, cloneURL, baseSHA, headSHA string) (modver.Result, error) {
	return func(ctx context.Context, cloneURL, baseSHA, headSHA string) (modver.Result, error) {
		return result, nil
	}
}

func ptr[T any](x T) *T {
	return &x
}
