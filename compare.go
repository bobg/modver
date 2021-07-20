package modver

import (
	"context"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"golang.org/x/tools/go/packages"
)

// Compare compares an "older" set of Go packages to a "newer" set of the same packages.
// It tells whether the changes from "older" to "newer" require an increase in the major, minor, or patchlevel version numbers,
// according to semver rules (https://semver.org/).
//
// The result is the _minimal_ change required.
// The actual change required may be greater.
// For example,
// if a new method is added to a type,
// this function will return Minor.
// However, if something also changed about an existing method that breaks the old contract -
// it accepts a narrower range of inputs, for example,
// or returns errors in some new cases -
// that may well require a major-version bump,
// and this function can't detect those cases.
//
// You can be assured, however,
// that if this function returns Major,
// a minor-version bump won't suffice,
// and if this function returns Minor,
// a patchlevel bump won't suffice,
// etc.
//
// The packages passed to this function should have no load errors
// (that is, len(p.Errors) should be 0 for each package p in `olders` and `newers`).
// If you are using packages.Load
// (see https://pkg.go.dev/golang.org/x/tools/go/packages#Load),
// you will need at least
//   packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo
// in your Config.Mode, and you'll probably also want:
//   append(os.Environ(), "GO111MODULE=off")
// in your Config.Env.
//
// The function transformPkgPath takes a package's "path"
// (the path by which it is imported, e.g. "github.com/bobg/modver"),
// or the path of something defined in it
// (such as a subpackage),
// and produces a canonical version of it for comparison purposes.
// This allows packages in `olders` and `newers` to compare equal
// even if they were loaded via two different paths.
// For example, if two different versions of this module are loaded,
// one from /tmp/older/github.com/bobg/modver
// and one from /tmp/newer/github.com/bobg/modver,
// transformPkgPath should be a function that strips off a "/tmp/older/" or a "/tmp/newer/" prefix from its input.
// If transformPkgPath is nil, package paths are not transformed.
func Compare(olders, newers []*packages.Package, transformPkgPath func(string) string) Result {
	// for _, o := range olders {
	// 	if len(o.Errors) > 0 {
	// 		packages.PrintErrors(olders)
	// 		os.Exit(1)
	// 	}
	// }
	// for _, n := range newers {
	// 	if len(n.Errors) > 0 {
	// 		packages.PrintErrors(newers)
	// 		os.Exit(1)
	// 	}
	// }

	if transformPkgPath == nil {
		transformPkgPath = func(in string) string { return in }
	}
	var (
		older = makePackageMap(olders, transformPkgPath)
		newer = makePackageMap(newers, transformPkgPath)
	)

	spew.Config.DisableMethods = true
	// fmt.Printf("xxx older:\n%s\nxxx newer:\n%s", spew.Sdump(older), spew.Sdump(newer))

	c := &comparer{samePackagePath: func(a, b string) bool { return transformPkgPath(a) == transformPkgPath(b) }}

	for pkgPath, pkg := range older {
		if strings.Contains(pkgPath, "/internal/") || strings.HasSuffix(pkgPath, "/internal") {
			// Nothing in an internal package or subpackage is part of the public API.
			continue
		}

		newPkg := newer[pkgPath]
		// fmt.Printf("xxx newPkg:\n%s", spew.Sdump(newPkg))

		for id, obj := range pkg.TypesInfo.Defs {
			// fmt.Printf("xxx checking %s: %#v\n", id, obj)

			if !ast.IsExported(id.Name) {
				continue
			}
			if obj == nil {
				continue
			}
			if isField(obj) {
				continue
			}
			if newPkg == nil {
				// fmt.Printf("xxx no new package %s\n", pkgPath)
				return Major
			}
			newObj := findDef(newPkg.TypesInfo.Defs, id.Name)
			if newObj == nil {
				// fmt.Printf("xxx new package %s does not have %s\n", pkgPath, id)
				return Major
			}
			// fmt.Printf("xxx newObj %#v\n", newObj)
			if res := c.compareTypes(obj.Type(), newObj.Type()); res == Major {
				return Major
			}
		}
	}

	// fmt.Printf("xxx no major changes\n")

	// Second, look for minor-version changes.
	for pkgPath, pkg := range newer {
		// fmt.Printf("xxx minor loop: pkgPath %s\n", pkgPath)

		if strings.Contains(pkgPath, "/internal/") || strings.HasSuffix(pkgPath, "/internal") {
			// Nothing in an internal package or subpackage is part of the public API.
			continue
		}

		oldPkg := older[pkgPath]

		for id, obj := range pkg.TypesInfo.Defs {
			// fmt.Printf("xxx checking %s\n", id)
			if !ast.IsExported(id.Name) {
				continue
			}
			if obj == nil {
				continue
			}
			if oldPkg == nil {
				return Minor
			}
			if isField(obj) {
				continue
			}
			oldObj := findDef(oldPkg.TypesInfo.Defs, id.Name)
			if oldObj == nil {
				return Minor
			}
			if res := c.compareTypes(oldObj.Type(), obj.Type()); res == Minor {
				return Minor
			}
		}
	}

	// Finally, look for patchlevel-version changes.
	for pkgPath, pkg := range older {
		newPkg := newer[pkgPath]
		if newPkg == nil {
			// fmt.Printf("xxx no new %s package\n", pkgPath)
			return Patchlevel
		}
		for id, obj := range pkg.TypesInfo.Defs {
			if obj == nil {
				continue
			}
			newObj := findDef(newPkg.TypesInfo.Defs, id.Name)
			if newObj == nil {
				// fmt.Printf("xxx no new %s obj. Old one is:\n%s", id.Name, spew.Sdump(obj))
				return Patchlevel
			}
			// fmt.Printf("xxx calling compareTypes on %s\n", id)
			if res := c.compareTypes(obj.Type(), newObj.Type()); res != None {
				return Patchlevel
			}
		}
	}

	return None
}

