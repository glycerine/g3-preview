// Source: src/smt/params/theory_array_params.h:31-59,61-75;
// src/smt/params/theory_array_params.cpp:22-40; src/smt/params/smt_params_helper.pyg:59-60.
package smtcore

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestSMTArrayConstructorElevenDefaultsDoesNotUpdate(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	module, e := g.Symbols.Intern("smt")
	paramsMust(t, e)
	global, e := g.Current.GetParams(module)
	paramsMust(t, e)
	paramsMust(t, c.SetBoolName(global, dynAckName("array.weak"), true))
	paramsMust(t, c.SetBoolName(global, dynAckName("array.extensional"), false))
	a, e := NewSMTArrayParams(b)
	paramsMust(t, e)
	defer func() { paramsMust(t, a.Close()) }()
	want := SMTArrayState{CanonizeSimplify: false, Simplify: true, Mode: 3, Weak: false, Extensional: true, Laziness: 1, DelayExpAxiom: true, CG: false, AlwaysPropUpward: true, LazyIEq: false, LazyIEqDelay: 10}
	state, e := a.State()
	paramsMust(t, e)
	if state != want {
		t.Fatal("ctor must not query/update module", state, want)
	}
	paramsMust(t, a.Update(c, p, g))
	state, e = a.State()
	paramsMust(t, e)
	want.Weak = true
	want.Extensional = false
	if state != want {
		t.Fatal("only actual two source assignments", state, want)
	}
	paramsMust(t, c.SetBoolName(p, dynAckName("array.extensional"), true))
	paramsMust(t, a.Update(c, p, g))
	state, e = a.State()
	paramsMust(t, e)
	if !state.Extensional || !state.Weak {
		t.Fatal("local-module-literal precedence", state)
	}
}
func TestSMTArrayDisplayExactlyNineFieldsAndSignedBooleanShowPos(t *testing.T) {
	_, _, _, b := dynAckSourceFixture(t)
	a, e := NewSMTArrayParams(b)
	paramsMust(t, e)
	defer func() { paramsMust(t, a.Close()) }()
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	defer func() { paramsMust(t, out.Close()) }()
	paramsMust(t, out.SetShowPos(true))
	paramsMust(t, a.Display(out))
	text, e := out.String()
	paramsMust(t, e)
	want := "m_array_mode=+3\nm_array_weak=+0\nm_array_extensional=+1\nm_array_laziness=1\nm_array_delay_exp_axiom=+1\nm_array_cg=+0\nm_array_always_prop_upward=+1\nm_array_lazy_ieq=+0\nm_array_lazy_ieq_delay=10\n"
	if text != want || strings.Contains(text, "canonize") || strings.Contains(text, "simplify") {
		t.Fatal("literal nine insertion chains/omitted fields", text, want)
	}
}

