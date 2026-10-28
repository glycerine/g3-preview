package sat

// Source: src/sat/sat_solver.cpp:1257–1345; src/sat/sat_model_converter.cpp:35–114; src/sat/sat_types.h:138; src/util/lbool.h.

import (
	"github.com/glycerine/g3/pkg/util/lbool"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestSATSolverNativeModelConverterMatchesSlice(t *testing.T) {
	mc, b := satConverterTestOwner(t)
	e, e1 := mc.Mk(satModelElimVar, 0)
	satOK(t, e1)
	satOK(t, mc.InsertBinary(e, 1, 2))
	e, e1 = mc.Mk(satModelBlockLit, 1)
	satOK(t, e1)
	satOK(t, mc.InsertBinary(e, 3, 4))
	r, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b})
	satOK(t, err)
	var col satUintColumn
	col.Bind(r)
	satOK(t, satSolverResizeModel(&col, 3))
	native := Model{lbool.Undef, lbool.Undef, lbool.False}
	negative := lbool.False
	satOK(t, col.Write(2, uint32(int32(negative))))
	satOK(t, mc.Apply(native))
	satOK(t, mc.applyModelSpan(satSolverModelSpan{column: &col}))
	for i, want := range native {
		got, err := (satSolverModelSpan{column: &col}).read(BoolVar(i))
		satOK(t, err)
		if got != want {
			t.Fatal(i, got, want)
		}
	}
	a, err := mc.CheckModel(native)
	satOK(t, err)
	z, err := mc.checkModelSpan(satSolverModelSpan{column: &col})
	satOK(t, err)
	if a != z {
		t.Fatal(a, z)
	}
	if r.Stats().LiveBlocks != 0 {
		t.Fatal("native model columns allocated Runtime backing", r.Stats())
	}
	satOK(t, col.Finalize())
	satOK(t, r.Close())
}
func TestSATSolverNativeModelResizeInitializesDefaultEnums(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b})
	satOK(t, e)
	var col satUintColumn
	col.Bind(r)
	satOK(t, satSolverResizeModel(&col, 5))
	n, e := col.Size()
	satOK(t, e)
	if n != 5 {
		t.Fatal(n)
	}
	for i := uint32(0); i < n; i++ {
		v, e := col.Read(i)
		satOK(t, e)
		if v != 0 {
			t.Fatal(i, v)
		}
	}
	satOK(t, satSolverResizeModel(&col, 0))
	satOK(t, col.Finalize())
	satOK(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
