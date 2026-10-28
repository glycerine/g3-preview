package model_test

// Source: src/model/func_interp.cpp:24-330; independent Boolean graph cases.

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"testing"
)

func modelTestFunction(t *testing.T, m *ast.ASTManager, arity uint32) *model.FunctionInterpretation {
	t.Helper()
	f, e := model.NewFunctionInterpretation(m, arity)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, f.Close()) })
	return f
}
func TestFunctionInterpretationEntriesReplaceAndReferences(t *testing.T) {
	m := astTestManager(t)
	f := modelTestFunction(t, m, 1)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	astTestPin(t, m, yes)
	astTestPin(t, m, no)
	before, e := m.GetRefCount(yes)
	astTestMust(t, e)
	astTestMust(t, f.InsertEntry([]ast.ASTHandle{yes}, no))
	entry, e := f.FindEntry([]ast.ASTHandle{yes})
	astTestMust(t, e)
	if entry == nil {
		t.Fatal("missing entry")
	}
	got, e := entry.Result()
	astTestMust(t, e)
	if got != no {
		t.Fatal(got)
	}
	astTestMust(t, f.InsertEntry([]ast.ASTHandle{yes}, yes))
	if f.NumEntries() != 1 {
		t.Fatal("replacement appended")
	}
	got, e = entry.Result()
	astTestMust(t, e)
	if got != yes {
		t.Fatal(got)
	}
	after, e := m.GetRefCount(yes)
	astTestMust(t, e)
	if after != before+2 {
		t.Fatal("argument and result references", before, after)
	}
	astTestMust(t, f.Close())
	after, e = m.GetRefCount(yes)
	astTestMust(t, e)
	if after != before {
		t.Fatal("graph leaked", before, after)
	}
}
func TestFunctionInterpretationElsePeelingCacheAndCompression(t *testing.T) {
	m := astTestManager(t)
	f := modelTestFunction(t, m, 1)
	b, e := m.MkUninterpretedSort(astTestSymbol(t, m, "graph-sort"), nil)
	astTestMust(t, e)
	v, e := m.MkVar(0, b)
	astTestMust(t, e)
	astTestPin(t, m, v)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	arg, e := m.MkConst(astTestSymbol(t, m, "graph-arg"), b)
	astTestMust(t, e)
	astTestPin(t, m, arg)
	eq, e := m.MkEq(v, arg)
	astTestMust(t, e)
	term, e := m.MkIte(eq, no, yes)
	astTestMust(t, e)
	astTestPin(t, m, term)
	astTestMust(t, f.SetElse(term))
	if f.NumEntries() != 1 {
		t.Fatal("source ITE graph extraction")
	}
	fallback, e := f.Else()
	astTestMust(t, e)
	if fallback != yes {
		t.Fatal(fallback)
	}
	interp, e := f.Interpretation()
	astTestMust(t, e)
	astTestPin(t, m, interp)
	cached, e := f.Interpretation()
	astTestMust(t, e)
	if cached != interp {
		t.Fatal("source cache identity")
	}
	astTestMust(t, f.InsertEntry([]ast.ASTHandle{arg}, yes))
	astTestMust(t, f.Compress())
	if f.NumEntries() != 0 {
		t.Fatal("redundant entry remained")
	}
	constant, e := f.IsConstant()
	astTestMust(t, e)
	if !constant {
		t.Fatal("constant graph")
	}
	cached, e = f.Interpretation()
	astTestMust(t, e)
	if cached != yes {
		t.Fatal("stale interpretation cache")
	}
}
func TestFunctionInterpretationElseDirectOrderAndCopy(t *testing.T) {
	m := astTestManager(t)
	f := modelTestFunction(t, m, 2)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	v, e := m.MkVar(0, b)
	astTestMust(t, e)
	astTestPin(t, m, v)
	astTestMust(t, f.SetElse(v))
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	out := astTestPin(t, m, no)
	ok, e := f.EvalElse([]ast.ASTHandle{yes, no}, out)
	astTestMust(t, e)
	if !ok {
		t.Fatal("non-ground else missing")
	}
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if got != yes {
		t.Fatal("eval_else must bind VAR 0 to args[0]", got)
	}
	copy, e := f.Copy()
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, copy.Close()) })
	astTestMust(t, f.SetElse(no))
	fallback, e := copy.Else()
	astTestMust(t, e)
	if fallback != v {
		t.Fatal("copy mutated with original")
	}
}

