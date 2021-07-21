package modver

import (
	"context"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"golang.org/x/tools/go/packages"
)

// Compare compares an "older" version of a Go module to a "newer" version of the same module.
// It tells whether the changes from "older" to "newer" require an increase in the major, minor, or patchlevel version numbers,
// according to semver rules (https://semver.org/).
//
// Briefly, a major-version bump is needed for incompatible changes in the public API,
// such as when a type is removed or renamed,
// or parameters or results are added to or removed from a function.
// Old callers cannot expect to use the new version without being updated.
//
// A minor-version bump is needed when new features are added to the public API,
// like a new entrypoint or new fields in an existing struct.
// Old callers _can_ continue using the new version without being updated,
// but callers depending on the new features cannot use the old version.
//
// A patchlevel bump is needed for most other changes.
//
// The result of Compare is the _minimal_ change required.
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
// in your Config.Mode.
// See CompareDirs for an example of how to call Compare with the result of packages.Load.
func Compare(olders, newers []*packages.Package) Result {
	var (
		older = makePackageMap(olders)
		newer = makePackageMap(newers)
	)

	var c comparer
	for pkgPath, pkg := range older {
		if strings.Contains(pkgPath, "/internal/") || strings.HasSuffix(pkgPath, "/internal") {
			// Nothing in an internal package or subpackage is part of the public API.
			continue
		}

		newPkg := newer[pkgPath]
		for id, obj := range pkg.TypesInfo.Defs {
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
				return Result{Code: Major, Why: fmt.Sprintf("no new version of package %s", pkgPath)}
			}
			newObj := findDef(newPkg.TypesInfo.Defs, id.Name)
			if newObj == nil {
				return Result{Code: Major, Why: fmt.Sprintf("no object %s in new version of package %s", id.Name, pkgPath)}
			}
			if res := c.compareTypes(obj.Type(), newObj.Type()); res.Code == Major {
				return res
			}
		}
	}

	// Second, look for minor-version changes.
	for pkgPath, pkg := range newer {
		if strings.Contains(pkgPath, "/internal/") || strings.HasSuffix(pkgPath, "/internal") {
			// Nothing in an internal package or subpackage is part of the public API.
			continue
		}

		oldPkg := older[pkgPath]

		for id, obj := range pkg.TypesInfo.Defs {
			if !ast.IsExported(id.Name) {
				continue
			}
			if obj == nil {
				continue
			}
			if oldPkg == nil {
				return Result{Code: Minor, Why: fmt.Sprintf("no old version of package %s", pkgPath)}
			}
			if isField(obj) {
				continue
			}
			oldObj := findDef(oldPkg.TypesInfo.Defs, id.Name)
			if oldObj == nil {
				return Result{Code: Minor, Why: fmt.Sprintf("no object %s in old version of package %s", id.Name, pkgPath)}
			}
			if res := c.compareTypes(oldObj.Type(), obj.Type()); res.Code >= Minor {
				return Result{Code: Minor, Why: res.Why}
			}
		}
	}

	// Finally, look for patchlevel-version changes.
	for pkgPath, pkg := range older {
		newPkg := newer[pkgPath]
		if newPkg == nil {
			return Result{Code: Patchlevel, Why: fmt.Sprintf("no new version of package %s", pkgPath)}
		}
		for id, obj := range pkg.TypesInfo.Defs {
			if obj == nil {
				continue
			}
			newObj := findDef(newPkg.TypesInfo.Defs, id.Name)
			if newObj == nil {
				return Result{Code: Patchlevel, Why: fmt.Sprintf("no object %s in new version of package %s", id.Name, pkgPath)}
			}
			if res := c.compareTypes(obj.Type(), newObj.Type()); res.Code != None {
				return Result{Code: Patchlevel, Why: res.Why}
			}
		}
	}

	return Result{}
}

// CompareDirs loads Go modules from the directories at older and newer
// and calls Compare on the results.
func CompareDirs(older, newer string) (Result, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo,
		Dir:  older,
	}
	olders, err := packages.Load(cfg, "./...")
	if err != nil {
		return Result{}, fmt.Errorf("loading %s/...: %w", older, err)
	}
	for _, p := range olders {
		if len(p.Errors) > 0 {
			return Result{}, errpkg{pkg: p}
		}
	}

	cfg.Dir = newer
	newers, err := packages.Load(cfg, "./...")
	if err != nil {
		return Result{}, fmt.Errorf("loading %s/...: %w", newer, err)
	}
	for _, p := range newers {
		if len(p.Errors) > 0 {
			return Result{}, errpkg{pkg: p}
		}
	}

	return Compare(olders, newers), nil
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
		return Result{}, fmt.Errorf("creating tmpdir: %w", err)
	}
	defer os.RemoveAll(parent)

	olderDir := filepath.Join(parent, "older")
	err = gitSetup(ctx, repoURL, olderDir, olderSHA)
	if err != nil {
		return Result{}, fmt.Errorf("setting up older clone: %w", err)
	}

	newerDir := filepath.Join(parent, "newer")
	err = gitSetup(ctx, repoURL, newerDir, newerSHA)
	if err != nil {
		return Result{}, fmt.Errorf("setting up newer clone: %w", err)
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
