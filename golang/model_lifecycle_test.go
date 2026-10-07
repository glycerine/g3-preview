// Source: src/model/model.cpp:28-40,60-66,111-150;
// src/model/model_core.h:31-45,70-76; src/tactic/model_converter.cpp:176-181.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func TestModelDerivedConstructorAndVirtualReferenceDestruction(t *testing.T) {
	m := astTestManager(t)
	before := m.budget.Used()
	md, e := newSolverModel(m)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, md.Close()) })
	if md.core.derived != md || md.core.references != 0 || len(md.core.functions.cells) != 8 || len(md.universes.cells) != 8 {
		t.Fatal("source member construction")
	}
	n, e := md.NumUninterpretedSorts()
	satOK(t, e)
	if n != 0 {
		t.Fatal(n)
	}
	var ref tacticModelRef
	satOK(t, ref.Set(&md.core))
	satOK(t, ref.Close())
	if !md.core.closed || !md.closed {
		t.Fatal("base final DEC did not invoke derived destructor")
	}
	if m.budget.Used() != before {
		t.Fatal("derived members leaked", before, m.budget.Used())
	}
}

func TestModelDerivedConstructorFinalMemberRefusalUnwinds(t *testing.T) {
	m := astTestManager(t)
	before := m.budget.Used()
	// Header, constant map and function map succeed; the derived universe
	// map's first reservation fails. This is one constructor boundary only.
	m.budget.FailAfter(3)
	t.Cleanup(func() { m.budget.FailAfter(-1) })
	md, e := newSolverModel(m)
	if !errors.Is(e, arena.ErrInjectedFailure) || md != nil {
		t.Fatal("source final member refusal", md, e)
	}
	if m.budget.Used() != before {
		t.Fatal("constructed member prefix leaked", before, m.budget.Used())
	}
	m.budget.FailAfter(-1)
}

func TestModelDerivedUniverseUpdatePreservesLiteralAppendAndCounts(t *testing.T) {
	m := astTestManager(t)
	md, e := newSolverModel(m)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, md.Close()) })
	s, e := m.MkUninterpretedSort(astTestSymbol(t, m, "update-u"), nil)
	satOK(t, e)
	v, e := m.GetSomeValue(s)
	satOK(t, e)
	satOK(t, md.RegisterUniverse(s, []ASTHandle{v}))
	// A caller reference makes the source's stale physical prefix observable
	// through destruction. The source update DEC's the old graph, then APPENDs
	// without reset; destruction therefore also consumes this caller reference.
	satOK(t, m.IncRef(v))
	satOK(t, md.RegisterUniverse(s, []ASTHandle{v}))
	u, e := md.Universe(s)
	satOK(t, e)
	if !reflect.DeepEqual(u, []ASTHandle{v, v}) {
		t.Fatal("source update was replaced with reset", u)
	}
	refs, e := m.GetRefCount(v)
	satOK(t, e)
	if refs != 2 {
		t.Fatal("invented retention of source stale prefix", refs)
	}
	n, e := md.NumUninterpretedSorts()
	satOK(t, e)
	if n != 1 {
		t.Fatal("source update duplicated sort", n)
	}
	satOK(t, md.Close())
	if _, e = m.GetKind(v); e == nil {
		t.Fatal("literal source destructor did not consume both physical entries")
	}
}

func TestModelDerivedUniverseOwnershipCopyAndInsertionOrder(t *testing.T) {
	m := astTestManager(t)
	md, e := newSolverModel(m)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, md.Close()) })
	sorts := []ASTHandle{}
	values := []ASTHandle{}
	for _, name := range []string{"u-first", "u-second", "u-third"} {
		s, e := m.MkUninterpretedSort(astTestSymbol(t, m, name), nil)
		satOK(t, e)
		v, e := m.GetSomeValue(s)
		satOK(t, e)
		satOK(t, md.RegisterUniverse(s, []ASTHandle{v}))
		sorts = append(sorts, s)
		values = append(values, v)
	}
	n, e := md.NumUninterpretedSorts()
	satOK(t, e)
	if n != 3 {
		t.Fatal(n)
	}
	for i, s := range sorts {
		got, e := md.UninterpretedSort(uint32(i))
		satOK(t, e)
		if got != s {
			t.Fatal("source sort insertion order")
		}
		u, e := md.Universe(s)
		satOK(t, e)
		if !reflect.DeepEqual(u, []ASTHandle{values[i]}) {
			t.Fatal(u)
		}
	}
	copy, e := newSolverModel(m)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, copy.Close()) })
	satOK(t, copy.CopyUniverseInterpretations(md))
	satOK(t, md.Close())
	for _, v := range values {
		_, e = m.GetKind(v)
		satOK(t, e)
	}
	var out tacticModelRef
	satOK(t, out.Set(&copy.core))
	satOK(t, out.Close())
	for i, v := range values {
		if _, e = m.GetKind(v); e == nil {
			t.Fatal("universe value leaked", i)
		}
		if _, e = m.GetKind(sorts[i]); e == nil {
			t.Fatal("universe sort leaked", i)
		}
	}
}

