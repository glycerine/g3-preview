// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
// Source model_converter body assertions and independent finite lifetime checks.
package g3

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func satConverterTestOwner(t *testing.T) (*satModelConverter, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	c, e := newSATModelConverter(arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	paramsMust(t, e)
	t.Cleanup(func() {
		paramsMust(t, c.Close())
		if b.Used() != 0 {
			t.Fatalf("model converter retained %d", b.Used())
		}
	})
	return c, b
}
func TestSATSolverModelConverterReversePatchAndBlockedFlip(t *testing.T) {
	c, _ := satConverterTestOwner(t)
	a, e := c.Mk(satModelElimVar, 0)
	paramsMust(t, e)
	paramsMust(t, c.InsertLiterals(a, []satLiteral{satMakeLiteral(0, false), satMakeLiteral(1, false)}))
	b, e := c.Mk(satModelElimVar, 1)
	paramsMust(t, e)
	paramsMust(t, c.InsertLiterals(b, []satLiteral{satMakeLiteral(1, false)}))
	m := satModel{lUndef, lUndef}
	paramsMust(t, c.Apply(m))
	if m[0] != lUndef || m[1] != lTrue {
		t.Fatalf("source reverse order and partial model %v", m)
	}
	ok, e := c.CheckModel(m)
	paramsMust(t, e)
	if !ok {
		t.Fatal("source saved clauses")
	}
	paramsMust(t, c.Reset())
	a, e = c.Mk(satModelBlockLit, 0)
	paramsMust(t, e)
	paramsMust(t, c.InsertBinary(a, satMakeLiteral(0, true), satMakeLiteral(1, false)))
	m = satModel{lTrue, lFalse}
	paramsMust(t, c.Apply(m))
	if m[0] != lFalse || m[1] != lFalse {
		t.Fatalf("source blocked flip %v", m)
	}
	ok, e = c.CheckModel(satModel{lTrue, lFalse})
	paramsMust(t, e)
	if ok {
		t.Fatal("check_model must detect original saved clause")
	}
}
func TestSATSolverModelConverterSourceOtherUndefinedAndDisplay(t *testing.T) {
	c, _ := satConverterTestOwner(t)
	a, e := c.Mk(satModelElimVar, 0)
	paramsMust(t, e)
	paramsMust(t, c.InsertBinary(a, satMakeLiteral(0, true), satMakeLiteral(2, true)))
	m := satModel{lUndef, lFalse, lUndef}
	paramsMust(t, c.Apply(m))
	if m[0] != lUndef || m[2] != lFalse {
		t.Fatalf("source assigns other undefined variable %v", m)
	}
	v, e := c.MaxVar(1)
	paramsMust(t, e)
	if v != 2 {
		t.Fatalf("source max saved literal var %d", v)
	}
	vars, e := c.AppendVars([]satBoolVar{99, 0})
	paramsMust(t, e)
	if !reflect.DeepEqual(vars, []satBoolVar{99, 0}) {
		t.Fatalf("source collect_vars insertion dedup %v", vars)
	}
	out, e := NewNumericTextStream(NumericTextStreamConfig{})
	paramsMust(t, e)
	defer out.Close()
	paramsMust(t, c.DisplayInto(out))
	text, e := out.String()
	paramsMust(t, e)
	if text != "(sat::model-converter\n  (elim 0\n    (-0 -2)))\n" {
		t.Fatalf("source display %q", text)
	}
	ok, e := c.CheckInvariant(3)
	paramsMust(t, e)
	if !ok {
		t.Fatal("source invariant")
	}
}
func TestSATSolverModelConverterOuterGrowthDeepCopies(t *testing.T) {
	c, _ := satConverterTestOwner(t)
	for i := uint32(0); i < 2; i++ {
		r, e := c.Mk(satModelElimVar, satBoolVar(i))
		paramsMust(t, e)
		paramsMust(t, c.InsertBinary(r, satMakeLiteral(satBoolVar(i), false), satMakeLiteral(3, false)))
	}
	old0 := c.entries[0].literals
	old1 := c.entries[1].literals
	cap0, e := c.store.Capacity(old0)
	paramsMust(t, e)
	oldRef, e := c.EntryAt(0)
	paramsMust(t, e)
	_, e = c.Mk(satModelElimVar, 2)
	paramsMust(t, e)
	if c.Capacity() != 3 || c.Size() != 3 || c.store.IsLive(old0) || c.store.IsLive(old1) {
		t.Fatal("source outer growth must deep-copy then destroy old nested vectors")
	}
	new0 := c.entries[0].literals
	if new0 == old0 {
		t.Fatal("source copy constructor replaced by move")
	}
	capNew, e := c.store.Capacity(new0)
	paramsMust(t, e)
	if capNew != cap0 {
		t.Fatal("source nested copy capacity")
	}
	if _, _, e = c.KindAndVar(oldRef); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatalf("source relocated entry reference lifetime %v", e)
	}
	paramsMust(t, arena.ValidateRecordType(reflect.TypeOf(satModelConversionEntry{})))
	paramsMust(t, arena.ValidateRecordType(reflect.TypeOf(satModelConversionEntryRef{})))
}
func TestSATSolverModelConverterDeepCopyFailurePrefixAndResume(t *testing.T) {
	foundPartial := false
	for prefix := int64(0); prefix < 24; prefix++ {
		b := arena.NewBudget(^uint64(0))
		c, e := newSATModelConverter(arena.VectorConfig{PageSize: 4, Descriptors: arena.Config{PageSize: 2, Budget: b}})
		paramsMust(t, e)
		for i := uint32(0); i < 2; i++ {
			r, e := c.Mk(satModelElimVar, satBoolVar(i))
			paramsMust(t, e)
			paramsMust(t, c.InsertBinary(r, satMakeLiteral(satBoolVar(i), false), satMakeLiteral(3, false)))
			if i == 1 {
				paramsMust(t, c.InsertBinary(r, satMakeLiteral(satBoolVar(i), false), satMakeLiteral(3, false)))
			}
		}
		// Occupy the warmed free class8 block so the second different-class
		// copy genuinely requires a reservation after old0 was destroyed.
		_, e = c.store.CloneList(c.entries[1].literals)
		paramsMust(t, e)
		old0 := c.entries[0].literals
		old1 := c.entries[1].literals
		before := b.Used()
		satConverterAssertActualCharges(t, c, b)
		b.FailAfter(prefix)
		_, e = c.Mk(satModelElimVar, 2)
		if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatalf("prefix %d: %v", prefix, e)
		}
		satConverterAssertActualCharges(t, c, b)
		if e != nil && c.pending != nil && c.copyCursor == 1 {
			foundPartial = true
			if c.store.IsLive(old0) || !c.store.IsLive(old1) || !c.store.IsLive(c.entries[0].literals) {
				t.Fatalf("ascending completed copy/destroy roots prefix %d", prefix)
			}
			if b.Used() < before {
				t.Fatalf("unaccounted owned roots before teardown prefix %d", prefix)
			}
			r, e := c.EntryAt(1)
			paramsMust(t, e)
			if _, _, e = c.KindAndVar(r); !errors.Is(e, ErrSATIncompleteGrowth) {
				t.Fatalf("undefined unconstructed access guard %v", e)
			}
			b.FailAfter(-1)
			_, e = c.Mk(satModelElimVar, 2)
			paramsMust(t, e)
			if c.pending != nil || c.Size() != 3 || c.store.IsLive(old1) {
				t.Fatal("documented continuation did not complete source remaining copies")
			}
		}
		b.FailAfter(-1)
		paramsMust(t, c.Close())
		if b.Used() != 0 {
			t.Fatalf("prefix %d cleanup actual roots %d", prefix, b.Used())
		}
	}
	if !foundPartial {
		t.Fatal("sweep never exercised source first-copy success then second-copy failure")
	}
}
func TestSATSolverModelConverterCopyAppendAndSourceBitFields(t *testing.T) {
	s, _ := satConverterTestOwner(t)
	d, _ := satConverterTestOwner(t)
	a, e := s.Mk(satModelElimVar, 0)
	paramsMust(t, e)
	paramsMust(t, s.InsertBinary(a, satMakeLiteral(0, false), satMakeLiteral(1, false)))
	_, e = d.Mk(satModelBlockLit, 2)
	paramsMust(t, e)
	paramsMust(t, d.Copy(s))
	paramsMust(t, d.Copy(s))
	if d.Size() != 3 {
		t.Fatal("source Copy appends instead of assigning")
	}
	paramsMust(t, s.Reset())
	ok, e := d.CheckModel(satModel{lTrue, lFalse, lUndef})
	paramsMust(t, e)
	if !ok {
		t.Fatal("copied nested payload borrowed source")
	}
	if e = d.Copy(d); !errors.Is(e, ErrSATSelfAppend) {
		t.Fatalf("source dangling self-copy adapter %v", e)
	}
	paramsMust(t, d.Reset())
	r, e := d.Mk(satModelElimVar, satBoolVar(^uint32(0)))
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatalf("source post-publication bitfield assertion %v", e)
		}
	} else {
		paramsMust(t, e)
	}
	if d.Size() != 1 {
		t.Fatal("post-publication source assertion prefix lost")
	}
	k, v, e := d.KindAndVar(r)
	paramsMust(t, e)
	if k != satModelElimVar || v != satNullBoolVar {
		t.Fatalf("source unsigned bitfield truncation k=%d v=%d", k, v)
	}
}

