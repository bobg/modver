// Package modver compares two versions of the same set of Go packages.
// It can tell whether the differences require at least a patchlevel version change,
// or a minor version change,
// or a major version change,
// according to semver rules
// (https://semver.org/).
package modver

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
)

func makePackageMap(pkgs []*packages.Package, keyFn func(string) string) map[string]*packages.Package {
	result := make(map[string]*packages.Package)
	for _, pkg := range pkgs {
		result[keyFn(pkg.PkgPath)] = pkg
	}
	return result
}

func (c *comparer) compareTypes(older, newer types.Type) Result {
	// fmt.Printf("xxx compareTypes: older:\n%s\nnewer:\n%s", spew.Sdump(older), spew.Sdump(newer))

	// fmt.Printf("xxx %scompareTypes older = %s; newer = %s\n", strings.Repeat("  ", c.depth), older, newer)
	// c.depth++
	// defer func() { c.depth-- }()

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
		// fmt.Printf("xxx old struct vs. new non-struct\n")
		return Major
	}
	if !c.assignableTo(newer, older) {
		// fmt.Printf("xxx types not assignable: %s vs. %s\n", older, newer)
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
			// fmt.Printf("xxx new struct has no %s field\n", name)
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
