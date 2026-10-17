// Source: src/smt/params/smt_params.h:75-318; src/smt/params/qi_params.h:31-109;
// src/qe/qe_sat_tactic.cpp:632; src/muz/pdr/pdr_farkas_learner.cpp:390.
package smtcore

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func smtCopyGraph(t *testing.T, g *uparams.GlobalParamsService, b *arena.Budget, owners ...*SMTParams) {
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
	for _, s := range owners {
		if s == nil {
			continue
		}
		if s.self != s || s.Budget != b {
			t.Fatal("exact copy owner")
		}
		n += s.header
		for _, m := range []*uparams.ModuleParamsOwner{&s.ModuleParamsOwner, &s.preprocess.ModuleParamsOwner, &s.preprocess.pattern.ModuleParamsOwner, &s.dyn.ModuleParamsOwner, &s.qi.ModuleParamsOwner, &s.arith.ModuleParamsOwner, &s.array.ModuleParamsOwner, &s.bv.ModuleParamsOwner, &s.str.ModuleParamsOwner, &s.pb.ModuleParamsOwner} {
			for f := m.Helpers; f != nil; f = f.Next {
				if f.Owner != m || f.Self != f || f.Invocation == 0 {
					t.Fatal("named exact helper")
				}
				n += f.Header
			}
		}
	}
	if g.Symbols.Budget == b {
		n += astTestSymbolGraph(g.Symbols) + g.Symbols.Stats().Table.RetainedBytes
	}
	if n != b.Used() {
		t.Fatal("exact live cold graph before faultlift", n, b.Used())
	}
}
func smtCopyLongSource(t *testing.T, c *uparams.ParamsContext, p *uparams.ParamsRef, s *SMTParams) {
	t.Helper()
	paramsMust(t, c.SetStringName(p, dynAckName("qi.cost"), dynAckName(strings.Repeat("c", 43))))
	paramsMust(t, c.SetStringName(p, dynAckName("mbqi.id"), dynAckName("borrowed-id")))
	paramsMust(t, s.qi.Update(p))
	// The generated update does not assign m_qi_new_gen.
	s.qi.newGen = strings.Repeat("g", 37)
	s.qi.cost = strings.Repeat("c", 43)
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
	if v.qi.cost != s.qi.cost || v.qi.newGen != s.qi.newGen {
		t.Fatal("copied Go strings")
	}
	if v.qi.state.MBQIId != s.qi.state.MBQIId || v.qi.self != &v.qi || v.preprocess.self != &v.preprocess || v.Helpers != nil || v.completedBases != 511 || !v.constructorReady {
		t.Fatal("fresh concrete owner, exact borrowed pointer")
	}
	paramsMust(t, s.Close())
	if v.qi.cost[42] != 'c' {
		t.Fatal("copied text outlives original")
	}
	paramsMust(t, c.Reset(p))
	if _, e = v.qi.state.MBQIId.Byte(0); e == nil {
		t.Fatal("copied borrowed pointer must retain original source lifetime")
	}
}
func TestSMTParamsCopyConstructorEveryActualAllocationAndReverseCompletedUnwind(t *testing.T) {
	for ordinal := int64(0); ordinal <= 1; ordinal++ {
		c, g, p, b := dynAckSourceFixture(t)
		src, e := NewSMTParams(c, p, g, b)
		paramsMust(t, e)
		smtCopyLongSource(t, c, p, src)
		b.FailAfter(ordinal)
		dst, e := src.Clone()
		smtCopyGraph(t, g, b, src, dst)
		if ordinal == 0 {
			if !errors.Is(e, arena.ErrInjectedFailure) || dst != nil {
				t.Fatal("parent header refusal", e)
			}
		} else if e != nil || !dst.constructorReady || dst.completedBases != 511 {
			t.Fatal("Go text copy requires only parent header", e)
		}
		b.FailAfter(-1)
		if dst != nil {
			paramsMust(t, dst.Close())
		}
		paramsMust(t, src.Close())
		smtCopyGraph(t, g, b)
	}
}

func TestSMTParamsAssignmentPreservesSourceOrderWithGoStrings(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	src, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	dst, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	smtCopyLongSource(t, c, p, src)
	src.preprocess.state.LiftITE = 77
	src.dyn.state.Threshold = 89
	src.qi.state.EagerThreshold = 987
	src.arith.state.Mode = 42
	src.state.RandomSeed = 654
	b.FailAfter(0)
	paramsMust(t, dst.Assign(src))
	if dst.preprocess.state != src.preprocess.state || dst.dyn.state != src.dyn.state || dst.qi.state != src.qi.state || dst.arith.state != src.arith.state || dst.state != src.state || len(dst.qi.cost) != 43 || len(dst.qi.newGen) != 37 {
		t.Fatal("Go assignment must finish independently of arena failures")
	}
	smtCopyGraph(t, g, b, src, dst)
	paramsMust(t, dst.Assign(dst))
	b.FailAfter(-1)
	paramsMust(t, dst.Close())
	paramsMust(t, src.Close())
	smtCopyGraph(t, g, b)
}

