// Source: src/ast/ast_ll_pp.cpp:168-184,217-236,292-297;
// src/ast/for_each_ast.h:40-125.
package ast

import (
	"errors"
	"math"
	"reflect"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/util"
)

func TestASTLowLevelPrinterActualPostorderAndBorrowedStreamFlags(t *testing.T) {
	m := astTestManager(t)
	bs, e := m.MkBoolSort()
	astTestMust(t, e)
	v, e := m.MkVar(7, bs)
	astTestMust(t, e)
	astTestPin(t, m, v)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "f"), []ASTHandle{bs}, bs, ASTFuncDeclInfo{})
	astTestMust(t, e)
	astTestPin(t, m, d)
	a, e := m.MkApp(d, []ASTHandle{v})
	astTestMust(t, e)
	astTestPin(t, m, a)
	b := arena.NewBudget(1 << 20)
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: b})
	astTestMust(t, e)
	defer out.Close()
	astTestMust(t, out.SetShowPos(true))
	astTestMust(t, m.DisplayLowLevel(out, a, true, true))
	id, e := m.GetASTID(v)
	astTestMust(t, e)
	want := "#" + strconv.FormatUint(uint64(id), 10) + " := (:var 7 Bool)\n(f #" + strconv.FormatUint(uint64(id), 10) + ")\n"
	got, e := out.String()
	astTestMust(t, e)
	if got != want {
		t.Fatal("literal postorder, unsigned field formatting", got, want)
	}
	if m.LowLevel.Active || m.LowLevel.capacity != 0 || m.LowLevel.marks.exprs.bytes != 0 || m.LowLevel.marks.decls.bytes != 0 {
		t.Fatal("source lexical stack/mark DT lost roots")
	}
}

func TestASTLowLevelPrinterDeclarationMarksUseActualDeclarationIndex(t *testing.T) {
	sy, q, _ := astTestOwners(t)
	b := arena.NewBudget(1 << 20)
	m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	astTestMust(t, e)
	defer m.Close()
	bs, e := m.MkBoolSort()
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "bounded-decl"), nil, bs, ASTFuncDeclInfo{})
	astTestMust(t, e)
	a, e := m.MkApp(d, nil)
	astTestMust(t, e)
	astTestMust(t, m.IncRef(a))
	defer func() { astTestMust(t, m.DecRef(a)) }()
	sb := arena.NewBudget(1 << 20)
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: sb})
	astTestMust(t, e)
	defer out.Close()
	astTestMust(t, m.DisplayLowLevel(out, a, true, true))
	got, e := out.String()
	astTestMust(t, e)
	if got != "bounded-decl\n" {
		t.Fatal("source declaration bitmap uses get_decl_id, not tagged global id", got)
	}
	if b.Used() != m.Stats().AllocatedBytes {
		t.Fatal("source lexical bitmap graph", b.Used(), m.Stats())
	}
}

