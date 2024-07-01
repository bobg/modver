package modver

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bobg/errors"
	"golang.org/x/tools/go/packages"
)

//go:embed *.go go.*
var embedded embed.FS

func withGoFiles(f func(string) error) error {
	tmpdir, err := os.MkdirTemp("", "modver")
	if err != nil {
		return errors.Wrap(err, "creating tempdir")
	}
	defer os.RemoveAll(tmpdir)

	entries, err := embedded.ReadDir(".")
	if err != nil {
		return errors.Wrap(err, "reading embedded dir")
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		if err = copyToTmpdir(tmpdir, name); err != nil {
			return errors.Wrapf(err, "copying embedded file %s to tmpdir %s", name, tmpdir)
		}
	}

	return f(tmpdir)
}

func copyToTmpdir(tmpdir string, filename string) error {
	in, err := embedded.Open(filename)
	if err != nil {
		return errors.Wrapf(err, "opening embedded file %s", filename)
	}
	defer in.Close()

	destname := filepath.Join(tmpdir, filename)
	out, err := os.OpenFile(destname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return errors.Wrapf(err, "opening %s for writing", destname)
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return errors.Wrap(err, "copying data")
}

func withPackage(f func(pkg *packages.Package) error) error {
	return withGoFiles(func(tmpdir string) error {
		config := &packages.Config{
			Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedTypesSizes | packages.NeedModule | packages.NeedEmbedFiles | packages.NeedEmbedPatterns,
			Dir:  tmpdir,
		}
		pkgs, err := packages.Load(config, ".")
		if err != nil {
			return errors.Wrapf(err, "loading Go package from %s", tmpdir)
		}
		if len(pkgs) != 1 {
			return fmt.Errorf("loading Go package in %s, got %d packages, want 1", tmpdir, len(pkgs))
		}
		pkg := pkgs[0]
		return f(pkg)
	})
}
