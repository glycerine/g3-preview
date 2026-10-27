package nlsat

import (
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func atomPolynomialFixture(t *testing.T) (*polynomial.MultivariatePolynomialCore, []polynomial.Polynomial) {
	t.Helper()
	am, budget := assignmentFixture(t)
	q, err := am.RationalManager()
	assignmentMust(t, err)
	mm, err := polynomial.NewMonomialManager(polynomial.MonomialManagerConfig{Budget: budget, Memory: q.Integers().RuntimeOwner(), Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: diagnostic.DebugBuild}})
	assignmentMust(t, err)
	assignmentMust(t, mm.IncManagerRef())
	t.Cleanup(func() { assignmentMust(t, mm.DecManagerRef()) })
	pm, err := polynomial.NewMultivariatePolynomialCore(polynomial.MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }})
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, pm.CloseChecked()) })
	var ps []polynomial.Polynomial
	for i := 0; i < 12; i++ {
		x, err := mm.MkVar()
		assignmentMust(t, err)
		p, err := pm.MkVariable(x, 1)
		assignmentMust(t, err)
		assignmentMust(t, pm.IncRef(p))
		t.Cleanup(func() { assignmentMust(t, pm.DecRef(p)) })
		ps = append(ps, p)
	}
	return pm, ps
}

func TestAtomKindsHeadersAndSignNormalization(t *testing.T) {
	for _, test := range []struct {
		k              AtomKind
		text           string
		eq, ineq, root bool
	}{
		{AtomEQ, "=", true, true, false}, {AtomLT, "<", false, true, false}, {AtomGT, ">", false, true, false},
		{AtomRootEQ, "= root", true, false, true}, {AtomRootLT, "< root", false, false, true}, {AtomRootGT, "> root", false, false, true},
		{AtomRootLE, "<= root", false, false, true}, {AtomRootGE, ">= root", false, false, true}, {7, "7", false, false, false}, {-1, "-1", false, true, false},
	} {
		var a Atom
		a.initialize(test.k, 29)
		if a.Kind() != test.k || test.k.String() != test.text || a.IsEquality() != test.eq || a.IsInequality() != test.ineq || a.IsRoot() != test.root {
			t.Fatal("kind", test)
		}
		if a.MaxVariable() != 29 || a.BooleanVariable() != sat.NullBoolVar || a.RefCount() != 0 {
			t.Fatal("initial header", a)
		}
		a.IncRef()
		a.IncRef()
		assignmentMust(t, a.DecRef())
		if a.RefCount() != 1 {
			t.Fatal("reference header", a)
		}
		a.refs = math.MaxUint32
		a.IncRef()
		if a.RefCount() != 0 {
			t.Fatal("source uint32 wrap", a)
		}
	}
	for _, k := range []AtomKind{AtomEQ, AtomLT, AtomGT} {
		flipped, err := k.Flip()
		assignmentMust(t, err)
		again, err := flipped.Flip()
		assignmentMust(t, err)
		if again != k || (k != AtomEQ && flipped == k) {
			t.Fatal("flip", k, flipped, again)
		}
	}
	for _, s := range []int{math.MinInt, -1234567, -1, 0, 1, 1234567, math.MaxInt} {
		n := normalizeSign(s)
		if n < -1 || n > 1 || (n < 0) != (s < 0) || (n == 0) != (s == 0) {
			t.Fatal("sign", s, n)
		}
	}
	if TrueLiteral.Variable() != TrueBooleanVariable || TrueLiteral.Sign() || FalseLiteral != TrueLiteral.Negated() || NullVariable != math.MaxUint32 {
		t.Fatal("literal sentinels")
	}
}

