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

// https://golang.org/ref/spec#Type_identity
func (c *comparer) identical(a, b types.Type) (res bool) {
	if res, ok := c.cache[typePair{a, b}]; ok {
		return res
	}
	if res, ok := c.cache[typePair{b, a}]; ok {
		return res
	}
	doCache := true
	defer func() {
		if doCache {
			c.cache[typePair{a, b}] = res
		}
	}()

	if types.Identical(a, b) {
		return true
	}

	// Break any infinite regress,
	// e.g. when checking type Node struct { children []*Node }
	for _, pair := range c.stack {
		if a == pair.a && b == pair.b {
			doCache = false
			return true
		}
	}
	c.stack = append(c.stack, typePair{a: a, b: b})
	defer func() { c.stack = c.stack[:len(c.stack)-1] }()

	if na, ok := a.(*types.Named); ok {
		if nb, ok := b.(*types.Named); ok {
			if na.Obj().Name() != nb.Obj().Name() {
				return false
			}
			// Can't return true yet just because the types have equal names.
			// Continue to checking their underlying types.
		} else {
			return false
		}
	}

	ua, ub := a.Underlying(), b.Underlying()

	if types.Identical(ua, ub) {
		return true
	}

	return c.underlyingIdentical(ua, ub)
}

func (c *comparer) underlyingIdentical(ua, ub types.Type) bool {
	switch ua := ua.(type) {

	case *types.Array:
		// Two array types are identical if they have identical element types and the same array length.
		if ub, ok := ub.(*types.Array); ok {
			return ua.Len() == ub.Len() && c.identical(ua.Elem(), ub.Elem())
		}
		return false

	case *types.Slice:
		// Two slice types are identical if they have identical element types.
		if ub, ok := ub.(*types.Slice); ok {
			return c.identical(ua.Elem(), ub.Elem())
		}
		return false

	case *types.Struct:
		return c.identicalStructs(ua, ub)

	case *types.Pointer:
		// Two pointer types are identical if they have identical base types.
		if ub, ok := ub.(*types.Pointer); ok {
			return c.identical(ua.Elem(), ub.Elem())
		}
		return false

	case *types.Signature:
		// Two function types are identical if they have the same number of parameters and result values,
		// corresponding parameter and result types are identical,
		// and either both functions are variadic or neither is.
		// Parameter and result names are not required to match.
		if ub, ok := ub.(*types.Signature); ok {
			identical, _ := c.identicalSigs(ua, ub)
			return identical
		}
		return false

	case *types.Interface:
		return c.identicalInterfaces(ua, ub)

	case *types.Map:
		return c.identicalMaps(ua, ub)

	case *types.Chan:
		// Two channel types are identical if they have identical element types and the same direction.
		if ub, ok := ub.(*types.Chan); ok {
			if ua.Dir() != ub.Dir() {
				return false
			}
			return c.identical(ua.Elem(), ub.Elem())
		}
	}

	return false
}

func (c *comparer) identicalStructs(ua *types.Struct, b types.Type) bool {
	ub, ok := b.(*types.Struct)
	if !ok {
		return false
	}

	// Two struct types are identical if they have the same sequence of fields,
	// and if corresponding fields have the same names,
	// and identical types,
	// and identical tags.
	// Non-exported field names from different packages are always different.

	if ua.NumFields() != ub.NumFields() {
		return false
	}
	for i := 0; i < ua.NumFields(); i++ {
		if ua.Tag(i) != ub.Tag(i) {
			return false
		}

		fa, fb := ua.Field(i), ub.Field(i)

		if fa.Name() != fb.Name() {
			return false
		}
		if !fa.Exported() && !c.samePackage(fa.Pkg(), fb.Pkg()) {
			return false
		}
		if !c.identical(fa.Type(), fb.Type()) {
			return false
		}
	}
	return true
}

func (c *comparer) identicalSigs(older, newer *types.Signature) (identical, addedOptionalParams bool) {
	identical, addedOptionalParams = true, true
	if older.Variadic() {
		if !newer.Variadic() {
			return false, false
		}
		addedOptionalParams = false
	} else if newer.Variadic() {
		identical = false
	}

	resultsIdentical, _ := c.identicalTuples(older.Results(), newer.Results())
	if !resultsIdentical {
		return false, false
	}

	paramsIdentical, addedParam := c.identicalTuples(older.Params(), newer.Params())
	return identical && paramsIdentical, addedOptionalParams && addedParam
}

func (c *comparer) identicalInterfaces(ua *types.Interface, b types.Type) bool {
	ub, ok := b.(*types.Interface)
	if !ok {
		return false
	}

	// Two interface types are identical if they have the same set of methods with the same names and identical function types.
	// Non-exported method names from different packages are always different.
	// The order of the methods is irrelevant.

	if ua.NumMethods() != ub.NumMethods() { // Warning: this panics on incomplete interfaces.
		return false
	}

	ma, mb := methodMap(ua), methodMap(ub)

	for aname, afn := range ma {
		bfn, ok := mb[aname]
		if !ok {
			return false
		}
		if !afn.Exported() && !c.samePackage(afn.Pkg(), bfn.Pkg()) {
			return false
		}
		if !c.identical(afn.Type(), bfn.Type()) {
			return false
		}
	}
	return true
}

func (c *comparer) identicalMaps(ua *types.Map, b types.Type) bool {
	ub, ok := b.(*types.Map)
	if !ok {
		return false
	}

	// Two map types are identical if they have identical key and element types.
	if !c.identical(ua.Key(), ub.Key()) {
		return false
	}
	return c.identical(ua.Elem(), ub.Elem())
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
