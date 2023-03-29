package main

import (
	"fmt"
	"os"
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
			if got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}