func TestFunctionInterpretationBooleanIffRemainsElseAndTieOrder(t *testing.T) {
	m := astTestManager(t)
	f := modelTestFunction(t, m, 1)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	v, e := m.MkVar(0, b)
	astTestMust(t, e)
	astTestPin(t, m, v)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	eq, e := m.MkEq(v, yes)
	astTestMust(t, e)
	term, e := m.MkIte(eq, no, yes)
	astTestMust(t, e)
	astTestPin(t, m, term)
	astTestMust(t, f.SetElse(term))
	if f.NumEntries() != 0 {
		t.Fatal("source OP_EQ predicate must reject OP_IFF")
	}
	fallback, e := f.Else()
	astTestMust(t, e)
	if fallback != term {
		t.Fatal(fallback)
	}
	// Non-value ground arguments remain legal and clear the graph value flag.
	for i, result := range []ast.ASTHandle{yes, no, no, yes} {
		arg := goalTestAtom(t, m, fmt.Sprint("tie-", i))
		astTestMust(t, f.InsertEntry([]ast.ASTHandle{arg}, result))
	}
	if f.ArgumentsAreValues() {
		t.Fatal("uninterpreted constants counted as values")
	}
	before := m.Budget().Used()
	m.Budget().FailAfter(0)
	most, e := f.MaxOccurrenceResult()
	m.Budget().FailAfter(-1)
	astTestMust(t, e)
	if most != no || m.Budget().Used() != before {
		t.Fatal("source running maximum tie without arena-backed map", most, m.Budget().Used(), before)
	}
}

func TestFunctionInterpretationNativeOccurrenceGrowthAndEmptyResult(t *testing.T) {
	m := astTestManager(t)
	f := modelTestFunction(t, m, 1)
	empty, e := f.MaxOccurrenceResult()
	astTestMust(t, e)
	if !empty.IsNull() {
		t.Fatal("empty graph occurrence result", empty)
	}
	results := make([]ast.ASTHandle, 16)
	for i := range results {
		results[i] = goalTestAtom(t, m, fmt.Sprint("occ-result-", i))
		arg := goalTestAtom(t, m, fmt.Sprint("occ-arg-", i))
		astTestMust(t, f.InsertEntry([]ast.ASTHandle{arg}, results[i]))
	}
	for i, result := range []ast.ASTHandle{results[5], results[8]} {
		arg := goalTestAtom(t, m, fmt.Sprint("occ-extra-", i))
		astTestMust(t, f.InsertEntry([]ast.ASTHandle{arg}, result))
	}
	before := m.Budget().Used()
	m.Budget().FailAfter(0)
	most, e := f.MaxOccurrenceResult()
	m.Budget().FailAfter(-1)
	astTestMust(t, e)
	if most != results[5] || m.Budget().Used() != before {
		t.Fatal("first running maximum across native map growth", most, results[5], m.Budget().Used(), before)
	}
}

func TestFunctionInterpretationMultiArgumentPeelingAndElseMiss(t *testing.T) {
	m := astTestManager(t)
	f := modelTestFunction(t, m, 2)
	sort, e := m.MkUninterpretedSort(astTestSymbol(t, m, "two-sort"), nil)
	astTestMust(t, e)
	args := make([]ast.ASTHandle, 2)
	vars := make([]ast.ASTHandle, 2)
	eqs := make([]ast.ASTHandle, 2)
	for i := range args {
		args[i], e = m.MkConst(astTestSymbol(t, m, fmt.Sprint("arg-", i)), sort)
		astTestMust(t, e)
		astTestPin(t, m, args[i])
		vars[i], e = m.MkVar(uint32(i), sort)
		astTestMust(t, e)
		eqs[i], e = m.MkEq(args[i], vars[i])
		astTestMust(t, e)
	}
	cond, e := m.MkAndArgs(eqs)
	astTestMust(t, e)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	term, e := m.MkIte(cond, yes, no)
	astTestMust(t, e)
	astTestPin(t, m, term)
	out := astTestPin(t, m, yes)
	ok, e := f.EvalElse(args, out)
	astTestMust(t, e)
	if ok {
		t.Fatal("partial graph evaluated else")
	}
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if got != yes {
		t.Fatal("miss must preserve output")
	}
	astTestMust(t, f.SetElse(term))
	entry, e := f.FindEntry(args)
	astTestMust(t, e)
	if entry == nil {
		t.Fatal("reversed equality operands not recognized")
	}
	got, e = entry.Result()
	astTestMust(t, e)
	if got != yes {
		t.Fatal(got)
	}
	interp, e := f.Interpretation()
	astTestMust(t, e)
	if interp.IsNull() {
		t.Fatal("missing ITE interpretation")
	}
	astTestMust(t, f.SetElse(ast.ASTHandle{}))
	interp, e = f.Interpretation()
	astTestMust(t, e)
	if !interp.IsNull() {
		t.Fatal("partial graph has no total interpretation")
	}
}