func arrayNamedColdGraph(t *testing.T, a *SMTArrayParams, g *util.GlobalParamsService, b *arena.Budget, extra ...*SMTArrayParams) {
	t.Helper()
	total := g.OwnerBytes + catalogGraph(g.Config.Catalog)
	if g.Current != nil {
		total += gparamsGraph(g.Current)
	}
	for v := g.Superseded; v != nil; v = v.RetainedNext {
		total += gparamsGraph(v)
	}
	for v := g.Failed; v != nil; v = v.RetainedNext {
		total += gparamsGraph(v)
	}
	owners := append([]*SMTArrayParams{a}, extra...)
	for _, a := range owners {
		if a == nil {
			continue
		}
		if a.self != a || a.ModuleParamsOwner.Self != &a.ModuleParamsOwner || a.Budget != b {
			t.Fatal("actual array/header identity")
		}
		total += a.header
		for f := a.Helpers; f != nil; f = f.Next {
			if f.Self != f || f.Owner != &a.ModuleParamsOwner || f.Invocation == 0 || f.Invocation > a.Serial {
				t.Fatal("exact helper owner/frame/invocation")
			}
			total += f.Header
		}
	}
	if total != b.Used() {
		t.Fatal("complete named cold graph", total, b.Used())
	}
}
func TestSMTArrayConstructorAndUpdateEveryRealReservationPrefix(t *testing.T) {
	firstCtor := -1
	for ordinal := int64(0); ordinal < 8; ordinal++ {
		_, g, _, b := dynAckSourceFixture(t)
		b.FailAfter(ordinal)
		a, e := NewSMTArrayParams(b)
		arrayNamedColdGraph(t, a, g, b)
		if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal("actual cold constructor leaf", e)
		}
		if a != nil && (a.Context != nil || a.Globals != nil || a.Helpers != nil || a.Serial != 0 || a.header != uint64(reflect.TypeOf(SMTArrayParams{}).Size())) {
			t.Fatal("constructor looked up module or duplicated inline header")
		}
		b.FailAfter(-1)
		if a != nil {
			paramsMust(t, a.Close())
			paramsMust(t, a.Close())
		}
		arrayNamedColdGraph(t, nil, g, b)
		if e == nil {
			firstCtor = int(ordinal)
			break
		}
	}
	if firstCtor != 1 {
		t.Fatal("one standalone actual Go header; no source scalar allocation", firstCtor)
	}
	firstUpdate := -1
	for ordinal := int64(0); ordinal < 8; ordinal++ {
		c, g, p, b := dynAckSourceFixture(t)
		a, e := NewSMTArrayParams(b)
		paramsMust(t, e)
		b.FailAfter(ordinal)
		cause := a.Update(c, p, g)
		arrayNamedColdGraph(t, a, g, b)
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal("actual lexical helper leaf", cause)
		}
		b.FailAfter(-1)
		paramsMust(t, a.Update(c, p, g))
		paramsMust(t, a.Close())
		arrayNamedColdGraph(t, nil, g, b)
		if cause == nil {
			firstUpdate = int(ordinal)
			break
		}
	}
	if firstUpdate != 1 {
		t.Fatal("actual warmed-budget helper acquisition", firstUpdate)
	}
}
func TestSMTArrayConstructorUninitializedGlobalsAndFunctionFailureRetry(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	paramsMust(t, g.FinalizeSource())
	a, e := NewSMTArrayParams(b)
	paramsMust(t, e)
	defer func() { paramsMust(t, a.Close()) }()
	before, e := a.State()
	paramsMust(t, e)
	cause := a.Update(c, p, g)
	want := util.ErrGlobalParamsUninitialized
	if util.GlobalParamsSourceDebug {
		want = util.ErrGlobalParamsAssertion
	}
	if !errors.Is(cause, want) || a.Helpers != nil {
		t.Fatal("actual helper constructor source assertion", cause)
	}
	after, e := a.State()
	paramsMust(t, e)
	if after != before {
		t.Fatal("helper ctor refusal mutated source fields", after, before)
	}
	arrayNamedColdGraph(t, a, g, b)
	paramsMust(t, g.InitializeSource())
	paramsMust(t, a.Update(c, p, g))
	// Non-updated public source fields may be changed directly by source consumers.
	a.state.CanonizeSimplify = true
	a.state.Simplify = false
	a.state.Laziness = 19
	a.state.LazyIEqDelay = 23
	before = a.state
	paramsMust(t, a.Update(c, p, g))
	if a.state != before {
		t.Fatal("updt_params modified a field absent from source body", before, a.state)
	}
	copy := *a
	if e = copy.Close(); !errors.Is(e, util.ErrParamsOwner) {
		t.Fatal("copied owning wrapper", e)
	}
	arrayNamedColdGraph(t, a, g, b)
}
func TestSMTModuleHelperBorrowedLocalExactFrameForeignAndStaleGuards(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	key, e := g.Symbols.Intern("smt")
	paramsMust(t, e)
	global, e := g.Current.GetParams(key)
	paramsMust(t, e)
	paramsMust(t, c.SetBoolName(global, dynAckName("array.weak"), true))
	paramsMust(t, c.SetBoolName(p, dynAckName("array.weak"), false))
	a, e := NewSMTArrayParams(b)
	paramsMust(t, e)
	other, e := NewSMTArrayParams(b)
	paramsMust(t, e)
	paramsMust(t, a.Bind(c, g))
	paramsMust(t, other.Bind(c, g))
	a.Running = true
	f, e := a.BeginHelper(p, "smt")
	paramsMust(t, e)
	other.Running = true
	foreign, e := other.BeginHelper(p, "smt")
	paramsMust(t, e)
	got, e := a.FrameBool(f, "array.weak", true)
	paramsMust(t, e)
	if got {
		t.Fatal("actual local overrides acquired module")
	}
	paramsMust(t, c.SetBoolName(p, dynAckName("array.weak"), true))
	got, e = a.FrameBool(f, "array.weak", false)
	paramsMust(t, e)
	if !got {
		t.Fatal("local was cloned instead of borrowed")
	}
	if _, e = other.FrameBool(f, "array.weak", false); !errors.Is(e, util.ErrParamsPrecondition) {
		t.Fatal("foreign frame admitted", e)
	}
	if e = a.Close(); !errors.Is(e, util.ErrParamsPrecondition) {
		t.Fatal("destructive callback admitted during helper execution", e)
	}
	arrayNamedColdGraph(t, a, g, b, other)
	paramsMust(t, a.FinishHelper(f, nil))
	a.Running = false
	if _, e = a.FrameBool(f, "array.weak", false); !errors.Is(e, util.ErrParamsPrecondition) {
		t.Fatal("ended lexical frame reused", e)
	}
	paramsMust(t, other.FinishHelper(foreign, nil))
	other.Running = false
	arrayNamedColdGraph(t, a, g, b, other)
	paramsMust(t, a.Close())
	paramsMust(t, other.Close())
	arrayNamedColdGraph(t, nil, g, b)
}

