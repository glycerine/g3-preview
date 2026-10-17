// Source model_converter body assertions and independent finite lifetime checks.
package sat

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/lbool"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func satConverterTestOwner(t *testing.T) (*satModelConverter, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	c, e := newSATModelConverter(b)
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
	paramsMust(t, c.InsertLiterals(a, []Literal{MakeLiteral(0, false), MakeLiteral(1, false)}))
	b, e := c.Mk(satModelElimVar, 1)
	paramsMust(t, e)
	paramsMust(t, c.InsertLiterals(b, []Literal{MakeLiteral(1, false)}))
	m := Model{lbool.Undef, lbool.Undef}
	paramsMust(t, c.Apply(m))
	if m[0] != lbool.Undef || m[1] != lbool.True {
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
	paramsMust(t, c.InsertBinary(a, MakeLiteral(0, true), MakeLiteral(1, false)))
	m = Model{lbool.True, lbool.False}
	paramsMust(t, c.Apply(m))
	if m[0] != lbool.False || m[1] != lbool.False {
		t.Fatalf("source blocked flip %v", m)
	}
	ok, e = c.CheckModel(Model{lbool.True, lbool.False})
	paramsMust(t, e)
	if ok {
		t.Fatal("check_model must detect original saved clause")
	}
}
func TestSATSolverModelConverterSourceOtherUndefinedAndDisplay(t *testing.T) {
	c, _ := satConverterTestOwner(t)
	a, e := c.Mk(satModelElimVar, 0)
	paramsMust(t, e)
	paramsMust(t, c.InsertBinary(a, MakeLiteral(0, true), MakeLiteral(2, true)))
	m := Model{lbool.Undef, lbool.False, lbool.Undef}
	paramsMust(t, c.Apply(m))
	if m[0] != lbool.Undef || m[2] != lbool.False {
		t.Fatalf("source assigns other undefined variable %v", m)
	}
	v, e := c.MaxVar(1)
	paramsMust(t, e)
	if v != 2 {
		t.Fatalf("source max saved literal var %d", v)
	}
	vars, e := c.AppendVars([]BoolVar{99, 0})
	paramsMust(t, e)
	if !reflect.DeepEqual(vars, []BoolVar{99, 0}) {
		t.Fatalf("source collect_vars insertion dedup %v", vars)
	}
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{})
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

func TestSATSolverModelConverterCopyAppendAndSourceBitFields(t *testing.T) {
	s, _ := satConverterTestOwner(t)
	d, _ := satConverterTestOwner(t)
	a, e := s.Mk(satModelElimVar, 0)
	paramsMust(t, e)
	paramsMust(t, s.InsertBinary(a, MakeLiteral(0, false), MakeLiteral(1, false)))
	_, e = d.Mk(satModelBlockLit, 2)
	paramsMust(t, e)
	paramsMust(t, d.Copy(s))
	paramsMust(t, d.Copy(s))
	if d.Size() != 3 {
		t.Fatal("source Copy appends instead of assigning")
	}
	paramsMust(t, s.Reset())
	ok, e := d.CheckModel(Model{lbool.True, lbool.False, lbool.Undef})
	paramsMust(t, e)
	if !ok {
		t.Fatal("copied nested payload borrowed source")
	}
	paramsMust(t, d.Copy(d))
	if d.Size() != 6 {
		t.Fatal("native finite self-copy", d.Size())
	}
	paramsMust(t, d.Reset())
	r, e := d.Mk(satModelElimVar, BoolVar(^uint32(0)))
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
	if k != satModelElimVar || v != NullBoolVar {
		t.Fatalf("source unsigned bitfield truncation k=%d v=%d", k, v)
	}
}

