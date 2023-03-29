package modver

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func TestCompare(t *testing.T) {
	cases := []struct {
		dir  string
		want ResultCode
	}{{
		dir: "major", want: Major,
	}, {
		dir: "minor", want: Minor,
	}, {
		dir: "patchlevel", want: Patchlevel,
	}, {
		dir: "none", want: None,
	}}

	for _, c := range cases {
		runtest(t, c.dir, c.want)
	}
}

func runtest(t *testing.T, typ string, want ResultCode) {
	tree := filepath.Join("testdata", typ)
	entries, err := os.ReadDir(tree)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".tmpl") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".tmpl")
		t.Run(fmt.Sprintf("%s/%s", typ, name), func(t *testing.T) {
			tmpls, err := template.ParseFiles(filepath.Join(tree, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}

			tmpdir, err := os.MkdirTemp("", "modver")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpdir)

			var (
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

			var sawGomod bool
			for _, tmpl := range tmpls.Templates() {
				// Skip the top-level template
				if strings.HasSuffix(tmpl.Name(), ".tmpl") {
					continue
				}

				sawGomod = sawGomod || (filepath.Base(tmpl.Name()) == "go.mod")

				parts := strings.Split(tmpl.Name(), "/")
				if !strings.Contains(parts[len(parts)-1], ".") {
					parts = append(parts, "x.go")
				}

				if len(parts) == 1 {
					// Only a filename is given.
					// Write it to both older and newer dirs.
					buf := new(bytes.Buffer)
					err = executeTmpl(tmpl, buf)
					if err != nil {
						t.Fatal(err)
					}
					for _, subdir := range []string{olderTestDir, newerTestDir} {
						filename := filepath.Join(subdir, parts[0])
						err = os.WriteFile(filename, buf.Bytes(), 0644)
						if err != nil {
							t.Fatal(err)
						}
					}
					continue
				}

				if len(parts) > 1 {
					dirparts := append([]string{tmpdir}, parts[:len(parts)-1]...)
					dirname := filepath.Join(dirparts...)
					err = os.MkdirAll(dirname, 0755)
					if err != nil {
						t.Fatal(err)
					}
				}
				fileparts := append([]string{tmpdir}, parts...)
				filename := filepath.Join(fileparts...)
				err = executeTmplToFile(tmpl, filename)
				if err != nil {
					t.Fatal(err)
				}
			}
			if !sawGomod {
				buf := new(bytes.Buffer)
				fmt.Fprintf(buf, "module %s\n\ngo 1.18\n", name)
				err = os.WriteFile(filepath.Join(olderTestDir, "go.mod"), buf.Bytes(), 0644)
				if err != nil {
					t.Fatal(err)
				}
				err = os.WriteFile(filepath.Join(newerTestDir, "go.mod"), buf.Bytes(), 0644)
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

func executeTmpl(tmpl *template.Template, w io.Writer) error {
	pr, pw := io.Pipe()
	go func() {
		err := tmpl.Execute(pw, nil)
		if err != nil {
			log.Printf("Error executing template: %s\n", err)
		}
		pw.Close()
	}()

	sc := bufio.NewScanner(pr)
	for sc.Scan() {
		line := sc.Text()
		line = strings.TrimPrefix(line, "//// ")
		fmt.Fprintln(w, line)
	}
	return sc.Err()
}

func executeTmplToFile(tmpl *template.Template, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return executeTmpl(tmpl, f)
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

	ctx := context.Background()

	// Do it once with the go-git library.
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		// For some reason, the go-git library fails when running under GitHub Actions.
		// TODO: figure out why.
		res, err := CompareGit(ctx, gitDir, "HEAD", "HEAD")
		if err != nil {
			t.Fatal(err)
		}
		if res.Code() != None {
			t.Errorf("want None, got %s", res)
		}
	}

	// Now with the git binary.
	ctx = WithGit(ctx, "git")
	res, err = CompareGit(ctx, gitDir, "HEAD", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if res.Code() != None {
		t.Errorf("want None, got %s", res)
	}
}