const arrayDisplayDefaults = "m_array_mode=3\nm_array_weak=0\nm_array_extensional=1\nm_array_laziness=1\nm_array_delay_exp_axiom=1\nm_array_cg=0\nm_array_always_prop_upward=1\nm_array_lazy_ieq=0\nm_array_lazy_ieq_delay=10\n"

func TestSMTArrayGoBufferOutputAndClosedFileBadbit(t *testing.T) {
	_, g, _, b := dynAckSourceFixture(t)
	a, e := NewSMTArrayParams(b)
	paramsMust(t, e)
	defer func() { paramsMust(t, a.Close()) }()
	first := -1
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		sb := arena.NewBudget(^uint64(0))
		out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: sb})
		paramsMust(t, e)
		sb.FailAfter(ordinal)
		cause := a.Display(out)
		text, e := out.String()
		paramsMust(t, e)
		if !strings.HasPrefix(arrayDisplayDefaults, text) {
			t.Fatal("literal separate insertion prefix", ordinal, text, cause)
		}
		if cause == nil && text != arrayDisplayDefaults {
			t.Fatal("complete nine fields", text)
		}
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal("actual output leaf", cause)
		}
		sb.FailAfter(-1)
		paramsMust(t, out.Close())
		if sb.Used() != 0 {
			t.Fatal("output cold/refund")
		}
		if cause == nil {
			first = int(ordinal)
			break
		}
	}
	if first < 0 {
		t.Fatal("Go buffer output did not complete", first)
	}
	file, e := os.CreateTemp(".", "smt-array-file-*")
	paramsMust(t, e)
	defer os.Remove(file.Name())
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b, BorrowedFile: file})
	paramsMust(t, e)
	paramsMust(t, file.Close())
	paramsMust(t, a.Display(out))
	if !out.FileBad || out.FilePosition != 0 {
		t.Fatal("real failed ostream stream", out.FileBad, out.FilePosition)
	}
	paramsMust(t, out.Close())
	arrayNamedColdGraph(t, a, g, b)
}
