package main

import "testing"

func TestGetTag(t *testing.T) {
	got, err := getTag("../..", "aa470e1b623810ea1434f51b569f37cf9a0782ab")
	if err != nil {
		t.Fatal(err)
	}

	const want = "v1.1.8"
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
