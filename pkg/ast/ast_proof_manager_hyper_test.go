// Licensed under the MIT license.
// Additional finite tests; no registered hyper-resolution source test exists.
package ast

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func hyperTestInputs(t *testing.T, m *ASTManager) ([]ASTHandle, ASTHandle, []ASTRefVector) {
	t.Helper()
	a, b, c := proofTestBool(t, m, "hyper-a"), proofTestBool(t, m, "hyper-b"), proofTestBool(t, m, "hyper-c")
	pa, e := m.MkProofHypothesis(a)
	astTestMust(t, e)
	pb, e := m.MkProofHypothesis(b)
	astTestMust(t, e)
	v, e := m.NewRefVectorFrom([]ASTHandle{a, b})
	astTestMust(t, e)
	w, e := m.NewRefVectorFrom([]ASTHandle{c})
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, m.DelRefVector(&v)); astTestMust(t, m.DelRefVector(&w)) })
	return []ASTHandle{pa, pb}, c, []ASTRefVector{v, w}
}

func TestASTProofHyperActualParametersAppendAndDisabled(t *testing.T) {
	m := astTestManager(t)
	// Source hyper-resolution directly calls mk_app even with proofs disabled.
	ps, f, substs := hyperTestInputs(t, m)
	positions := []ASTProofPosition{{First: math.MaxUint32, Second: 1 << 31}}
	p, e := m.MkProofHyperResolve(ps, f, positions, substs)
	astTestMust(t, e)
	if p.IsNull() {
		t.Fatal("source has no disabled guard")
	}
	proofTestFact(t, m, p, f)
	d, e := m.GetDecl(p)
	astTestMust(t, e)
	info := m.infoAt(m.declAt(d.slot).info.slot)
	if info.count != 5 {
		t.Fatal("flattened parameter count", info.count)
	}
	for i, want := range []int32{-1, math.MinInt32} {
		param, e := m.infoParameter(info, uint32(i+2))
		astTestMust(t, e)
		record, e := m.parameter(param)
		astTestMust(t, e)
		if record.Kind != ASTParameterInt || record.integer != want {
			t.Fatal("uint32-to-int32 source position conversion", i, record.integer)
		}
	}
	out, e := NewASTProofHyperOutput(m)
	astTestMust(t, e)
	defer out.Close()
	premises, e := m.NewRefVector()
	astTestMust(t, e)
	defer m.DelRefVector(&premises)
	conclusion, e := m.NewRef(ASTHandle{})
	astTestMust(t, e)
	defer m.DelRef(&conclusion)
	for iteration := uint32(0); iteration < 2; iteration++ {
		yes, e := m.MatchProofHyperResolve(p, premises, conclusion, out)
		astTestMust(t, e)
		if !yes || out.PositionsLen() != iteration+1 || out.SubstitutionsLen() != 2*(iteration+1) {
			t.Fatal("append-only outputs", yes, out.PositionsLen(), out.SubstitutionsLen())
		}
		pos, e := out.Position(iteration)
		astTestMust(t, e)
		if pos != positions[0] {
			t.Fatal("signed integer decode", pos)
		}
		for j, input := range substs {
			v, e := out.Substitution(2*iteration + uint32(j))
			astTestMust(t, e)
			n, e := m.RefVectorSize(input)
			astTestMust(t, e)
			gotN, e := m.RefVectorSize(v)
			astTestMust(t, e)
			if gotN != n {
				t.Fatal("substitution lengths")
			}
			for k := uint32(0); k < n; k++ {
				want, e := m.RefVectorGet(input, k)
				astTestMust(t, e)
				got, e := m.RefVectorGet(v, k)
				astTestMust(t, e)
				if want != got {
					t.Fatal("substitution order")
				}
			}
		}
	}
	n, e := m.RefVectorSize(premises)
	astTestMust(t, e)
	if n != 4 {
		t.Fatal("premises append", n)
	}
	got, e := m.RefGet(conclusion)
	astTestMust(t, e)
	if got != f {
		t.Fatal("conclusion assignment")
	}
	yes, e := m.MatchProofHyperResolve(f, premises, conclusion, nil)
	astTestMust(t, e)
	if yes {
		t.Fatal("unmatched should not touch outputs")
	}
}

