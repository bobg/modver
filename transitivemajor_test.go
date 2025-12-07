package modver_test

import (
	"testing"

	"github.com/bobg/modver/v2"
)

func TestTransitiveMajor(t *testing.T) {
	result, err := modver.CompareDirs("testdata/_transitivemajor/a1", "testdata/_transitivemajor/a2")
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Code(); got != modver.Major {
		t.Errorf("got %s, want Major", got)
	}
}
