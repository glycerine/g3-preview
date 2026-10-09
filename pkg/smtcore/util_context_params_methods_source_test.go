// Source: src/cmd_context/context_params.cpp:42-135,158-196;
// src/cmd_context/context_params.h:26-80.
package smtcore

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/util"
)

func TestContextParamsSetASCIINormalizationLiteralBoolAndLP64UIntParsing(t *testing.T) {
	c, g, _, b := dynAckSourceFixture(t)
	s, e := NewContextParams(c, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, s.Close()) }()
	for _, tc := range []struct {
		key, value string
		want       uint32
	}{{"TIMEOUT", "", 0}, {"TIMEOUT", "00017", 17}, {"TIMEOUT", "4294967296", 0}, {"TIMEOUT", "4294967295", ^uint32(0)}, {"TIMEOUT", "9223372036854775807", ^uint32(0)}, {"TIMEOUT", "9223372036854775808", ^uint32(0)}, {"TIMEOUT", "18446744073709551615", ^uint32(0)}, {"rLiMiT", "42\x00junk", 42}} {
		paramsMust(t, s.Set(&tc.key, &tc.value))
		got, e := s.State()
		paramsMust(t, e)
		n := got.Timeout
		if tc.key == "rLiMiT" {
			n = got.RLimit
		}
		if n != tc.want {
			t.Fatal("literal unsigned parse/cast", tc, n)
		}
	}
	k, v := "Well-Sorted-Check", "true"
	paramsMust(t, s.Set(&k, &v))
	if !s.state.WellSortedCheck {
		t.Fatal("source ASCII normalization")
	}
	k, v = "MODEL", "false"
	paramsMust(t, s.Set(&k, &v))
	if s.state.Model {
		t.Fatal("literal bool")
	}
	for _, bad := range []string{"TRUE", " false", "false ", "1", ""} {
		old := s.state
		k = "MoDeL"
		e = s.Set(&k, &bad)
		var ex util.DefaultException
		want := "invalid value '" + bad + "' for Boolean parameter 'MoDeL'"
		if !errors.As(e, &ex) || ex.Message() != want || s.state != old {
			t.Fatal("original name/value exception, no field publication", bad, e)
		}
	}
	for _, bad := range []string{"+1", "-1", " 1", "1 ", "0x10", "1x2", "1_0"} {
		old := s.state
		k = "TIMEOUT"
		e = s.Set(&k, &bad)
		var ex util.DefaultException
		if !errors.As(e, &ex) || ex.Message() != "invalid value '"+bad+"' for unsigned int parameter 'TIMEOUT'" || s.state != old {
			t.Fatal("scan all digits then reject", bad, e)
		}
	}
	contextParamsCold(t, s, g, b)
}
func TestContextParamsUnknownSetNormalizedKeyLeavesUnusedValueUnreadAndListsExactDescriptors(t *testing.T) {
	c, g, _, b := dynAckSourceFixture(t)
	s, e := NewContextParams(c, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, s.Close()) }()
	key := "UNKNOWN-OPTION"
	e = s.Set(&key, nil)
	var ex util.DefaultException
	if !errors.As(e, &ex) || !strings.HasPrefix(ex.Message(), "unknown parameter 'unknown_option'\nLegal parameters are:\n") {
		t.Fatal("unknown name branch does not dereference unused value", e)
	}
	if strings.Count(ex.Message(), " (default:") != 15 || !strings.Contains(ex.Message(), "  timeout (unsigned int) (default: 4294967295)\n") || strings.Contains(ex.Message(), "interpolants") {
		t.Fatal("actual fifteen collected descriptors", ex.Message())
	}
	contextParamsCold(t, s, g, b)
}
func TestContextParamsMergeDefaultAutoConfigUsesContainsAnyKindAndExactOwnedCopy(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	s, e := NewContextParams(c, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, s.Close()) }()
	key, val := "AUTO-CONFIG", "false"
	paramsMust(t, s.Set(&key, &val))
	key = "untouched"
	paramsMust(t, c.SetUIntName(p, &key, 19))
	result, e := s.MergeDefaultParams(p)
	paramsMust(t, e)
	present, e := c.ContainsName(p, dynAckName("auto_config"))
	paramsMust(t, e)
	if present {
		t.Fatal("input remains unchanged")
	}
	got, e := c.GetBoolName(&result, dynAckName("auto_config"), true)
	paramsMust(t, e)
	if got {
		t.Fatal("copied output gets false")
	}
	n, e := c.GetUIntName(&result, &key, 0)
	paramsMust(t, e)
	if n != 19 {
		t.Fatal("literal COW copy")
	}
	paramsMust(t, c.Del(&result))
	paramsMust(t, c.SetUIntName(p, dynAckName("auto_config"), 13))
	result, e = s.MergeDefaultParams(p)
	paramsMust(t, e)
	if result.Slot != p.Slot || result.Generation != p.Generation {
		t.Fatal("contains ignores kind; returned copy aliases")
	}
	paramsMust(t, c.Del(&result))
	contextParamsCold(t, s, g, b)
}
func TestContextParamsGetSolverProofShortCircuitAndPublicationBeforeMergedParams(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	s, e := NewContextParams(c, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, s.Close()) }()
	m := astTestManager(t)
	paramsMust(t, m.ToggleProofMode(ast.ASTProofDisabled))
	proofs, models, core := true, true, true
	var invalid util.ParamsRef
	e = s.GetSolverParams(m, &invalid, &proofs, &models, &core)
	if !errors.Is(e, util.ErrParamsOwner) || proofs || !models || !core {
		t.Fatal("disabled proof skips p getter, later model reaches invalid p", e, proofs, models, core)
	}
	paramsMust(t, c.SetBoolName(p, dynAckName("proof"), true))
	paramsMust(t, c.SetBoolName(p, dynAckName("model"), false))
	paramsMust(t, c.SetBoolName(p, dynAckName("unsat_core"), true))
	paramsMust(t, s.GetSolverParams(m, p, &proofs, &models, &core))
	if proofs || models || !core {
		t.Fatal("real AST proof mode and local getters")
	}
	paramsMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	paramsMust(t, s.GetSolverParams(m, p, &proofs, &models, &core))
	if !proofs || models || !core {
		t.Fatal("actual enabled proofs")
	}
	contextParamsCold(t, s, g, b)
}

