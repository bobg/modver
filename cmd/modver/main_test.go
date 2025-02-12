package main

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/bobg/modver/v3"
)

func TestDoShowResult(t *testing.T) {
	cases := []struct {
		res          modver.Result
		opts         options
		wantExitCode int
		want         string
	}{{
		res:          modver.Patchlevel,
		opts:         options{quiet: true},
		wantExitCode: int(modver.Patchlevel),
	}, {
		res:  modver.Patchlevel,
		want: modver.Patchlevel.String() + "\n",
	}, {
		res:  modver.None,
		opts: options{v1: "v1.0.0", v2: "v1.0.1"},
		want: "OK None\n",
	}, {
		res:          modver.None,
		opts:         options{v1: "v1.0.1", v2: "v1.0.0"},
		want:         "ERR None\n",
		wantExitCode: 1,
	}, {
		res:  modver.Patchlevel,
		opts: options{v1: "v1.0.0", v2: "v1.0.1"},
		want: "OK Patchlevel\n",
	}, {
		res:          modver.Patchlevel,
		opts:         options{v1: "v1.0.0", v2: "v1.0.0"},
		want:         "ERR Patchlevel\n",
		wantExitCode: 1,
	}, {
		res:  modver.Minor,
		opts: options{v1: "v1.0.0", v2: "v1.1.0"},
		want: "OK Minor\n",
	}, {
		res:          modver.Minor,
		opts:         options{v1: "v1.0.0", v2: "v1.0.1"},
		want:         "ERR Minor\n",
		wantExitCode: 1,
	}, {
		res:  modver.Major,
		opts: options{v1: "v1.0.0", v2: "v2.0.0"},
		want: "OK Major\n",
	}, {
		res:          modver.Major,
		opts:         options{v1: "v1.0.0", v2: "v1.1.0"},
		want:         "ERR Major\n",
		wantExitCode: 1,
	}}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case_%02d", i+1), func(t *testing.T) {
			buf := new(bytes.Buffer)
			exitCode := doShowResult(buf, tc.res, tc.opts)
			if exitCode != tc.wantExitCode {
				t.Errorf("got exit code %d, want %d", exitCode, tc.wantExitCode)
			}
			if buf.String() != tc.want {
				t.Errorf("got %s, want %s", buf, tc.want)
			}
		})
	}
}