func TestASTProofHyperUnusedFactsVectorFailsBeforeConclusionDiagnostic(t *testing.T) {
	m := astTestManager(t)
	ps, _, substs := hyperTestInputs(t, m)
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	_, e := m.MkProofHyperResolve(ps, ASTHandle{}, []ASTProofPosition{{}}, substs)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || m.BudgetField.Used() != before {
		t.Fatal("unused fmls is first allocation, before Bool assertion", e, m.BudgetField.Used(), before)
	}
}

func TestASTProofHyperReaderPremiseIncrementBeforeGrowthFailure(t *testing.T) {
	m := astTestManager(t)
	ps, f, substs := hyperTestInputs(t, m)
	p, e := m.MkProofHyperResolve(ps, f, []ASTProofPosition{{}}, substs)
	astTestMust(t, e)
	out, e := NewASTProofHyperOutput(m)
	astTestMust(t, e)
	defer out.Close()
	vec, e := m.NewRefVector()
	astTestMust(t, e)
	defer m.DelRefVector(&vec)
	conclusion, e := m.NewRef(ASTHandle{})
	astTestMust(t, e)
	defer m.DelRef(&conclusion)
	before, e := m.GetRefCount(ps[0])
	astTestMust(t, e)
	m.BudgetField.FailAfter(0)
	yes, e := m.MatchProofHyperResolve(p, vec, conclusion, out)
	m.BudgetField.FailAfter(-1)
	after, err := m.GetRefCount(ps[0])
	astTestMust(t, err)
	n, err := m.RefVectorSize(vec)
	astTestMust(t, err)
	h, err := m.RefGet(conclusion)
	astTestMust(t, err)
	if yes || !errors.Is(e, arena.ErrInjectedFailure) || after != before+1 || n != 0 || !h.IsNull() || out.SubstitutionsLen() != 0 {
		t.Fatal("source INC-before-growth output prefix", yes, e, after, before, n, h, out.SubstitutionsLen())
	}
	// Explicit caller balancing is permitted; vector destruction cannot consume
	// the failed push's source-retained reference.
	astTestMust(t, m.DecRef(ps[0]))
}

func TestASTProofHyperLexicalRawVectorsAndCache(t *testing.T) {
	for after := int64(0); after < 5; after++ {
		m := astTestManager(t)
		ps, f, _ := hyperTestInputs(t, m)
		empty, e := m.NewRefVector()
		astTestMust(t, e)
		before := m.BudgetField.Used()
		nodes, parameters, infos := m.liveNodes, m.liveParameters, m.liveInfos
		// With two premises and an empty substitution, the first five
		// reserves are: fmls(2), sorts(2), args(2), sorts(3), args(3).
		// No declaration/parameter action occurs before these finish.
		m.BudgetField.FailAfter(after)
		_, e = m.MkProofHyperResolve(ps, f, nil, []ASTRefVector{empty})
		m.BudgetField.FailAfter(-1)
		if !errors.Is(e, arena.ErrInjectedFailure) || m.BudgetField.Used() != before || m.liveNodes != nodes || m.liveParameters != parameters || m.liveInfos != infos {
			t.Fatal("lexical raw vectors leak or omit source growth", after, e, m.BudgetField.Used(), before)
		}
		astTestMust(t, m.DelRefVector(&empty))
	}
	m := astTestManager(t)
	ps, f, _ := hyperTestInputs(t, m)
	empty, e := m.NewRefVector()
	astTestMust(t, e)
	defer m.DelRefVector(&empty)
	p, e := m.MkProofHyperResolve(ps[:1], f, nil, []ASTRefVector{empty})
	astTestMust(t, e)
	q, e := m.MkProofHyperResolve(ps[:1], f, nil, []ASTRefVector{empty})
	astTestMust(t, e)
	if p != q {
		t.Fatal("actual structural interning")
	}
	// Zero-parameter hyper declarations use the source fixed cache, whose
	// first parent count remains fixed; no invented variable-arity repair.
	_, e = m.MkProofHyperResolve(ps, f, nil, []ASTRefVector{empty})
	if !errors.Is(e, ErrASTDomain) {
		t.Fatal("source zero-parameter cache arity", e)
	}
}

