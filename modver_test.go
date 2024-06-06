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

	"github.com/bobg/errors"
)

func TestCompare(t *testing.T) {
	tbCompare(t)
}

func BenchmarkCompare(b *testing.B) {
	tbCompare(b)
}

func tbCompare(tb testing.TB) {
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
		runtest(tb, c.dir, c.want)
	}
}

func tbRun(tb testing.TB, name string, f func(testing.TB)) {
	switch tb := tb.(type) {
	case *testing.T:
		tb.Run(name, func(t *testing.T) { f(tb) })
	case *testing.B:
		tb.Run(name, func(b *testing.B) { f(tb) })
	}
}

func runtest(tb testing.TB, typ string, want ResultCode) {
	b, _ := tb.(*testing.B)

	tree := filepath.Join("testdata", typ)
	entries, err := os.ReadDir(tree)
	if err != nil {
		tb.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".tmpl") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".tmpl")
		tbRun(tb, fmt.Sprintf("%s/%s", typ, name), func(tb testing.TB) {
			err := withTestDirs(tree, name, func(olderTestDir, newerTestDir string) {
				if b != nil {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, err := CompareDirs(olderTestDir, newerTestDir)
						if err != nil {
							b.Fatal(err)
						}
					}
					return
				}

				got, err := CompareDirs(olderTestDir, newerTestDir)
				if err != nil {
					tb.Fatal(err)
				}
				if got.Code() != want {
					tb.Errorf("want %s, got %s", want, got)
				} else {
					tb.Log(got)
				}
			})
			if err != nil {
				tb.Fatal(err)
			}
		})
	}
}

func withTestDirs(tree, name string, f func(olderTestDir, newerTestDir string)) error {
	tmpls, err := template.ParseFiles(filepath.Join(tree, name+".tmpl"))
	if err != nil {
		return errors.Wrap(err, "parsing templates")
	}

	tmpdir, err := os.MkdirTemp("", "modver")
	if err != nil {
		return errors.Wrap(err, "creating temp dir")
	}
	defer os.RemoveAll(tmpdir)

	var (
		olderTestDir = filepath.Join(tmpdir, "older")
		newerTestDir = filepath.Join(tmpdir, "newer")
	)

	if err = os.Mkdir(olderTestDir, 0755); err != nil {
		return errors.Wrap(err, "creating older test dir")
	}
	if err = os.Mkdir(newerTestDir, 0755); err != nil {
		return errors.Wrap(err, "creating newer test dir")
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
			if err = executeTmpl(tmpl, buf); err != nil {
				return errors.Wrap(err, "executing template")
			}
			for _, subdir := range []string{olderTestDir, newerTestDir} {
				filename := filepath.Join(subdir, parts[0])
				if err = os.WriteFile(filename, buf.Bytes(), 0644); err != nil {
					return errors.Wrapf(err, "writing file %s", filename)
				}
			}
			continue
		}

		if len(parts) > 1 {
			dirparts := append([]string{tmpdir}, parts[:len(parts)-1]...)
			dirname := filepath.Join(dirparts...)
			if err = os.MkdirAll(dirname, 0755); err != nil {
				return errors.Wrapf(err, "creating dir %s", dirname)
			}
		}
		fileparts := append([]string{tmpdir}, parts...)
		filename := filepath.Join(fileparts...)
		if err = executeTmplToFile(tmpl, filename); err != nil {
			return errors.Wrapf(err, "executing template to file %s", filename)
		}
	}
	if !sawGomod {
		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "module %s\n\ngo 1.18\n", name)
		if err = os.WriteFile(filepath.Join(olderTestDir, "go.mod"), buf.Bytes(), 0644); err != nil {
			return errors.Wrap(err, "writing older go.mod")
		}
		if err = os.WriteFile(filepath.Join(newerTestDir, "go.mod"), buf.Bytes(), 0644); err != nil {
			return errors.Wrap(err, "writing newer go.mod")
		}
	}

	f(olderTestDir, newerTestDir)

	return nil
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
	res, err := CompareGit(ctx, gitDir, "HEAD", "HEAD")
	var cberr cloneBugErr
	if errors.As(err, &cberr) {
		// Workaround for an apparent bug in go-git. See https://github.com/go-git/go-git/issues/726.
		t.Logf("Encountered clone bug, trying workaround: %s", cberr)
		res, err = CompareGit(ctx, "https://github.com/bobg/modver", "HEAD", "HEAD")
	}
	if err != nil {
		t.Fatal(err)
	}
	if res.Code() != None {
		t.Errorf("want None, got %s", res)
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
