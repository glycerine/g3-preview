// Source: src/smt/params/smt_params.h:75-318; src/smt/params/qi_params.h:31-109;
// src/qe/qe_sat_tactic.cpp:632; src/muz/pdr/pdr_farkas_learner.cpp:390.
// Source author: Leonardo de Moura (2008-02-20; 2008-06-15).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"strings"
	"testing"
)

func smtCopyGraph(t *testing.T, g *GlobalParamsService, b *arena.Budget, owners ...*SMTParams) {
	t.Helper()
	n := g.ownerBytes + catalogGraph(g.config.Catalog)
	if g.current != nil {
		n += gparamsGraph(g.current)
	}
	for x := g.failed; x != nil; x = x.retainedNext {
		n += gparamsGraph(x)
	}
	for x := g.superseded; x != nil; x = x.retainedNext {
		n += gparamsGraph(x)
	}
	for _, s := range owners {
		if s == nil {
			continue
		}
		if s.self != s || s.budget != b {
			t.Fatal("exact copy owner")
		}
		n += s.header + uint64(len(s.qi.cost.external)+len(s.qi.cost.pending)+len(s.qi.newGen.external)+len(s.qi.newGen.pending))
		for _, m := range []*smtModuleParamsOwner{&s.smtModuleParamsOwner, &s.preprocess.smtModuleParamsOwner, &s.preprocess.pattern.smtModuleParamsOwner, &s.dyn.smtModuleParamsOwner, &s.qi.smtModuleParamsOwner, &s.arith.smtModuleParamsOwner, &s.array.smtModuleParamsOwner, &s.bv.smtModuleParamsOwner, &s.str.smtModuleParamsOwner, &s.pb.smtModuleParamsOwner} {
			for f := m.helpers; f != nil; f = f.next {
				if f.owner != m || f.self != f || f.invocation == 0 {
					t.Fatal("named exact helper")
				}
				n += f.header
			}
		}
	}
	if g.symbols.budget == b {
		n += astTestSymbolGraph(g.symbols) + g.symbols.Stats().Table.RetainedBytes
	}
	if n != b.Used() {
		t.Fatal("exact live cold graph before faultlift", n, b.Used())
	}
}
func smtCopyLongSource(t *testing.T, c *ParamsContext, p *ParamsRef, s *SMTParams) {
	t.Helper()
	paramsMust(t, c.SetStringName(p, dynAckName("qi.cost"), dynAckName(strings.Repeat("c", 43))))
	paramsMust(t, c.SetStringName(p, dynAckName("mbqi.id"), dynAckName("borrowed-id")))
	paramsMust(t, s.qi.Update(p))
	// Source public m_qi_new_gen is not assigned by updt_params; establish
	// its value through the actual selected std::string assignment engine.
	paramsMust(t, s.qi.newGen.assign(literalParamsStringView(strings.Repeat("g", 37))))
	// A later shorter public string assignment retains larger capacity; copy
	// construction must allocate source length, not source capacity.
	paramsMust(t, s.qi.cost.assign(literalParamsStringView(strings.Repeat("c", 44))))
	paramsMust(t, s.qi.cost.assign(literalParamsStringView(strings.Repeat("c", 43))))
}
func TestSMTParamsCopyConstructorPreservesAllBasesAndBorrowedPointerWithoutGetters(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	s, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, s.Close()) }()
	smtCopyLongSource(t, c, p, s)
	s.state.RandomSeed = 123
	s.dyn.state.Threshold = 98
	s.qi.state.Ematching = false
	s.qi.state.EmatchingInitialized = false
	paramsMust(t, g.Close()) // Copy must not use module/default getters or reopen a closed service.
	v, e := s.Clone()
	paramsMust(t, e)
	defer func() { paramsMust(t, v.Close()) }()
	smtCopyGraph(t, g, b, s, v)
	if v.state != s.state || v.preprocess.state != s.preprocess.state || v.preprocess.pattern.state != s.preprocess.pattern.state || v.preprocess.bit != s.preprocess.bit || v.dyn.state != s.dyn.state || v.qi.state != s.qi.state || v.arith.state != s.arith.state || v.array.state != s.array.state || v.bv.state != s.bv.state || v.str.state != s.str.state || v.pb.state != s.pb.state || v.datatype != s.datatype {
		t.Fatal("literal base/member values")
	}
	if s.qi.cost.capacity != 86 || v.qi.cost.capacity != 43 || v.qi.newGen.capacity != 37 || &v.qi.cost.external[0] == &s.qi.cost.external[0] || &v.qi.newGen.external[0] == &s.qi.newGen.external[0] {
		t.Fatal("independent exact-length strings")
	}
	if v.qi.state.MBQIId != s.qi.state.MBQIId || v.qi.self != &v.qi || v.preprocess.self != &v.preprocess || v.helpers != nil || v.completedBases != 511 || !v.constructorReady {
		t.Fatal("fresh concrete owner, exact borrowed pointer")
	}
	paramsMust(t, s.Close())
	if ch, e := v.qi.cost.view().Byte(42); e != nil || ch != 'c' {
		t.Fatal("copied owned text outlives original", e)
	}
	paramsMust(t, c.Reset(p))
	if _, e = v.qi.state.MBQIId.Byte(0); e == nil {
		t.Fatal("copied borrowed pointer must retain original source lifetime")
	}
}
func TestSMTParamsCopyConstructorEveryActualAllocationAndReverseCompletedUnwind(t *testing.T) {
	for ordinal := int64(0); ordinal <= 3; ordinal++ {
		c, g, p, b := dynAckSourceFixture(t)
		s, e := NewSMTParams(c, p, g, b)
		paramsMust(t, e)
		smtCopyLongSource(t, c, p, s)
		b.FailAfter(ordinal)
		v, e := s.Clone()
		smtCopyGraph(t, g, b, s, v)
		if ordinal < 3 && !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal("actual header,cost,newGen frontier", ordinal, e)
		}
		if ordinal == 1 && (v == nil || v.completedBases != 3 || v.endedBases != 3 || v.qi.completed != 0) {
			t.Fatal("older preproc/dyn source unwind", v)
		}
		if ordinal == 2 && (v == nil || v.qi.completed != 1 || !v.qi.cost.closed || v.qi.newGen.ready || len(v.qi.cost.external) != 0 || v.completedBases != 3) {
			t.Fatal("completed cost automatic destruction before return")
		}
		if ordinal == 3 && (e != nil || v.completedBases != 511 || !v.constructorReady) {
			t.Fatal("first complete source copy", e)
		}
		b.FailAfter(-1)
		if v != nil {
			paramsMust(t, v.Close())
		}
		paramsMust(t, s.Close())
		smtCopyGraph(t, g, b)
	}
}
func TestSMTParamsAssignmentPublishesOnlyLiteralPrefixAtSecondStringRefusal(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	src, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	dst, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	defer func() { paramsMust(t, dst.Close()); paramsMust(t, src.Close()) }()
	smtCopyLongSource(t, c, p, src)
	src.preprocess.state.LiftITE = 77
	src.dyn.state.Threshold = 89
	src.qi.state.Ematching = true
	src.qi.state.EmatchingInitialized = true
	src.qi.state.EagerThreshold = 987
	src.arith.state.Mode = 42
	src.state.RandomSeed = 654
	oldqi := dst.qi.state
	oldarith := dst.arith.state
	oldparent := dst.state
	b.FailAfter(1)
	e = dst.Assign(src)
	smtCopyGraph(t, g, b, src, dst)
	if !errors.Is(e, arena.ErrInjectedFailure) || dst.preprocess.state != src.preprocess.state || dst.dyn.state != src.dyn.state || !dst.qi.state.Ematching || !dst.qi.state.EmatchingInitialized || dst.qi.cost.length != 43 || dst.qi.newGen.length != 4 || dst.qi.state.EagerThreshold != oldqi.EagerThreshold || dst.arith.state != oldarith || dst.state != oldparent {
		t.Fatal("literal member assignment frontier", e)
	}
	b.FailAfter(-1)
	paramsMust(t, dst.Assign(src))
	if dst.qi.state != src.qi.state || dst.state != src.state || dst.arith.state != src.arith.state {
		t.Fatal("source-valid retry finishes assignment")
	}
	smtCopyGraph(t, g, b, src, dst)
	b.FailAfter(0)
	paramsMust(t, dst.Assign(dst))
	smtCopyGraph(t, g, b, src, dst)
	b.FailAfter(-1)
}