func contextMethodsCold(t *testing.T, s *ContextParams, g *util.GlobalParamsService, b *arena.Budget) {
	t.Helper()
	n := g.OwnerBytes + catalogGraph(g.Config.Catalog)
	if g.Current != nil {
		n += gparamsGraph(g.Current)
	}
	for x := g.Failed; x != nil; x = x.RetainedNext {
		n += gparamsGraph(x)
	}
	for x := g.Superseded; x != nil; x = x.RetainedNext {
		n += gparamsGraph(x)
	}
	if s != nil {
		n += s.header + uint64(s.dotProof.StorageBytes()+s.traceFile.StorageBytes())
		for f := s.frames; f != nil; f = f.next {
			if f.self != f || f.owner != s || f.invocation == 0 {
				t.Fatal("actual lexical owner binding")
			}
			n += f.header + f.key.StorageBytes() + uint64(len(f.buffer.State.Heap))
			d := &f.descriptor
			if d.Self != nil {
				if d.Self != d || d.OwnerBytes != 0 {
					t.Fatal("embedded actual descriptor, no second wrapper charge")
				}
				if d.Texts != nil {
					n += gparamsVectorGraph(d.Texts.Stats())
				}
			}
		}
	}
	if n != b.Used() {
		t.Fatal("immediate once-only cold graph", n, b.Used())
	}
}
func TestContextParamsEveryUnknownDescriptorAndErrorBufferReservation(t *testing.T) {
	success := -1
	for ordinal := int64(0); ordinal < 160; ordinal++ {
		c, g, _, b := dynAckSourceFixture(t)
		s, e := NewContextParams(c, g, b)
		paramsMust(t, e)
		key := "AN-ACTUAL-UNKNOWN-PARAMETER-WITH-EXTERNAL-STRING-STORAGE"
		old := s.state
		b.FailAfter(ordinal)
		e = s.Set(&key, nil)
		contextMethodsCold(t, s, g, b)
		var ex util.DefaultException
		if errors.As(e, &ex) {
			success = int(ordinal)
			if !strings.Contains(ex.Message(), "unknown parameter 'an_actual_unknown_parameter_with_external_string_storage'") {
				t.Fatal("normalized owned key", e)
			}
		} else if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal("real known leaf failure", ordinal, e)
		}
		if s.state != old || s.executing {
			t.Fatal("no invented field publication")
		}
		for f := s.frames; f != nil; f = f.next {
			if !f.descriptorAttempted || f.descriptor.ConstructorReady() || f.descriptor.Closed() || f.keyComplete && !f.key.Closed {
				t.Fatal("incomplete descriptor context retained, older completed key ended")
			}
		}
		b.FailAfter(-1)
		paramsMust(t, s.Close())
		contextMethodsCold(t, nil, g, b)
		if success >= 0 {
			break
		}
	}
	if success < 0 {
		t.Fatal("actual entire unknown branch horizon did not close")
	}
	t.Log("first complete unknown diagnostic reservation ordinal", success)
}
func TestContextParamsActualLongStringSetEveryAllocationAndErrorScanRefusal(t *testing.T) {
	for ordinal := int64(0); ordinal <= 1; ordinal++ {
		c, g, _, b := dynAckSourceFixture(t)
		s, e := NewContextParams(c, g, b)
		paramsMust(t, e)
		old, e := s.TraceFile().String()
		paramsMust(t, e)
		key, value := "TRACE-FILE-NAME", strings.Repeat("f", 81)
		b.FailAfter(ordinal)
		e = s.Set(&key, &value)
		contextMethodsCold(t, s, g, b)
		if ordinal == 0 && !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal("real lexical frame before members", e)
		}
		if ordinal == 1 && e != nil {
			t.Fatal("Go string replacement after frame", e)
		}
		got, x := s.TraceFile().String()
		paramsMust(t, x)
		if (ordinal == 0 && got != old) || (ordinal == 1 && got != value) {
			t.Fatal("header refusal versus successful Go text assignment", got)
		}
		b.FailAfter(-1)
		paramsMust(t, s.Set(&key, &value))
		got, x = s.TraceFile().String()
		paramsMust(t, x)
		if got != value {
			t.Fatal(got)
		}
		contextMethodsCold(t, s, g, b)
		paramsMust(t, s.Close())
	}
}
func TestContextParamsMergeEveryContextCOWReservationPreservesInputAndExactReturnOwnership(t *testing.T) {
	success := -1
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		_, _, q := paramsTestOwner(t) // unrelated borrowed numeral service survives.
		bc, bs, b := arena.NewBudget(1<<24), arena.NewBudget(1<<24), arena.NewBudget(1<<24)
		sy, e := util.NewSymbolManager(util.SymbolConfig{Budget: bs, PageSize: 64, DescriptorPageSize: 2})
		paramsMust(t, e)
		c, e := util.NewParamsContext(sy, q, util.ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
		paramsMust(t, e)
		catalog, e := util.NewGlobalParamsCatalog(sy, catalogTestConfig(b))
		paramsMust(t, e)
		cfg := gparamsTestConfig(b)
		cfg.Catalog = catalog
		g, e := util.NewGlobalParamsService(sy, c, cfg)
		paramsMust(t, e)
		paramsMust(t, g.InitializeSource())
		pv, e := c.NewValue()
		paramsMust(t, e)
		p := &pv
		s, e := NewContextParams(c, g, b)
		paramsMust(t, e)
		key, val := "auto_config", "false"
		paramsMust(t, s.Set(&key, &val))
		paramsMust(t, c.SetUIntName(p, dynAckName("unchanged"), 97))
		// Class-frame allocation is a separate named owner; this campaign refuses
		// every actual COW block/parallel metadata/payload acquisition on c's budget.
		bc.FailAfter(ordinal)
		out, e := s.MergeDefaultParams(p)
		contextMethodsCold(t, s, g, b)
		cs := c.Stats()
		cold := cs.HeaderBytes + cs.BlockBytes + cs.ReceiptBytes + cs.SourceFrameBytes + cs.SourceSetBytes + cs.ReplacementBytes + cs.PageBytes + cs.DirectoryBytes + cs.FormatBytes + gparamsVectorGraph(cs.Text)
		if cold != bc.Used() || astTestSymbolGraph(sy) != bs.Used() {
			t.Fatal("immediate named COW and separate borrowed-symbol cold owners", cold, bc.Used())
		}
		present, x := c.ContainsName(p, dynAckName("auto_config"))
		paramsMust(t, x)
		n, x := c.GetUIntName(p, dynAckName("unchanged"), 0)
		paramsMust(t, x)
		if present || n != 97 {
			t.Fatal("source input is caller-owned and unchanged")
		}
		if e == nil {
			success = int(ordinal)
			got, x := c.GetBoolName(&out, dynAckName("auto_config"), true)
			paramsMust(t, x)
			if got {
				t.Fatal("actual copied field")
			}
		} else if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal("COW source reservation", ordinal, e)
		}
		bc.FailAfter(-1)
		if e == nil {
			paramsMust(t, c.Del(&out))
		}
		paramsMust(t, s.Close())
		contextMethodsCold(t, nil, g, b)
		paramsMust(t, c.Del(p))
		paramsMust(t, g.Close())
		paramsMust(t, catalog.Close())
		paramsMust(t, c.Close())
		paramsMust(t, sy.Close())
		if bc.Used() != 0 || bs.Used() != 0 || b.Used() != 0 {
			t.Fatal("exact prepared owners refunded")
		}
		if success >= 0 {
			break
		}
	}
	if success < 0 {
		t.Fatal("actual COW horizon did not close")
	}
	t.Log("first successful COW reservation ordinal", success)
}

