// Source: src/math/polynomial/upolynomial_factorization.cpp:979-1017;
// src/util/mpz.h; src/util/mpz.cpp; src/util/mpq.h.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func mignotteNamedIntegerPhysicalAudit(t *testing.T, m *IntegerManager, external []memory.Block, public ...*Int) {
	t.Helper()
	cells := map[numericCellRef]bool{}
	add := func(r numericCellRef) {
		if r == 0 || m.isFailureRef(r) {
			return
		}
		if !validNumericRef(m.cells, r) {
			t.Fatal("invalid ordinary integer root", r)
		}
		cells[r] = true
	}
	for _, r := range m.tmp {
		add(r)
	}
	for _, r := range m.arg {
		add(r)
	}
	for i := range m.rawDisplay.roots {
		add(m.rawDisplay.roots[i].cell)
	}
	if m.matchesDividesNumeralInvocation(&m.dividesNumeral.receipt) {
		add(m.dividesNumeral.numeral.cell)
	}
	for _, s := range []*integerRawScope{&m.divisionRaw, &m.extendedRaw, &m.gcdRaw} {
		for i := uint8(0); i < s.count; i++ {
			add(s.values[i].cell)
		}
	}
	if m.matchesAddMulInvocation(&m.gcdRaw.addMul) || m.matchesAddMulInvocation(&m.addMulInvocation.receipt) {
		add(m.addMulInvocation.tmp.cell)
	}
	add(m.powerRaw.value.cell)
	for i := range m.rootRaw.values {
		add(m.rootRaw.values[i].cell)
	}
	for i := range m.perfectSquare.values {
		add(m.perfectSquare.values[i].cell)
	}
	add(m.smt2Display.value.cell)
	add(m.intMin.cell)
	add(m.two64.cell)
	for _, v := range public {
		if v != nil {
			add(v.cell)
		}
	}
	if uint32(len(cells)) != m.cells.Stats().Live {
		t.Fatal("unnamed actual integer descriptor", len(cells), m.cells.Stats().Live)
	}
	objects := map[memory.ObjectRef]bool{}
	for ref := range cells {
		c := m.cell(ref)
		if c.payload != (arena.VectorRef{}) {
			t.Fatal("phantom cell Words payload")
		}
		if c.released {
			continue
		}
		object, e := m.cellObject(ref)
		if e != nil {
			t.Fatal("invalid actual object root", ref, e)
		}
		if objects[object] {
			t.Fatal("duplicate physical numeric object", object)
		}
		objects[object] = true
	}
	if m.pendingCell.object != (memory.ObjectRef{}) {
		if objects[m.pendingCell.object] {
			t.Fatal("duplicated pending physical object")
		}
		if _, e := m.allocator.ObjectExtent(m.pendingCell.object); e != nil {
			t.Fatal("invalid pending object", e)
		}
	}
	words := map[arena.VectorRef]bool{}
	for _, r := range m.natural.formatScratch {
		if r == (arena.VectorRef{}) {
			continue
		}
		if !m.words.IsLive(r) {
			t.Fatal("invalid format Words root")
		}
		words[r] = true
	}
	if uint32(len(words)) != m.words.Stats().Live {
		t.Fatal("unnamed real formatting Words root", len(words), m.words.Stats().Live)
	}
	raw := map[memory.Block]bool{}
	if m.sourceFinalized || m.constructorUnwound {
		for ref := range cells {
			c := m.cell(ref)
			if !c.released && c.backing.offset == 0 {
				if _, e := m.memory.Size(c.backing.raw); e != nil {
					t.Fatal("invalid named surviving direct source cell", e)
				}
				raw[c.backing.raw] = true
			}
		}
	}

	buffers := []*sourceLimbBuffer{&m.natural.division[0], &m.natural.division[1], &m.natural.division[2], &m.natural.division[3]}
	if m.natural.verificationActive {
		buffers = append(buffers, &m.natural.verification)
	}
	if m.natural.formatting.active || m.natural.formatting.pending {
		for i := range m.natural.formatting.locals {
			buffers = append(buffers, &m.natural.formatting.locals[i])
		}
	}
	for _, s := range buffers {
		for _, r := range []memory.Block{s.raw, s.pending} {
			if r == (memory.Block{}) {
				continue
			}
			if raw[r] {
				t.Fatal("duplicate actual Natural raw root")
			}
			raw[r] = true
			if _, e := m.memory.ReadUint32(r, 0); e != nil && !errors.Is(e, memory.ErrRuntimeUninitialized) {
				t.Fatal("invalid Natural raw root", e)
			}
		}
	}
	for _, r := range []memory.Block{m.natural.charFormatting.buffer.raw, m.natural.charFormatting.buffer.pending} {
		if r == (memory.Block{}) {
			continue
		}
		if raw[r] {
			t.Fatal("duplicate character raw root")
		}
		raw[r] = true
		if _, e := m.memory.Read(r, 0); e != nil && !errors.Is(e, memory.ErrRuntimeUninitialized) {
			t.Fatal("invalid character raw root", e)
		}
	}
	for _, block := range external {
		if block == (memory.Block{}) || raw[block] {
			t.Fatal("duplicate or empty named external block", block)
		}
		if _, e := m.memory.Size(block); e != nil {
			t.Fatal("invalid actual external raw root", block, e)
		}
		raw[block] = true
	}
	allocator := m.allocator.Stats()
	if m.memory.Stats().LiveBlocks != allocator.Chunks+allocator.DirectObjects+allocator.PendingBlocks+uint32(len(raw)) {
		t.Fatal("unnamed physical runtime allocation", m.memory.Stats(), allocator, len(raw))
	}
}