func satConverterAssertActualCharges(t *testing.T, c *satModelConverter, b *arena.Budget) {
	t.Helper()
	w := c.store.words.Stats()
	d := w.Descriptors
	want := uint64(reflect.TypeOf(satModelConverter{}).Size()) + uint64(cap(c.entries)+cap(c.pending))*uint64(reflect.TypeOf(satModelConversionEntry{}).Size())
	want += uint64(reflect.TypeOf(satSolverLiteralStore{}).Size()) + uint64(cap(c.store.records))*uint64(reflect.TypeOf(satSolverLiteralListRecord{}).Size())
	want += w.OwnerBytes + w.PayloadBytes + w.BlockMetadataBytes + w.DirectoryBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
	if b.Used() != want {
		t.Fatalf("immediate actual header/backing/root charges: %d != %d", b.Used(), want)
	}
}
func TestSATSolverModelConverterIndependentChainModels(t *testing.T) {
	c, b := satConverterTestOwner(t)
	for mask := uint32(0); mask < 32; mask++ {
		paramsMust(t, c.Reset())
		var originals [8][2]satLiteral
		model := make(satModel, 5)
		for i := 0; i < 4; i++ {
			kind := satModelElimVar
			model[i] = lUndef
			if mask&(1<<uint(i)) != 0 {
				kind = satModelBlockLit
				model[i] = lTrue
			}
			r, e := c.Mk(kind, satBoolVar(i))
			paramsMust(t, e)
			originals[2*i] = [2]satLiteral{satMakeLiteral(satBoolVar(i), false), satMakeLiteral(satBoolVar(i+1), false)}
			originals[2*i+1] = [2]satLiteral{satMakeLiteral(satBoolVar(i), true), satMakeLiteral(satBoolVar(i+1), true)}
			paramsMust(t, c.InsertBinary(r, originals[2*i][0], originals[2*i][1]))
			paramsMust(t, c.InsertBinary(r, originals[2*i+1][0], originals[2*i+1][1]))
		}
		model[4] = lFalse
		if mask&16 != 0 {
			model[4] = lTrue
		}
		paramsMust(t, c.Apply(model))
		for _, clause := range originals {
			good := false
			for _, l := range clause {
				good = good || model[l.variable()] == lTrue && !l.sign() || model[l.variable()] == lFalse && l.sign()
			}
			if !good {
				t.Fatalf("independent original clause mask%d %v model%v", mask, clause, model)
			}
		}
		for i := 0; i < 4; i++ {
			if model[i] == model[i+1] || model[i] == lUndef {
				t.Fatalf("independent parity oracle %v", model)
			}
		}
		satConverterAssertActualCharges(t, c, b)
	}
}
func TestSATSolverModelConverterActualClauseAndWrapperRoutes(t *testing.T) {
	c, b := satConverterTestOwner(t)
	a := satStore(t, nil)
	clause := satMk(t, a, []satLiteral{satMakeLiteral(0, false), satMakeLiteral(1, true), satMakeLiteral(2, false)}, false)
	r, e := c.Mk(satModelBlockLit, 0)
	paramsMust(t, e)
	paramsMust(t, c.InsertClause(r, a, clause))
	paramsMust(t, c.InsertWrapper(r, a, newSATClauseWrapper(clause)))
	paramsMust(t, c.InsertWrapper(r, a, newSATBinaryWrapper(satMakeLiteral(0, false), satMakeLiteral(3, false))))
	n, e := c.store.Size(c.entries[0].literals)
	paramsMust(t, e)
	if n != 11 {
		t.Fatalf("actual overload scalar stream length%d", n)
	}
	for _, i := range []uint32{3, 7, 10} {
		v, e := c.store.Read(c.entries[0].literals, i)
		paramsMust(t, e)
		if v != satDefaultLiteral() {
			t.Fatal("source clause delimiter")
		}
	}
	paramsMust(t, a.Delete(clause))
	ok, e := c.CheckModel(satModel{lTrue, lFalse, lFalse, lFalse})
	paramsMust(t, e)
	if !ok {
		t.Fatal("owned saved data died with clause")
	}
	satConverterAssertActualCharges(t, c, b)
}
func TestSATSolverModelConverterConstructorAndInsertFaults(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 8; prefix++ {
		b := arena.NewBudget(^uint64(0))
		b.FailAfter(prefix)
		c, e := newSATModelConverter(arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}})
		if e == nil {
			paramsMust(t, c.Close())
			success = true
		} else if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatalf("constructor %d %v", prefix, e)
		}
		if b.Used() != 0 {
			t.Fatalf("constructor unwind immediate%d %d", prefix, b.Used())
		}
		if success {
			break
		}
	}
	if !success {
		t.Fatal("constructor sweep no success")
	}
	c, b := satConverterTestOwner(t)
	r, e := c.Mk(satModelBlockLit, 0)
	paramsMust(t, e)
	// Real source pushes each literal before its delimiter. A failed later
	// expansion retains the completed literal prefix and published capacity.
	paramsMust(t, c.InsertLiterals(r, []satLiteral{satMakeLiteral(0, false), satMakeLiteral(1, false), satMakeLiteral(2, false)}))
	_, e = c.store.CloneList(c.entries[0].literals)
	paramsMust(t, e)
	b.FailAfter(0)
	e = c.InsertBinary(r, satMakeLiteral(0, true), satMakeLiteral(1, true))
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatalf("insert failure %v", e)
	}
	// Current length4/cap5: first new literal publishes, then second literal
	// needs a fresh class8 block; the warmed block is explicitly occupied.
	// Completed original delimiter and the new first-literal prefix stay live.
	n, e := c.store.Size(c.entries[0].literals)
	paramsMust(t, e)
	if n != 5 {
		t.Fatalf("failed append prefix%d", n)
	}
	v, e := c.store.Read(c.entries[0].literals, 4)
	paramsMust(t, e)
	if v != satMakeLiteral(0, true) {
		t.Fatal("completed source first literal publication")
	}
	satConverterAssertActualCharges(t, c, b)
	b.FailAfter(-1)
}

