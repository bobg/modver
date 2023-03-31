package modver

import (
	"fmt"
	"testing"
)

func TestIsPublic(t *testing.T) {
	cases := []struct {
		inp  string
		want bool
	}{{
		inp:  "main",
		want: false,
	}, {
		inp:  "internal",
		want: false,
	}, {
		inp:  "mainx",
		want: true,
	}, {
		inp:  "internalx",
		want: true,
	}, {
		inp:  "foo/main",
		want: false,
	}, {
		inp:  "main/foo",
		want: true,
	}, {
		inp:  "foo/mainx",
		want: true,
	}, {
		inp:  "mainx/foo",
		want: true,
	}, {
		inp:  "foo/internal",
		want: false,
	}, {
		inp:  "internal/foo",
		want: false,
	}, {
		inp:  "foo/internal/bar",
		want: false,
	}, {
		inp:  "foo/internalx",
		want: true,
	}, {
		inp:  "internalx/foo",
		want: true,
	}, {
		inp:  "foo/xinternal/bar",
		want: true,
	}, {
		inp:  "foo/xinternal",
		want: true,
	}, {
		inp:  "xinternal/foo",
		want: true,
	}, {
		inp:  "foo/xinternal/bar",
		want: true,
	}}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("case_%02d", i+1), func(t *testing.T) {
			got := isPublic(tc.inp)
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
