package smtcore

// Source: src/cmd_context/context_params.h:26-55;
// src/cmd_context/context_params.cpp:26-40,137-156.

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func contextParamsCold(t *testing.T, v *ContextParams, g *uparams.GlobalParamsService, b *arena.Budget) {
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
	if v != nil {
		if v.self != v || v.budget != b {
			t.Fatal("exact owner")
		}
		n += v.header
	}
	if n != b.Used() {
		t.Fatal("before-lift named cold graph", n, b.Used())
	}
}
func TestContextParamsDefaultConstructorAndGlobalUpdate(t *testing.T) {
	c, g, _, b := dynAckSourceFixture(t)
	global, e := g.SourceGetRef()
	paramsMust(t, e)
	key := "auto_config"
	paramsMust(t, c.SetBoolName(global, &key, false))
	v, e := NewContextParams(c, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	s, e := v.State()
	paramsMust(t, e)
	want := ContextParamsState{AutoConfig: false, Model: true, Timeout: ^uint32(0), DefaultsInitialized: true}
	if s != want {
		t.Fatal("source defaults/global constructor update", s, want)
	}
	a := v.TraceFile()
	d := v.DotProofFile()
	if a != "z3.log" || d != "proof.dot" {
		t.Fatal(a, d)
	}
	contextParamsCold(t, v, g, b)
}
func TestContextParamsNestedBoolAndCurrentScalarFallbackButLiteralStrings(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewContextParams(c, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	key := "well_sorted_check"
	paramsMust(t, c.SetBoolName(p, &key, true))
	key = "type_check"
	paramsMust(t, c.SetBoolName(p, &key, false))
	key = "timeout"
	paramsMust(t, c.SetUIntName(p, &key, 17))
	key = "trace_file_name"
	val := "custom"
	paramsMust(t, c.SetStringName(p, &key, &val))
	paramsMust(t, v.Update(p))
	s, e := v.State()
	paramsMust(t, e)
	if s.WellSortedCheck || s.Timeout != 17 {
		t.Fatal(s)
	}
	empty, e := c.NewValue()
	paramsMust(t, e)
	defer func() { paramsMust(t, c.Del(&empty)) }()
	paramsMust(t, v.Update(&empty))
	s, e = v.State()
	paramsMust(t, e)
	a := v.TraceFile()
	if s.Timeout != 17 || a != "z3.log" {
		t.Fatal("scalar current fallback versus fixed string literal", s, a)
	}
	// The nested fallback is evaluated even when outer type_check has a value;
	// both are source reads, with no extra module reference or lexical frame.
	contextParamsCold(t, v, g, b)
}

func TestContextParamsGoStringSnapshotsAndCStringBoundary(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewContextParams(c, g, b)
	paramsMust(t, e)
	oldTrace, oldProof := v.TraceFile(), v.DotProofFile()
	traceKey, trace := "trace_file_name", "trace.log\x00ignored"
	proofKey, proof := "dot_proof_file", "proof.dot\x00ignored"
	paramsMust(t, c.SetStringName(p, &traceKey, &trace))
	paramsMust(t, c.SetStringName(p, &proofKey, &proof))
	paramsMust(t, v.Update(p))
	if v.TraceFile() != "trace.log" || v.DotProofFile() != "proof.dot" {
		t.Fatal("C string parameter boundary", v.TraceFile(), v.DotProofFile())
	}
	if oldTrace != "z3.log" || oldProof != "proof.dot" {
		t.Fatal("earlier Go string snapshots", oldTrace, oldProof)
	}
	name := astTraceFilename{owned: v.TraceFile(), ownedMode: true}
	length, e := name.Length()
	paramsMust(t, e)
	last, e := name.Byte(length)
	paramsMust(t, e)
	if length != uint64(len("trace.log")) || last != 0 {
		t.Fatal("trace filename terminator", length, last)
	}
	contextParamsCold(t, v, g, b)
	paramsMust(t, v.Close())
	if name.owned != "trace.log" {
		t.Fatal("borrowed string snapshot changed at close", name.owned)
	}
	contextParamsCold(t, nil, g, b)
}

func TestContextParamsCompilerCopyIsIndependentAndKeepsCurrentValues(t *testing.T) {
	c, g, _, b := dynAckSourceFixture(t)
	source, e := NewContextParams(c, g, b)
	paramsMust(t, e)
	key, value := "auto_config", "false"
	paramsMust(t, source.Set(&key, &value))
	key, value = "trace_file_name", "copied.log"
	paramsMust(t, source.Set(&key, &value))
	copy, e := source.Clone()
	paramsMust(t, e)
	defer func() { paramsMust(t, copy.Close()) }()
	key, value = "auto_config", "true"
	paramsMust(t, source.Set(&key, &value))
	key, value = "trace_file_name", "source.log"
	paramsMust(t, source.Set(&key, &value))
	paramsMust(t, source.Close())
	state, e := copy.State()
	paramsMust(t, e)
	if state.AutoConfig || copy.TraceFile() != "copied.log" {
		t.Fatal("compiler copy did not preserve the copied snapshot", state, copy.TraceFile())
	}
	key, value = "model", "false"
	paramsMust(t, copy.Set(&key, &value))
	state, e = copy.State()
	paramsMust(t, e)
	if state.Model {
		t.Fatal("copied context parameters are not independently mutable")
	}
}

func TestContextParamsEveryConstructorReservationAndGoStringUnwind(t *testing.T) {
	for ordinal := int64(0); ; ordinal++ {
		c, g, _, b := dynAckSourceFixture(t)
		p, e := g.SourceGetRef()
		paramsMust(t, e)
		key := "trace_file_name"
		val := strings.Repeat("t", 41)
		paramsMust(t, c.SetStringName(p, &key, &val))
		key = "dot_proof_file"
		val = strings.Repeat("d", 47)
		paramsMust(t, c.SetStringName(p, &key, &val))
		b.FailAfter(ordinal)
		v, e := NewContextParams(c, g, b)
		contextParamsCold(t, v, g, b)
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
			if v != nil && !v.state.DefaultsInitialized {
				t.Fatal("constructor state before return")
			}
			if v != nil {
				before := b.Used()
				if x := v.Update(p); !errors.Is(x, uparams.ErrParamsPrecondition) {
					t.Fatal("failed ctor ordinary Update", x)
				}
				if x := v.UpdateGlobal(); !errors.Is(x, uparams.ErrParamsPrecondition) {
					t.Fatal("failed ctor global read", x)
				}
				if b.Used() != before {
					t.Fatal("incomplete ctor allocation erased")
				}
			}
		}
		b.FailAfter(-1)
		if v != nil {
			paramsMust(t, v.Close())
		}
		contextParamsCold(t, nil, g, b)
		if e == nil {
			if ordinal != 1 {
				t.Fatal("actual header; text is Go-managed", ordinal)
			}
			break
		}
		if ordinal > 3 {
			t.Fatal("no success")
		}
	}
}
func TestContextParamsUpdateStringRefusalPrefixAndIncompleteOwnerGuards(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewContextParams(c, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	key := "timeout"
	paramsMust(t, c.SetUIntName(p, &key, 29))
	key = "unsat_core"
	paramsMust(t, c.SetBoolName(p, &key, true))
	key = "trace_file_name"
	value := strings.Repeat("q", 90)
	paramsMust(t, c.SetStringName(p, &key, &value))
	before := b.Used()
	b.FailAfter(0)
	paramsMust(t, v.Update(p))
	state, e := v.State()
	paramsMust(t, e)
	text := v.TraceFile()
	if state.Timeout != 29 || !state.UnsatCore || text != value || b.Used() != before {
		t.Fatal("Go strings must not stop the project update on an arena fault", state, text)
	}
	contextParamsCold(t, v, g, b)
	b.FailAfter(-1)
	copied := *v
	if !errors.Is(copied.Close(), uparams.ErrParamsOwner) {
		t.Fatal("copied owner")
	}
}

func TestContextParamsEveryUpdateAllocationAndGoStringSnapshots(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	v, e := NewContextParams(c, g, b)
	paramsMust(t, e)
	key := "trace_file_name"
	trace := strings.Repeat("a", 41)
	paramsMust(t, c.SetStringName(p, &key, &trace))
	key = "dot_proof_file"
	dot := strings.Repeat("b", 47)
	paramsMust(t, c.SetStringName(p, &key, &dot))
	key = "unsat_core"
	paramsMust(t, c.SetBoolName(p, &key, true))
	old := v.TraceFile()
	b.FailAfter(0)
	paramsMust(t, v.Update(p))
	state, e := v.State()
	paramsMust(t, e)
	text := v.TraceFile()
	proof := v.DotProofFile()
	if text != trace || proof != dot || !state.UnsatCore {
		t.Fatal("complete Go text update", state, text, proof)
	}
	if old != "z3.log" {
		t.Fatal("previous Go string snapshot", old)
	}
	contextParamsCold(t, v, g, b)
	current := v.TraceFile()
	b.FailAfter(-1)
	paramsMust(t, v.Close())
	contextParamsCold(t, nil, g, b)
	if current != trace {
		t.Fatal("closed owner does not mutate Go string snapshot", current)
	}
}

func TestContextParamsAllBooleanUpdateFieldsWrongKindsAndGlobalReRead(t *testing.T) {
	keys := []string{"auto_config", "proof", "model", "model_validate", "dump_models", "trace", "unsat_core", "debug_ref_count", "smtlib2_compliant"}
	for _, key := range keys {
		for _, value := range []bool{false, true} {
			c, g, p, b := dynAckSourceFixture(t)
			v, e := NewContextParams(c, g, b)
			paramsMust(t, e)
			paramsMust(t, c.SetBoolName(p, &key, value))
			paramsMust(t, v.Update(p))
			s, e := v.State()
			paramsMust(t, e)
			actual := map[string]bool{"auto_config": s.AutoConfig, "proof": s.Proof, "model": s.Model, "model_validate": s.ModelValidate, "dump_models": s.DumpModels, "trace": s.Trace, "unsat_core": s.UnsatCore, "debug_ref_count": s.DebugRefCount, "smtlib2_compliant": s.SMTLib2Compliant}
			if actual[key] != value {
				t.Fatal(key, value, s)
			}
			paramsMust(t, c.SetUIntName(p, &key, 99))
			paramsMust(t, v.Update(p))
			after, e := v.State()
			paramsMust(t, e)
			if after != s {
				t.Fatal("wrong-kind preserves current scalar", key, s, after)
			}
			global, e := g.SourceGetRef()
			paramsMust(t, e)
			paramsMust(t, c.SetBoolName(global, &key, !value))
			paramsMust(t, v.UpdateGlobal())
			after, e = v.State()
			paramsMust(t, e)
			actual = map[string]bool{"auto_config": after.AutoConfig, "proof": after.Proof, "model": after.Model, "model_validate": after.ModelValidate, "dump_models": after.DumpModels, "trace": after.Trace, "unsat_core": after.UnsatCore, "debug_ref_count": after.DebugRefCount, "smtlib2_compliant": after.SMTLib2Compliant}
			if actual[key] == value {
				t.Fatal("actual fresh global ref", key, after)
			}
			paramsMust(t, v.SetRLimit(73))
			after, e = v.State()
			paramsMust(t, e)
			if after.RLimit != 73 || after.InterpolantsInitialized {
				t.Fatal("scalar setter/never-initialized interpolants", after)
			}
			contextParamsCold(t, v, g, b)
			paramsMust(t, v.Close())
			contextParamsCold(t, nil, g, b)
		}
	}
}