func TestASTProofHyperReaderRetainsConclusionBeforeOuterGrowth(t *testing.T) {
	m := astTestManager(t)
	ps, f, substs := hyperTestInputs(t, m)
	p, e := m.MkProofHyperResolve(ps, f, []ASTProofPosition{{1, 2}}, substs)
	astTestMust(t, e)
	out, e := NewASTProofHyperOutput(m)
	astTestMust(t, e)
	defer out.Close()
	vec, e := m.NewRefVectorFrom(ps)
	astTestMust(t, e)
	// Resize down retains physical capacity for the actual premise pushes.
	astTestMust(t, m.RefVectorResize(vec, 0))
	defer m.DelRefVector(&vec)
	conclusion, e := m.NewRef(ASTHandle{})
	astTestMust(t, e)
	defer m.DelRef(&conclusion)
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	yes, e := m.MatchProofHyperResolve(p, vec, conclusion, out)
	m.BudgetField.FailAfter(-1)
	n, err := m.RefVectorSize(vec)
	astTestMust(t, err)
	got, err := m.RefGet(conclusion)
	astTestMust(t, err)
	if yes || !errors.Is(e, arena.ErrInjectedFailure) || n != 2 || got != f || out.SubstitutionsLen() != 0 || out.PositionsLen() != 0 || m.BudgetField.Used() != before {
		t.Fatal("premises/conclusion precede outer substitution growth", yes, e, n, got)
	}
}

func TestASTProofHyperPositionPublishedBeforeNextSubstitutionFailure(t *testing.T) {
	matchedPrefix := false
	for after := int64(0); after < 24; after++ {
		m := astTestManager(t)
		ps, f, _ := hyperTestInputs(t, m)
		empty, e := m.NewRefVector()
		astTestMust(t, e)
		p, e := m.MkProofHyperResolve(ps, f, []ASTProofPosition{{3, 4}, {5, 6}}, []ASTRefVector{empty, empty, empty})
		astTestMust(t, e)
		out, e := NewASTProofHyperOutput(m)
		astTestMust(t, e)
		vec, e := m.NewRefVectorFrom(ps)
		astTestMust(t, e)
		astTestMust(t, m.RefVectorResize(vec, 0))
		conclusion, e := m.NewRef(ASTHandle{})
		astTestMust(t, e)
		m.BudgetField.FailAfter(after)
		yes, err := m.MatchProofHyperResolve(p, vec, conclusion, out)
		m.BudgetField.FailAfter(-1)
		if err != nil && !errors.Is(err, arena.ErrInjectedFailure) {
			t.Fatal("unexpected failure", after, err)
		}
		if out.PositionsLen() == 2 && out.SubstitutionsLen() == 2 && err != nil {
			matchedPrefix = true
			got, e := out.Position(1)
			astTestMust(t, e)
			if got != (ASTProofPosition{5, 6}) {
				t.Fatal("completed second position lost", got)
			}
		}
		if yes && (out.PositionsLen() != 2 || out.SubstitutionsLen() != 3) {
			t.Fatal("complete parse outputs")
		}
		astTestMust(t, out.Close())
		astTestMust(t, m.DelRefVector(&vec))
		astTestMust(t, m.DelRef(&conclusion))
		astTestMust(t, m.DelRefVector(&empty))
	}
	if !matchedPrefix {
		t.Fatal("did not exercise position publication before next outer-vector failure")
	}
}

func TestASTProofHyperOutputPhysicalChargesAndEscapedCleanup(t *testing.T) {
	m := astTestManager(t)
	before := m.BudgetField.Used()
	out, e := NewASTProofHyperOutput(m)
	astTestMust(t, e)
	if m.BudgetField.Used()-before != out.ownerBytes {
		t.Fatal("cold header charge")
	}
	m.mu.Lock()
	for i := uint32(0); i < 6; i++ {
		astTestMust(t, out.pushPosition(m, ASTProofPosition{i, i + 1}))
	}
	m.mu.Unlock()
	// svector capacities 2,3,5,8; scalar pair is exactly eight bytes.
	if cap(out.positions) != 8 || out.positionBytes != 64 || m.BudgetField.Used()-before != out.ownerBytes+64 {
		t.Fatal("source capacity and physical charge")
	}
	copyOwner := *out
	if !errors.Is(copyOwner.Close(), ErrASTHandle) {
		t.Fatal("copied cold owner accepted")
	}
	astTestMust(t, out.Close())
	if m.BudgetField.Used() != before {
		t.Fatal("outer owner not refunded")
	}
	escaped, e := NewASTProofHyperOutput(m)
	astTestMust(t, e)
	m.mu.Lock()
	astTestMust(t, escaped.pushSubstitution(m))
	m.mu.Unlock()
	retained := escaped.ownerBytes + escaped.substitutionBytes
	borrowedOwners := m.BudgetField.Used() - m.Stats().AllocatedBytes - retained
	astTestMust(t, m.Close())
	if m.BudgetField.Used() != borrowedOwners+retained {
		t.Fatal("manager frees its own vector graph only", m.BudgetField.Used(), retained)
	}
	astTestMust(t, escaped.Close())
	if m.BudgetField.Used() != borrowedOwners {
		t.Fatal("escaped output refunds caller storage")
	}
}

