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

func runtest(t *testing.T, typ string, want ResultCode) {
	tree := filepath.Join("testdata", typ)
	entries, err := os.ReadDir(tree)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		t.Run(filepath.Join(typ, entry.Name()), func(t *testing.T) {
			olderDir := filepath.Join(tree, entry.Name(), "older")
			newerDir := filepath.Join(tree, entry.Name(), "newer")
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

	gitDir := filepath.Join(pwd, ".git")
	_, err = os.Stat(gitDir)
	if os.IsNotExist(err) {
		t.Skip()
	}
	if err != nil {
		t.Fatal(err)
	}

	res, err := CompareGit(context.Background(), gitDir, "HEAD", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if res.Code() != None {
		t.Errorf("want None, got %s", res)
	}
}
