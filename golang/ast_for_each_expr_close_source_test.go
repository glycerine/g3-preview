// Licensed under the MIT license; see LICENSE.txt.
// Source: src/ast/for_each_expr.h:27-105; src/ast/ast_ll_pp.cpp:292-297; src/ast/ast.cpp:1397-1479.
// Source author: Leonardo de Moura.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/mpnum"
	"math"
	"reflect"
	"testing"
)

func TestASTExprVisitorHistorySurvivesActualLaterNumeralCloseStop(t *testing.T) {
	bn := arena.NewBudget(1 << 28)
	bs := arena.NewBudget(1 << 24)
	ba := arena.NewBudget(1 << 24)
	var r *memory.Runtime
	var m *ASTManager
	var q *mpnum.RationalService
	armed, called := false, false
	var callbackClose, callbackNumericClose, callbackNumericRead error
	var e error
	r, e = memory.NewRuntime(memory.RuntimeConfig{Budget: bn, ThreadLocal: true, SizeTBits: 64, Transport: func(f memory.Failure) error {
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
	q, e = mpnum.NewRationalService(mpnum.RationalServiceConfig{Integers: mpnum.IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := NewSymbolManager(SymbolConfig{Budget: bs})
	paramsMust(t, e)
	anchor, e := q.NewUPolynomialManager(mpnum.UPolynomialConfig{Checkpoint: func() bool { panic("unused checkpoint") }, Cooperation: func() error { panic("unused cooperation") }})
	paramsMust(t, e)
	scoped, e := mpnum.NewScopedInt(anchor.IntegerManager())
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
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: ba})
	paramsMust(t, e)
	// Actual source traversal refusal then ordinary retry retains one exact
	// source-ended mark action. No count/registry inference establishes it.
	visitArgs := make([]ASTHandle, 18)
	visitDomain := make([]ASTHandle, 18)
	for i := range visitArgs {
		visitArgs[i], e = m.MkVar(uint32(69000+i), sort)
		paramsMust(t, e)
		paramsMust(t, m.IncRef(visitArgs[i]))
		visitDomain[i] = sort
	}
	vd, e := m.MkFuncDecl(astTestSymbol(t, m, "close-history-f"), visitDomain, sort, ASTFuncDeclInfo{})
	paramsMust(t, e)
	vr, e := m.MkApp(vd, visitArgs)
	paramsMust(t, e)
	ba.FailAfter(0)
	_, e = m.GetNumExprs(vr)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	tuple := m.exprVisit.pendingMark
	markOwner := m.exprVisit.pendingOwner
	invocation := m.exprVisit.invocation
	bodyHistory := m.exprVisit.bodyCause
	if ba.Used() != m.Stats().AllocatedBytes+out.Stats().OwnerBytes+out.Stats().ScratchBytes+out.Stats().OutputOwnerBytes+out.Stats().OutputOverflowBytes {
		t.Fatal("before-lift exact visitor graph")
	}
	ba.FailAfter(-1)
	_, e = m.GetNumExprs(vr)
	if astSourceDebug && !errors.Is(e, ErrASTDomain) || !astSourceDebug && e != nil {
		t.Fatal("actual profile retry", e)
	}
	history := m.exprVisitHistory
	if history == nil || history.node != tuple || history.owner != markOwner || history.invocation != invocation || history.bodyCause != bodyHistory {
		t.Fatal("missing exact archived action")
	}
	// A second actual failed traversal remains in the current lexical frame,
	// rather than an archive node. The same public Close must preserve its
	// exact diagnostics after ACK through the later older-member failure.
	currentArgs := make([]ASTHandle, 18)
	currentDomain := make([]ASTHandle, 18)
	for i := range currentArgs {
		currentArgs[i], e = m.MkVar(uint32(70000+i), sort)
		paramsMust(t, e)
		paramsMust(t, m.IncRef(currentArgs[i]))
		currentDomain[i] = sort
	}
	cd, e := m.MkFuncDecl(astTestSymbol(t, m, "current-close-history-f"), currentDomain, sort, ASTFuncDeclInfo{})
	paramsMust(t, e)
	cr, e := m.MkApp(cd, currentArgs)
	paramsMust(t, e)
	ba.FailAfter(0)
	_, e = m.GetNumExprs(cr)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("real current-frame refusal", e)
	}
	currentNode := m.exprVisit.unlistedNode
	currentOwner := m.exprVisit.unlistedOwner
	currentSerial := m.exprVisit.invocation
	currentBody := m.exprVisit.bodyCause
	currentSource := m.exprVisit.sourceCause
	if currentNode.slot == 0 || currentNode == tuple || currentBody == nil || currentSource != nil || ba.Used() != m.Stats().AllocatedBytes+out.Stats().OwnerBytes+out.Stats().ScratchBytes+out.Stats().OutputOwnerBytes+out.Stats().OutputOverflowBytes {
		t.Fatal("current exact stopped graph")
	}
	ba.FailAfter(-1)
	// An actual foreign mark blocks owning acknowledgment and stamps first own
	// cause. Reset is performed by that foreign mark owner, not by this frame.
	ph := m.external(tuple)
	paramsMust(t, m.ResetNodeMark1(ph, markOwner))
	const foreignMarker = uint64(9223372036854774000)
	paramsMust(t, m.MarkNode1(ph, true, foreignMarker))
	if err := m.Close(); !errors.Is(err, ErrASTDomain) || !errors.Is(history.ownCause, ErrASTDomain) || m.closed {
		t.Fatal("actual foreign owner refusal", err)
	}
	ownHistory := history.ownCause
	paramsMust(t, m.ResetNodeMark1(ph, foreignMarker))
	// A separate live foreign mark must survive the younger visitor ACK and
	// later LL close stop; its bit/token is not an owning discovery mechanism.
	foreignNode := visitArgs[0]
	foreignRef, e := m.internal(foreignNode)
	paramsMust(t, e)
	paramsMust(t, m.MarkNode1(foreignNode, true, foreignMarker))
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
	if !errors.Is(e, arena.ErrBounds) || !errors.Is(m.lowLevel.destructorCause, arena.ErrBounds) || !m.lowLevel.active || !m.lowLevel.numeralAcquired || m.lowLevel.capacity == 0 {
		t.Fatal("younger source stop destroyed older lexical owner", e, m.lowLevel)
	}
	z := reflect.ValueOf(anchor.IntegerManager()).Elem()
	natural := z.FieldByName("natural").Elem()
	completed := natural.FieldByName("formatting").FieldByName("completed").Uint()
	cell := reflect.ValueOf(&m.lowLevel.numeral).Elem().FieldByName("raw").FieldByName("num").FieldByName("cell").Uint()
	cells := z.FieldByName("cells").Elem()
	pageSize := cells.FieldByName("config").FieldByName("PageSize").Uint()
	slot := uint64(uint32(cell))
	released := cells.FieldByName("pages").Index(int((slot - 1) / pageSize)).FieldByName("records").Index(int((slot - 1) % pageSize)).FieldByName("released").Bool()
	t.Log("actual source formatting prefix", completed, natural.FieldByName("charFormatting").FieldByName("completed").Bool(), released)
	if completed != 0 && released {
		t.Fatal("younger source destructor stop consumed the older lexical numeral")
	}
	upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{anchor}, nil, upolyPhysicalExtraOwners{ASTManagers: []*ASTManager{m}, RationalServices: []*mpnum.RationalService{q}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &big}}})
	if ba.Used() != m.Stats().AllocatedBytes+out.Stats().OwnerBytes+out.Stats().ScratchBytes+out.Stats().OutputOwnerBytes+out.Stats().OutputOverflowBytes {
		t.Fatal("immediate older physical/cold graph", ba.Used(), m.Stats(), out.Stats())
	}
	// Real enclosing Close ACKs visitor first, then reaches the stopped
	// numeric source child. Inspect original charged receipt BEFORE any retry.
	laterClose := m.Close()
	if !errors.Is(laterClose, arena.ErrBounds) || m.closed {
		t.Fatal("actual later AST Close stop", laterClose)
	}
	if m.exprVisitHistory != history || history.node != tuple || history.owner != markOwner || history.invocation != invocation || history.bodyCause != bodyHistory || history.ownCause != ownHistory || !history.acknowledged {
		t.Fatal("enclosing Close refunded/lost acknowledged visitor history")
	}
	if m.exprVisit.active || m.exprVisit.unlistedNode != currentNode || m.exprVisit.unlistedOwner != currentOwner || m.exprVisit.invocation != currentSerial || m.exprVisit.bodyCause != currentBody || m.exprVisit.sourceCause != currentSource || m.exprVisit.ownCause != nil {
		t.Fatal("current unarchived tuple/history lost after later Close stop")
	}
	if !m.nodeAt(foreignRef.slot).mark1 || m.nodeAt(foreignRef.slot).markOwner1 != foreignMarker {
		t.Fatal("foreign mark consumed by older close stop")
	}
	if ba.Used() != m.Stats().AllocatedBytes+out.Stats().OwnerBytes+out.Stats().ScratchBytes+out.Stats().OutputOwnerBytes+out.Stats().OutputOverflowBytes {
		t.Fatal("stopped enclosing close cold graph")
	}
	upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{anchor}, nil, upolyPhysicalExtraOwners{ASTManagers: []*ASTManager{m}, RationalServices: []*mpnum.RationalService{q}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &big}}})
	// A later foreign mark at the acknowledged archive's exact node must
	// remain unread by repeated owning Close, while the older numeric child
	// reaches its next real consumed-accounting continuation.
	paramsMust(t, m.MarkNode1(ph, true, foreignMarker))
	repeatedClose := m.Close()
	if !errors.Is(repeatedClose, arena.ErrBounds) || m.closed {
		t.Fatal("actual repeated older Close stop", repeatedClose)
	}
	if m.exprVisitHistory != history || !history.acknowledged || history.bodyCause != bodyHistory || history.ownCause != ownHistory || !m.nodeAt(tuple.slot).mark1 || m.nodeAt(tuple.slot).markOwner1 != foreignMarker || m.exprVisit.unlistedNode != currentNode || m.exprVisit.bodyCause != currentBody {
		t.Fatal("repeated Close replayed exact visitor ACK or replaced immutable history")
	}
	if ba.Used() != m.Stats().AllocatedBytes+out.Stats().OwnerBytes+out.Stats().ScratchBytes+out.Stats().OutputOwnerBytes+out.Stats().OutputOverflowBytes {
		t.Fatal("repeated stopped enclosing graph")
	}
	upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{anchor}, nil, upolyPhysicalExtraOwners{ASTManagers: []*ASTManager{m}, RationalServices: []*mpnum.RationalService{q}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &big}}})
	// Match the actual younger source-value cursor first. This explicit ACK
	// ends only completed child locals/char, before older NUM/stack DT.
	paramsDecFinish(t, func() error { return q.CompleteNumeralInsertion(&m.lowLevel.numeralInsertion, false) })
	if !m.lowLevel.numeralAcquired || m.lowLevel.capacity == 0 {
		t.Fatal("false ACK consumed older AST values")
	}
	if natural.FieldByName("formatting").FieldByName("completed").Uint() != 0 || natural.FieldByName("charFormatting").FieldByName("completed").Bool() {
		t.Fatal("actual source child values not ended")
	}
	// A later source-valid new char constructor is foreign to the old receipt.
	// Its real consumed-old-buffer signed-accounting guard retains exact
	// unjoined allocation roots; this is Go guard-domain, not native OOM.
	var foreign mpnum.NumeralInsertionReceipt
	if e = q.InsertIntoTracked(out, &big, &foreign); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual foreign char constructor", e)
	}
	foreignBefore := foreign
	charBuffer := natural.FieldByName("charFormatting").FieldByName("buffer")
	foreignToken := func() [6]uint64 {
		var tuple [6]uint64
		for i, name := range []string{"raw", "pending"} {
			block := charBuffer.FieldByName(name)
			tuple[3*i] = block.FieldByName("origin").Uint()
			tuple[3*i+1] = block.FieldByName("slot").Uint()
			tuple[3*i+2] = block.FieldByName("generation").Uint()
		}
		return tuple
	}
	foreignRaw := foreignToken()
	if foreignRaw == ([6]uint64{}) {
		t.Fatal("foreign constructor has no actual raw acquisition")
	}
	upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{anchor}, nil, upolyPhysicalExtraOwners{ASTManagers: []*ASTManager{m}, RationalServices: []*mpnum.RationalService{q}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &big}}})
	if ba.Used() != m.Stats().AllocatedBytes+out.Stats().OwnerBytes+out.Stats().ScratchBytes+out.Stats().OutputOwnerBytes+out.Stats().OutputOverflowBytes {
		t.Fatal("foreign-before-old-ACK cold graph", ba.Used(), m.Stats(), out.Stats())
	}
	if !foreign.Active() {
		t.Fatal("foreign unjoined ctor receipt lost")
	}
	paramsDecFinish(t, m.Close)
	if m.exprVisitHistory != nil || !history.acknowledged || history.node != tuple || history.owner != markOwner || history.invocation != invocation || history.bodyCause != bodyHistory || history.ownCause != ownHistory || m.exprVisit.unlistedNode != currentNode || m.exprVisit.bodyCause != currentBody || m.exprVisit.sourceCause != currentSource {
		t.Fatal("final Close replayed/lost retired tuple or source/own history")
	}
	if !reflect.DeepEqual(foreign, foreignBefore) || foreignToken() != foreignRaw {
		t.Fatal("old true ACK changed foreign invocation/history/raw tuple", foreign, foreignBefore, foreignToken(), foreignRaw)
	}
	upolyHenselAuditPhysical(t, []*mpnum.UPolynomialManager{anchor}, nil, upolyPhysicalExtraOwners{RationalServices: []*mpnum.RationalService{q}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &big}}})
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
