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
		pr             func(*testing.T, *int) prType
		compareGitWith func(*testing.T, *int) compareGitWithType
		compareDirs    func(*testing.T, *int) compareDirsType
	}{{
		opts: options{
			pr:      "https://github.com/foo/bar/pull/17",
			ghtoken: "token",
		},
		pr: mockPR("foo", "bar", 17),
	}, {
		opts: options{
			pr:      "https://github.com/foo/bar/baz/pull/17",
			ghtoken: "token",
		},
		wantErr: true,
	}, {
		opts: options{
			pr: "https://github.com/foo/bar/pull/17",
		},
		wantErr: true,
	}, {
		opts: options{
			gitRepo: ".git",
			args:    []string{"older", "newer"},
		},
		compareGitWith: mockCompareGitWith(".git", "older", "newer"),
	}, {
		opts: options{
			gitRepo: ".git",
			args:    []string{"older", "newer", "evenmorenewer"},
		},
		wantErr: true,
	}, {
		opts: options{
			args: []string{"older", "newer"},
		},
		compareDirs: mockCompareDirs("older", "newer"),
	}, {
		opts: options{
			args: []string{"older"},
		},
		wantErr: true,
	}}

	ctx := context.Background()

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case_%02d", i+1), func(t *testing.T) {
			var (
				pr             prType
				compareGitWith compareGitWithType
				compareDirs    compareDirsType
				calls          int
			)
			if tc.pr != nil {
				pr = tc.pr(t, &calls)
			}
			if tc.compareGitWith != nil {
				compareGitWith = tc.compareGitWith(t, &calls)
			}
			if tc.compareDirs != nil {
				compareDirs = tc.compareDirs(t, &calls)
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
			if calls != 1 {
				t.Errorf("got %d calls, want 1", calls)
			}
		})
	}
}

func mockNewClient(ctx context.Context, host, token string) (*github.Client, error) {
	return nil, nil
}

func mockPR(wantOwner, wantRepo string, wantPRNum int) func(*testing.T, *int) prType {
	return func(t *testing.T, calls *int) prType {
		return func(ctx context.Context, gh *github.Client, owner, reponame string, prnum int) (modver.Result, error) {
			*calls++
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

func mockCompareGitWith(wantGitRepo, wantOlder, wantNewer string) func(*testing.T, *int) compareGitWithType {
	return func(t *testing.T, calls *int) compareGitWithType {
		return func(ctx context.Context, repoURL, olderRev, newerRev string, f func(older, newer string) (modver.Result, error)) (modver.Result, error) {
			*calls++
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

func mockCompareDirs(wantOlder, wantNewer string) func(*testing.T, *int) compareDirsType {
	return func(t *testing.T, calls *int) compareDirsType {
		return func(older, newer string) (modver.Result, error) {
			*calls++
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
