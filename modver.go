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

	"github.com/davecgh/go-spew/spew"
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

func Compare(olders, newers []*packages.Package) Result {
	return compare(olders, newers, nil)
}

// This is factored out for the sake of testing.
func compare(olders, newers []*packages.Package, pkgMapKey func(string) string) Result {
	var (
		older = makePackageMap(olders, pkgMapKey)
		newer = makePackageMap(newers, pkgMapKey)
	)

	spew.Config.DisableMethods = true // xxx
	// fmt.Printf("xxx older:\n%s\nnewer:\n%s", spew.Sdump(older), spew.Sdump(newer))

	for pkgPath, pkg := range older {
		if strings.Contains(pkgPath, "/internal/") || strings.HasSuffix(pkgPath, "/internal") {
			// Nothing in an internal package or subpackage is part of the public API.
			continue
		}

		newPkg := newer[pkgPath]
		// fmt.Printf("xxx newPkg:\n%s", spew.Sdump(newPkg))

		for id, obj := range pkg.TypesInfo.Defs {
			if !ast.IsExported(id.Name) {
				continue
			}
			if obj == nil {
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
			if res := compareTypes(obj.Type(), newObj.Type()); res == Major {
				return Major
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
				return Minor
			}
			oldObj := findDef(oldPkg.TypesInfo.Defs, id.Name)
			if oldObj == nil {
				return Minor
			}
			if res := compareTypes(oldObj.Type(), obj.Type()); res == Minor {
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
			if res := compareTypes(obj.Type(), newObj.Type()); res != None {
				return Patchlevel
			}
		}
	}

	return None
}

func makePackageMap(pkgs []*packages.Package, keyFn func(string) string) map[string]*packages.Package {
	result := make(map[string]*packages.Package)
	for _, pkg := range pkgs {
		key := pkg.PkgPath
		if keyFn != nil {
			key = keyFn(key)
		}
		result[key] = pkg
	}
	return result
}

func compareTypes(older, newer types.Type) Result {
	// fmt.Printf("xxx compareTypes: older:\n%s\nnewer:\n%s", spew.Sdump(older), spew.Sdump(newer))

	if olderStruct, ok := older.(*types.Struct); ok {
		if newerStruct, ok := newer.(*types.Struct); ok {
			return compareStructs(olderStruct, newerStruct)
		}
		fmt.Printf("xxx old struct vs. new non-struct\n")
		return Major
	}
	if !types.Identical(older, newer) { // xxx should this be IdenticalIgnoreTags?
		return Major
	}
	return None
}

func compareStructs(older, newer *types.Struct) Result {
	var (
		olderMap = structMap(older)
		newerMap = structMap(newer)
	)

	for name := range olderMap {
		_, ok := newerMap[name]
		if !ok {
			return Major
		}
		// xxx test newField type against field type
	}

	for name := range newerMap {
		_, ok := olderMap[name]
		if !ok {
			return Minor
		}
	}

	if !types.Identical(older, newer) {
		return Patchlevel
	}

	return None
}

func methodMap(t *types.MethodSet) map[string]*types.Selection {
	result := make(map[string]*types.Selection)
	for i := 0; i < t.Len(); i++ {
		sel := t.At(i)
		result[sel.String()] = sel
	}
	return result
}

func structMap(t *types.Struct) map[string]*types.Var {
	result := make(map[string]*types.Var)
	for i := 0; i < t.NumFields(); i++ {
		f := t.Field(i)
		result[f.String()] = f // xxx we want the name - what is this string?
	}
	return result
}

func findDef(defs map[*ast.Ident]types.Object, name string) types.Object {
	for k, v := range defs {
		if k.Name == name {
			return v
		}
	}
	return nil
}
