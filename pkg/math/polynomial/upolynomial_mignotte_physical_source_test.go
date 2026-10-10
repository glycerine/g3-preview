// Source: src/math/polynomial/upolynomial_factorization.cpp:979-1017;
// src/util/mpz.h; src/util/mpz.cpp; src/util/mpq.h.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func mignotteNamedIntegerPhysicalAudit(t *testing.T, m *unumeral.IntegerManager, external []umemory.Block, public ...*unumeral.Int) {
	t.Helper()
	cells := map[unumeral.NumericCellRef]bool{}
	add := func(r unumeral.NumericCellRef) {
		if r == 0 || m.IsFailureRef(r) {
			return
		}
		if !m.IsLiveCellToken(uint64(r)) {
			t.Fatal("invalid ordinary integer root", r)
		}
		cells[r] = true
	}
	for _, r := range m.Tmp {
		add(r)
	}
	for _, r := range m.Arg {
		add(r)
	}
	for i := range m.RawDisplay.Roots {
		add(m.RawDisplay.Roots[i].Cell)
	}
	if m.MatchesDividesNumeralInvocation(&m.DividesNumeral.Receipt) {
		add(m.DividesNumeral.Numeral.Cell)
	}
	for _, s := range []*unumeral.IntegerRawScope{&m.DivisionRaw, &m.ExtendedRaw, &m.GcdRaw} {
		for i := uint8(0); i < s.Count; i++ {
			add(s.Values[i].Cell)
		}
	}
	if m.MatchesAddMulInvocation(&m.GcdRaw.AddMul) || m.MatchesAddMulInvocation(&m.AddMulInvocation.Receipt) {
		add(m.AddMulInvocation.Tmp.Cell)
	}
	add(m.PowerRaw.Value.Cell)
	for i := range m.RootRaw.Values {
		add(m.RootRaw.Values[i].Cell)
	}
	for i := range m.PerfectSquare.Values {
		add(m.PerfectSquare.Values[i].Cell)
	}
	add(m.Smt2Display.Value.Cell)
	add(m.IntMin.Cell)
	add(m.Two64.Cell)
	for _, v := range public {
		if v != nil {
			add((v).Cell)
		}
	}
	if uint32(len(cells)) != m.Cells.Stats().Live {
		t.Fatal("unnamed actual integer descriptor", len(cells), m.Cells.Stats().Live)
	}
	objects := map[umemory.ObjectRef]bool{}
	for ref := range cells {
		c := m.Cell(ref)
		if c.Payload != (arena.VectorRef{}) {
			t.Fatal("phantom cell Words payload")
		}
		if c.Released {
			continue
		}
		object, e := m.CellObject(ref)
		if e != nil {
			t.Fatal("invalid actual object root", ref, e)
		}
		if objects[object] {
			t.Fatal("duplicate physical numeric object", object)
		}
		objects[object] = true
	}
	if m.PendingAllocationObject() != (umemory.ObjectRef{}) {
		if objects[m.PendingAllocationObject()] {
			t.Fatal("duplicated pending physical object")
		}
		if _, e := m.SourceAllocator().ObjectExtent(m.PendingAllocationObject()); e != nil {
			t.Fatal("invalid pending object", e)
		}
	}
	raw := map[umemory.Block]bool{}
	if m.SourceFinalized || m.ConstructorUnwound {
		for ref := range cells {
			c := m.Cell(ref)
			if !c.Released && c.Backing.Offset == 0 {
				if _, e := m.RuntimeOwner().Size(c.Backing.Raw); e != nil {
					t.Fatal("invalid named surviving direct source cell", e)
				}
				raw[c.Backing.Raw] = true
			}
		}
	}

	for _, block := range external {
		if block == (umemory.Block{}) || raw[block] {
			t.Fatal("duplicate or empty named external block", block)
		}
		if _, e := m.RuntimeOwner().Size(block); e != nil {
			t.Fatal("invalid actual external raw root", block, e)
		}
		raw[block] = true
	}
	allocator := m.SourceAllocator().Stats()
	if m.RuntimeOwner().Stats().LiveBlocks != allocator.Chunks+allocator.DirectObjects+allocator.PendingBlocks+uint32(len(raw)) {
		t.Fatal("unnamed physical runtime allocation", m.RuntimeOwner().Stats(), allocator, len(raw))
	}
}

func mignotteScopePhysicalAudit(t *testing.T, q *unumeral.RationalManager, s *UPolynomialFactorizationService, caller ...*unumeral.Int) {
	mignotteScopePhysicalAuditExtra(t, q, s, nil, caller...)
}
func mignotteScopePhysicalAuditExtra(t *testing.T, q *unumeral.RationalManager, s *UPolynomialFactorizationService, external []umemory.Block, caller ...*unumeral.Int) {
	u := s.manager
	c := s.scope
	roots := append([]*unumeral.Int{}, caller...)
	roots = append(roots, &q.NTmp, &q.AddTmp1, &q.AddTmp2, &q.AddMulTmp.Num, &q.AddMulTmp.Den, &q.LtTmp1.Num, &q.LtTmp1.Den, &q.LtTmp2.Num, &q.LtTmp2.Den, &(*u.modular.BorrowModulus()), &u.modular.Lower, &u.modular.Upper, &u.modular.InvTmp1, &u.modular.InvTmp2, &u.modular.InvTmp3, &u.modular.DivTmp)
	states := []*upolyVectorState{&u.db, &u.dbab1, &u.dbab2, &u.tr, &u.push, &u.basic, &u.div1, &u.div2, &u.exactDiv, &u.gcd1, &u.gcd2, &u.cra, &u.mgcd[0], &u.mgcd[1], &u.mgcd[2], &u.mgcd[3], &u.mgcd[4], &u.mgcd[5], &u.sqf1, &u.sqf2, &u.pw}
	if c != nil {
		states = append(states, &c.fpp, &c.trial, &c.quotient)
		roots = append(roots, &c.mignotte.norm, &c.mignotte.bound, &c.mignotte.tmp, &c.p, &c.bestP, &c.gcdTmp, &c.leading, &c.trialContent)
	}
	for _, v := range states {
		if v.backing != 0 {
			a := u.values(v)
			for i := uint32(0); uint64(i) < uint64(len(a)); i++ {
				roots = append(roots, &a[i])
			}
		}
	}
	mignotteNamedIntegerPhysicalAudit(t, q.Integers(), external, roots...)
	n := q.HeaderBytes + integerActualOwnedBytes(q.Integers()) + u.header + u.modular.Header + u.store.header + s.header + s.config.Primes.Stats().TotalBytes
	if c != nil {
		n += c.header
	}

	if n != q.Budget().Used() {
		t.Fatal("actual enclosing cold/backing equation", n, q.Budget().Used())
	}
}
