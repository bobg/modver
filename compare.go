package modver

import (
	"context"
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"golang.org/x/mod/semver"
	"golang.org/x/tools/go/packages"
)

func Compare(olders, newers []*packages.Package) Report {
	c := newComparer(olders, newers)
	return c.run()
}

func isPublic(pkgpath string) bool {
	switch pkgpath {
	case "internal", "main":
		return false
	}
	if strings.HasSuffix(pkgpath, "/main") {
		return false
	}
	if strings.HasPrefix(pkgpath, "internal/") {
		return false
	}
	if strings.HasSuffix(pkgpath, "/internal") {
		return false
	}
	if strings.Contains(pkgpath, "/internal/") {
		return false
	}
	return true
}

func (c *comparer) compareMajor(older, newer map[string]*packages.Package) Result {
	for pkgPath, pkg := range c.older {
		x := c.compareMajorPkg(pkgPath, pkg)
		// xxx accumulate x into result
	}
}

func (c *comparer) compareMajorPkg(pkgPath string, pkg *packages.Package) Xxx {
	if !isPublic(pkgPath) {
		return nil
	}

	newer, ok := c.newer[pkgPath]
	if !ok {
		// xxx return "no new version of package"
	}

	var (
		oldTopObjs = c.oldTopObjs[pkgPath]
		newTopObjs = c.newTopObjs[pkgPath]
	)
	for id, obj := range oldTopObjs {
		if !isExported(id) {
			continue
		}

		newObj := newTopObjs[id]
		if newObj == nil {
			// xxx accumulate "no object %s in new version of package %s"
			continue
		}

		xxx := c.compareTypes(obj.Type(), newObj.Type())
		// xxx accumulate xxx into result
	}

	return xxx
}

func (c *comparer) compareMinor(older, newer map[string]*packages.Package) Result {
	for pkgPath, pkg := range newer {
		if !isPublic(pkgPath) {
			continue
		}

		oldPkg := older[pkgPath]
		if oldPkg != nil {
			if oldMod, newMod := oldPkg.Module, pkg.Module; oldMod != nil && newMod != nil {
				if cmp := semver.Compare("v"+oldMod.GoVersion, "v"+newMod.GoVersion); cmp < 0 {
					return rwrapf(Minor, "minimum Go version changed from %s to %s", oldMod.GoVersion, newMod.GoVersion)
				}
			}
		}

		var (
			topObjs    = makeTopObjs(pkg)
			oldTopObjs map[string]types.Object
		)

		for id, obj := range topObjs {
			if !isExported(id) {
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

// CompareGit compares the Go packages in two revisions of a Git repository at the given URL.
func CompareGit(ctx context.Context, repoURL, olderRev, newerRev string) (Result, error) {
	return CompareGitWith(ctx, repoURL, olderRev, newerRev, CompareDirs)
}

// CompareGit2 compares the Go packages in one revision each of two Git repositories.
func CompareGit2(ctx context.Context, olderRepoURL, olderRev, newerRepoURL, newerRev string) (Result, error) {
	return CompareGit2With(ctx, olderRepoURL, olderRev, newerRepoURL, newerRev, CompareDirs)
}

// CompareGitWith compares the Go packages in two revisions of a Git repository at the given URL.
// It uses the given callback function to perform the comparison.
//
// The callback function receives the paths to two directories,
// containing two clones of the repo:
// one checked out at the older revision
// and one checked out at the newer revision.
//
// Note that CompareGit(...) is simply CompareGitWith(..., CompareDirs).
func CompareGitWith(ctx context.Context, repoURL, olderRev, newerRev string, f func(older, newer string) (Result, error)) (Result, error) {
	return CompareGit2With(ctx, repoURL, olderRev, repoURL, newerRev, f)
}

// CompareGit2With compares the Go packages in one revision each of two Git repositories.
// It uses the given callback function to perform the comparison.
//
// The callback function receives the paths to two directories,
// each containing a clone of one of the repositories at its selected revision.
//
// Note that CompareGit2(...) is simply CompareGit2With(..., CompareDirs).
func CompareGit2With(ctx context.Context, olderRepoURL, olderRev, newerRepoURL, newerRev string, f func(older, newer string) (Result, error)) (Result, error) {
	parent, err := os.MkdirTemp("", "modver")
	if err != nil {
		return None, fmt.Errorf("creating tmpdir: %w", err)
	}
	defer os.RemoveAll(parent)

	olderDir := filepath.Join(parent, "older")
	newerDir := filepath.Join(parent, "newer")

	err = gitSetup(ctx, olderRepoURL, olderDir, olderRev)
	if err != nil {
		return None, fmt.Errorf("setting up older clone: %w", err)
	}

	err = gitSetup(ctx, newerRepoURL, newerDir, newerRev)
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

	gitCmd := GetGit(ctx)
	if gitCmd != "" {
		found, err := exec.LookPath(gitCmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot resolve git command %s, falling back to go-git library: %s\n", gitCmd, err)
			gitCmd = ""
		} else {
			gitCmd = found
		}
	}

	if gitCmd != "" {
		cmd := exec.CommandContext(ctx, gitCmd, "clone", repoURL, dir)
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("native git cloning %s into %s: %w", repoURL, dir, err)
		}

		cmd = exec.CommandContext(ctx, gitCmd, "checkout", rev)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("in native git checkout %s: %w", rev, err)
		}
	} else {
		cloneOpts := &git.CloneOptions{URL: repoURL, NoCheckout: true}
		repo, err := git.PlainCloneContext(ctx, dir, false, cloneOpts)
		if err != nil {
			return cloneBugErr{repoURL: repoURL, dir: dir, err: err}
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
	}

	return nil
}

type cloneBugErr struct {
	repoURL, dir string
	err          error
}

func (cb cloneBugErr) Error() string {
	return fmt.Sprintf("cloning %s into %s: %s", cb.repoURL, cb.dir, cb.err)
}

func (cb cloneBugErr) Unwrap() error {
	return cb.err
}

// Calls ast.IsExported on the final element of name
// (which may be package/type-qualified).
func isExported(name string) bool {
	if i := strings.LastIndex(name, "."); i > 0 {
		name = name[i+1:]
	}
	return ast.IsExported(name)
}