func TestContextParamsGetSolverCopyOOMRetainsExactUntransferredNumeral(t *testing.T) {
	bn, bc, bs, b := arena.NewBudget(1<<26), arena.NewBudget(1<<24), arena.NewBudget(1<<24), arena.NewBudget(1<<24)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: bn, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: r, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := util.NewSymbolManager(util.SymbolConfig{Budget: bs, PageSize: 64, DescriptorPageSize: 2})
	paramsMust(t, e)
	c, e := util.NewParamsContext(sy, q, util.ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
	paramsMust(t, e)
	m, e := polynomial.NewUPolynomialManagerFromRationalService(q, polynomial.UPolynomialConfig{Checkpoint: func() bool { panic("unreached polynomial checkpoint") }, Cooperation: func() error { panic("unreached polynomial cooperation") }})
	paramsMust(t, e)
	catalog, e := util.NewGlobalParamsCatalog(sy, catalogTestConfig(b))
	paramsMust(t, e)
	cfg := gparamsTestConfig(b)
	cfg.Catalog = catalog
	g, e := util.NewGlobalParamsService(sy, c, cfg)
	paramsMust(t, e)
	paramsMust(t, g.InitializeSource())
	s, e := NewContextParams(c, g, b)
	paramsMust(t, e)
	p, e := c.NewValue()
	paramsMust(t, e)
	t.Cleanup(func() {
		bn.FailAfter(-1)
		bc.FailAfter(-1)
		b.FailAfter(-1)
		paramsDecFinish(t, s.Close)
		paramsDecFinish(t, g.Close)
		paramsDecFinish(t, catalog.Close)
		paramsDecFinish(t, c.Close)
		paramsDecFinish(t, m.Close)
		paramsDecFinish(t, q.Close)
		paramsDecFinish(t, r.Close)
		paramsMust(t, sy.Close())
		if bn.Used() != 0 || bc.Used() != 0 || bs.Used() != 0 || b.Used() != 0 {
			t.Fatal("actual caller/provider refunds")
		}
	})
	scoped, e := util.NewScopedInt(m.IntegerManager())
	paramsMust(t, e)
	num, e := scoped.Get()
	paramsMust(t, e)
	limbs := make([]uint32, 32768)
	limbs[0], limbs[len(limbs)-1] = 17, 1
	paramsMust(t, m.IntegerManager().SetLimbs(num, limbs))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	key, e := sy.Intern("source-numeral")
	paramsMust(t, e)
	paramsMust(t, c.SetRat(&p, key, &big))
	paramsMust(t, q.Del(&big))
	paramsMust(t, scoped.Close())
	auto, val := "auto_config", "false"
	paramsMust(t, s.Set(&auto, &val))
	ast := astTestManager(t)
	proof, model, core := true, false, true
	paramsMust(t, r.SetMaxSize(1))
	e = s.GetSolverParams(ast, &p, &proof, &model, &core)
	var failure util.Failure
	if !errors.As(e, &failure) || failure.ErrorCode() != 101 || proof || !model || core || s.frames == nil {
		t.Fatalf("actual postphysical COW copy refusal: %T %v; failure=%v, frames=%+v, flags=%v/%v/%v", e, e, failure, s.frames, proof, model, core)
	}
	f := s.frames
	if f.sourceReceipt.Invocation() == 0 || f.valueOwned || f.destructorCause != nil {
		t.Fatal("failed init reference held separately from completed temporary destructor")
	}
	slot := f.sourceReceipt.Slot
	leaf := &c.SetFrames[slot-1]
	if !leaf.Active || !leaf.Acquired || leaf.Cause == nil {
		t.Fatal("actual copied POD numeral retained before append")
	}
	extras := upolyPhysicalExtraOwners{RationalServices: []*util.RationalService{q}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &leaf.Next.Rat}}, ParamsValues: []paramsPhysicalValueOwner{{c, &p}}}
	upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{m}, nil, extras)
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	contextMethodsCold(t, s, g, b)
	if f.setReceipt.Invocation() != leaf.Invocation || f.setReceipt.Slot != slot || f.setReceipt.Generation != leaf.Generation {
		t.Fatal("exact acquired numeral invocation")
	}
	bn.FailAfter(0)
	bc.FailAfter(0)
	b.FailAfter(0)
	actual := f.setReceipt
	originalSource := f.sourceReceipt
	originalBody := f.bodyCause
	originalDT := f.destructorCause
	wrong := actual
	wrong.InvocationField++
	if e = c.CleanupSet(wrong); !errors.Is(e, util.ErrParamsPrecondition) || !leaf.Active || !leaf.Acquired {
		t.Fatal("foreign primitive invocation cannot release copied numeral", e)
	}
	upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{m}, nil, extras)
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	contextMethodsCold(t, s, g, b)
	paramsMust(t, r.SetMaxSize(0))
	paramsDecFinish(t, s.Close)
	if f.setReceipt != actual || f.sourceReceipt != originalSource || !f.setAcknowledged || !f.sourceAcknowledged || f.bodyCause == nil || f.ownCause != nil || f.bodyCause != originalBody || f.destructorCause != originalDT {
		t.Fatal("own ACK erased actual acquired tuple or immutable cause history")
	}
	if leaf.Active || leaf.Acquired {
		t.Fatal("parent owning close omitted exact failed copied numeral")
	}
	upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{RationalServices: []*util.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &p}}})
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	bn.FailAfter(-1)
	bc.FailAfter(-1)
	b.FailAfter(-1)
	// A separate caller-owned failed COW action must not be adopted by a later
	// context call merely because its returned alias names the same live block.
	foreign, x := c.NewValue()
	paramsMust(t, x)
	paramsMust(t, c.Assign(&foreign, &p))
	s2, x := NewContextParams(c, g, b)
	paramsMust(t, x)
	paramsMust(t, s2.Set(&auto, &val))
	paramsMust(t, r.SetMaxSize(1))
	if x = c.SetBoolName(&foreign, &auto, false); x == nil {
		t.Fatal("actual foreign copy stop")
	}
	foreignSet, x := c.StoppedSet(&foreign)
	paramsMust(t, x)
	foreignSource, x := c.StoppedSource(&foreign)
	paramsMust(t, x)
	if _, x = s2.MergeDefaultParams(&foreign); !errors.Is(x, util.ErrParamsPrecondition) {
		t.Fatal("stopped caller mutation guard", x)
	}
	paramsMust(t, r.SetMaxSize(0))
	paramsDecFinish(t, s2.Close)
	if !c.SetFrames[foreign.Slot-1].Active || !c.SourceFrames[foreign.Slot-1].Active {
		t.Fatal("parent adopted unrelated failed caller invocation")
	}
	paramsMust(t, c.CleanupSet(foreignSet))
	paramsMust(t, c.CleanupSource(foreignSource))
	paramsMust(t, c.Del(&foreign))
	// Exercise the actual lexical epilogue engine with a genuine final-reference
	// DT stop. This is the Runtime signed-counter guarded profile, not a claim
	// that source native signed overflow is a valid destruction domain.
	s3, x := NewContextParams(c, g, b)
	paramsMust(t, x)
	older, x := s3.beginSourceFrame()
	paramsMust(t, x)
	paramsMust(t, older.key.Construct(b, util.LiteralParamsStringView(strings.Repeat("k", 81))))
	older.keyComplete = true
	younger, x := s3.beginSourceFrame()
	paramsMust(t, x)
	paramsMust(t, s3.mergeSourceFrame(younger, &p))
	paramsMust(t, c.Del(&p)) // exact temporary now holds the actual final ref.
	paramsDecTLSNearMinimum(t, r)
	body := errors.New("actual outer body diagnostic")
	x = s3.endSourceFrame(younger, body)
	if !errors.Is(x, body) || !errors.Is(x, arena.ErrBounds) || younger.decrement.InvocationField == 0 || !younger.valueOwned {
		t.Fatal("actual younger DT stop retains body and exact leaf", x)
	}
	otherBody := errors.New("older body diagnostic")
	x = s3.endSourceFrame(older, otherBody)
	if !errors.Is(x, otherBody) || !errors.Is(x, arena.ErrBounds) || older.key.Closed || older.key.Length != 81 {
		t.Fatal("younger stop protects older lexical storage", x)
	}
	upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{RationalServices: []*util.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &younger.value}}})
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	contextMethodsCold(t, s3, g, b)
	bn.FailAfter(0)
	bc.FailAfter(0)
	b.FailAfter(0)
	paramsDecFinish(t, s3.Close)
	if younger.bodyCause != body || !errors.Is(younger.destructorCause, arena.ErrBounds) || older.bodyCause != otherBody || !errors.Is(s3.destructorCause, arena.ErrBounds) {
		t.Fatal("own ACK preserves each immutable body/source cause")
	}
	upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{RationalServices: []*util.RationalService{q}})
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	contextMethodsCold(t, nil, g, b)
	bn.FailAfter(-1)
	bc.FailAfter(-1)
	b.FailAfter(-1)
}

