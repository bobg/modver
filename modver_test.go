package modver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMajor(t *testing.T) {
	runtest(t, "major", Major)
}

func TestMinor(t *testing.T) {
	runtest(t, "minor", Minor)
}

func TestPatchlevel(t *testing.T) {
	runtest(t, "patchlevel", Patchlevel)
}

func TestNone(t *testing.T) {
	runtest(t, "none", None)
}

func runtest(t *testing.T, subtree string, want ResultCode) {
	olderTree := filepath.Join("testdata", subtree, "older")
	entries, err := os.ReadDir(olderTree)
	if err != nil {
		t.Fatal(err)
	}
	newerTree := filepath.Join("testdata", subtree, "newer")
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			olderDir := filepath.Join(olderTree, entry.Name())
			newerDir := filepath.Join(newerTree, entry.Name())
			got, err := CompareDirs(olderDir, newerDir)
			if err != nil {
				t.Fatal(err)
			}
			if got.Code() != want {
				t.Errorf("want %s, got %s", want, got)
			} else {
				t.Log(got)
			}
		})
	}
}

func TestGit(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	const sha = "9a72a6127e397434ab4e4eb34673ff08f143b609"
	res, err := CompareGit(context.Background(), filepath.Join(pwd, ".git"), sha, sha)
	if err != nil {
		t.Fatal(err)
	}
	if res.Code() != None {
		t.Errorf("want None, got %s", res)
	}
}