func TestSMTParamsAssignmentEveryActualAllocationPreservesReceiverIdentityAndStickyBodyCause(t *testing.T) {
	for ordinal := int64(0); ordinal <= 2; ordinal++ {
		c, g, p, b := dynAckSourceFixture(t)
		src, e := NewSMTParams(c, p, g, b)
		paramsMust(t, e)
		dst, e := NewSMTParams(c, p, g, b)
		paramsMust(t, e)
		smtCopyLongSource(t, c, p, src)
		owner, header := dst.self, dst.header
		oldcost, oldgen := dst.qi.cost.view(), dst.qi.newGen.view()
		b.FailAfter(ordinal)
		e = dst.Assign(src)
		smtCopyGraph(t, g, b, src, dst)
		if ordinal < 2 && !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal("actual two external assignments", ordinal, e)
		}
		if ordinal == 0 && (dst.qi.cost.capacity != 21 || dst.qi.newGen.capacity != 15) {
			t.Fatal("no replacement before first allocation")
		}
		if ordinal == 1 && (dst.qi.cost.capacity != 43 || dst.qi.newGen.capacity != 15) {
			t.Fatal("only cost replaced")
		}
		if ordinal == 2 && (e != nil || dst.qi.cost.capacity != 43 || dst.qi.newGen.capacity != 37) {
			t.Fatal("first success", e)
		}
		if dst.self != owner || dst.header != header || dst.completedBases != 511 || dst.helpers != nil {
			t.Fatal("source assignment must not replace Go owner")
		}
		if ordinal == 0 {
			if _, err := oldcost.Byte(0); err != nil {
				t.Fatal("failed allocation preserves old pointer", err)
			}
		} else {
			if _, err := oldcost.Byte(0); err == nil {
				t.Fatal("actual replaced pointer invalidated")
			}
		}
		if ordinal < 2 {
			if _, err := oldgen.Byte(0); err != nil {
				t.Fatal("unreplaced later pointer", err)
			}
		}
		cause := dst.copyCause
		b.FailAfter(-1)
		paramsMust(t, dst.Assign(src))
		if dst.copyCause != cause {
			t.Fatal("first body cause stays sticky")
		}
		copy, e := dst.Clone()
		paramsMust(t, e)
		if copy.copyCause != nil || copy.constructorCause != nil {
			t.Fatal("history must not be copied")
		}
		smtCopyGraph(t, g, b, src, dst, copy)
		paramsMust(t, copy.Close())
		paramsMust(t, dst.Close())
		paramsMust(t, src.Close())
		smtCopyGraph(t, g, b)
	}
}
func TestSMTParamsCopyGuardsAndActualSelfAssignmentLeaveProviderAndAllocationsUnread(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	s, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	// Source self string assignment is a real no-op; no budget reservation.
	b.FailAfter(0)
	paramsMust(t, s.Assign(s))
	smtCopyGraph(t, g, b, s)
	shallow := *s
	if _, e = shallow.Clone(); !errors.Is(e, ErrParamsOwner) {
		t.Fatal("accidental Go copy cannot acquire or release", e)
	}
	if e = shallow.Assign(s); !errors.Is(e, ErrParamsOwner) {
		t.Fatal(e)
	}
	if e = s.Assign(&shallow); !errors.Is(e, ErrParamsOwner) {
		t.Fatal(e)
	}
	smtCopyGraph(t, g, b, s)
	b.FailAfter(-1)
	_, g2, p2, b2 := dynAckSourceFixture(t)
	other, e := NewSMTParams(g2.context, p2, g2, b2)
	paramsMust(t, e)
	before := s.state
	if e = s.Assign(other); !errors.Is(e, ErrParamsOwner) {
		t.Fatal("foreign global source namespace", e)
	}
	if s.state != before {
		t.Fatal("foreign refusal before publication")
	}
	smtCopyGraph(t, g, b, s)
	smtCopyGraph(t, g2, b2, other)
	paramsMust(t, other.Close())
	paramsMust(t, s.SourceFinalize())
	if _, e = s.Clone(); !errors.Is(e, ErrParamsOwner) {
		t.Fatal("ended source lifetime", e)
	}
	paramsMust(t, s.Close())
}

