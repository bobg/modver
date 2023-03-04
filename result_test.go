package modver

import (
	"bytes"
	"fmt"
	"testing"
)

func TestPretty(t *testing.T) {
	cases := []struct {
		r    Result
		want string
	}{{
		r:    Minor,
		want: "Minor\n",
	}, {
		r:    rwrap(Minor, "foo"),
		want: "foo\n  Minor\n",
	}, {
		r:    ResultList{Minor, rwrap(Minor, "foo")},
		want: "Minor\n  Minor\n  foo\n    Minor\n",
	}}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case_%02d", i+1), func(t *testing.T) {
			buf := new(bytes.Buffer)
			Pretty(buf, tc.r)
			if buf.String() != tc.want {
				t.Errorf("got %s, want %s", buf, tc.want)
			}
		})
	}
}
