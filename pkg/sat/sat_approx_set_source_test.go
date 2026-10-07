// Source: src/util/approx_set.h:43-158; src/sat/sat_types.h:115-119;
// src/sat/sat_solver.h:419; src/smt/smt_conflict_resolution.h:34.
package sat

import (
	"reflect"
	"testing"
)

func TestSATApproxSetAllSourceConstructorsAndLiteralMapping(t *testing.T) {
	var empty satApproxSet
	if !empty.empty() || reflect.TypeOf(empty).Size() != 4 {
		t.Fatal("source32 default word")
	}
	for _, raw := range []uint32{0, 1, 31, 32, 63, 64, ^uint32(0) - 1, ^uint32(0)} {
		s := newSATApproxSet(raw)
		if !s.mayContain(raw%32) || !s.mayEqual(satApproxSetFromUnsigned([]uint32{raw, raw + 32})) {
			t.Fatal(raw, s)
		}
		lit := Literal(raw)
		a := satApproxSetFromLiterals([]Literal{lit, lit})
		if !a.equivalent(s) || !a.mayContainLiteral(lit) {
			t.Fatal("literal2unsigned uses entire index, not variable", raw, a, s)
		}
		var b satApproxSet
		b.insertLiteral(lit)
		if b.mustNotEqual(a) {
			t.Fatal("actual literal conversion", raw)
		}
	}
	a := satApproxSetFromLiterals([]Literal{Literal(0), Literal(1)})
	if !a.mayContain(0) || !a.mayContain(1) || a.mayContain(2) {
		t.Fatal("positive and negative source indices remain distinct")
	}
	variables := []BoolVar{0, 31, 32, NullBoolVar}
	v := satApproxSetFromVariables(variables)
	if !v.equivalent(satApproxSetFromUnsigned([]uint32{0, 31, 32, uint32(NullBoolVar)})) {
		t.Fatal("bool_var uses identity unsigned conversion")
	}
	if !satApproxSetFromLiterals([]Literal{DefaultLiteral()}).mayContain(30) || satApproxSetFromLiterals([]Literal{DefaultLiteral()}).mayContain(31) {
		t.Fatal("literal null index differs from null Boolean variable residue")
	}
	copy := a
	copy.reset()
	if !copy.empty() || a.empty() {
		t.Fatal("source value copy")
	}
}

func TestSATApproxSetIndependentCompleteFiniteAlgebra(t *testing.T) {
	values := []uint32{0, 1, 9, 15, 16, 31, 32, ^uint32(0)}
	var sets [256]satApproxSet
	var oracle [256][32]bool
	for mask := 0; mask < 256; mask++ {
		var input []uint32
		for i, v := range values {
			if mask&(1<<i) != 0 {
				input = append(input, v)
				oracle[mask][v%32] = true
			}
		}
		sets[mask] = satApproxSetFromUnsigned(input)
	}
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			union := sets[a].unionValue(sets[b])
			intersection := sets[a].intersectionValue(sets[b])
			ua := sets[a]
			ua.union(sets[b])
			ia := sets[a]
			ia.intersect(sets[b])
			difference := sets[a]
			difference.subtract(sets[b])
			if ua != union || ia != intersection {
				t.Fatal("source value and mutation operations", a, b)
			}
			subset, equal, disjoint, empty := true, true, true, true
			for r := 0; r < 32; r++ {
				av, bv := oracle[a][r], oracle[b][r]
				if union.mayContain(uint32(r)) != (av || bv) || intersection.mayContain(uint32(r)) != (av && bv) || difference.mayContain(uint32(r)) != (av && !bv) || sets[a].mustNotContain(uint32(r)) == av {
					t.Fatal(a, b, r)
				}
				if av && !bv {
					subset = false
				}
				if av != bv {
					equal = false
				}
				if av && bv {
					disjoint = false
				}
				if av {
					empty = false
				}
			}
			if sets[a].mustNotSubset(sets[b]) == subset || sets[a].mustNotSubsume(sets[b]) == subset || sets[a].approxSubset(sets[b]) != subset || sets[a].mayEqual(sets[b]) != equal || sets[a].mustNotEqual(sets[b]) == equal || sets[a].equivalent(sets[b]) != equal || sets[a].emptyIntersection(sets[b]) != disjoint || sets[a].empty() != empty {
				t.Fatal("independent residue predicates", a, b)
			}
		}
	}
}