func TestSATSolverModelConverterOwnerEnumAndCloseGuards(t *testing.T) {
	c, b := satConverterTestOwner(t)
	other, _ := satConverterTestOwner(t)
	_, e := c.Mk(satModelConversionKind(2), 0)
	if !errors.Is(e, ErrSATKind) || c.Size() != 0 {
		t.Fatalf("Go enum-domain guard for source-invalid enum conversion: %v size%d", e, c.Size())
	}
	r, e := c.Mk(satModelElimVar, 0)
	paramsMust(t, e)
	if _, _, e = other.KindAndVar(r); !errors.Is(e, ErrSATOwner) {
		t.Fatalf("foreign entry %v", e)
	}
	copyOwner := *c
	if _, e = copyOwner.Mk(satModelElimVar, 0); !errors.Is(e, ErrSATOwner) {
		t.Fatalf("cold owner copied %v", e)
	}
	before := b.Used()
	c.epoch = ^uint64(0)
	if e = c.Reset(); !errors.Is(e, arena.ErrSlotLimit) || b.Used() != before || c.Size() != 1 {
		t.Fatalf("epoch exhaustion must not destroy roots %v", e)
	}
	paramsMust(t, c.Close())
	if b.Used() != 0 || c.epoch != ^uint64(0) {
		t.Fatal("Close cleanup at exhaustion must not reuse epoch")
	}
	if _, e = c.Mk(satModelElimVar, 0); !errors.Is(e, arena.ErrClosed) {
		t.Fatalf("closed before enum/storage mutation %v", e)
	}
	badBudget := arena.NewBudget(^uint64(0))
	_, e = newSATModelConverter(arena.VectorConfig{PageSize: 3, Descriptors: arena.Config{Budget: badBudget}})
	if !errors.Is(e, arena.ErrConfig) || badBudget.Used() != 0 {
		t.Fatalf("nested invalid-config unwind %v %d", e, badBudget.Used())
	}
}

