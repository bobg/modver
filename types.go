package modver

import (
	"fmt"
	"go/ast"
	"go/types"
	"regexp"

	"golang.org/x/tools/go/packages"
)

type (
	comparer struct {
		stack []typePair
		cache map[typePair]bool
	}
	typePair struct{ a, b types.Type }
)

func newComparer() *comparer {
	return &comparer{cache: make(map[typePair]bool)}
}

func (c *comparer) compareTypes(older, newer types.Type) Result {
	if c.identical(older, newer) {
		return None
	}
	if olderNamed, ok := older.(*types.Named); ok {
		if newerNamed, ok := newer.(*types.Named); ok {
			// We already know they have the same name and package.
			return c.compareTypes(olderNamed.Underlying(), newerNamed.Underlying())
		}
		// This is probably impossible.
		// How can newer not be *types.Named if older is,
		// and newer has the same name?
		return Major.wrap(fmt.Sprintf("%s went from defined type to non-defined type", older))
	}
	if olderStruct, ok := older.(*types.Struct); ok {
		if newerStruct, ok := newer.(*types.Struct); ok {
			return c.compareStructs(olderStruct, newerStruct)
		}
		return Major.wrap(fmt.Sprintf("%s went from struct to non-struct", older))
	}
	if olderIntf, ok := older.(*types.Interface); ok {
		if newerIntf, ok := newer.(*types.Interface); ok {
			if c.assignableTo(newerIntf, olderIntf) {
				return Minor.wrap(fmt.Sprintf("new interface %s is a superset of old", newer))
			}
			return Major.wrap(fmt.Sprintf("new interface %s is not assignable to old", newer))
		}
		return Major.wrap(fmt.Sprintf("%s went from interface to non-interface", older))
	}
	if olderSig, ok := older.(*types.Signature); ok {
		if newerSig, ok := newer.(*types.Signature); ok {
			if _, added := c.identicalSigs(olderSig, newerSig); added {
				return Minor.wrap(fmt.Sprintf("%s adds optional parameters", newer))
			}
		} else {
			return Major.wrap(fmt.Sprintf("%s went from function to non-function", older))
		}
	}
	if !c.assignableTo(newer, older) {
		return Major.wrap(fmt.Sprintf("%s is not assignable to %s", newer, older))
	}
	return None
}

func (c *comparer) compareStructs(older, newer *types.Struct) Result {
	var (
		olderMap = structMap(older)
		newerMap = structMap(newer)
	)

	for i := 0; i < older.NumFields(); i++ {
		field := older.Field(i)
		newFieldIndex, ok := newerMap[field.Name()]
		if !ok {
			return Major.wrap(fmt.Sprintf("old struct field %s was removed from %s", field.Name(), older))
		}
		newField := newer.Field(newFieldIndex)
		if !c.identical(field.Type(), newField.Type()) {
			return Major.wrap(fmt.Sprintf("struct field %s changed in %s", field.Name(), older))
		}
		var (
			tag    = older.Tag(i)
			newTag = newer.Tag(newFieldIndex)
		)
		if res := c.compareStructTags(tag, newTag); res.Code() == Major {
			return res.wrap(fmt.Sprintf("tag change in field %s of %s", field.Name(), older))
		}
	}

	for i := 0; i < newer.NumFields(); i++ {
		field := newer.Field(i)
		oldFieldIndex, ok := olderMap[field.Name()]
		if !ok {
			return Minor.wrap(fmt.Sprintf("struct field %s was added to %s", field.Name(), newer))
		}
		var (
			oldTag = older.Tag(oldFieldIndex)
			tag    = newer.Tag(i)
		)
		if res := c.compareStructTags(oldTag, tag); res.Code() == Minor {
			return res.wrap(fmt.Sprintf("tag change in field %s of %s", field.Name(), older))
		}
	}

	if !c.identical(older, newer) {
		return Patchlevel.wrap(fmt.Sprintf("old and new versions of %s are not identical", older))
	}

	return None
}

func (c *comparer) compareStructTags(a, b string) Result {
	if a == b {
		return None
	}
	var (
		amap = tagMap(a)
		bmap = tagMap(b)
	)
	for k, av := range amap {
		if bv, ok := bmap[k]; ok {
			if av != bv {
				return Major.wrap(fmt.Sprintf(`struct tag changed the value for key "%s" from "%s" to "%s"`, k, av, bv))
			}
		} else {
			return Major.wrap(fmt.Sprintf("struct tag %s was removed", k))
		}
	}
	for k := range bmap {
		if _, ok := amap[k]; !ok {
			return Minor.wrap(fmt.Sprintf("struct tag %s was added", k))
		}
	}
	return None
}

// https://golang.org/ref/spec#Assignability
func (c *comparer) assignableTo(v, t types.Type) bool {
	if types.AssignableTo(v, t) {
		return true
	}

	// "x's type is identical to T"
	if c.identical(v, t) {
		return true
	}

	// "x's type V and T have identical underlying types
	// and at least one of V or T is not a defined type"
	uv, ut := v.Underlying(), t.Underlying()
	if c.identical(uv, ut) {
		if _, ok := v.(*types.Named); !ok {
			return true
		}
		if _, ok := t.(*types.Named); !ok {
			return true
		}
	}

	// "T is an interface type and x implements T"
	if intf, ok := ut.(*types.Interface); ok {
		if c.implements(v, intf) {
			return true
		}
	}

	if c.assignableChan(v, t, uv, ut) {
		return true
	}

	return c.assignableBasic(v, t, uv, ut)
}