func TestASTProofHyperOutputAscendingDestructionAndIDReuse(t *testing.T) {
	m := astTestManager(t)
	s, e := m.MkBoolSort()
	astTestMust(t, e)
	a, e := m.MkVar(701, s)
	astTestMust(t, e)
	b, e := m.MkVar(702, s)
	astTestMust(t, e)
	c, e := m.MkVar(703, s)
	astTestMust(t, e)
	ids := [3]uint32{}
	for i, h := range []ASTHandle{a, b, c} {
		ids[i], e = m.GetASTID(h)
		astTestMust(t, e)
	}
	out, e := NewASTProofHyperOutput(m)
	astTestMust(t, e)
	m.mu.Lock()
	astTestMust(t, out.pushSubstitution(m))
	astTestMust(t, m.pushVector(out.substitutions[0], astNodeRef{a.slot, a.generation}))
	astTestMust(t, m.pushVector(out.substitutions[0], astNodeRef{b.slot, b.generation}))
	astTestMust(t, out.pushSubstitution(m))
	astTestMust(t, m.pushVector(out.substitutions[1], astNodeRef{c.slot, c.generation}))
	m.mu.Unlock()
	astTestMust(t, out.Close())
	// Source outer vector destroys elements ascending; each inner owning
	// ref-vector decrements values ascending. Thus the ID recycle stack is
	// a,b,c and reuse is c,b,a; this probes actual production destruction.
	for i := uint32(0); i < 3; i++ {
		h, e := m.MkVar(710+i, s)
		astTestMust(t, e)
		id, e := m.GetASTID(h)
		astTestMust(t, e)
		if id != ids[2-i] {
			t.Fatal("source lexical destruction/ID recycle order", i, id, ids)
		}
	}
}

func TestASTProofHyperMalformedTrailingPairRetainsPriorOutputs(t *testing.T) {
	m := astTestManager(t)
	ps, f, substs := hyperTestInputs(t, m)
	a, e := m.RefVectorGet(substs[0], 0)
	astTestMust(t, e)
	pa, e := m.NewParameterAST(a)
	astTestMust(t, e)
	pi, e := m.NewParameterInt(-1)
	astTestMust(t, e)
	info, e := m.NewFuncDeclInfo(0, ASTPrHyperResolve, []ASTParameter{pa, pi}, ASTFuncFlags{}, false)
	astTestMust(t, e)
	proofSort := m.external(m.proofSort)
	boolSort, e := m.MkBoolSort()
	astTestMust(t, e)
	d, e := m.MkFuncDeclWithInfo(astTestSymbol(t, m, "malformed-hyper"), []ASTHandle{proofSort, boolSort}, proofSort, info)
	astTestMust(t, e)
	p, e := m.MkApp(d, []ASTHandle{ps[0], f})
	astTestMust(t, e)
	astTestMust(t, m.DelFuncDeclInfo(&info))
	astTestMust(t, m.DelParameter(&pa))
	astTestMust(t, m.DelParameter(&pi))
	out, e := NewASTProofHyperOutput(m)
	astTestMust(t, e)
	defer out.Close()
	premises, e := m.NewRefVector()
	astTestMust(t, e)
	defer m.DelRefVector(&premises)
	conclusion, e := m.NewRef(ASTHandle{})
	astTestMust(t, e)
	defer m.DelRef(&conclusion)
	yes, err := m.MatchProofHyperResolve(p, premises, conclusion, out)
	wantErr := arena.ErrBounds
	if ASTSourceDebug {
		wantErr = ErrASTDomain
	}
	if yes || !errors.Is(err, wantErr) {
		t.Fatal("trailing pair source assertion/OOB adapter", yes, err)
	}
	n, e := m.RefVectorSize(premises)
	astTestMust(t, e)
	got, e := m.RefGet(conclusion)
	astTestMust(t, e)
	v, e := out.Substitution(0)
	astTestMust(t, e)
	vn, e := m.RefVectorSize(v)
	astTestMust(t, e)
	value, e := m.RefVectorGet(v, 0)
	astTestMust(t, e)
	if n != 1 || got != f || out.SubstitutionsLen() != 1 || out.PositionsLen() != 0 || vn != 1 || value != a {
		t.Fatal("reader must not prevalidate/rollback parameter suffix")
	}
}