func TestSATSolverModelConverterSelfCopyCapturedEndAndInvalidationPrefix(t *testing.T) {
	m, _ := satConverterTestOwner(t)
	r, e := m.Mk(satModelElimVar, 4)
	paramsMust(t, e)
	paramsMust(t, m.InsertBinary(r, satMakeLiteral(4, false), satMakeLiteral(5, true)))
	old, e := m.entry(r)
	paramsMust(t, e)
	oldPayload := old.literals
	paramsMust(t, m.Copy(m))
	if m.Size() != 2 || m.sourceCapacity != 2 {
		t.Fatalf("finite self copy size/cap %d/%d", m.Size(), m.sourceCapacity)
	}
	q, e := m.EntryAt(1)
	paramsMust(t, e)
	copy, e := m.entry(q)
	paramsMust(t, e)
	if copy.literals == oldPayload {
		t.Fatal("self copy aliases nested owner")
	}
	paramsMust(t, m.store.Write(oldPayload, 0, satMakeLiteral(6, false)))
	v, e := m.store.Read(copy.literals, 0)
	paramsMust(t, e)
	if v != satMakeLiteral(4, false) {
		t.Fatal("self copied data aliases")
	}
	paramsMust(t, m.Reset())
	_, e = m.Mk(satModelElimVar, 0)
	paramsMust(t, e)
	_, e = m.Mk(satModelElimVar, 1)
	paramsMust(t, e)
	_, e = m.Mk(satModelElimVar, 2)
	paramsMust(t, e)
	e = m.Copy(m)
	if !errors.Is(e, ErrSATSelfAppend) {
		t.Fatal("size3/cap3 relocation should guard", e)
	}
}
func TestSATSolverModelConverterSelfCopyRelocationBoundary(t *testing.T) {
	m, b := satConverterTestOwner(t)
	_, e := m.Mk(satModelElimVar, 0)
	paramsMust(t, e)
	_, e = m.Mk(satModelElimVar, 1)
	paramsMust(t, e)
	_, e = m.Mk(satModelElimVar, 2)
	paramsMust(t, e)
	// Shrink only the fixture's trivially scalar outer view to size2/cap3; the
	// discarded entry owns an empty list and must be explicitly destroyed.
	discarded := m.entries[2].literals
	paramsMust(t, m.store.Free(discarded))
	m.entries = m.entries[:2]
	e = m.Copy(m)
	if !errors.Is(e, ErrSATSelfAppend) {
		t.Fatalf("dangling source iterator guard %v", e)
	}
	if m.Size() != 3 || m.sourceCapacity != 5 {
		t.Fatalf("completed append/growth prefixes %d/%d", m.Size(), m.sourceCapacity)
	}
	r, e := m.EntryAt(2)
	paramsMust(t, e)
	_, v, e := m.KindAndVar(r)
	paramsMust(t, e)
	if v != 0 {
		t.Fatal("first valid append lost")
	}
	satConverterAssertActualCharges(t, m, b)
}