func TestContextParamsAllSetBooleanFieldsAndReachedBorrowedStringLifetime(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	s, e := NewContextParams(c, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, s.Close()) }()
	expected := s.state
	for _, key := range []string{"type_check", "auto_config", "proof", "model", "model_validate", "dump_models", "trace", "unsat_core", "debug_ref_count", "smtlib2_compliant"} {
		for _, v := range []bool{true, false} {
			text := "false"
			if v {
				text = "true"
			}
			paramsMust(t, s.Set(&key, &text))
			switch key {
			case "type_check":
				expected.WellSortedCheck = v
			case "auto_config":
				expected.AutoConfig = v
			case "proof":
				expected.Proof = v
			case "model":
				expected.Model = v
			case "model_validate":
				expected.ModelValidate = v
			case "dump_models":
				expected.DumpModels = v
			case "trace":
				expected.Trace = v
			case "unsat_core":
				expected.UnsatCore = v
			case "debug_ref_count":
				expected.DebugRefCount = v
			case "smtlib2_compliant":
				expected.SMTLib2Compliant = v
			}
			if s.state != expected {
				t.Fatal("literal independent field publication", key, v, s.state, expected)
			}
		}
	}
	key, value := "borrow", "true"
	paramsMust(t, c.SetStringName(p, &key, &value))
	view, e := c.GetStringViewFallbackName(p, &key, nil, nil)
	paramsMust(t, e)
	paramsMust(t, c.Reset(p))
	old := s.state
	e = s.SetViews(util.LiteralParamsStringView("model"), view)
	if e == nil || s.state != old {
		t.Fatal("reached stale value must fail before bool publication", e)
	}
	e = s.SetViews(util.LiteralParamsStringView("unknown"), view)
	var ex util.DefaultException
	if !errors.As(e, &ex) {
		t.Fatal("unknown branch must leave stale unused value unread", e)
	}
	e = s.SetViews(view, util.ParamsStringView{})
	if e == nil || s.state != old {
		t.Fatal("key constructor reaches stale borrow before value", e)
	}
	contextMethodsCold(t, s, g, b)
}

