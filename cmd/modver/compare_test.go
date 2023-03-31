package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-github/v50/github"

	"github.com/bobg/modver/v2"
)

func TestDoCompare(t *testing.T) {
	cases := []struct {
		opts           options
		wantErr        bool
		pr             func(t *testing.T) prType
		compareGitWith func(t *testing.T) compareGitWithType
		compareDirs    func(t *testing.T) compareDirsType
	}{{
		opts: options{
			pr:      "https://github.com/foo/bar/pull/17",
			ghtoken: "token",
		},
		pr: mockPR("foo", "bar", 17),
	}, {
		opts: options{
			gitRepo: ".git",
			args:    []string{"older", "newer"},
		},
		compareGitWith: mockCompareGitWith(".git", "older", "newer"),
	}, {
		opts: options{
			args: []string{"older", "newer"},
		},
		compareDirs: mockCompareDirs("older", "newer"),
	}}

	ctx := context.Background()

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case_%02d", i+1), func(t *testing.T) {
			var (
				pr             prType
				compareGitWith compareGitWithType
				compareDirs    compareDirsType
			)
			if tc.pr != nil {
				pr = tc.pr(t)
			}
			if tc.compareGitWith != nil {
				compareGitWith = tc.compareGitWith(t)
			}
			if tc.compareDirs != nil {
				compareDirs = tc.compareDirs(t)
			}

			_, err := doCompareHelper(ctx, tc.opts, mockNewClient, pr, compareGitWith, compareDirs)
			if err != nil {
				if !tc.wantErr {
					t.Errorf("got error %s, wanted none", err)
				}
				return
			}
			if tc.wantErr {
				t.Error("got no error, wanted one")
				return
			}
		})
	}
}

func mockNewClient(ctx context.Context, host, token string) (*github.Client, error) {
	return nil, nil
}

func mockPR(wantOwner, wantRepo string, wantPRNum int) func(t *testing.T) prType {
	return func(t *testing.T) prType {
		return func(ctx context.Context, gh *github.Client, owner, reponame string, prnum int) (modver.Result, error) {
			if owner != wantOwner {
				t.Errorf("got owner %s, want %s", owner, wantOwner)
			}
			if reponame != wantRepo {
				t.Errorf("got repo %s, want %s", reponame, wantRepo)
			}
			if wantPRNum != prnum {
				t.Errorf("got PR number %d, want %d", prnum, wantPRNum)
			}
			return modver.None, nil
		}
	}
}

func mockCompareGitWith(wantGitRepo, wantOlder, wantNewer string) func(t *testing.T) compareGitWithType {
	return func(t *testing.T) compareGitWithType {
		return func(ctx context.Context, repoURL, olderRev, newerRev string, f func(older, newer string) (modver.Result, error)) (modver.Result, error) {
			if repoURL != wantGitRepo {
				t.Errorf("got repo URL %s, want %s", repoURL, wantGitRepo)
			}
			if olderRev != wantOlder {
				t.Errorf("got older rev %s, want %s", olderRev, wantOlder)
			}
			if newerRev != wantNewer {
				t.Errorf("got newer rev %s, want %s", newerRev, wantNewer)
			}
			return modver.None, nil
		}
	}
}

func mockCompareDirs(wantOlder, wantNewer string) func(t *testing.T) compareDirsType {
	return func(t *testing.T) compareDirsType {
		return func(older, newer string) (modver.Result, error) {
			if older != wantOlder {
				t.Errorf("got older dir %s, want %s", older, wantOlder)
			}
			if newer != wantNewer {
				t.Errorf("got newer dir %s, want %s", newer, wantNewer)
			}
			return modver.None, nil
		}
	}
}
