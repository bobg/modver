package main

import (
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	ghtok := os.Getenv("GITHUB_TOKEN")

	cases := []struct {
		args    []string
		wantErr bool
		want    options
	}{{
		want: options{
			ghtoken: ghtok,
			gitCmd:  "git",
		},
	}, {
		args: []string{"-pr", "foo"},
		want: options{
			pr:      "foo",
			ghtoken: ghtok,
			gitCmd:  "git",
		},
	}, {
		args:    []string{"-pr", "foo", "-git", "bar"},
		wantErr: true,
	}, {
		args:    []string{"-pr", "foo", "-v1", "bar"},
		wantErr: true,
	}, {
		args:    []string{"-pr", "foo", "-v2", "bar"},
		wantErr: true,
	}, {
		args:    []string{"-pr", "foo", "-versions"},
		wantErr: true,
	}, {
		args: []string{"-v1", "1", "-v2", "2"},
		want: options{
			v1:      "v1",
			v2:      "v2",
			ghtoken: ghtok,
			gitCmd:  "git",
		},
	}, {
		args:    []string{"-v1", "1", "-v2", "bar"},
		wantErr: true,
	}, {
		args:    []string{"-v1", "foo", "-v2", "2"},
		wantErr: true,
	}}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case_%02d", i+1), func(t *testing.T) {
			got, err := parseArgsHelper(tc.args)
			if err != nil {
				if !tc.wantErr {
					t.Errorf("got error %v, wanted no error", err)
				}
				return
			}
			if tc.wantErr {
				t.Fatal("got no error but wanted one")
			}
			if len(got.args) == 0 {
				got.args = nil // not []string{}
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestParsePR(t *testing.T) {
	cases := []struct {
		inp             string
		wantErr         bool
		owner, reponame string
		prnum           int
	}{{
		wantErr: true,
	}, {
		inp:     "https://x/y",
		wantErr: true,
	}, {
		inp:     "https://github.com/bobg/modver/bleah/17",
		wantErr: true,
	}, {
		inp:      "https://github.com/bobg/modver/pull/17",
		owner:    "bobg",
		reponame: "modver",
		prnum:    17,
	}}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case_%02d", i+1), func(t *testing.T) {
			owner, reponame, prnum, err := parsePR(tc.inp)
			if err != nil {
				if !tc.wantErr {
					t.Errorf("got error %v, wanted no error", err)
				}
				return
			}
			if tc.wantErr {
				t.Fatal("got no error but wanted one")
			}
			if owner != tc.owner {
				t.Errorf("got owner %s, want %s", owner, tc.owner)
			}
			if reponame != tc.reponame {
				t.Errorf("got repo %s, want %s", reponame, tc.reponame)
			}
			if prnum != tc.prnum {
				t.Errorf("got PR number %d, want %d", prnum, tc.prnum)
			}
		})
	}
}