func TestASTProofHyperBuilderAllFailurePrefixesImmediateOwnership(t *testing.T) {
	failures, successes := 0, 0
	for after := int64(0); after < 64; after++ {
		m := astTestManager(t)
		ps, f, substs := hyperTestInputs(t, m)
		// Warm the source declaration spelling so its legitimate interned
		// lifetime cannot be mistaken for an unaccounted numeric temporary.
		_, e := m.Symbols.Intern("hyper-res")
		astTestMust(t, e)
		borrowed := m.BudgetField.Used() - m.Stats().AllocatedBytes
		qBefore, e := m.numerals.Stats()
		astTestMust(t, e)
		m.BudgetField.FailAfter(after)
		_, err := m.MkProofHyperResolve(ps, f, []ASTProofPosition{{7, 9}}, substs)
		m.BudgetField.FailAfter(-1)
		if err != nil {
			failures++
			if !errors.Is(err, arena.ErrInjectedFailure) {
				t.Fatal(after, err)
			}
		} else {
			successes++
		}
		// Inspect immediately, before Del/Close/String: completed declaration
		// publication may legitimately survive a subsequent app failure.
		// Every surviving parameter must belong to a published node's info;
		// lexical fmls/sorts/args/params arrays must already be refunded.
		ownedInfos, ownedParameters := uint32(0), uint32(0)
		for slot := uint32(1); slot < m.nextNode; slot++ {
			r := m.nodeAt(slot)
			if !r.live {
				continue
			}
			var infoSlot uint32
			if r.kind == ASTSort {
				infoSlot = m.sortAt(slot).info.slot
			} else if r.kind == ASTFuncDecl {
				infoSlot = m.declAt(slot).info.slot
			}
			if infoSlot != 0 {
				ownedInfos++
				ownedParameters += m.infoAt(infoSlot).count
			}
		}
		qAfter, e := m.numerals.Stats()
		astTestMust(t, e)
		if m.liveInfos != ownedInfos || m.liveParameters != ownedParameters || qAfter.LiveCells != qBefore.LiveCells || qAfter.LivePayloads != qBefore.LivePayloads || m.BudgetField.Used() != borrowed+m.Stats().AllocatedBytes {
			t.Fatal("unreachable temporary before cleanup", after, err, m.liveInfos, ownedInfos, m.liveParameters, ownedParameters, m.BudgetField.Used(), borrowed+m.Stats().AllocatedBytes)
		}
	}
	if failures == 0 || successes == 0 {
		t.Fatal("did not traverse failure and success boundaries", failures, successes)
	}
}