// The existing Runtime ErrorTransport is a supported Go boundary, not a
// native ostream callback. Only public Runtime actions establish the signed
// accounting guard; no allocator fields or acquired owner tuple are edited.
func TestASTLowLevelPrinterYoungerNumeralStopProtectsActualOlderFrame(t *testing.T) {
	bn := arena.NewBudget(1 << 28)
	bs := arena.NewBudget(1 << 24)
	ba := arena.NewBudget(1 << 24)
	var r *util.Runtime
	var m *ASTManager
	var q *util.RationalService
	armed, called := false, false
	var callbackClose, callbackNumericClose, callbackNumericRead error
	var e error
	r, e = util.NewRuntime(util.RuntimeConfig{Budget: bn, ThreadLocal: true, SizeTBits: 64, Transport: func(f util.Failure) error {
		if armed {
			armed = false
			called = true
			paramsMust(t, r.SetMaxSize(0))
			callbackClose = m.Close()
			func() {
				defer func() {
					if got := recover(); got != ErrASTPending {
						t.Fatal("callback Stats typed guard", got)
					}
				}()
				m.Stats()
			}()
			func() {
				defer func() {
					if got := recover(); got != ErrASTPending {
						t.Fatal("callback constructor observer typed guard", got)
					}
				}()
				m.ConstructorState()
			}()
			callbackNumericClose = q.Close()
			_, callbackNumericRead = q.Stats()
			// Include the real preexisting large numeral allocations in the
			// source delta, unlike the small baseline Params fixture.
			a, err := r.Allocate(8)
			paramsMust(t, err)
			bb, err := r.Allocate(8)
			paramsMust(t, err)
			delta := -r.State().GlobalSize - 1000000
			if err = r.Reallocate(&a, uint64(16)+uint64(delta)-8); !errors.Is(err, arena.ErrBounds) {
				t.Fatal(err)
			}
			paramsMust(t, r.Deallocate(&a))
			state := r.State()
			if state.LocalSize != 0 || state.GlobalSize >= 0 {
				t.Fatal("actual first sync", state)
			}
			delta = math.MinInt64 + 1000 - state.GlobalSize + 16
			if err = r.Reallocate(&bb, uint64(delta)+16-8); !errors.Is(err, arena.ErrBounds) {
				t.Fatal(err)
			}
			paramsMust(t, r.Deallocate(&bb))
			if state = r.State(); state.LocalSize != 0 || state.GlobalSize != math.MinInt64+1000 {
				t.Fatal("actual guarded lane", state)
			}
		}
		return f
	}})
	paramsMust(t, e)
	q, e = util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: r, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := util.NewSymbolManager(util.SymbolConfig{Budget: bs})
	paramsMust(t, e)
	anchor, e := polynomial.NewUPolynomialManagerFromRationalService(q, polynomial.UPolynomialConfig{Checkpoint: func() bool { panic("unused checkpoint") }, Cooperation: func() error { panic("unused cooperation") }})
	paramsMust(t, e)
	scoped, e := util.NewScopedInt(anchor.IntegerManager())
	paramsMust(t, e)
	num, e := scoped.Get()
	paramsMust(t, e)
	limbs := make([]uint32, 32768)
	limbs[0], limbs[len(limbs)-1] = 17, 1
	paramsMust(t, anchor.IntegerManager().SetLimbs(num, limbs))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	paramsMust(t, scoped.Close())
	m, e = NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: ba})
	paramsMust(t, e)
	sort, e := m.MkBoolSort()
	paramsMust(t, e)
	rat, e := m.NewParameterRational(&big)
	paramsMust(t, e)
	integer, e := m.NewParameterInt(1)
	paramsMust(t, e)
	sym, e := sy.Intern("arith")
	paramsMust(t, e)
	fid, e := m.families.MkID(sym)
	paramsMust(t, e)
	info, e := m.NewFuncDeclInfo(fid, 0, []ASTParameter{rat, integer}, ASTFuncFlags{}, false)
	paramsMust(t, e)
	name, e := sy.Intern("numeral")
	paramsMust(t, e)
	decl, e := m.MkFuncDecl(name, nil, sort, info)
	paramsMust(t, e)
	app, e := m.MkApp(decl, nil)
	paramsMust(t, e)
	paramsMust(t, m.IncRef(app))
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: ba})
	paramsMust(t, e)
	before := r.State()
	paramsMust(t, r.SetMaxSize(uint64(before.GlobalSize+1000000)))
	armed = true
	e = m.DisplayLowLevel(out, app, true, true)
	if !called {
		t.Fatal("actual later numeral allocation never reached transport", e, r.State(), before)
	}
	if !errors.Is(callbackNumericClose, arena.ErrInvalidSlot) || !errors.Is(callbackNumericRead, arena.ErrInvalidSlot) {
		t.Fatal("same-service insertion callback guard", callbackNumericClose, callbackNumericRead)
	}
	if !errors.Is(callbackClose, ErrASTPending) {
		t.Fatal("executing callback cleanup guard", callbackClose)
	}
	if !errors.Is(e, arena.ErrBounds) || !errors.Is(m.LowLevel.destructorCause, arena.ErrBounds) || !m.LowLevel.Active || !m.LowLevel.NumeralAcquired || m.LowLevel.capacity == 0 {
		t.Fatal("younger source stop destroyed older lexical owner", e, m.LowLevel)
	}
	z := reflect.ValueOf(anchor.IntegerManager()).Elem()
	natural := testFieldBySourceName(z, "natural").Elem()
	completed := testFieldBySourceName(testFieldBySourceName(natural, "formatting"), "completed").Uint()
	cell := testFieldBySourceName(testFieldBySourceName(testFieldBySourceName(reflect.ValueOf(&m.LowLevel.Numeral).Elem(), "raw"), "num"), "cell").Uint()
	cells := testFieldBySourceName(z, "cells").Elem()
	pageSize := testFieldBySourceName(testFieldBySourceName(cells, "config"), "PageSize").Uint()
	slot := uint64(uint32(cell))
	released := testFieldBySourceName(testFieldBySourceName(testFieldBySourceName(cells, "pages").Index(int((slot-1)/pageSize)), "records").Index(int((slot-1)%pageSize)), "released").Bool()
	t.Log("actual source formatting prefix", completed, testFieldBySourceName(testFieldBySourceName(natural, "charFormatting"), "completed").Bool(), released)
	if completed != 0 && released {
		t.Fatal("younger source destructor stop consumed the older lexical numeral")
	}
	upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{anchor}, nil, upolyPhysicalExtraOwners{ASTManagers: []*ASTManager{m}, RationalServices: []*util.RationalService{q}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &big}}})
	if ba.Used() != m.Stats().AllocatedBytes+out.Stats().OwnerBytes+out.Stats().ScratchBytes+out.Stats().OutputOwnerBytes+out.Stats().OutputOverflowBytes {
		t.Fatal("immediate older physical/cold graph", ba.Used(), m.Stats(), out.Stats())
	}
	// Match the actual younger source-value cursor first. This explicit ACK
	// ends only completed child locals/char, before older NUM/stack DT.
	paramsDecFinish(t, func() error { return q.CompleteNumeralInsertion(&m.LowLevel.numeralInsertion, false) })
	if !m.LowLevel.NumeralAcquired || m.LowLevel.capacity == 0 {
		t.Fatal("false ACK consumed older AST values")
	}
	if testFieldBySourceName(testFieldBySourceName(natural, "formatting"), "completed").Uint() != 0 || testFieldBySourceName(testFieldBySourceName(natural, "charFormatting"), "completed").Bool() {
		t.Fatal("actual source child values not ended")
	}
	// A later source-valid new char constructor is foreign to the old receipt.
	// Its real consumed-old-buffer signed-accounting guard retains exact
	// unjoined allocation roots; this is Go guard-domain, not native OOM.
	var foreign util.NumeralInsertionReceipt
	if e = q.InsertIntoTracked(out, &big, &foreign); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual foreign char constructor", e)
	}
	foreignBefore := foreign
	charBuffer := testFieldBySourceName(testFieldBySourceName(natural, "charFormatting"), "buffer")
	foreignToken := func() [6]uint64 {
		var tuple [6]uint64
		for i, name := range []string{"raw", "pending"} {
			block := testFieldBySourceName(charBuffer, name)
			tuple[3*i] = testFieldBySourceName(block, "origin").Uint()
			tuple[3*i+1] = testFieldBySourceName(block, "slot").Uint()
			tuple[3*i+2] = testFieldBySourceName(block, "generation").Uint()
		}
		return tuple
	}
	foreignRaw := foreignToken()
	if foreignRaw == ([6]uint64{}) {
		t.Fatal("foreign constructor has no actual raw acquisition")
	}
	upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{anchor}, nil, upolyPhysicalExtraOwners{ASTManagers: []*ASTManager{m}, RationalServices: []*util.RationalService{q}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &big}}})
	if ba.Used() != m.Stats().AllocatedBytes+out.Stats().OwnerBytes+out.Stats().ScratchBytes+out.Stats().OutputOwnerBytes+out.Stats().OutputOverflowBytes {
		t.Fatal("foreign-before-old-ACK cold graph", ba.Used(), m.Stats(), out.Stats())
	}
	if !foreign.Active() {
		t.Fatal("foreign unjoined ctor receipt lost")
	}
	paramsDecFinish(t, m.Close)
	if !reflect.DeepEqual(foreign, foreignBefore) || foreignToken() != foreignRaw {
		t.Fatal("old true ACK changed foreign invocation/history/raw tuple", foreign, foreignBefore, foreignToken(), foreignRaw)
	}
	upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{anchor}, nil, upolyPhysicalExtraOwners{RationalServices: []*util.RationalService{q}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &big}}})
	if ba.Used() != out.Stats().OwnerBytes+out.Stats().ScratchBytes+out.Stats().OutputOwnerBytes+out.Stats().OutputOverflowBytes {
		t.Fatal("foreign-after-old-ACK cold graph", ba.Used(), out.Stats())
	}
	paramsDecFinish(t, func() error { return q.CompleteNumeralInsertion(&foreign, false) })
	paramsDecFinish(t, func() error { return q.CompleteNumeralInsertion(&foreign, true) })
	paramsMust(t, out.Close())
	paramsDecFinish(t, func() error { return q.Del(&big) })
	paramsDecFinish(t, anchor.Close)
	paramsDecFinish(t, q.Close)
	paramsDecFinish(t, r.Close)
	paramsMust(t, sy.Close())
	if bn.Used() != 0 || bs.Used() != 0 || ba.Used() != 0 {
		t.Fatal("owning roots lost", bn.Used(), bs.Used(), ba.Used())
	}
}