func TestSATSolverModelConverterDebugAssertionBeforeLaterInvalidLiteral(t *testing.T) {
	m, b := satConverterTestOwner(t)
	r, e := m.Mk(satModelBlockLit, 0)
	paramsMust(t, e)
	paramsMust(t, m.InsertBinary(r, satMakeLiteral(0, false), satMakeLiteral(1, false)))
	paramsMust(t, m.InsertBinary(r, satMakeLiteral(0, true), satMakeLiteral(0, true)))
	paramsMust(t, m.InsertBinary(r, satMakeLiteral(5, false), satMakeLiteral(0, false)))
	model := satModel{lTrue, lFalse}
	e = m.Apply(model)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatalf("source per-separator assertion must precede later invalid var: %v", e)
		}
	} else {
		paramsMust(t, e)
	}
	if model[0] != lFalse || model[1] != lFalse {
		t.Fatalf("completed source assignment prefix %v", model)
	}
	if _, e = m.CheckModel(model); !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("CheckModel must still traverse later invalid variable: %v", e)
	}
	satConverterAssertActualCharges(t, m, b)
}

func TestSATSolverModelConverterImplicitCloneExactCapacityAndLifetime(t *testing.T) {
	src, _ := satConverterTestOwner(t)
	for i := uint32(0); i < 4; i++ {
		r, e := src.Mk(satModelBlockLit, satBoolVar(i))
		paramsMust(t, e)
		paramsMust(t, src.InsertBinary(r, satMakeLiteral(satBoolVar(i), false), satMakeLiteral(satBoolVar(i+4), true)))
	}
	// Source vector<entry>::shrink(3) destroys entry3 while retaining cap5.
	paramsMust(t, src.store.Free(src.entries[3].literals))
	src.entries = src.entries[:3]
	b := arena.NewBudget(^uint64(0))
	clone, e := src.Clone(arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	paramsMust(t, e)
	t.Cleanup(func() {
		paramsMust(t, clone.Close())
		if b.Used() != 0 {
			t.Fatal("clone leaked", b.Used())
		}
	})
	if clone.Size() != 3 || clone.sourceCapacity != 5 || cap(clone.entries) != 5 {
		t.Fatalf("implicit copy exact outer capacity %d/%d/%d", clone.Size(), clone.sourceCapacity, cap(clone.entries))
	}
	for i := uint32(0); i < 3; i++ {
		r, e := clone.EntryAt(i)
		paramsMust(t, e)
		x, e := clone.entry(r)
		paramsMust(t, e)
		sc, e := src.store.Capacity(src.entries[i].literals)
		paramsMust(t, e)
		cc, e := clone.store.Capacity(x.literals)
		paramsMust(t, e)
		if cc != sc {
			t.Fatal("nested capacity changed")
		}
		v, e := clone.store.Read(x.literals, 0)
		paramsMust(t, e)
		if v != satMakeLiteral(satBoolVar(i), false) {
			t.Fatal(v)
		}
	}
	paramsMust(t, src.store.Write(src.entries[0].literals, 0, satMakeLiteral(99, false)))
	v, e := clone.store.Read(clone.entries[0].literals, 0)
	paramsMust(t, e)
	if v != satMakeLiteral(0, false) {
		t.Fatal("clone shallow copied ownership")
	}
	paramsMust(t, src.Close())
	v, e = clone.store.Read(clone.entries[2].literals, 0)
	paramsMust(t, e)
	if v != satMakeLiteral(2, false) {
		t.Fatal("clone retained borrowed owner")
	}
	satConverterAssertActualCharges(t, clone, b)
}
func TestSATSolverModelConverterImplicitCopyEmptyRetainedCapacity(t *testing.T) {
	src, _ := satConverterTestOwner(t)
	for i := uint32(0); i < 3; i++ {
		_, e := src.Mk(satModelBlockLit, satBoolVar(i))
		paramsMust(t, e)
	}
	for i := range src.entries {
		paramsMust(t, src.store.Free(src.entries[i].literals))
	}
	src.entries = src.entries[:0]
	b := arena.NewBudget(^uint64(0))
	c, e := src.Clone(arena.VectorConfig{Descriptors: arena.Config{Budget: b}})
	paramsMust(t, e)
	if !c.Empty() || c.sourceCapacity != 3 || cap(c.entries) != 3 {
		t.Fatal("copy lost allocated empty vector")
	}
	satConverterAssertActualCharges(t, c, b)
	paramsMust(t, c.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	dst, db := satConverterTestOwner(t)
	_, e = dst.Mk(satModelElimVar, 8)
	paramsMust(t, e)
	old := dst.entries[0].literals
	paramsMust(t, dst.Assign(src))
	if !dst.Empty() || dst.sourceCapacity != 3 || dst.store.IsLive(old) {
		t.Fatal("assignment did not destroy and copy empty backing")
	}
	satConverterAssertActualCharges(t, dst, db)
	paramsMust(t, dst.Assign(dst))
}
func TestSATSolverModelConverterImplicitAssignmentFaultPrefixes(t *testing.T) {
	src, _ := satConverterTestOwner(t)
	for i := uint32(0); i < 3; i++ {
		r, e := src.Mk(satModelBlockLit, satBoolVar(i))
		paramsMust(t, e)
		ls := []satLiteral{satMakeLiteral(satBoolVar(i), false), 2, 4, 6, 8, 10, 12, 14, 16}
		paramsMust(t, src.InsertLiterals(r, ls))
	}
	foundPartial, foundSuccess := false, false
	for fail := int64(0); fail < 35; fail++ {
		b := arena.NewBudget(^uint64(0))
		dst, e := newSATModelConverter(arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}})
		paramsMust(t, e)
		old, e := dst.Mk(satModelElimVar, 9)
		paramsMust(t, e)
		oldPayload := dst.entries[0].literals
		b.FailAfter(fail)
		e = dst.Assign(src)
		if dst.store.IsLive(oldPayload) {
			t.Fatal("destination destruction must precede copying", fail)
		}
		if _, _, guard := dst.KindAndVar(old); !errors.Is(guard, arena.ErrInvalidSlot) {
			t.Fatal("assignment kept prior entry reference", guard)
		}
		if e == nil {
			foundSuccess = true
			if dst.Size() != 3 || dst.sourceCapacity != 3 {
				t.Fatal("complete assignment shape")
			}
		} else {
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
			if dst.sourceCapacity == 0 {
				if dst.Size() != 0 {
					t.Fatal("allocation refusal left header")
				}
			} else {
				if dst.Size() != 3 || dst.sourceCapacity != 3 {
					t.Fatal("copy_core publication shape")
				}
				if dst.copyCursor > 0 && dst.copyCursor < 3 {
					foundPartial = true
				}
				for i := uint32(0); i < 3; i++ {
					r, e := dst.EntryAt(i)
					paramsMust(t, e)
					_, v, e := dst.KindAndVar(r)
					if i < dst.copyCursor {
						paramsMust(t, e)
						if uint32(v) != i {
							t.Fatal("ascending copied prefix")
						}
					} else if !errors.Is(e, ErrSATIncompleteGrowth) {
						t.Fatal("unconstructed entry guard", i, e)
					}
				}
			}
		}
		satConverterAssertActualCharges(t, dst, b)
		paramsMust(t, dst.Close())
		if b.Used() != 0 {
			t.Fatal("assignment fault leaked", fail, b.Used())
		}
	}
	if !foundPartial || !foundSuccess {
		t.Fatalf("did not exercise finite copy prefixes/success %v/%v", foundPartial, foundSuccess)
	}
}

