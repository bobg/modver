package modver

import (
	"bytes"
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
