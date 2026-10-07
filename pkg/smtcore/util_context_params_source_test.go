// Source: src/cmd_context/context_params.h:26-55;
// src/cmd_context/context_params.cpp:26-40,137-156.
// Source author: Leonardo de Moura (2012-12-01).
// Licensed under the MIT license; see LICENSE.txt.
package smtcore

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func contextParamsCold(t *testing.T, v *ContextParams, g *util.GlobalParamsService, b *arena.Budget) {
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
		n += v.header + uint64(len(v.dotProof.External)+len(v.dotProof.Pending)+len(v.traceFile.External)+len(v.traceFile.Pending))
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
	a, e := v.TraceFile().String()
	paramsMust(t, e)
	d, e := v.DotProofFile().String()
	paramsMust(t, e)
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
	a, e := v.TraceFile().String()
	paramsMust(t, e)
	if s.Timeout != 17 || a != "z3.log" {
		t.Fatal("scalar current fallback versus fixed string literal", s, a)
	}
	// The nested fallback is evaluated even when outer type_check has a value;
	// both are source reads, with no extra module reference or lexical frame.
	contextParamsCold(t, v, g, b)
}
func TestContextParamsEveryConstructorReservationAndReverseStringUnwind(t *testing.T) {
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
			if v != nil && (!v.dotProof.Closed || !v.traceFile.Closed || !v.state.DefaultsInitialized) {
				t.Fatal("completed strings source unwind before return")
			}
			if v != nil {
				before := b.Used()
				if x := v.Update(p); !errors.Is(x, util.ErrParamsPrecondition) {
					t.Fatal("failed ctor ordinary Update", x)
				}
				if x := v.UpdateGlobal(); !errors.Is(x, util.ErrParamsPrecondition) {
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
			if ordinal != 3 {
				t.Fatal("actual header and two external arrays", ordinal)
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
	val := strings.Repeat("q", 90)
	paramsMust(t, c.SetStringName(p, &key, &val))
	before := b.Used()
	b.FailAfter(0)
	e = v.Update(p)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	s, x := v.State()
	paramsMust(t, x)
	a, x := v.TraceFile().String()
	paramsMust(t, x)
	if s.Timeout != 29 || s.UnsatCore || a != "z3.log" || b.Used() != before {
		t.Fatal("prefix scalar before trace assignment; suffix unread", s, a)
	}
	contextParamsCold(t, v, g, b)
	b.FailAfter(-1)
	paramsMust(t, v.Update(p))
	s, x = v.State()
	paramsMust(t, x)
	if !s.UnsatCore {
		t.Fatal(s)
	}
	copyOwner := *v
	if e := copyOwner.Close(); !errors.Is(e, util.ErrParamsOwner) {
		t.Fatal("shallow-copy guard", e)
	}
}

func TestContextParamsEveryUpdateAllocationAndReachedStringViewLifetime(t *testing.T) {
	for ordinal := int64(0); ; ordinal++ {
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
		b.FailAfter(ordinal)
		e = v.Update(p)
		contextParamsCold(t, v, g, b)
		s, x := v.State()
		paramsMust(t, x)
		a, x := v.TraceFile().String()
		paramsMust(t, x)
		d, x := v.DotProofFile().String()
		paramsMust(t, x)
		if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(e)
		}
		if ordinal == 0 && (a != "z3.log" || d != "proof.dot" || s.UnsatCore) {
			t.Fatal("first string refusal", a, d, s)
		}
		if ordinal == 1 && (a != trace || d != "proof.dot" || s.UnsatCore) {
			t.Fatal("first string published before second refusal", a, d, s)
		}
		if ordinal >= 1 {
			if _, x := old.Byte(0); !errors.Is(x, arena.ErrInvalidSlot) {
				t.Fatal("real replacement invalidates captured view", x)
			}
		}
		closedView := v.TraceFile()
		b.FailAfter(-1)
		paramsMust(t, v.Close())
		contextParamsCold(t, nil, g, b)
		if _, x := closedView.Byte(0); x == nil {
			t.Fatal("closed borrowed view")
		}
		if e == nil {
			if ordinal != 2 || a != trace || d != dot || !s.UnsatCore {
				t.Fatal("exact two arrays and suffix", ordinal, a, d, s)
			}
			break
		}
		if ordinal > 2 {
			t.Fatal("bounded horizon")
		}
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