func (c *comparer) assignableChan(v, t, uv, ut types.Type) bool {
	// "x is a bidirectional channel value,
	// T is a channel type,
	// x's type V and T have identical element types,
	// and at least one of V or T is not a defined type"
	if chv, ok := uv.(*types.Chan); ok && chv.Dir() == types.SendRecv {
		if cht, ok := ut.(*types.Chan); ok && c.identical(chv.Elem(), cht.Elem()) {
			if _, ok := v.(*types.Named); !ok {
				return true
			}
			if _, ok := t.(*types.Named); !ok {
				return true
			}
		}
	}
	return false
}

func (c *comparer) assignableBasic(v, t, uv, ut types.Type) bool {
	b, ok := v.(*types.Basic)
	if !ok {
		return false
	}

	// "x is the predeclared identifier nil
	// and T is a pointer, function, slice, map, channel, or interface type"
	if b.Kind() == types.UntypedNil {
		switch ut.(type) {
		case *types.Pointer:
			return true
		case *types.Signature:
			return true
		case *types.Slice:
			return true
		case *types.Map:
			return true
		case *types.Chan:
			return true
		case *types.Interface:
			return true
		}
	}

	// "x is an untyped constant representable by a value of type T"
	switch b.Kind() {
	case types.UntypedBool, types.UntypedInt, types.UntypedRune, types.UntypedFloat, types.UntypedComplex, types.UntypedString:
		return representable(b, t)
	}

	return false
}

// https://golang.org/ref/spec#Method_sets
func (c *comparer) implements(v types.Type, t *types.Interface) bool {
	if types.Implements(v, t) {
		return true
	}

	mv, mt := methodMap(v), methodMap(t)
	for tname, tfn := range mt {
		vfn, ok := mv[tname]
		if !ok {
			return false
		}
		if !c.identical(vfn.Type(), tfn.Type()) {
			return false
		}
	}

	return true
}

func (c *comparer) identicalTuples(a, b *types.Tuple) (identical, added1 bool) {
	identical, added1 = true, true
	la, lb := a.Len(), b.Len()
	if la != lb {
		if la+1 != lb {
			return false, false
		}
		identical = false
	}
	for i := 0; i < la; i++ {
		va, vb := a.At(i), b.At(i)
		if !c.identical(va.Type(), vb.Type()) {
			return false, false
		}
	}
	return identical, added1
}

func (c *comparer) samePackage(a, b *types.Package) bool {
	return a.Path() == b.Path()
}

// https://golang.org/ref/spec#Representability
// TODO: Add range checking of literals.
func representable(x *types.Basic, t types.Type) bool {
	tb, ok := t.Underlying().(*types.Basic)
	if !ok {
		return false
	}

	switch x.Kind() {
	case types.UntypedBool:
		return (tb.Info() & types.IsBoolean) == types.IsBoolean

	case types.UntypedInt:
		return (tb.Info() & types.IsNumeric) == types.IsNumeric

	case types.UntypedRune:
		switch tb.Kind() {
		case types.Int8, types.Int16, types.Uint8, types.Uint16:
			return false
		}
		return (tb.Info() & types.IsNumeric) == types.IsNumeric

	case types.UntypedFloat:
		if (tb.Info() & types.IsInteger) == types.IsInteger {
			return false
		}
		return (tb.Info() & types.IsNumeric) == types.IsNumeric

	case types.UntypedComplex:
		return (tb.Info() & types.IsComplex) == types.IsComplex

	case types.UntypedString:
		return (tb.Info() & types.IsString) == types.IsString
	}

	return false
}

func methodMap(t types.Type) map[string]types.Object {
	ms := types.NewMethodSet(t)
	result := make(map[string]types.Object)
	for i := 0; i < ms.Len(); i++ {
		fnobj := ms.At(i).Obj()
		result[fnobj.Name()] = fnobj
	}
	return result
}

func makePackageMap(pkgs []*packages.Package) map[string]*packages.Package {
	result := make(map[string]*packages.Package)
	for _, pkg := range pkgs {
		result[pkg.PkgPath] = pkg
	}
	return result
}

func makeTopObjs(pkg *packages.Package) map[string]types.Object {
	res := make(map[string]types.Object)
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					switch spec := spec.(type) {
					case *ast.ValueSpec:
						for _, name := range spec.Names {
							res[name.Name] = pkg.TypesInfo.Defs[name]
						}

					case *ast.TypeSpec:
						res[spec.Name.Name] = pkg.TypesInfo.Defs[spec.Name]
					}
				}

			case *ast.FuncDecl:
				res[decl.Name.Name] = pkg.TypesInfo.Defs[decl.Name]
			}
		}
	}

	return res
}

func structMap(t *types.Struct) map[string]int {
	result := make(map[string]int)
	for i := 0; i < t.NumFields(); i++ {
		f := t.Field(i)
		result[f.Name()] = i
	}
	return result
}

var tagRE = regexp.MustCompile(`([^ ":[:cntrl:]]+):"(([^"]|\\")*)"`)

func tagMap(inp string) map[string]string {
	res := make(map[string]string)
	matches := tagRE.FindAllStringSubmatch(inp, -1)
	for _, match := range matches {
		res[match[1]] = match[2]
	}
	return res
}