// CompareDirs loads Go packages from the directories at older and newer
// and calls Compare on the results.
func CompareDirs(older, newer string) (Result, error) {
	// fmt.Printf("xxx CompareDirs(%s, %s)\n", older, newer)

	tmpdir, err := os.MkdirTemp("", "modver")
	if err != nil {
		return None, fmt.Errorf("creating tempdir: %w", err)
	}
	defer os.RemoveAll(tmpdir)

	srcdir := filepath.Join(tmpdir, "src")
	err = os.Mkdir(srcdir, 0755)
	if err != nil {
		return None, fmt.Errorf("creating %s: %w", srcdir, err)
	}

	olderFull, err := filepath.Abs(older)
	if err != nil {
		return None, fmt.Errorf("making %s absolute: %w", older, err)
	}
	err = os.Symlink(olderFull, filepath.Join(srcdir, "older"))
	if err != nil {
		return None, fmt.Errorf("linking %s/older to %s: %w", srcdir, olderFull, err)
	}

	newerFull, err := filepath.Abs(newer)
	if err != nil {
		return None, fmt.Errorf("making %s absolute: %w", newer, err)
	}
	err = os.Symlink(newerFull, filepath.Join(srcdir, "newer"))
	if err != nil {
		return None, fmt.Errorf("linking %s/newer to %s: %w", srcdir, newerFull, err)
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo,
		Env:  append(os.Environ(), "GO111MODULE=off"),
		Dir:  srcdir,
	}

	olders, err := packages.Load(cfg, "./older/...")
	if err != nil {
		return None, fmt.Errorf("loading %s/older/...: %w", srcdir, err)
	}
	// fmt.Printf("xxx olders: %+v\n", olders)
	// fmt.Printf("xxx pkgpath %s\n", olders[0].PkgPath)
	for _, p := range olders {
		if len(p.Errors) > 0 {
			return None, errpkg{pkg: p}
		}
	}

	newers, err := packages.Load(cfg, "./newer/...")
	if err != nil {
		return None, fmt.Errorf("loading %s/newer/...: %w", srcdir, err)
	}
	for _, p := range newers {
		if len(p.Errors) > 0 {
			return None, errpkg{pkg: p}
		}
	}
	return Compare(olders, newers, func(inp string) string {
		if result := strings.TrimPrefix(inp, "_"+srcdir+"/older"); result != inp {
			return "_" + result
		}
		if result := strings.TrimPrefix(inp, "_"+srcdir+"/newer"); result != inp {
			return "_" + result
		}
		return inp
	}), nil
}

type errpkg struct {
	pkg *packages.Package
}

func (p errpkg) Error() string {
	strs := make([]string, 0, len(p.pkg.Errors))
	for _, e := range p.pkg.Errors {
		strs = append(strs, e.Error())
	}
	return fmt.Sprintf("error(s) loading package %s: %s", p.pkg.PkgPath, strings.Join(strs, "; "))
}

// CompareGit compares the Go packages in two revisions of a Git repo at the given URL.
func CompareGit(ctx context.Context, repoURL, olderSHA, newerSHA string) (Result, error) {
	parent, err := os.MkdirTemp("", "modver")
	if err != nil {
		return None, fmt.Errorf("creating tmpdir: %w", err)
	}
	defer os.RemoveAll(parent)

	olderDir := filepath.Join(parent, "older")
	err = gitSetup(ctx, repoURL, olderDir, olderSHA)
	if err != nil {
		return None, fmt.Errorf("setting up older clone: %w", err)
	}

	newerDir := filepath.Join(parent, "newer")
	err = gitSetup(ctx, repoURL, newerDir, newerSHA)
	if err != nil {
		return None, fmt.Errorf("setting up newer clone: %w", err)
	}

	return CompareDirs(olderDir, newerDir)
}

func gitSetup(ctx context.Context, repoURL, dir, sha string) error {
	err := os.Mkdir(dir, 0755)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	cloneOpts := &git.CloneOptions{URL: repoURL, NoCheckout: true}
	repo, err := git.PlainCloneContext(ctx, dir, false, cloneOpts)
	if err != nil {
		return fmt.Errorf("cloning %s into %s: %w", repoURL, dir, err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree from %s: %w", dir, err)
	}
	err = worktree.Checkout(&git.CheckoutOptions{Hash: plumbing.NewHash(sha)})
	if err != nil {
		return fmt.Errorf("checking out %s in %s: %w", sha, dir, err)
	}
	return nil
}