func TestContextParamsEveryLiteralBoolAndUIntDiagnosticOutputReservation(t *testing.T) {
	for _, key := range []string{"MoDeL", "TIMEOUT"} {
		success := -1
		for ordinal := int64(0); ordinal < 32; ordinal++ {
			c, g, _, b := dynAckSourceFixture(t)
			s, e := NewContextParams(c, g, b)
			paramsMust(t, e)
			old := s.state
			value := strings.Repeat("x", 143)
			b.FailAfter(ordinal)
			e = s.Set(&key, &value)
			contextMethodsCold(t, s, g, b)
			if s.state != old {
				t.Fatal("diagnostic never publishes scalar")
			}
			var ex util.DefaultException
			if errors.As(e, &ex) {
				kind := "Boolean"
				if key == "TIMEOUT" {
					kind = "unsigned int"
				}
				if ex.Message() != "invalid value '"+value+"' for "+kind+" parameter '"+key+"'" {
					t.Fatal("literal separate insertions", ex.Message())
				}
				success = int(ordinal)
			} else if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal("real buffer reservation", ordinal, e)
			}
			b.FailAfter(-1)
			paramsMust(t, s.Close())
			contextMethodsCold(t, nil, g, b)
			if success >= 0 {
				break
			}
		}
		if success < 0 {
			t.Fatal("diagnostic reservation horizon", key)
		}
		t.Log(key, "diagnostic first successful ordinal", success)
	}
}

