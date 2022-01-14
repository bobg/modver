package modver

import "go/types"

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
			tpa, tpb := na.TypeParams(), nb.TypeParams()
			if tpa.Len() != tpb.Len() {
				return false
			}
			for i := 0; i < tpa.Len(); i++ {
				if !c.identical(tpa.At(i).Constraint(), tpb.At(i).Constraint()) {
					return false
				}
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
		return c.identicalChans(ua, ub)
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

func (c *comparer) identicalChans(ua *types.Chan, b types.Type) bool {
	ub, ok := b.(*types.Chan)
	if !ok {
		return false
	}

	// Two channel types are identical if they have identical element types and the same direction.
	if ua.Dir() != ub.Dir() {
		return false
	}
	return c.identical(ua.Elem(), ub.Elem())
}
