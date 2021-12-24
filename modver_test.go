package modver

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cp "github.com/otiai10/copy"
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
			tmpdir, err := os.MkdirTemp("", "modver")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpdir)

			var (
				srcdir       = filepath.Join(tree, entry.Name())
				olderSrcDir  = filepath.Join(srcdir, "older")
				newerSrcDir  = filepath.Join(srcdir, "newer")
				olderTestDir = filepath.Join(tmpdir, "older")
				newerTestDir = filepath.Join(tmpdir, "newer")
			)
			err = os.Mkdir(olderTestDir, 0755)
			if err != nil {
				t.Fatal(err)
			}
			err = os.Mkdir(newerTestDir, 0755)
			if err != nil {
				t.Fatal(err)
			}

			err = cp.Copy(olderSrcDir, olderTestDir)
			if err != nil {
				t.Fatal(err)
			}
			err = cp.Copy(newerSrcDir, newerTestDir)
			if err != nil {
				t.Fatal(err)
			}

			var (
				gomodSrc   = filepath.Join(srcdir, "go.mod")
				gomodOlder = filepath.Join(olderTestDir, "go.mod")
				gomodNewer = filepath.Join(newerTestDir, "go.mod")
			)

			_, err = os.Stat(gomodSrc)
			if os.IsNotExist(err) {
				b := new(bytes.Buffer)
				fmt.Fprintf(b, "module %s\n\ngo 1.18\n", entry.Name())
				err = os.WriteFile(gomodOlder, b.Bytes(), 0644)
				if err != nil {
					t.Fatal(err)
				}
				err = os.WriteFile(gomodNewer, b.Bytes(), 0644)
				if err != nil {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			} else {
				err = cp.Copy(gomodSrc, gomodOlder)
				if err != nil {
					t.Fatal(err)
				}
				err = cp.Copy(gomodSrc, gomodNewer)
				if err != nil {
					t.Fatal(err)
				}
			}

			gosumSrc := filepath.Join(srcdir, "go.sum")
			_, err = os.Stat(gosumSrc)
			if err == nil {
				err = cp.Copy(gosumSrc, filepath.Join(olderTestDir, "go.sum"))
				if err != nil {
					t.Fatal(err)
				}
				err = cp.Copy(gosumSrc, filepath.Join(newerTestDir, "go.sum"))
				if err != nil {
					t.Fatal(err)
				}
			}

			got, err := CompareDirs(olderTestDir, newerTestDir)
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