func TestSMTParamsAssignmentEveryActualAllocationPreservesReceiverIdentityAndStickyBodyCause(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	src, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	dst, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	smtCopyLongSource(t, c, p, src)
	owner, header := dst.self, dst.header
	oldcost, oldgen := dst.qi.cost, dst.qi.newGen
	b.FailAfter(0)
	paramsMust(t, dst.Assign(src))
	if dst.self != owner || dst.header != header || dst.completedBases != 511 || dst.Helpers != nil || dst.copyCause != nil {
		t.Fatal("Go assignment changed owner or created a false failure")
	}
	if oldcost != "(+ weight generation)" {
		t.Fatal("old cost snapshot", oldcost)
	}
	if oldgen != "cost" {
		t.Fatal("old generation snapshot", oldgen)
	}
	smtCopyGraph(t, g, b, src, dst)
	b.FailAfter(-1)
	clone, e := dst.Clone()
	paramsMust(t, e)
	if clone.copyCause != nil || clone.constructorCause != nil {
		t.Fatal("fresh clone history")
	}
	smtCopyGraph(t, g, b, src, dst, clone)
	paramsMust(t, clone.Close())
	paramsMust(t, dst.Close())
	paramsMust(t, src.Close())
	smtCopyGraph(t, g, b)
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
	if _, e = shallow.Clone(); !errors.Is(e, uparams.ErrParamsOwner) {
		t.Fatal("accidental Go copy cannot acquire or release", e)
	}
	if e = shallow.Assign(s); !errors.Is(e, uparams.ErrParamsOwner) {
		t.Fatal(e)
	}
	if e = s.Assign(&shallow); !errors.Is(e, uparams.ErrParamsOwner) {
		t.Fatal(e)
	}
	smtCopyGraph(t, g, b, s)
	b.FailAfter(-1)
	_, g2, p2, b2 := dynAckSourceFixture(t)
	other, e := NewSMTParams(g2.Context, p2, g2, b2)
	paramsMust(t, e)
	before := s.state
	if e = s.Assign(other); !errors.Is(e, uparams.ErrParamsOwner) {
		t.Fatal("foreign global source namespace", e)
	}
	if s.state != before {
		t.Fatal("foreign refusal before publication")
	}
	smtCopyGraph(t, g, b, s)
	smtCopyGraph(t, g2, b2, other)
	paramsMust(t, other.Close())
	paramsMust(t, s.SourceFinalize())
	if _, e = s.Clone(); !errors.Is(e, uparams.ErrParamsOwner) {
		t.Fatal("ended source lifetime", e)
	}
	paramsMust(t, s.Close())
}

func TestSMTParamsStringCopyUsesCountedBytesAndGoAssignment(t *testing.T) {
	c, g, p, b := dynAckSourceFixture(t)
	src, e := NewSMTParams(c, p, g, b)
	paramsMust(t, e)
	smtCopyLongSource(t, c, p, src)
	// Copying a Go string retains suffix bytes after an embedded zero.
	src.qi.cost = src.qi.cost[:1] + "\x00" + src.qi.cost[2:]
	dst, e := src.Clone()
	paramsMust(t, e)
	if len(dst.qi.cost) != 43 || dst.qi.cost[1] != 0 || dst.qi.cost[42] != 'c' {
		t.Fatal("copy is counted string, never strlen")
	}
	smtCopyGraph(t, g, b, src, dst)
	src.qi.cost = "tiny"
	src.qi.newGen = ""
	// Go assignment preserves values independently of the numeric arena.

	b.FailAfter(0)
	paramsMust(t, dst.Assign(src))
	smtCopyGraph(t, g, b, src, dst)
	if len(dst.qi.cost) != 4 || len(dst.qi.newGen) != 0 {
		t.Fatal("inplace source member assignment")
	}
	b.FailAfter(-1)
	b.FailAfter(1)
	small, e := src.Clone()
	paramsMust(t, e)
	smtCopyGraph(t, g, b, src, dst, small)
	if small.qi.cost != "tiny" || small.qi.newGen != "" {
		t.Fatal("Go text copy only charges actual parent header")
	}
	b.FailAfter(-1)
	paramsMust(t, small.Close())
	paramsMust(t, dst.Close())
	paramsMust(t, src.Close())
	smtCopyGraph(t, g, b)
}
