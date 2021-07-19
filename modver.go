// Package modver compares two versions of the same set of Go packages.
// It can tell whether the differences require at least a patchlevel version change,
// or a minor version change,
// or a major version change,
// according to semver rules
// (https://semver.org/).
package modver

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

type Result int

const (
	None Result = iota
	Patchlevel
	Minor
	Major
)

func (r Result) String() string {
	switch r {
	case None:
		return "None"
	case Patchlevel:
		return "Patchlevel"
	case Minor:
		return "Minor"
	case Major:
		return "Major"
	default:
		return "unknown Result value"
	}
}

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

	// spew.Config.DisableMethods = true

	c := &comparer{samePackagePath: func(a, b string) bool { return transformPkgPath(a) == transformPkgPath(b) }}

	for pkgPath, pkg := range older {
		if strings.Contains(pkgPath, "/internal/") || strings.HasSuffix(pkgPath, "/internal") {
			// Nothing in an internal package or subpackage is part of the public API.
			continue
		}

		newPkg := newer[pkgPath]
		// fmt.Printf("xxx newPkg:\n%s", spew.Sdump(newPkg))

		for id, obj := range pkg.TypesInfo.Defs {
			fmt.Printf("xxx checking %s: %#v\n", id, obj)

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
				fmt.Printf("xxx no new package %s\n", pkgPath)
				return Major
			}
			newObj := findDef(newPkg.TypesInfo.Defs, id.Name)
			if newObj == nil {
				fmt.Printf("xxx new package %s does not have %s\n", pkgPath, id)
				return Major
			}
			fmt.Printf("xxx newObj %#v\n", newObj)
			if res := c.compareTypes(obj.Type(), newObj.Type()); res == Major {
				return Major
			}
		}
	}

	fmt.Printf("xxx no major changes\n")

	// Second, look for minor-version changes.
	for pkgPath, pkg := range newer {
		fmt.Printf("xxx minor loop: pkgPath %s\n", pkgPath)

		if strings.Contains(pkgPath, "/internal/") || strings.HasSuffix(pkgPath, "/internal") {
			// Nothing in an internal package or subpackage is part of the public API.
			continue
		}

		oldPkg := older[pkgPath]

		for id, obj := range pkg.TypesInfo.Defs {
			fmt.Printf("xxx checking %s\n", id)
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
			fmt.Printf("xxx no new %s package\n", pkgPath)
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
			fmt.Printf("xxx calling compareTypes on %s\n", id)
			if res := c.compareTypes(obj.Type(), newObj.Type()); res != None {
				return Patchlevel
			}
		}
	}

	return None
}

func makePackageMap(pkgs []*packages.Package, keyFn func(string) string) map[string]*packages.Package {
	result := make(map[string]*packages.Package)
	for _, pkg := range pkgs {
		result[keyFn(pkg.PkgPath)] = pkg
	}
	return result
}

func (c *comparer) compareTypes(older, newer types.Type) Result {
	// fmt.Printf("xxx compareTypes: older:\n%s\nnewer:\n%s", spew.Sdump(older), spew.Sdump(newer))

	fmt.Printf("xxx %scompareTypes older = %s; newer = %s\n", strings.Repeat("  ", c.depth), older, newer)
	c.depth++
	defer func() { c.depth-- }()

	if olderNamed, ok := older.(*types.Named); ok {
		if newerNamed, ok := newer.(*types.Named); ok {
			// We already know they have the same name and package.
			return c.compareTypes(olderNamed.Underlying(), newerNamed.Underlying())
		}
		// This is probably impossible.
		// How can newer not be *types.Named if older is,
		// and newer has the same name?
		return Major
	}
	if olderStruct, ok := older.(*types.Struct); ok {
		if newerStruct, ok := newer.(*types.Struct); ok {
			return c.compareStructs(olderStruct, newerStruct)
		}
		fmt.Printf("xxx old struct vs. new non-struct\n")
		return Major
	}
	if !c.assignableTo(newer, older) {
		fmt.Printf("xxx types not assignable: %s vs. %s\n", older, newer)
		return Major
	}
	return None
}

func (c *comparer) compareStructs(older, newer *types.Struct) Result {
	var (
		olderMap = structMap(older)
		newerMap = structMap(newer)
	)

	for name, field := range olderMap {
		newerField, ok := newerMap[name]
		if !ok {
			fmt.Printf("xxx new struct has no %s field\n", name)
			return Major
		}
		if !c.identical(field.Type(), newerField.Type()) {
			return Major
		}
		// xxx what about field tags? Parse them for major vs minor changes?
	}

	for name := range newerMap {
		_, ok := olderMap[name]
		if !ok {
			return Minor
		}
	}

	if !c.identical(older, newer) {
		return Patchlevel
	}

	return None
}

func structMap(t *types.Struct) map[string]*types.Var {
	result := make(map[string]*types.Var)
	for i := 0; i < t.NumFields(); i++ {
		f := t.Field(i)
		result[f.Name()] = f
	}
	return result
}

func findDef(defs map[*ast.Ident]types.Object, name string) types.Object {
	for k, v := range defs {
		if k.Name == name && !isField(v) {
			return v
		}
	}
	return nil
}

func isField(obj types.Object) bool {
	if obj.Parent() != nil {
		return false
	}
	_, ok := obj.Type().(*types.Signature)
	return !ok
}