func TestInequalityAtomIdentityParityBorrowingAndHash(t *testing.T) {
	pm, polys := atomPolynomialFixture(t)
	duplicate, err := pm.MkVariable(0, 1)
	assignmentMust(t, err)
	assignmentMust(t, pm.IncRef(duplicate))
	t.Cleanup(func() { assignmentMust(t, pm.DecRef(duplicate)) })
	if duplicate == polys[0] {
		t.Fatal("fixture requires distinct cells for the same polynomial")
	}
	for n := 1; n <= len(polys); n++ {
		ps := append([]polynomial.Polynomial(nil), polys[:n]...)
		even := make([]bool, n)
		for i := range even {
			even[i] = i%2 == 0
		}
		for _, kind := range []AtomKind{AtomEQ, AtomLT, AtomGT} {
			var a, b InequalityAtom
			assignmentMust(t, a.initialize(kind, ps, even, 11, make([]InequalityFactor, n)))
			assignmentMust(t, b.initialize(kind, ps, even, 999, make([]InequalityFactor, n)))
			b.boolVar = 91
			b.IncRef()
			if !a.Equal(&b) || a.Size() != uint32(n) {
				t.Fatal("identity depends on metadata", n, kind)
			}
			h, err := a.Hash(pm)
			assignmentMust(t, err)
			hb, err := b.Hash(pm)
			assignmentMust(t, err)
			if h != hb {
				t.Fatal("equal hashes")
			}
			b.factors[n-1].Even = !b.factors[n-1].Even
			hb, err = b.Hash(pm)
			assignmentMust(t, err)
			if a.Equal(&b) || h != hb {
				t.Fatal("parity must affect equality but not hashing", n, kind)
			}
			b.factors[n-1].Even = !b.factors[n-1].Even
			for i := range ps {
				p, err := a.Polynomial(uint32(i))
				assignmentMust(t, err)
				odd, err := a.IsOdd(uint32(i))
				assignmentMust(t, err)
				isEven, err := a.IsEven(uint32(i))
				assignmentMust(t, err)
				if p != polys[i] || odd == even[i] || isEven != even[i] {
					t.Fatal("factor", n, i)
				}
				info, err := pm.Info(p)
				assignmentMust(t, err)
				if info.RefCount != 1 {
					t.Fatal("atom constructor acquired polynomial refs", info)
				}
			}
			b.factors[0].Polynomial = polys[len(polys)-1]
			if a.Equal(&b) {
				t.Fatal("polynomial identity ignored")
			}
		}
		// The placement constructor copies inputs, rather than borrowing either
		// caller input array. Only the explicitly supplied destination is shared.
		var a InequalityAtom
		storage := make([]InequalityFactor, n)
		assignmentMust(t, a.initialize(AtomEQ, ps, even, 11, storage))
		ps[0] = polynomial.Polynomial{}
		even[0] = !even[0]
		if storage[0].Polynomial != polys[0] || storage[0].Even != true {
			t.Fatal("input alias")
		}
	}
	var a, b InequalityAtom
	assignmentMust(t, a.initialize(AtomLT, polys[:2], []bool{false, true}, 11, make([]InequalityFactor, 2)))
	assignmentMust(t, b.initialize(AtomLT, []polynomial.Polynomial{duplicate, polys[1]}, []bool{false, true}, 11, make([]InequalityFactor, 2)))
	if a.Equal(&b) {
		t.Fatal("structural equality replaced polynomial cell identity")
	}
	b.factors[0] = a.factors[1]
	b.factors[1] = a.factors[0]
	if a.Equal(&b) {
		t.Fatal("factor order ignored")
	}
	b.factors[0] = a.factors[0]
	b.factors[1] = a.factors[1]
	b.kind = AtomGT
	if a.Equal(&b) {
		t.Fatal("inequality kind ignored")
	}
	ha, err := a.Hash(pm)
	assignmentMust(t, err)
	hb, err := b.Hash(pm)
	assignmentMust(t, err)
	if ha == hb {
		t.Fatal("inequality hash kind ignored")
	}
}

func TestRootAtomIdentityAndSourceHashCollisions(t *testing.T) {
	pm, ps := atomPolynomialFixture(t)
	for _, k := range []AtomKind{AtomRootEQ, AtomRootLT, AtomRootGT, AtomRootLE, AtomRootGE} {
		var a, b RootAtom
		assignmentMust(t, a.initialize(k, 11, 1, ps[0]))
		assignmentMust(t, b.initialize(k, 11, 1, ps[0]))
		b.refs = 77
		b.boolVar = 19
		if !a.Equal(&b) || a.Variable() != 11 || a.MaxVariable() != 11 || a.Index() != 1 || a.Polynomial() != ps[0] {
			t.Fatal("root identity/header")
		}
		h, err := a.Hash(pm)
		assignmentMust(t, err)
		// Source shifts by two in a uint32 before ORing the kind: the top two
		// index bits disappear from hashing, but remain part of equality.
		b.index |= 0x40000000
		hb, err := b.Hash(pm)
		assignmentMust(t, err)
		if a.Equal(&b) || h != hb {
			t.Fatal("root index collision semantics")
		}
		b.index = 1
		b.polynomial = ps[1]
		if a.Equal(&b) {
			t.Fatal("root polynomial ignored")
		}
		info, err := pm.Info(ps[0])
		assignmentMust(t, err)
		if info.RefCount != 1 {
			t.Fatal("root constructor acquired polynomial refs")
		}
	}
	var a, b RootAtom
	assignmentMust(t, a.initialize(AtomRootEQ, 11, 1, ps[0]))
	assignmentMust(t, b.initialize(AtomRootGE, 11, 1, ps[0]))
	h, err := a.Hash(pm)
	assignmentMust(t, err)
	hb, err := b.Hash(pm)
	assignmentMust(t, err)
	if a.Equal(&b) || h != hb {
		t.Fatal("source kind/index OR collision")
	}
}
