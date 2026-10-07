// Source: src/util/approx_set.h:43-154; src/smt/smt_literal.h:114-116.
// Source author: Leonardo de Moura (2007-03-02; 2008-02-18).
// Licensed under the MIT license; see LICENSE.txt.
package smtcore

import (
	"reflect"
	"testing"
)

func TestSMTLiteralApproxSetSource64CollisionAndSentinels(t *testing.T) {
	var s SMTLiteralApproxSet
	if !s.Empty() || reflect.TypeOf(s).Size() != 8 {
		t.Fatal("source default/literal specialization layout")
	}
	s.Insert(SMTLiteralFromIndex(0))
	if !s.MayContain(SMTLiteralFromIndex(64)) || s.MayContain(SMTLiteralFromIndex(32)) {
		t.Fatal("64-bit collisions differ from SAT32")
	}
	s.Insert(DefaultSMTLiteral())
	s.Insert(DefaultSMTLiteral().Negated())
	if !s.MayContain(SMTLiteralFromIndex(62)) || !s.MayContain(SMTLiteralFromIndex(63)) {
		t.Fatal("source unsigned negative sentinel conversion")
	}
	for _, raw := range []int32{-2147483648, -2, -1, 0, 1, 2147483647} {
		if SMTLiteralToUnsigned(SMTLiteralFromIndex(raw)) != uint32(raw) {
			t.Fatal(raw)
		}
	}
	copy := s
	copy.Reset()
	if !copy.Empty() || s.Empty() {
		t.Fatal("source copied inline set")
	}
	one := SMTLiteralApproxSingleton(SMTLiteralFromIndex(32))
	if !one.MayContain(SMTLiteralFromIndex(96)) || one.MayContain(SMTLiteralFromIndex(0)) {
		t.Fatal("single constructor")
	}
	all := SMTLiteralApproxFrom([]SMTLiteral{SMTLiteralFromIndex(0), SMTLiteralFromIndex(64), DefaultSMTLiteral(), DefaultSMTLiteral().Negated()})
	if !all.Equivalent(s) || !all.MayEqual(s) || all.MustNotEqual(s) {
		t.Fatal("duplicate/collision range constructor")
	}
}

func TestSMTLiteralApproxSetIndependentFiniteResidueAlgebra(t *testing.T) {
	values := []int32{-2, -1, 0, 1, 32, 63, 64, 127}
	var sets [256]SMTLiteralApproxSet
	var oracle [256][64]bool
	for mask := 0; mask < 256; mask++ {
		var row []SMTLiteral
		for i, raw := range values {
			if mask&(1<<i) != 0 {
				row = append(row, SMTLiteralFromIndex(raw))
				oracle[mask][uint32(raw)%64] = true
			}
		}
		sets[mask] = SMTLiteralApproxFrom(row)
	}
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			union := sets[a].Union(sets[b])
			intersection := sets[a].Intersection(sets[b])
			difference := sets[a]
			difference.SubtractAssign(sets[b])
			u := sets[a]
			u.UnionAssign(sets[b])
			i := sets[a]
			i.IntersectAssign(sets[b])
			if u != union || i != intersection {
				t.Fatal("in-place source operators")
			}
			subset, equal, emptyIntersection := true, true, true
			for residue := 0; residue < 64; residue++ {
				av, bv := oracle[a][residue], oracle[b][residue]
				l := SMTLiteralFromIndex(int32(residue))
				if union.MayContain(l) != (av || bv) || intersection.MayContain(l) != (av && bv) || difference.MayContain(l) != (av && !bv) || sets[a].MustNotContain(l) == av {
					t.Fatal(a, b, residue)
				}
				if av && !bv {
					subset = false
				}
				if av != bv {
					equal = false
				}
				if av && bv {
					emptyIntersection = false
				}
			}
			if sets[a].ApproxSubset(sets[b]) != subset || sets[a].MustNotSubset(sets[b]) == subset || sets[a].MustNotSubsume(sets[b]) == subset || sets[a].Equivalent(sets[b]) != equal || sets[a].MayEqual(sets[b]) != equal || sets[a].MustNotEqual(sets[b]) == equal || sets[a].EmptyIntersection(sets[b]) != emptyIntersection {
				t.Fatal("source approximate predicates", a, b)
			}
		}
	}
	if !SMTLiteralApproxFrom(nil).Empty() {
		t.Fatal("empty range constructor")
	}
}