func mignotteScopePhysicalAudit(t *testing.T, q *RationalManager, s *UPolynomialFactorizationService, caller ...*Int) {
	mignotteScopePhysicalAuditExtra(t, q, s, nil, caller...)
}
func mignotteScopePhysicalAuditExtra(t *testing.T, q *RationalManager, s *UPolynomialFactorizationService, external []memory.Block, caller ...*Int) {
	u := s.manager
	c := s.scope
	roots := append([]*Int{}, caller...)
	roots = append(roots, &q.nTmp, &q.addTmp1, &q.addTmp2, &q.addMulTmp.num, &q.addMulTmp.den, &q.ltTmp1.num, &q.ltTmp1.den, &q.ltTmp2.num, &q.ltTmp2.den, &u.modular.p, &u.modular.lower, &u.modular.upper, &u.modular.invTmp1, &u.modular.invTmp2, &u.modular.invTmp3, &u.modular.divTmp)
	states := []*upolyVectorState{&u.db, &u.dbab1, &u.dbab2, &u.tr, &u.push, &u.basic, &u.div1, &u.div2, &u.exactDiv, &u.gcd1, &u.gcd2, &u.cra, &u.mgcd[0], &u.mgcd[1], &u.mgcd[2], &u.mgcd[3], &u.mgcd[4], &u.mgcd[5], &u.sqf1, &u.sqf2, &u.pw}
	if c != nil {
		states = append(states, &c.fpp, &c.trial, &c.quotient)
		roots = append(roots, &c.mignotte.norm, &c.mignotte.bound, &c.mignotte.tmp, &c.p, &c.bestP, &c.gcdTmp, &c.leading, &c.trialContent)
	}
	for _, v := range states {
		if v.backing != 0 {
			a := u.values(v)
			for i := uint32(0); i < v.initialized; i++ {
				roots = append(roots, &a[i])
			}
		}
	}
	mignotteNamedIntegerPhysicalAudit(t, q.z, external, roots...)
	n := q.headerBytes + integerActualOwnedBytes(q.z) + u.header + u.modular.header + u.store.header + s.header + s.config.Primes.Stats().TotalBytes
	n += uint64(cap(u.pages))*uint64(reflect.TypeOf((*upolyDescriptorPage)(nil)).Size()) + uint64(len(u.pages))*uint64(reflect.TypeOf(upolyDescriptorPage{}).Size()) + uint64(cap(u.bindings))*uint64(reflect.TypeOf(upolyStoreBinding{}).Size()) + uint64(cap(u.store.backings))*uint64(reflect.TypeOf(upolyBacking{}).Size())
	for _, b := range u.store.backings {
		if b.live {
			n += uint64(cap(b.values)) * uint64(reflect.TypeOf(Int{}).Size())
		}
	}
	if c != nil {
		n += c.header
	}

	if n != q.budget.Used() {
		t.Fatal("actual enclosing cold/backing equation", n, q.budget.Used())
	}
}
