package modver

import (
	"context"
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"golang.org/x/mod/semver"
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
//   packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo
// in your Config.Mode.
// See CompareDirs for an example of how to call Compare with the result of packages.Load.
func Compare(olders, newers []*packages.Package) Result {
	var (
		older = makePackageMap(olders)
		newer = makePackageMap(newers)
	)

	c := newComparer()

	// Look for major-version changes.
	if res := c.compareMajor(older, newer); res != nil {
		return res
	}

	// Look for minor-version changes.
	if res := c.compareMinor(older, newer); res != nil {
		return res
	}

	// Finally, look for patchlevel-version changes.
	if res := c.comparePatchlevel(older, newer); res != nil {
		return res
	}

	return None
}

func (c *comparer) compareMajor(older, newer map[string]*packages.Package) Result {
	for pkgPath, pkg := range older {
		if strings.Contains(pkgPath, "/internal/") || strings.HasSuffix(pkgPath, "/internal") {
			// Nothing in an internal package or subpackage is part of the public API.
			continue
		}

		newPkg := newer[pkgPath]
		if newPkg != nil {
			if oldMod, newMod := pkg.Module, newPkg.Module; oldMod != nil && newMod != nil {
				if oldMod.Path != newMod.Path {
					return rwrapf(Major, "module name changed from %s to %s", oldMod.Path, newMod.Path)
				}
				if cmp := semver.Compare("v"+oldMod.GoVersion, "v"+newMod.GoVersion); cmp < 0 {
					return rwrapf(Major, "minimum Go version changed from %s to %s", oldMod.GoVersion, newMod.GoVersion)
				}
			}
		}

		var (
			topObjs    = makeTopObjs(pkg)
			newTopObjs map[string]types.Object
		)

		for id, obj := range topObjs {
			if !ast.IsExported(id) {
				continue
			}
			if newPkg == nil {
				return rwrapf(Major, "no new version of package %s", pkgPath)
			}
			if newTopObjs == nil {
				newTopObjs = makeTopObjs(newPkg)
			}
			newObj := newTopObjs[id]
			if newObj == nil {
				return rwrapf(Major, "no object %s in new version of package %s", id, pkgPath)
			}
			if res := c.compareTypes(obj.Type(), newObj.Type()); res.Code() == Major {
				return rwrapf(res, "checking %s", id)
			}
		}
	}

	return nil
}

func (c *comparer) compareMinor(older, newer map[string]*packages.Package) Result {
	for pkgPath, pkg := range newer {
		if strings.Contains(pkgPath, "/internal/") || strings.HasSuffix(pkgPath, "/internal") {
			// Nothing in an internal package or subpackage is part of the public API.
			continue
		}

		var (
			topObjs    = makeTopObjs(pkg)
			oldPkg     = older[pkgPath]
			oldTopObjs map[string]types.Object
		)

		for id, obj := range topObjs {
			if !ast.IsExported(id) {
				continue
			}
			if oldPkg == nil {
				return rwrapf(Minor, "no old version of package %s", pkgPath)
			}
			if oldTopObjs == nil {
				oldTopObjs = makeTopObjs(oldPkg)
			}
			oldObj := oldTopObjs[id]
			if oldObj == nil {
				return rwrapf(Minor, "no object %s in old version of package %s", id, pkgPath)
			}
			if res := c.compareTypes(oldObj.Type(), obj.Type()); res.Code() >= Minor {
				return rwrapf(res.sub(Minor), "checking %s", id)
			}
		}
	}

	return nil
}

func (c *comparer) comparePatchlevel(older, newer map[string]*packages.Package) Result {
	for pkgPath, pkg := range older {
		var (
			topObjs = makeTopObjs(pkg)
			newPkg  = newer[pkgPath]
		)
		if newPkg == nil {
			return rwrapf(Patchlevel, "no new version of package %s", pkgPath)
		}
		newTopObjs := makeTopObjs(newPkg)
		for id, obj := range topObjs {
			newObj := newTopObjs[id]
			if newObj == nil {
				return rwrapf(Patchlevel, "no object %s in new version of package %s", id, pkgPath)
			}
			if res := c.compareTypes(obj.Type(), newObj.Type()); res.Code() != None {
				return rwrapf(res.sub(Patchlevel), "checking %s", id)
			}
		}
	}

	return nil
}

// CompareDirs loads Go modules from the directories at older and newer
// and calls Compare on the results.
func CompareDirs(older, newer string) (Result, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedModule,
		Dir:  older,
	}
	olders, err := packages.Load(cfg, "./...")
	if err != nil {
		return None, fmt.Errorf("loading %s/...: %w", older, err)
	}
	for _, p := range olders {
		if len(p.Errors) > 0 {
			return None, errpkg{pkg: p}
		}
	}

	cfg.Dir = newer
	newers, err := packages.Load(cfg, "./...")
	if err != nil {
		return None, fmt.Errorf("loading %s/...: %w", newer, err)
	}
	for _, p := range newers {
		if len(p.Errors) > 0 {
			return None, errpkg{pkg: p}
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
func CompareGit(ctx context.Context, repoURL, olderRev, newerRev string) (Result, error) {
	return CompareGitWith(ctx, repoURL, olderRev, newerRev, CompareDirs)
}

// CompareGitWith compares the Go packages in two revisions of a Git repo at the given URL.
// It uses the given callback function to perform the comparison.
//
// The callback function receives the paths to two directories,
// containing two clones of the repo:
// one checked out at the older revision
// and one checked out at the newer revision.
//
// Note that CompareGit(...) is simply CompareGitWith(..., CompareDirs).
func CompareGitWith(ctx context.Context, repoURL, olderRev, newerRev string, f func(older, newer string) (Result, error)) (Result, error) {
	parent, err := os.MkdirTemp("", "modver")
	if err != nil {
		return None, fmt.Errorf("creating tmpdir: %w", err)
	}
	defer os.RemoveAll(parent)

	olderDir := filepath.Join(parent, "older")
	err = gitSetup(ctx, repoURL, olderDir, olderRev)
	if err != nil {
		return None, fmt.Errorf("setting up older clone: %w", err)
	}

	newerDir := filepath.Join(parent, "newer")
	err = gitSetup(ctx, repoURL, newerDir, newerRev)
	if err != nil {
		return None, fmt.Errorf("setting up newer clone: %w", err)
	}

	return f(olderDir, newerDir)
}

func gitSetup(ctx context.Context, repoURL, dir, rev string) error {
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
	hash, err := repo.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		return fmt.Errorf(`resolving revision "%s": %w`, rev, err)
	}
	err = worktree.Checkout(&git.CheckoutOptions{Hash: *hash})
	if err != nil {
		return fmt.Errorf(`checking out "%s" in %s: %w`, rev, dir, err)
	}
	return nil
}