func satConverterAssertActualCharges(t *testing.T, c *satModelConverter, b *arena.Budget) {
	t.Helper()
	want := uint64(reflect.TypeOf(satModelConverter{}).Size())
	if b.Used() != want {
		t.Fatalf("immediate actual header/backing/root charges: %d != %d", b.Used(), want)
	}
}
func TestSATSolverModelConverterIndependentChainModels(t *testing.T) {
	c, b := satConverterTestOwner(t)
	for mask := uint32(0); mask < 32; mask++ {
		paramsMust(t, c.Reset())
		var originals [8][2]Literal
		model := make(Model, 5)
		for i := 0; i < 4; i++ {
			kind := satModelElimVar
			model[i] = lbool.Undef
			if mask&(1<<uint(i)) != 0 {
				kind = satModelBlockLit
				model[i] = lbool.True
			}
			r, e := c.Mk(kind, BoolVar(i))
			paramsMust(t, e)
			originals[2*i] = [2]Literal{MakeLiteral(BoolVar(i), false), MakeLiteral(BoolVar(i+1), false)}
			originals[2*i+1] = [2]Literal{MakeLiteral(BoolVar(i), true), MakeLiteral(BoolVar(i+1), true)}
			paramsMust(t, c.InsertBinary(r, originals[2*i][0], originals[2*i][1]))
			paramsMust(t, c.InsertBinary(r, originals[2*i+1][0], originals[2*i+1][1]))
		}
		model[4] = lbool.False
		if mask&16 != 0 {
			model[4] = lbool.True
		}
		paramsMust(t, c.Apply(model))
		for _, clause := range originals {
			good := false
			for _, l := range clause {
				good = good || model[l.variable()] == lbool.True && !l.sign() || model[l.variable()] == lbool.False && l.sign()
			}
			if !good {
				t.Fatalf("independent original clause mask%d %v model%v", mask, clause, model)
			}
		}
		for i := 0; i < 4; i++ {
			if model[i] == model[i+1] || model[i] == lbool.Undef {
				t.Fatalf("independent parity oracle %v", model)
			}
		}
		satConverterAssertActualCharges(t, c, b)
	}
}
func TestSATSolverModelConverterActualClauseAndWrapperRoutes(t *testing.T) {
	c, b := satConverterTestOwner(t)
	a := satStore(t, nil)
	clause := satMk(t, a, []Literal{MakeLiteral(0, false), MakeLiteral(1, true), MakeLiteral(2, false)}, false)
	r, e := c.Mk(satModelBlockLit, 0)
	paramsMust(t, e)
	paramsMust(t, c.InsertClause(r, a, clause))
	paramsMust(t, c.InsertWrapper(r, a, newSATClauseWrapper(clause)))
	paramsMust(t, c.InsertWrapper(r, a, newSATBinaryWrapper(MakeLiteral(0, false), MakeLiteral(3, false))))
	n := len(c.entries[0].literals)
	if n != 11 {
		t.Fatalf("actual overload scalar stream length%d", n)
	}
	for _, i := range []uint32{3, 7, 10} {
		v := c.entries[0].literals[i]
		if v != DefaultLiteral() {
			t.Fatal("source clause delimiter")
		}
	}
	paramsMust(t, a.Delete(clause))
	ok, e := c.CheckModel(Model{lbool.True, lbool.False, lbool.False, lbool.False})
	paramsMust(t, e)
	if !ok {
		t.Fatal("owned saved data died with clause")
	}
	satConverterAssertActualCharges(t, c, b)
}
func TestSATSolverModelConverterColdRefusalsAndNativeInsert(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 8; prefix++ {
		b := arena.NewBudget(^uint64(0))
		b.FailAfter(prefix)
		c, e := newSATModelConverter(b)
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
	paramsMust(t, c.InsertLiterals(r, []Literal{0, 2, 4}))
	before := b.Used()
	b.FailAfter(0)
	paramsMust(t, c.InsertBinary(r, 1, 3))
	want := []Literal{0, 2, 4, DefaultLiteral(), 1, 3, DefaultLiteral()}
	n := len(c.entries[0].literals)
	if int(n) != len(want) || b.Used() != before {
		t.Fatal("native insert reserved backing", n, b.Used(), before)
	}
	for i, l := range want {
		v := c.entries[0].literals[i]
		if v != l {
			t.Fatal(i, v, l)
		}
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
	nativeBudget := arena.NewBudget(^uint64(0))
	native, e := newSATModelConverter(nativeBudget)
	paramsMust(t, e)
	paramsMust(t, native.Close())
	if nativeBudget.Used() != 0 {
		t.Fatal("native literal store retained cold header")
	}

}

func TestSATSolverModelConverterDebugAssertionBeforeLaterInvalidLiteral(t *testing.T) {
	m, b := satConverterTestOwner(t)
	r, e := m.Mk(satModelBlockLit, 0)
	paramsMust(t, e)
	paramsMust(t, m.InsertBinary(r, MakeLiteral(0, false), MakeLiteral(1, false)))
	paramsMust(t, m.InsertBinary(r, MakeLiteral(0, true), MakeLiteral(0, true)))
	paramsMust(t, m.InsertBinary(r, MakeLiteral(5, false), MakeLiteral(0, false)))
	model := Model{lbool.True, lbool.False}
	e = m.Apply(model)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatalf("source per-separator assertion must precede later invalid var: %v", e)
		}
	} else {
		paramsMust(t, e)
	}
	if model[0] != lbool.False || model[1] != lbool.False {
		t.Fatalf("completed source assignment prefix %v", model)
	}
	if _, e = m.CheckModel(model); !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("CheckModel must still traverse later invalid variable: %v", e)
	}
	satConverterAssertActualCharges(t, m, b)
}

