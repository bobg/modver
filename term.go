package modver

// This file duplicates logic from go/types that is sadly unexported.

import "go/types"

// termSubset reports whether x ⊆ y.
func (c *Comparer) termSubset(x, y *types.Term) bool {
	// easy cases
	switch {
	case x == nil:
		return true // ∅ ⊆ y == true
	case y == nil:
		return false // x ⊆ ∅ == false since x != ∅
	case y.Type() == nil:
		return true // x ⊆ 𝓤 == true
	case x.Type() == nil:
		return false // 𝓤 ⊆ y == false since y != 𝓤
	}
	// ∅ ⊂ x, y ⊂ 𝓤

	if c.termDisjoint(x, y) {
		return false // x ⊆ y == false if x ∩ y == ∅
	}
	// x.typ == y.typ

	// ~t ⊆ ~t == true
	// ~t ⊆ T == false
	//  T ⊆ ~t == true
	//  T ⊆  T == true
	return !x.Tilde() || y.Tilde()
}

// termDisjoint reports whether x ∩ y == ∅.
// x.typ and y.typ must not be nil.
func (c *Comparer) termDisjoint(x, y *types.Term) bool {
	ux := x.Type()
	if y.Tilde() {
		ux = ux.Underlying()
	}
	uy := y.Type()
	if x.Tilde() {
		uy = uy.Underlying()
	}
	return !c.identical(ux, uy)
}

// termListSubset reports whether xl ⊆ yl.
func (c *Comparer) termListSubset(xl, yl []*types.Term) bool {
	if len(yl) == 0 {
		return len(xl) == 0
	}

	// each term x of xl must be a subset of yl
	for _, x := range xl {
		if !c.termListSuperset(yl, x) {
			return false // x is not a subset yl
		}
	}
	return true
}

// termListSuperset reports whether y ⊆ xl.
func (c *Comparer) termListSuperset(xl []*types.Term, y *types.Term) bool {
	for _, x := range xl {
		if c.termSubset(y, x) {
			return true
		}
	}
	return false
}