func TestSATSolverModelConverterImplicitCloneFailureUnwindAndSelfAssign(t *testing.T) {
	src, sb := satConverterTestOwner(t)
	for i := uint32(0); i < 3; i++ {
		r, e := src.Mk(satModelBlockLit, satBoolVar(i))
		paramsMust(t, e)
		paramsMust(t, src.InsertBinary(r, satMakeLiteral(satBoolVar(i), false), satMakeLiteral(satBoolVar(i+3), true)))
	}
	sourceBytes := sb.Used()
	success := false
	for fail := int64(0); fail < 40; fail++ {
		b := arena.NewBudget(^uint64(0))
		b.FailAfter(fail)
		clone, e := src.Clone(arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}})
		if e == nil {
			success = true
			satConverterAssertActualCharges(t, clone, b)
			paramsMust(t, clone.Close())
		} else {
			if !errors.Is(e, arena.ErrInjectedFailure) || clone != nil {
				t.Fatal("clone refusal", e)
			}
		}
		if b.Used() != 0 || sb.Used() != sourceBytes {
			t.Fatal("unpublished clone changed ownership", fail, b.Used(), sb.Used())
		}
	}
	if !success {
		t.Fatal("no successful clone")
	}
	sb.FailAfter(0)
	if n := testing.AllocsPerRun(100, func() {
		if e := src.Assign(src); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatal("self assignment allocated", n)
	}
	satConverterAssertActualCharges(t, src, sb)
	copied := *src
	before := src.entries[0].literals
	if e := src.Assign(&copied); !errors.Is(e, ErrSATOwner) || !src.store.IsLive(before) {
		t.Fatal("copied cold owner mutated destination", e)
	}
	paramsMust(t, src.Close())
	b := arena.NewBudget(^uint64(0))
	b.FailAfter(0)
	if _, e := src.Clone(arena.VectorConfig{Descriptors: arena.Config{Budget: b}}); !errors.Is(e, arena.ErrClosed) || b.Used() != 0 {
		t.Fatal("closed source charged constructor", e)
	}
}
func TestSATSolverModelConverterImplicitInterruptedCopyAndRestart(t *testing.T) {
	full, _ := satConverterTestOwner(t)
	for i := uint32(0); i < 3; i++ {
		r, e := full.Mk(satModelBlockLit, satBoolVar(i))
		paramsMust(t, e)
		paramsMust(t, full.InsertBinary(r, satMakeLiteral(satBoolVar(i), false), satMakeLiteral(satBoolVar(i+3), true)))
	}
	var partial *satModelConverter
	var pb *arena.Budget
	for fail := int64(0); fail < 35; fail++ {
		b := arena.NewBudget(^uint64(0))
		c, e := newSATModelConverter(arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}})
		paramsMust(t, e)
		b.FailAfter(fail)
		e = c.Assign(full)
		if errors.Is(e, arena.ErrInjectedFailure) && c.copyCursor > 0 && c.copyCursor < 3 {
			partial, pb = c, b
			break
		}
		paramsMust(t, c.Close())
		if b.Used() != 0 {
			t.Fatal("fixture retained failed owner")
		}
	}
	if partial == nil {
		t.Fatal("no interrupted constructor fixture")
	}
	defer func() {
		paramsMust(t, partial.Close())
		if pb.Used() != 0 {
			t.Fatal("partial cleanup", pb.Used())
		}
	}()
	prefix := partial.copyCursor
	sourceBytes := pb.Used()
	cb := arena.NewBudget(^uint64(0))
	clone, e := partial.Clone(arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: cb}})
	if clone != nil || !errors.Is(e, ErrSATIncompleteGrowth) || cb.Used() != 0 || pb.Used() != sourceBytes {
		t.Fatal("late guarded source clone cleanup", e)
	}
	dst, db := satConverterTestOwner(t)
	_, e = dst.Mk(satModelBlockLit, 9)
	paramsMust(t, e)
	old := dst.entries[0].literals
	e = dst.Assign(partial)
	if !errors.Is(e, ErrSATIncompleteGrowth) || dst.Size() != 3 || dst.copyCursor != prefix || dst.store.IsLive(old) {
		t.Fatal("late guarded assignment prefix", e, dst.copyCursor)
	}
	satConverterAssertActualCharges(t, dst, db)
	satConverterAssertActualCharges(t, partial, pb)
	if _, e = partial.Mk(satModelBlockLit, 9); !errors.Is(e, ErrSATIncompleteGrowth) || pb.Used() != sourceBytes {
		t.Fatal("unsafe partial append continuation", e)
	}
	pb.FailAfter(-1)
	paramsMust(t, partial.Assign(full))
	if partial.copying || partial.copyCursor != 0 || partial.Size() != 3 {
		t.Fatal("explicit restart did not reconstruct whole owner")
	}
	satConverterAssertActualCharges(t, partial, pb)
	paramsMust(t, dst.Reset())
	if dst.Size() != 0 || dst.sourceCapacity != 0 {
		t.Fatal("partial Reset did not clean prefix")
	}
	satConverterAssertActualCharges(t, dst, db)
}