func TestContextParamsActualCOWAppendRefusalCleanupStopRetainsImmutableChildTuples(t *testing.T) {
	bn, bc, bs, b := arena.NewBudget(1<<26), arena.NewBudget(1<<24), arena.NewBudget(1<<24), arena.NewBudget(1<<24)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: bn, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: r, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: bn}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bn}}}})
	paramsMust(t, e)
	sy, e := util.NewSymbolManager(util.SymbolConfig{Budget: bs, PageSize: 64, DescriptorPageSize: 2})
	paramsMust(t, e)
	c, e := util.NewParamsContext(sy, q, util.ParamsConfig{Budget: bc, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: bc}}})
	paramsMust(t, e)
	m, e := polynomial.NewUPolynomialManagerFromRationalService(q, polynomial.UPolynomialConfig{Checkpoint: func() bool { panic("unreached polynomial checkpoint") }, Cooperation: func() error { panic("unreached polynomial cooperation") }})
	paramsMust(t, e)
	catalog, e := util.NewGlobalParamsCatalog(sy, catalogTestConfig(b))
	paramsMust(t, e)
	cfg := gparamsTestConfig(b)
	cfg.Catalog = catalog
	g, e := util.NewGlobalParamsService(sy, c, cfg)
	paramsMust(t, e)
	paramsMust(t, g.InitializeSource())
	s, e := NewContextParams(c, g, b)
	paramsMust(t, e)
	p, e := c.NewValue()
	paramsMust(t, e)
	t.Cleanup(func() {
		bn.FailAfter(-1)
		bc.FailAfter(-1)
		b.FailAfter(-1)
		paramsDecFinish(t, s.Close)
		paramsDecFinish(t, g.Close)
		paramsDecFinish(t, catalog.Close)
		paramsDecFinish(t, c.Close)
		paramsDecFinish(t, m.Close)
		paramsDecFinish(t, q.Close)
		paramsDecFinish(t, r.Close)
		paramsMust(t, sy.Close())
		if bn.Used() != 0 || bc.Used() != 0 || bs.Used() != 0 || b.Used() != 0 {
			t.Fatal("actual caller/provider refunds")
		}
	})
	scoped, e := util.NewScopedInt(m.IntegerManager())
	paramsMust(t, e)
	num, e := scoped.Get()
	paramsMust(t, e)
	limbs := make([]uint32, 32768)
	limbs[0], limbs[len(limbs)-1] = 17, 1
	paramsMust(t, m.IntegerManager().SetLimbs(num, limbs))
	big, e := q.NewFromScopedInt(scoped)
	paramsMust(t, e)
	key, e := sy.Intern("source-numeral")
	paramsMust(t, e)
	paramsMust(t, c.SetUIntName(&p, dynAckName("prefix-zero"), 10))
	paramsMust(t, c.SetUIntName(&p, dynAckName("prefix-one"), 11))
	paramsMust(t, c.SetRat(&p, key, &big))
	paramsMust(t, q.Del(&big))
	paramsMust(t, scoped.Close())
	auto, val := "auto_config", "false"
	paramsMust(t, s.Set(&auto, &val))
	// This call completes the copied NUM, then refuses its actual append growth.
	// Its source POD has no automatic NUM DT; the exact acquisition remains live.
	s4 := s
	var x error
	beforeCopy := upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{RationalServices: []*util.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &p}}})
	bc.FailAfter(1)
	_, x = s4.MergeDefaultParams(&p)
	if !errors.Is(x, arena.ErrInjectedFailure) || s4.frames == nil {
		t.Fatal("actual NUM-before-append refusal", x)
	}
	h := s4.frames
	if h.setReceipt.InvocationField == 0 || h.sourceReceipt.InvocationField == 0 || h.valueOwned {
		t.Fatal("exact failed-copy tuples before lexical DT completion")
	}
	hLeaf := &c.SetFrames[h.setReceipt.Slot-1]
	hExtras := upolyPhysicalExtraOwners{RationalServices: []*util.RationalService{q}, ParamsRationalValues: []paramsPhysicalRationalOwner{{q, &hLeaf.Next.Rat}}, ParamsValues: []paramsPhysicalValueOwner{{c, &p}}}
	backedCopy := upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{m}, nil, hExtras)
	if backedCopy.Objects != beforeCopy.Objects+1 {
		t.Fatal("actual new copied NUM backing before TLS stop", beforeCopy, backedCopy)
	}
	oldValue := &c.Entry(&c.Blocks[p.Slot-1], 2).Rat
	equal, e := q.Equal(&hLeaf.Next.Rat, oldValue)
	paramsMust(t, e)
	if !equal {
		t.Fatal("fully backed acquired NUM matches original actual payload before stop")
	}
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	contextMethodsCold(t, s4, g, b)
	bc.FailAfter(-1) // after the full stopped-body graph, prepare the real TLS guard.
	paramsDecTLSNearMinimum(t, r)
	bn.FailAfter(0)
	bc.FailAfter(0)
	b.FailAfter(0)
	setTuple, sourceTuple, bodyTuple, dtTuple := h.setReceipt, h.sourceReceipt, h.bodyCause, h.destructorCause
	x = s4.Close()
	if !errors.Is(x, arena.ErrBounds) || h.setAcknowledged || h.sourceAcknowledged || h.setReceipt != setTuple || h.sourceReceipt != sourceTuple {
		t.Fatal("actual CleanupSet stop with unchanged parent acquisition tuples", x)
	}
	if !errors.Is(h.ownCause, arena.ErrBounds) || !errors.Is(bodyTuple, arena.ErrInjectedFailure) || dtTuple != nil || !hLeaf.Active || !hLeaf.Acquired || !errors.Is(hLeaf.CleanupCause, arena.ErrBounds) {
		t.Fatal("actual immutable body/own causes and consumed NUM leaf before retry", h.ownCause, bodyTuple, dtTuple)
	}
	ownTuple := h.ownCause
	upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{m}, nil, hExtras)
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	contextMethodsCold(t, s4, g, b)
	paramsDecFinish(t, s4.Close)
	if !h.setAcknowledged || !h.sourceAcknowledged || h.setReceipt != setTuple || h.sourceReceipt != sourceTuple || h.bodyCause != bodyTuple || h.destructorCause != dtTuple || h.ownCause != ownTuple {
		t.Fatal("matched owning retry preserves exact original tuples and body/source/own history")
	}
	paramsMust(t, s4.Close())
	upolyHenselAuditPhysical(t, []*polynomial.UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{RationalServices: []*util.RationalService{q}, ParamsValues: []paramsPhysicalValueOwner{{c, &p}}})
	paramsDecAuditColdGraph(t, q, m, r, bn, c, bc, sy, bs)
	contextMethodsCold(t, nil, g, b)
	bn.FailAfter(-1)
	bc.FailAfter(-1)
	b.FailAfter(-1)
	if e = c.Del(&p); e != nil {
		if !errors.Is(e, arena.ErrBounds) {
			t.Fatal("actual caller destruction guard", e)
		}
		paramsDecFinish(t, func() error { return c.CleanupDecrement(&p) })
	}
}