func TestModelDerivedEmptyUniverseUpdateAndHashGrowth(t *testing.T) {
	m := astTestManager(t)
	md, e := newSolverModel(m)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, md.Close()) })
	for i := 0; i < 40; i++ {
		s, e := m.MkUninterpretedSort(astTestSymbol(t, m, string(rune('a'+i))), nil)
		satOK(t, e)
		satOK(t, md.RegisterUniverse(s, nil))
		yes, e := md.HasUninterpretedSort(s)
		satOK(t, e)
		if !yes {
			t.Fatal("empty universe is present")
		}
		v, e := m.GetSomeValue(s)
		satOK(t, e)
		satOK(t, md.RegisterUniverse(s, []ASTHandle{v}))
		u, e := md.Universe(s)
		satOK(t, e)
		if len(u) != 1 || u[0] != v {
			t.Fatal("source append to empty universe")
		}
	}
	n, e := md.NumUninterpretedSorts()
	satOK(t, e)
	if n != 40 || len(md.universes.cells) <= 8 {
		t.Fatal("replacement duplicated sort or failed hash growth", n)
	}
}

func TestModelConverterToModelOriginalAllocationAndApplication(t *testing.T) {
	m := astTestManager(t)
	old, e := newSolverModel(m)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, old.Close()) })
	var out tacticModelRef
	satOK(t, out.Set(&old.core))
	t.Cleanup(func() { satOK(t, out.Close()) })
	before := m.budget.Used()
	m.budget.FailAfter(0)
	satOK(t, tacticModelConverterToModel(m, nil, &out))
	got, e := out.Get()
	satOK(t, e)
	if got != &old.core || m.budget.Used() != before {
		t.Fatal("null converter changed result")
	}
	m.budget.FailAfter(-1)
	calls := []string{}
	leaf := &modelConverterTestLeaf{name: "inspect-empty", calls: &calls}
	leaf.run = func(r *tacticModelRef, idx uint32) error {
		md, e := r.Get()
		if e != nil {
			return e
		}
		if idx != 0 || md.derived == nil || md == &old.core || !old.closed || len(md.constants.cells) != 8 || len(md.functions.cells) != 8 {
			t.Fatal("source allocation/assignment must precede call", idx)
		}
		return nil
	}
	satOK(t, tacticModelConverterToModel(m, leaf, &out))
	got, e = out.Get()
	satOK(t, e)
	if got.derived == nil {
		t.Fatal("converter received a substitute core")
	}
	first := got.derived
	sentinel := errors.New("source converter application failed")
	failing := &modelConverterTestLeaf{name: "fail-after-assignment", calls: &calls, fail: sentinel}
	if e = tacticModelConverterToModel(m, failing, &out); e != sentinel {
		t.Fatal("converter error lost", e)
	}
	got, e = out.Get()
	satOK(t, e)
	if !first.closed || got.derived == nil || got.derived == first || got.references != 1 {
		t.Fatal("source assigned model lost on converter error")
	}
	stored, e := newSolverModel(m)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, stored.Close()) })
	c, e := newTacticStoredModel(&stored.core)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	transient := got.derived
	satOK(t, tacticModelConverterToModel(m, c, &out))
	if !transient.closed {
		t.Fatal("previous model not released")
	}
	got, e = out.Get()
	satOK(t, e)
	if got != &stored.core {
		t.Fatal("stored converter application")
	}
	// Retire the retrieved reference before the fixture's direct owner cleanup.
	satOK(t, out.Close())
}