func TestFunctionInterpretationInsertionRefusalsReleaseReferences(t *testing.T) {
	m := astTestManager(t)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	astTestPin(t, m, yes)
	astTestPin(t, m, no)
	args := make([]ast.ASTHandle, 5)
	for i := range args {
		args[i] = goalTestAtom(t, m, fmt.Sprint("fault-", i))
	}
	before := make([]uint32, len(args)+2)
	for i, h := range append(append([]ast.ASTHandle{}, args...), yes, no) {
		before[i], e = m.GetRefCount(h)
		astTestMust(t, e)
	}
	baseline := m.Budget().Used()
	success := false
	for fault := int64(0); fault < 128; fault++ {
		m.Budget().FailAfter(fault)
		f, err := model.NewFunctionInterpretation(m, 1)
		if err == nil {
			for i, arg := range args {
				result := yes
				if i%2 == 0 {
					result = no
				}
				err = f.InsertEntry([]ast.ASTHandle{arg}, result)
				if err != nil {
					break
				}
			}
		}
		if err == nil {
			err = f.SetElse(yes)
		}
		m.Budget().FailAfter(-1)
		if f != nil {
			astTestMust(t, f.Close())
		}
		if m.Budget().Used() != baseline {
			t.Fatal("retained graph allocation", fault, baseline, m.Budget().Used())
		}
		for i, h := range append(append([]ast.ASTHandle{}, args...), yes, no) {
			count, e := m.GetRefCount(h)
			astTestMust(t, e)
			if count != before[i] {
				t.Fatal("retained entry references", fault, i, count, before[i])
			}
		}
		if err == nil {
			success = true
			break
		}
		if !errors.Is(err, arena.ErrInjectedFailure) {
			t.Fatal("unexpected failure", fault, err)
		}
	}
	if !success {
		t.Fatal("never completed graph insertion")
	}
}

func TestFunctionInterpretationCopyEntriesAndOwningOutput(t *testing.T) {
	m := astTestManager(t)
	f := modelTestFunction(t, m, 1)
	yes, e := m.MkTrue()
	astTestMust(t, e)
	no, e := m.MkFalse()
	astTestMust(t, e)
	astTestPin(t, m, yes)
	astTestPin(t, m, no)
	astTestMust(t, f.InsertEntry([]ast.ASTHandle{yes}, no))
	astTestMust(t, f.SetElse(yes))
	copied, e := f.Copy()
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, copied.Close()) })
	astTestMust(t, f.InsertEntry([]ast.ASTHandle{yes}, yes))
	entry, e := copied.FindEntry([]ast.ASTHandle{yes})
	astTestMust(t, e)
	value, e := entry.Result()
	astTestMust(t, e)
	if value != no {
		t.Fatal("copy shared mutable entry")
	}
	interp, e := copied.Interpretation()
	astTestMust(t, e)
	out := astTestPin(t, m, interp)
	astTestMust(t, copied.Close())
	value, e = m.RefGet(out)
	astTestMust(t, e)
	if value != interp {
		t.Fatal("owning output did not survive graph close")
	}
	accidental := *f
	if _, e = accidental.Else(); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("copied graph owner", e)
	}
	astTestMust(t, f.Close())
	if e = f.SetElse(yes); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed graph owner", e)
	}
}