func TestSATSolverModelConverterReadsProcessAssertionSwitchAtReachedSites(t *testing.T) {
	m, _ := satConverterTestOwner(t)
	r, e := m.Mk(satModelElimVar, 0)
	paramsMust(t, e)
	paramsMust(t, m.InsertBinary(r, MakeLiteral(0, false), MakeLiteral(1, false)))
	debug, _, _ := satIntegrityActionFixture(t, "")
	model := Model{lbool.True, lbool.False}
	if e = m.Apply(model); satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatalf("enabled eliminated-variable assertion: %v", e)
		}
	} else {
		paramsMust(t, e)
	}
	paramsMust(t, debug.EnableAssertions(false))
	paramsMust(t, m.Apply(model))
	if satStorageSourceDebug {
		x := &satModelConversionEntry{literals: []Literal{DefaultLiteral()}}
		paramsMust(t, m.assertEntryModelSpan(x, satSolverModelSpan{slice: model}))
		paramsMust(t, debug.EnableAssertions(true))
		if e = m.assertEntryModelSpan(x, satSolverModelSpan{slice: model}); !errors.Is(e, ErrSATAssertion) {
			t.Fatalf("enabled per-clause assertion: %v", e)
		}
		paramsMust(t, debug.Close())
		if e = m.Apply(model); !errors.Is(e, arena.ErrClosed) {
			t.Fatalf("reached eliminated-variable switch on closed owner: %v", e)
		}
		if e = m.assertEntryModelSpan(x, satSolverModelSpan{slice: model}); !errors.Is(e, arena.ErrClosed) {
			t.Fatalf("reached per-clause switch on closed owner: %v", e)
		}
		satisfied := &satModelConversionEntry{literals: []Literal{MakeLiteral(0, false), DefaultLiteral()}}
		if e = m.assertEntryModelSpan(satisfied, satSolverModelSpan{slice: model}); !errors.Is(e, arena.ErrClosed) {
			t.Fatalf("satisfied clause must still read its reached assertion switch: %v", e)
		}
	}
}

