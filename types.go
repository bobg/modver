package modver

import (
	"go/ast"
	"go/types"
	"regexp"

	"golang.org/x/tools/go/packages"
)

type (
	comparer struct {
		stack []typePair
		cache map[typePair]Result
	}
	typePair struct{ a, b types.Type }
)

func newComparer() *comparer {
	return &comparer{cache: make(map[typePair]Result)}
}

func (c *comparer) compareTypes(older, newer types.Type) (res Result) {
	pair := typePair{a: older, b: newer}
	if res, ok := c.cache[pair]; ok {
		if res == nil {
			// Break an infinite regress,
			// e.g. when checking type Node struct { children []*Node }
			return None
		}
		return res
	}

	c.cache[pair] = nil

	defer func() {
		c.cache[pair] = res
	}()

	switch older := older.(type) {
	case *types.Named:
		if newer, ok := newer.(*types.Named); ok {
			return c.compareNamed(older, newer)
		}
		// This is probably impossible.
		// How can newer not be *types.Named if older is,
		// and newer has the same name?
		return rwrapf(Major, "%s went from defined type to non-defined type", older)

	case *types.Struct:
		if newer, ok := newer.(*types.Struct); ok {
			return c.compareStructs(older, newer)
		}
		return rwrapf(Major, "%s went from struct to non-struct", older)

	case *types.Interface:
		if newer, ok := newer.(*types.Interface); ok {
			return c.compareInterfaces(older, newer)
		}
		return rwrapf(Major, "%s went from interface to non-interface", older)

	case *types.Signature:
		if newer, ok := newer.(*types.Signature); ok {
			return c.compareSignatures(older, newer)
		}
		return rwrapf(Major, "%s went from function to non-function", older)

	default:
		if !c.assignableTo(newer, older) {
			return rwrapf(Major, "%s is not assignable to %s", newer, older)
		}
		return None
	}
}

func (c *comparer) compareNamed(older, newer *types.Named) Result {
	res := c.compareTypeParamLists(older.TypeParams(), newer.TypeParams())
	if r := c.compareTypes(older.Underlying(), newer.Underlying()); r.Code() > res.Code() {
		res = r
	}

	return rwrapf(res, "in type %s", older)
}

func (c *comparer) compareStructs(older, newer *types.Struct) Result {
	var (
		olderMap = structMap(older)
		newerMap = structMap(newer)
	)

	var res Result = None

	for i := 0; i < older.NumFields(); i++ {
		field := older.Field(i)
		newFieldIndex, ok := newerMap[field.Name()]
		if !ok {
			return rwrapf(Major, "old struct field %s was removed from %s", field.Name(), older)
		}
		newField := newer.Field(newFieldIndex)

		if r := c.compareTypes(field.Type(), newField.Type()); r.Code() > res.Code() {
			res = rwrapf(r, "struct field %s changed in %s", field.Name(), older)
			if res.Code() == Major {
				return res
			}
		}

		var (
			tag    = older.Tag(i)
			newTag = newer.Tag(newFieldIndex)
		)
		if r := c.compareStructTags(tag, newTag); r.Code() == Major {
			return rwrapf(r, "tag change in field %s of %s", field.Name(), older)
		}
	}

	for i := 0; i < newer.NumFields(); i++ {
		field := newer.Field(i)
		oldFieldIndex, ok := olderMap[field.Name()]
		if !ok {
			return rwrapf(Minor, "struct field %s was added to %s", field.Name(), newer)
		}
		var (
			oldTag = older.Tag(oldFieldIndex)
			tag    = newer.Tag(i)
		)
		if res := c.compareStructTags(oldTag, tag); res.Code() == Minor {
			return rwrapf(res, "tag change in field %s of %s", field.Name(), older)
		}
	}

	if !c.identical(older, newer) {
		return rwrapf(Patchlevel, "old and new versions of %s are not identical", older)
	}

	return None
}

func (c *comparer) compareInterfaces(older, newer *types.Interface) Result {
	var res Result = None

	if c.implements(newer, older) {
		if !c.implements(older, newer) {
			res = rwrapf(Minor, "new interface %s is a superset of older", newer)
		}
	} else {
		return rwrapf(Major, "new interface %s does not implement old", newer)
	}

	if older.IsMethodSet() {
		if newer.IsMethodSet() {
			return res
		}
		return rwrap(Major, "new interface is a constraint, old one is not")
	}
	if newer.IsMethodSet() {
		return rwrap(Major, "old interface is a constraint, new one is not")
	}

	return rwrap(Major, "comparison of constraint type sets not yet implemented")
}

func (c *comparer) compareSignatures(older, newer *types.Signature) Result {
	var (
		typeParamsRes = c.compareTypeParamLists(older.TypeParams(), newer.TypeParams())
		paramsRes     = c.compareTuples(older.Params(), newer.Params(), !older.Variadic() && newer.Variadic())
		resultsRes    = c.compareTuples(older.Results(), newer.Results(), false)
	)

	res := rwrapf(typeParamsRes, "in type parameters of %s", older)
	if paramsRes.Code() > res.Code() {
		res = rwrapf(paramsRes, "in parameters of %s", older)
	}
	if resultsRes.Code() > res.Code() {
		res = rwrapf(resultsRes, "in results of %s", older)
	}
	return res
}

func (c *comparer) compareTuples(older, newer *types.Tuple, variadicCheck bool) Result {
	la, lb := older.Len(), newer.Len()

	maybeVariadic := variadicCheck && (la+1 == lb)

	if la != lb && !maybeVariadic {
		return rwrapf(Major, "%d param(s) to %d", la, lb)
	}

	var res Result = None
	for i := 0; i < la; i++ {
		va, vb := older.At(i), newer.At(i)
		thisRes := c.compareTypes(va.Type(), vb.Type())
		if thisRes.Code() == Major {
			return thisRes
		}
		if thisRes.Code() > res.Code() {
			res = thisRes
		}
	}

	if res.Code() < Minor && maybeVariadic {
		return rwrap(Minor, "added optional parameters")
	}
	return res
}

func (c *comparer) compareTypeParamLists(older, newer *types.TypeParamList) (res Result) {
	if older.Len() != newer.Len() {
		return rwrapf(Major, "went from %d type parameters to %d", older.Len(), newer.Len())
	}

	res = None
	for i := 0; i < older.Len(); i++ {
		thisRes := c.compareTypes(older.At(i).Constraint(), newer.At(i).Constraint())
		if thisRes.Code() > res.Code() {
			res = thisRes
			if res.Code() == Major {
				break
			}
		}
	}

	return res
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
				return rwrapf(Major, `struct tag changed the value for key "%s" from "%s" to "%s"`, k, av, bv)
			}
		} else {
			return rwrapf(Major, "struct tag %s was removed", k)
		}
	}
	for k := range bmap {
		if _, ok := amap[k]; !ok {
			return rwrapf(Minor, "struct tag %s was added", k)
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
