package modver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
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

func runtest(t *testing.T, subtree string, want Result) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo,
	}

	olderTree := filepath.Join("testdata", subtree, "older")
	entries, err := os.ReadDir(olderTree)
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
		t.Run(entry.Name(), func(t *testing.T) {
			olderDir := filepath.Join(olderTree, entry.Name())
			older, err := packages.Load(cfg, "./"+olderDir+"/...")
			if err != nil {
				t.Fatal(err)
			}

			newerDir := filepath.Join("testdata", subtree, "newer", entry.Name())
			newer, err := packages.Load(cfg, "./"+newerDir+"/...")
			if err != nil {
				t.Fatal(err)
			}

			got := Compare(older, newer, testPkgMapKey)
			if got != want {
				t.Errorf("got %s, want %s", got, want)
			}
		})
	}
}

func testPkgMapKey(inp string) string {
	result := strings.Replace(inp, "/older/", "/", 1)
	result = strings.Replace(result, "/newer/", "/", 1)
	return result
}