func TestSMTParamsStringCopyUsesCountedBytesAndInlineAssignmentRetainsCapacity(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	src, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	smtCopyLongSource(t, c, p, src)
	// Actual public std::string indexed mutation preserves its counted length;
	// copying the string must retain suffix bytes after an embedded zero.
	src.qi.cost.put(1, 0)
	dst, e := src.Clone()
	paramsMust(t, e)
	if dst.qi.cost.length != 43 || dst.qi.cost.external[1] != 0 || dst.qi.cost.external[42] != 'c' {
		t.Fatal("copy is counted string, never strlen")
	}
	smtCopyGraph(t, g, b, src, dst)
	paramsMust(t, src.qi.cost.assign(literalParamsStringView("tiny")))
	paramsMust(t, src.qi.newGen.assign(literalParamsStringView("")))
	// Existing-capacity assignment doesn't deallocate or shrink the destination.
	capCost, capGen := dst.qi.cost.capacity, dst.qi.newGen.capacity
	b.FailAfter(0)
	paramsMust(t, dst.Assign(src))
	smtCopyGraph(t, g, b, src, dst)
	if dst.qi.cost.capacity != capCost || dst.qi.newGen.capacity != capGen || dst.qi.cost.length != 4 || dst.qi.newGen.length != 0 {
		t.Fatal("inplace source member assignment")
	}
	b.FailAfter(-1)
	b.FailAfter(1)
	small, e := src.Clone()
	paramsMust(t, e)
	smtCopyGraph(t, g, b, src, dst, small)
	if small.qi.cost.external != nil || small.qi.newGen.external != nil || small.qi.cost.capacity != 15 || small.qi.newGen.capacity != 15 || small.qi.newGen.inline[0] != 0 {
		t.Fatal("inline copy only allocates actual parent header")
	}
	b.FailAfter(-1)
	paramsMust(t, small.Close())
	paramsMust(t, dst.Close())
	paramsMust(t, src.Close())
	smtCopyGraph(t, g, b)
}
