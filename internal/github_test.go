package internal

import (
	"fmt"
	"testing"
)

func TestParsePR(t *testing.T) {
	cases := []struct {
		inp                   string
		wantErr               bool
		host, owner, reponame string
		prnum                 int
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
		host:     "github.com",
		owner:    "bobg",
		reponame: "modver",
		prnum:    17,
	}}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case_%02d", i+1), func(t *testing.T) {
			host, owner, reponame, prnum, err := ParsePR(tc.inp)
			if err != nil {
				if !tc.wantErr {
					t.Errorf("got error %v, wanted no error", err)
				}
				return
			}
			if tc.wantErr {
				t.Fatal("got no error but wanted one")
			}
			if host != tc.host {
				t.Errorf("got host %s, want %s", host, tc.host)
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