// Source: src/ast/ast_ll_pp.cpp:292-297; src/util/mpz.cpp:1385-1410;
// src/ast/ast.cpp:1397-1479. Failed char construction has no source DT;
// source AST DT ends its own members, then owning Close ACKs exact unjoined child.
func TestASTSourceFinalizeKeepsFailedNumeralConstructorForExactOwnClose(t *testing.T) {
	bn, ba, bs := arena.NewBudget(1<<28), arena.NewBudget(1<<24), arena.NewBudget(1<<20)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: bn, ThreadLocal: true, SizeTBits: 64})
	paramsMust(t, e)
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: r, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := util.NewSymbolManager(util.SymbolConfig{Budget: bs})
	paramsMust(t, e)
	anchor, e := polynomial.NewUPolynomialManagerFromRationalService(q, polynomial.UPolynomialConfig{Checkpoint: func() bool { panic("unused") }, Cooperation: func() error { panic("unused") }})
	paramsMust(t, e)
	scoped, e := util.NewScopedInt(anchor.IntegerManager())
	paramsMust(t, e)
	value, e := scoped.Get()
	paramsMust(t, e)
	limbs := make([]uint32, 32768)
	limbs[0], limbs[32767] = 7, 1
	paramsMust(t, anchor.IntegerManager().SetLimbs(value, limbs))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	paramsMust(t, scoped.Close())
	m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: ba})
	paramsMust(t, e)
	sort, e := m.MkBoolSort()
	paramsMust(t, e)
	rat, e := m.NewParameterRational(&big)
	paramsMust(t, e)
	integer, e := m.NewParameterInt(1)
	paramsMust(t, e)
	arith, e := sy.Intern("arith")
	paramsMust(t, e)
	family, e := m.families.MkID(arith)
	paramsMust(t, e)
	info, e := m.NewFuncDeclInfo(family, 0, []ASTParameter{rat, integer}, ASTFuncFlags{}, false)
	paramsMust(t, e)
	symbol, e := sy.Intern("numeral")
	paramsMust(t, e)
	decl, e := m.MkFuncDecl(symbol, nil, sort, info)
	paramsMust(t, e)
	app, e := m.MkApp(decl, nil)
	paramsMust(t, e)
	paramsMust(t, m.IncRef(app))
	out, e := util.NewNumericTextStream(util.NumericTextStreamConfig{Budget: ba})
	paramsMust(t, e)
	paramsMust(t, r.SetMaxSize(uint64(r.State().GlobalSize+200000)))
	body := m.DisplayLowLevel(out, app, true, true)
	var failure util.Failure
	if !errors.As(body, &failure) || failure.Kind != util.FailureMemout {
		t.Fatal("real failed char constructor", body)
	}
	if !m.LowLevel.Active || !m.LowLevel.sourceDone || !m.LowLevel.numeralInsertion.Active() || m.LowLevel.NumeralAcquired || m.LowLevel.capacity != 0 {
		t.Fatal("failed ctor vs completed older source DT", m.LowLevel)
	}
	upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{anchor}, nil, upolyPhysicalExtraOwners{ASTManagers: []*ASTManager{m}, RationalServices: []*util.RationalService{q}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &big}}})
	if ba.Used() != m.Stats().AllocatedBytes+out.Stats().OwnerBytes+out.Stats().ScratchBytes+out.Stats().OutputOwnerBytes+out.Stats().OutputOverflowBytes {
		t.Fatal("immediate failed ctor cold graph")
	}
	captured := m.LowLevel.numeralInsertion
	paramsMust(t, m.SourceFinalize())
	if !m.sourceDTCompleted || !m.LowLevel.numeralInsertion.Active() || !reflect.DeepEqual(captured, m.LowLevel.numeralInsertion) {
		t.Fatal("source DT silently owning-ACKed incomplete child")
	}
	upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{anchor}, nil, upolyPhysicalExtraOwners{ASTManagers: []*ASTManager{m}, RationalServices: []*util.RationalService{q}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &big}}})
	paramsMust(t, m.Close())
	if m.LowLevel.numeralInsertion.Active() || m.LowLevel.Active {
		t.Fatal("own Close skipped exact retained child")
	}
	paramsMust(t, out.Close())
	paramsMust(t, q.Del(&big))
	paramsMust(t, anchor.Close())
	paramsMust(t, q.Close())
	paramsMust(t, r.Close())
	paramsMust(t, sy.Close())
	if bn.Used() != 0 || ba.Used() != 0 || bs.Used() != 0 {
		t.Fatal("actual named owner refunds", bn.Used(), ba.Used(), bs.Used())
	}
}