func TestSATSolverNativeModelEntryGrowthStableRefsAndCopy(t *testing.T) {
	src, sb := satConverterTestOwner(t)
	dst, db := satConverterTestOwner(t)
	before, other := sb.Used(), db.Used()
	sb.FailAfter(0)
	db.FailAfter(0)
	refs := make([]satModelConversionEntryRef, 4097)
	for i := range refs {
		r, e := src.Mk(satModelBlockLit, BoolVar(i))
		paramsMust(t, e)
		refs[i] = r
		paramsMust(t, src.InsertBinary(r, MakeLiteral(BoolVar(i), false), MakeLiteral(BoolVar(i+1), true)))
	}
	for i, r := range refs {
		k, v, e := src.KindAndVar(r)
		paramsMust(t, e)
		if k != satModelBlockLit || v != BoolVar(i) {
			t.Fatal("native growth retired stable entry index", i, k, v)
		}
	}
	paramsMust(t, dst.Copy(src))
	if dst.Size() != src.Size() {
		t.Fatal("copy size")
	}
	for i := range refs {
		want := []Literal{MakeLiteral(BoolVar(i), false), MakeLiteral(BoolVar(i+1), true), DefaultLiteral()}
		if dst.entries[i].varKind != src.entries[i].varKind || len(dst.entries[i].literals) != len(want) {
			t.Fatal("native copied entry", i)
		}
		for j, l := range want {
			if dst.entries[i].literals[j] != l {
				t.Fatal("deep copy values", i, j)
			}
		}
	}
	src.entries[0].literals[0] = 123
	if dst.entries[0].literals[0] != 0 {
		t.Fatal("native copy aliases source literals")
	}
	first, e := dst.EntryAt(0)
	paramsMust(t, e)
	paramsMust(t, dst.Copy(dst))
	if dst.Size() != 8194 {
		t.Fatal("self-copy did not capture original count", dst.Size())
	}
	dst.entries[0].literals[0] = 321
	if dst.entries[4097].literals[0] != 0 {
		t.Fatal("self-copy aliases nested backing")
	}
	if _, _, e = dst.KindAndVar(first); e != nil {
		t.Fatal("native append changed prior entry lifetime", e)
	}
	paramsMust(t, src.Reset())
	if _, _, e = src.KindAndVar(refs[0]); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("reset revived entry", e)
	}
	if dst.entries[4097].literals[0] != 0 {
		t.Fatal("source reset changed independent copied values")
	}
	if sb.Used() != before || db.Used() != other {
		t.Fatal("native entries/copy charged backing", sb.Used(), db.Used())
	}
	sb.FailAfter(-1)
	db.FailAfter(-1)
}

func TestSATSolverNativeModelColdBudgetAndIndependentReconstruction(t *testing.T) {
	b := arena.NewBudget(uint64(reflect.TypeOf(satModelConverter{}).Size()))
	b.FailAfter(1)
	src, e := newSATModelConverter(b)
	paramsMust(t, e)
	for i := uint32(0); i < 257; i++ {
		r, e := src.Mk(satModelElimVar, BoolVar(i))
		paramsMust(t, e)
		paramsMust(t, src.InsertBinary(r, MakeLiteral(BoolVar(i), false), MakeLiteral(BoolVar(i+1), true)))
	}
	dst, db := satConverterTestOwner(t)
	db.FailAfter(0)
	paramsMust(t, dst.Copy(src))
	db.FailAfter(-1)
	paramsMust(t, src.Close())
	paramsMust(t, src.Close())
	if b.Used() != 0 {
		t.Fatal("cold header refund", b.Used())
	}
	// Each clause is v_i OR !v_(i+1); with terminal true, reverse patch makes all true.
	model := make(Model, 258)
	for i := range model {
		model[i] = lbool.Undef
	}
	model[257] = lbool.True
	paramsMust(t, dst.Apply(model))
	for i, v := range model {
		if v != lbool.True {
			t.Fatal("independent implication-chain reconstruction", i, v)
		}
	}
	ok, e := dst.CheckModel(model)
	paramsMust(t, e)
	if !ok {
		t.Fatal("reconstructed model rejected")
	}
}