func TestSATSolverModelConverterImplicitCopyUninitializedCapacityGuard(t *testing.T) {
	var src *satModelConverter
	var sb *arena.Budget
	for fail := int64(0); fail < 25; fail++ {
		b := arena.NewBudget(^uint64(0))
		c, e := newSATModelConverter(arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}})
		paramsMust(t, e)
		r, e := c.Mk(satModelBlockLit, 0)
		paramsMust(t, e)
		paramsMust(t, c.InsertBinary(r, 0, 2))
		r, e = c.Mk(satModelBlockLit, 1)
		paramsMust(t, e)
		paramsMust(t, c.InsertLiterals(r, []satLiteral{2, 4, 6, 8, 10, 12, 14, 16, 18}))
		b.FailAfter(fail)
		_, e = c.Mk(satModelBlockLit, 2)
		if errors.Is(e, arena.ErrInjectedFailure) && c.pending != nil && c.copyCursor == 1 {
			src, sb = c, b
			break
		}
		paramsMust(t, c.Close())
		if b.Used() != 0 {
			t.Fatal("capacity fixture leaks")
		}
	}
	if src == nil {
		t.Fatal("no first-copy/unfinished-capacity source fixture")
	}
	defer func() {
		paramsMust(t, src.Close())
		if sb.Used() != 0 {
			t.Fatal("pending source teardown")
		}
	}()
	before := sb.Used()
	b := arena.NewBudget(^uint64(0))
	out, e := src.Clone(arena.VectorConfig{Descriptors: arena.Config{Budget: b}})
	if out != nil || !errors.Is(e, ErrSATIncompleteGrowth) || b.Used() != 0 || sb.Used() != before {
		t.Fatal("undefined capacity read must not copy old scalar capacity", e)
	}
	dst, db := satConverterTestOwner(t)
	_, e = dst.Mk(satModelBlockLit, 9)
	paramsMust(t, e)
	old := dst.entries[0].literals
	e = dst.Assign(src)
	if !errors.Is(e, ErrSATIncompleteGrowth) || dst.Size() != 0 || dst.sourceCapacity != 0 || dst.store.IsLive(old) {
		t.Fatal("assignment must destroy before source capacity guard", e)
	}
	satConverterAssertActualCharges(t, dst, db)
	satConverterAssertActualCharges(t, src, sb)
}
func TestSATSolverModelConverterImplicitCloneSharedBorrowedBudget(t *testing.T) {
	src, b := satConverterTestOwner(t)
	r, e := src.Mk(satModelBlockLit, 0)
	paramsMust(t, e)
	paramsMust(t, src.InsertBinary(r, 0, 2))
	before := b.Used()
	clone, e := src.Clone(arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}})
	paramsMust(t, e)
	if b.Used() <= before {
		t.Fatal("clone omitted its actual graph")
	}
	paramsMust(t, clone.Close())
	if b.Used() != before {
		t.Fatal("shared-budget close charged borrowed source graph")
	}
	v, e := src.store.Read(src.entries[0].literals, 0)
	paramsMust(t, e)
	if v != 0 {
		t.Fatal("shared-budget source root lost")
	}
	b.FailAfter(0)
	if _, e = src.Clone(arena.VectorConfig{Descriptors: arena.Config{Budget: b}}); !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != before {
		t.Fatal("shared-budget failed preparation was not refunded", e)
	}
}

func TestSATSolverModelConverterImplicitAssignmentAscendingDestroyAndCopy(t *testing.T) {
	src, _ := satConverterTestOwner(t)
	dst, b := satConverterTestOwner(t)
	for i := uint32(0); i < 3; i++ {
		_, e := src.Mk(satModelBlockLit, satBoolVar(i))
		paramsMust(t, e)
		_, e = dst.Mk(satModelBlockLit, satBoolVar(i+10))
		paramsMust(t, e)
	}
	old := [3]satSolverLiteralListRef{dst.entries[0].literals, dst.entries[1].literals, dst.entries[2].literals}
	paramsMust(t, dst.Assign(src))
	for i := uint32(0); i < 3; i++ {
		r, e := dst.EntryAt(i)
		paramsMust(t, e)
		_, v, e := dst.KindAndVar(r)
		paramsMust(t, e)
		n := dst.entries[i].literals
		prior := old[2-i]
		if uint32(v) != i || n.slot != prior.slot || n.generation != prior.generation+1 || dst.store.IsLive(prior) {
			t.Fatalf("ascending destruction/LIFO adapter allocation order slot%d: %+v prior%+v var%d", i, n, prior, v)
		}
	}
	satConverterAssertActualCharges(t, dst, b)
}