func TestASTProofHyperReaderAllFailurePrefixesImmediateReferences(t *testing.T) {
	failures, successes := 0, 0
	for after := int64(0); after < 48; after++ {
		m := astTestManager(t)
		ps, f, substs := hyperTestInputs(t, m)
		p, e := m.MkProofHyperResolve(ps, f, []ASTProofPosition{{11, 13}}, substs)
		astTestMust(t, e)
		out, e := NewASTProofHyperOutput(m)
		astTestMust(t, e)
		premises, e := m.NewRefVector()
		astTestMust(t, e)
		conclusion, e := m.NewRef(ASTHandle{})
		astTestMust(t, e)
		inputs := []ASTHandle{ps[0], ps[1], f}
		for _, v := range substs {
			n, e := m.RefVectorSize(v)
			astTestMust(t, e)
			for i := uint32(0); i < n; i++ {
				h, e := m.RefVectorGet(v, i)
				astTestMust(t, e)
				found := false
				for _, old := range inputs {
					if old == h {
						found = true
					}
				}
				if !found {
					inputs = append(inputs, h)
				}
			}
		}
		before := make([]uint32, len(inputs))
		for i, h := range inputs {
			before[i], e = m.GetRefCount(h)
			astTestMust(t, e)
		}
		borrowed := m.BudgetField.Used() - m.Stats().AllocatedBytes - out.ownerBytes
		m.BudgetField.FailAfter(after)
		yes, err := m.MatchProofHyperResolve(p, premises, conclusion, out)
		m.BudgetField.FailAfter(-1)
		if err != nil {
			failures++
			if !errors.Is(err, arena.ErrInjectedFailure) {
				t.Fatal(after, err)
			}
		} else {
			successes++
			if !yes {
				t.Fatal("matched hyper failed")
			}
		}
		// Compute the references actually published into caller outputs before
		// any cleanup. The only permitted extra reference is the ONE source
		// INC-before-growth failure of the current push.
		expected := make([]uint32, len(inputs))
		count := func(h ASTHandle) {
			for i, old := range inputs {
				if old == h {
					expected[i]++
					return
				}
			}
			t.Fatal("unexpected output token", h)
		}
		n, e := m.RefVectorSize(premises)
		astTestMust(t, e)
		for i := uint32(0); i < n; i++ {
			h, e := m.RefVectorGet(premises, i)
			astTestMust(t, e)
			count(h)
		}
		h, e := m.RefGet(conclusion)
		astTestMust(t, e)
		if !h.IsNull() {
			count(h)
		}
		for _, v := range out.substitutions {
			n, e := m.RefVectorSize(v)
			astTestMust(t, e)
			for i := uint32(0); i < n; i++ {
				h, e := m.RefVectorGet(v, i)
				astTestMust(t, e)
				count(h)
			}
		}
		extra := uint32(0)
		extraIndex := -1
		for i, h := range inputs {
			got, e := m.GetRefCount(h)
			astTestMust(t, e)
			if got < before[i]+expected[i] {
				t.Fatal("published reference omitted", after, i, got, before[i], expected[i])
			}
			delta := got - before[i] - expected[i]
			extra += delta
			if delta != 0 {
				extraIndex = i
			}
		}
		if extra > 1 || (err == nil && extra != 0) {
			t.Fatal("temporary reference retained", after, err, extra)
		}
		outer := out.ownerBytes + out.positionBytes + out.substitutionBytes
		if m.BudgetField.Used() != borrowed+m.Stats().AllocatedBytes+outer {
			t.Fatal("unaccounted owner before cleanup", after, m.BudgetField.Used(), borrowed+m.Stats().AllocatedBytes+outer)
		}
		if extra != 0 {
			var next ASTHandle
			if n < uint32(len(ps)) {
				next = ps[n]
			} else if len(out.substitutions) != 0 {
				index := len(out.substitutions) - 1
				if index < len(substs) {
					gotN, e := m.RefVectorSize(out.substitutions[index])
					astTestMust(t, e)
					wantN, e := m.RefVectorSize(substs[index])
					astTestMust(t, e)
					if gotN < wantN {
						next, e = m.RefVectorGet(substs[index], gotN)
						astTestMust(t, e)
					}
				}
			}
			if next.IsNull() || inputs[extraIndex] != next {
				t.Fatal("retained INC belongs to wrong action", after, inputs[extraIndex], next)
			}
			astTestMust(t, m.DecRef(inputs[extraIndex]))
		}
		astTestMust(t, out.Close())
		astTestMust(t, m.DelRefVector(&premises))
		astTestMust(t, m.DelRef(&conclusion))
	}
	if failures == 0 || successes == 0 {
		t.Fatal("missing predictable failure boundary", failures, successes)
	}
}

func TestASTProofHyperZeroPremisesAndDebugGuardOrder(t *testing.T) {
	m := astTestManager(t)
	f := m.external(m.trueApp)
	empty, e := m.NewRefVector()
	astTestMust(t, e)
	defer m.DelRefVector(&empty)
	p, e := m.MkProofHyperResolve(nil, f, nil, []ASTRefVector{empty})
	astTestMust(t, e)
	proofTestFact(t, m, p, f)
	out, e := NewASTProofHyperOutput(m)
	astTestMust(t, e)
	defer out.Close()
	premises, e := m.NewRefVector()
	astTestMust(t, e)
	defer m.DelRefVector(&premises)
	conclusion, e := m.NewRef(ASTHandle{})
	astTestMust(t, e)
	defer m.DelRef(&conclusion)
	yes, e := m.MatchProofHyperResolve(p, premises, conclusion, out)
	astTestMust(t, e)
	n, e := m.RefVectorSize(premises)
	astTestMust(t, e)
	if !yes || n != 0 || out.SubstitutionsLen() != 1 {
		t.Fatal("source zero-premise app and initial empty substitution")
	}
	if ASTSourceDebug {
		ps, _, _ := hyperTestInputs(t, m)
		before := m.BudgetField.Used()
		m.BudgetField.FailAfter(0)
		_, e = m.MkProofHyperResolve(ps, f, nil, nil)
		m.BudgetField.FailAfter(-1)
		if !errors.Is(e, ErrASTDomain) || m.BudgetField.Used() != before {
			t.Fatal("position count assertion precedes fmls allocation", e)
		}
	}
}
