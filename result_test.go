package modver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

func TestPretty(t *testing.T) {
	buf := new(bytes.Buffer)
	Pretty(buf, Minor)
	if buf.String() != "Minor\n" {
		t.Errorf("got %s, want Minor\\n", buf)
	}

	buf.Reset()

	res := rwrap(Minor, "foo")
	Pretty(buf, res)
	const want = "foo\n  Minor\n"
	if buf.String() != want {
		t.Errorf("got %s, want %s", buf, want)
	}
}

func TestMarshalResultCode(t *testing.T) {
	cases := []struct {
		rc      ResultCode
		want    string
		wantErr bool
	}{{
		rc:   None,
		want: `"None"`,
	}, {
		rc:   Patchlevel,
		want: `"Patchlevel"`,
	}, {
		rc:   Minor,
		want: `"Minor"`,
	}, {
		rc:   Major,
		want: `"Major"`,
	}, {
		rc:      ResultCode(42),
		wantErr: true,
	}}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case_%02d", i+1), func(t *testing.T) {
			got, err := json.Marshal(tc.rc)
			if err != nil && tc.wantErr {
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tc.wantErr {
				t.Fatal("got no error but want one")
			}
			if string(got) != tc.want {
				t.Errorf("marshaling: got %s, want %s", string(got), tc.want)
			}

			var rc ResultCode
			if err := json.Unmarshal(got, &rc); err != nil {
				t.Fatal(err)
			}
			if rc != tc.rc {
				t.Errorf("unmarshaling: got %v, want %v", rc, tc.rc)
			}
		})
	}
}
