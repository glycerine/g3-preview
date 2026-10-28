package model_test

// Source: src/model/func_interp.cpp:143-149; src/util/trace.h:40-46.

import "testing"

func TestFunctionInterpretationSetElseEntrySourceTrace(t *testing.T) {
	m, tags, output := functionInterpretationTraceFixture(t)
	sort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "trace-sort"), nil)
	astTestMust(t, err)
	v, err := m.MkVar(0, sort)
	astTestMust(t, err)
	a, err := m.MkConst(astTestSymbol(t, m, "trace-a"), sort)
	astTestMust(t, err)
	b, err := m.MkConst(astTestSymbol(t, m, "trace-b"), sort)
	astTestMust(t, err)
	yes, err := m.MkTrue()
	astTestMust(t, err)
	no, err := m.MkFalse()
	astTestMust(t, err)
	astTestPin(t, m, v)
	astTestPin(t, m, a)
	astTestPin(t, m, b)
	astTestPin(t, m, yes)
	astTestPin(t, m, no)
	eqYes, err := m.MkEq(v, a)
	astTestMust(t, err)
	eqNo, err := m.MkEq(v, b)
	astTestMust(t, err)
	inner, err := m.MkIte(eqNo, no, yes)
	astTestMust(t, err)
	outer, err := m.MkIte(eqYes, yes, inner)
	astTestMust(t, err)
	astTestPin(t, m, outer)
	f := modelTestFunction(t, m, 1)
	astTestMust(t, f.SetElse(outer))
	if output.Len() != 0 || f.NumEntries() != 2 {
		t.Fatalf("disabled source trace or peeling: entries=%d records=%q", f.NumEntries(), output.String())
	}
	tags.Enable("func_interp")
	f2 := modelTestFunction(t, m, 1)
	astTestMust(t, f2.SetElse(outer))
	if f2.NumEntries() != 2 {
		t.Fatal("enabled trace changed graph", f2.NumEntries())
	}
	if !tags.ShouldTrace("func_interp") {
		if output.Len() != 0 {
			t.Fatal("erased source trace", output.String())
		}
		return
	}
	got := output.String()
	want := "-------- [func_interp] set_else src/model/func_interp.cpp:146 ---------\n" +
		"fi entry expr: (ite (= (:var 0) trace-a) true (ite (= (:var 0) trace-b) false true))\n" +
		"------------------------------------------------\n" +
		"-------- [func_interp] set_else src/model/func_interp.cpp:146 ---------\n" +
		"fi entry expr: (ite (= (:var 0) trace-b) false true)\n" +
		"------------------------------------------------\n"
	if got != want {
		t.Fatalf("source ITE peel records: got %q, want %q", got, want)
	}
}
