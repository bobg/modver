package modver

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
)

type (
	comparer struct {
		samePackagePath func(a, b string) bool
		identicalStack  []typePair
	}
	typePair struct{ a, b types.Type }
)

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

// https://golang.org/ref/spec#Assignability
func (c *comparer) assignableTo(v, t types.Type) bool {
	if types.AssignableTo(v, t) {
		return true
	}

	// "x's type is identical to T"
	if c.identical(v, t) {
		return true
	}

	// "x's type V and T have identical underlying types and at least one of V or T is not a defined type"
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

	// "x is a bidirectional channel value, T is a channel type, x's type V and T have identical element types, and at least one of V or T is not a defined type"
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

	if b, ok := v.(*types.Basic); ok {
		// "x is the predeclared identifier nil and T is a pointer, function, slice, map, channel, or interface type"
		if b.Kind() == types.UntypedNil { // xxx ?
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
			if representable(b, t) {
				return true
			}
		}
	}

	return false
}

// https://golang.org/ref/spec#Type_identity
func (c *comparer) identical(a, b types.Type) bool {
	// indent := strings.Repeat("  ", c.depth)
	// fmt.Printf("xxx %sidentical:\n%s%s\n%svs.\n%s%s\n", indent, indent, a, indent, indent, b)
	// c.depth++
	// defer func() { c.depth-- }()

	if types.Identical(a, b) {
		return true
	}

	// Break any infinite regress,
	// e.g. when checking type Node struct { children []*Node }
	for _, pair := range c.identicalStack {
		if a == pair.a && b == pair.b {
			return true
		}
	}
	c.identicalStack = append(c.identicalStack, typePair{a: a, b: b})
	defer func() { c.identicalStack = c.identicalStack[:len(c.identicalStack)-1] }()

	// xxx check for defined types first

	ua, ub := a.Underlying(), b.Underlying()

	if types.Identical(ua, ub) {
		return true
	}

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
		// Two struct types are identical if they have the same sequence of fields, and if corresponding fields have the same names, and identical types, and identical tags. Non-exported field names from different packages are always different.
		if ub, ok := ub.(*types.Struct); ok {
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
		return false

	case *types.Pointer:
		// Two pointer types are identical if they have identical base types.
		if ub, ok := ub.(*types.Pointer); ok {
			return c.identical(ua.Elem(), ub.Elem())
		}
		return false

	case *types.Signature:
		// Two function types are identical if they have the same number of parameters and result values, corresponding parameter and result types are identical, and either both functions are variadic or neither is. Parameter and result names are not required to match.
		if ub, ok := ub.(*types.Signature); ok {
			if ua.Variadic() != ub.Variadic() {
				return false
			}
			if !c.tupleTypesIdentical(ua.Params(), ub.Params()) {
				return false
			}
			if !c.tupleTypesIdentical(ua.Results(), ub.Results()) {
				return false
			}
			return true
		}
		return false

	case *types.Interface:
		// Two interface types are identical if they have the same set of methods with the same names and identical function types. Non-exported method names from different packages are always different. The order of the methods is irrelevant.
		if ub, ok := ub.(*types.Interface); ok {
			if ua.NumMethods() != ub.NumMethods() { // xxx panics on incomplete interfaces
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
		return false

	case *types.Map:
		// Two map types are identical if they have identical key and element types.
		if ub, ok := ub.(*types.Map); ok {
			if !c.identical(ua.Key(), ub.Key()) {
				return false
			}
			return c.identical(ua.Elem(), ub.Elem())
		}
		return false

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

func (c *comparer) tupleTypesIdentical(a, b *types.Tuple) bool {
	if a.Len() != b.Len() {
		return false
	}
	for i := 0; i < a.Len(); i++ {
		va, vb := a.At(i), b.At(i)
		if !c.identical(va.Type(), vb.Type()) {
			return false
		}
	}
	return true
}

func (c *comparer) samePackage(a, b *types.Package) bool {
	return c.samePackagePath(a.Path(), b.Path())
}

// https://golang.org/ref/spec#Representability
// xxx no range checking of literals here
func representable(x *types.Basic, t types.Type) bool {
	tb, ok := t.(*types.Basic) // xxx use t.Underlying() here?
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

func makePackageMap(pkgs []*packages.Package, keyFn func(string) string) map[string]*packages.Package {
	result := make(map[string]*packages.Package)
	for _, pkg := range pkgs {
		result[keyFn(pkg.PkgPath)] = pkg
	}
	return result
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
