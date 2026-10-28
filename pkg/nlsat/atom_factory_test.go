package nlsat

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func atomFactoryFixture(t *testing.T) (*Evaluator, *atomFactory) {
	t.Helper()
	return atomFactoryFixtureWithColdAllocator(t, false)
}

func atomFactoryFixtureWithColdAllocator(t *testing.T, cold bool) (*Evaluator, *atomFactory) {
	t.Helper()
	e := evaluatorFixture(t)
	allocator := e.ism.allocator
	if cold {
		var err error
		allocator, err = umemory.NewSmallAllocator(umemory.SmallAllocatorConfig{Runtime: allocator.RuntimeOwner(), Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: diagnostic.DebugBuild}})
		assignmentMust(t, err)
		t.Cleanup(func() { assignmentMust(t, allocator.Close()) })
	}
	pc, err := polynomial.NewPolynomialCache(e.pm)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, pc.Close()) })
	f, err := newAtomFactory(e.pm, pc, allocator)
	if f != nil {
		t.Cleanup(func() {
			assignmentMust(t, f.cleanup())
			for _, a := range f.bools.atoms {
				if a != nil {
					assignmentMust(t, f.deleteAtom(a))
				}
			}
			assignmentMust(t, f.close())
		})
	}
	assignmentMust(t, err)
	return e, f
}

func TestBooleanVariablesIndependentReuseAndParallelArrays(t *testing.T) {
	_, f := atomFactoryFixture(t)
	v := &f.bools
	random := rand.New(rand.NewPCG(81, 206))
	var free []sat.BoolVar
	var live []sat.BoolVar
	next := sat.BoolVar(0)
	for step := 0; step < 2000; step++ {
		if len(live) != 0 && random.IntN(3) == 0 {
			index := random.IntN(len(live))
			b := live[index]
			v.values[b] = lbool.True
			v.levels[b] = 91
			v.justifications[b], _ = NewDecisionJustification()
			assignmentMust(t, v.erase(b))
			if !v.dead[b] || v.atoms[b] != nil || v.values[b] != lbool.True || v.levels[b] != 91 || !v.justifications[b].IsDecision() {
				t.Fatal("deletion changed fields outside its source prefix", step, b)
			}
			free = append(free, b)
			live = append(live[:index], live[index+1:]...)
		} else {
			want := next
			if len(free) != 0 {
				want = free[len(free)-1]
				free = free[:len(free)-1]
			} else {
				next++
			}
			b, err := f.mkBoolVar()
			assignmentMust(t, err)
			if b != want || v.dead[b] || v.atoms[b] != nil || v.values[b] != lbool.Undef || v.levels[b] != ^uint32(0) || !v.justifications[b].IsNull() || len(v.watches[b]) != 0 {
				t.Fatal("new or recycled variable differs from independent model", step, b, want)
			}
			live = append(live, b)
		}
		if int(v.count) != len(live) || len(v.atoms) != int(next) || len(v.values) != int(next) || len(v.levels) != int(next) || len(v.justifications) != int(next) || len(v.watches) != int(next) || len(v.dead) != int(next) {
			t.Fatal("parallel extent/live count", step)
		}
	}
}

func TestAtomFactoryOuterRegistrationCanonicalRootsAndReferences(t *testing.T) {
	e, f := atomFactoryFixture(t)
	zero, err := f.mkBoolVar()
	assignmentMust(t, err)
	if zero != TrueBooleanVariable {
		t.Fatal("source true-variable slot")
	}
	for _, k := range []AtomKind{AtomEQ, AtomLT, AtomGT} {
		before, used := f.bools.count, f.ineq.budget.Used()
		literal, err := f.mkInequalityLiteral(k, nil, nil)
		assignmentMust(t, err)
		want := FalseLiteral
		if k == AtomGT {
			want = TrueLiteral
		}
		if literal != want || f.bools.count != before || f.ineq.budget.Used() != used {
			t.Fatal("empty product constructed an atom", k, literal)
		}
	}
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 2}, evaluatorTerm{-2, 0, 0})
	copy := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 2}, evaluatorTerm{-2, 0, 0})
	negative := evaluatorPolynomial(t, e, evaluatorTerm{-1, 0, 2}, evaluatorTerm{2, 0, 0})
	b, err := f.mkInequality(AtomGT, []polynomial.Polynomial{p}, []bool{false})
	assignmentMust(t, err)
	if b != 1 || f.bools.count != 2 || arithmeticAtomHeader(f.bools.atoms[b]).boolVar != b {
		t.Fatal("outer inequality registration")
	}
	again, err := f.mkInequality(AtomLT, []polynomial.Polynomial{negative}, []bool{false})
	assignmentMust(t, err)
	if again != b || f.bools.count != 2 {
		t.Fatal("duplicate inequality allocated a Boolean slot")
	}
	for _, kind := range []AtomKind{AtomRootEQ, AtomRootLT, AtomRootGT, AtomRootLE, AtomRootGE} {
		for index := uint32(1); index <= 3; index++ {
			r, err := f.mkRoot(kind, 1, index, p)
			assignmentMust(t, err)
			a := f.bools.atoms[r].(*RootAtom)
			if a.Kind() != kind || a.Index() != index || a.BooleanVariable() != r || a.RefCount() != 0 {
				t.Fatal("root header publication")
			}
			assignmentMust(t, f.incRef(r))
			before, err := e.core.Info(a.Polynomial())
			assignmentMust(t, err)
			count := f.bools.count
			for _, duplicate := range []polynomial.Polynomial{copy, negative, p} {
				found, err := f.mkRoot(kind, 1, index, duplicate)
				assignmentMust(t, err)
				after, err := e.core.Info(a.Polynomial())
				assignmentMust(t, err)
				if found != r || f.bools.count != count || a.RefCount() != 1 || after.RefCount != before.RefCount {
					t.Fatal("duplicate root changed identity, kind, slot, or references")
				}
			}
			assignmentMust(t, f.decRef(r))
			if !f.bools.dead[r] || f.bools.atoms[r] != nil || f.ownedRoots[a] != nil {
				t.Fatal("last DEC did not retire atom and Boolean slot")
			}
			reused, err := f.mkBoolVar()
			assignmentMust(t, err)
			if reused != r {
				t.Fatal("root deletion did not recycle Boolean ID", reused, r)
			}
			assignmentMust(t, f.bools.erase(reused))
			reborn, err := f.mkRoot(kind, 1, index, p)
			assignmentMust(t, err)
			if f.bools.dead[reborn] || f.bools.atoms[reborn] == nil || f.ownedRoots[f.bools.atoms[reborn].(*RootAtom)] == nil {
				t.Fatal("canonical table retained a retired root")
			}
			assignmentMust(t, f.deleteAtom(f.bools.atoms[reborn]))
		}
	}
	assignmentMust(t, f.incRef(sat.NullBoolVar))
	assignmentMust(t, f.decRef(sat.NullBoolVar))
	assignmentMust(t, f.incRef(zero))
	assignmentMust(t, f.decRef(zero))
}

func TestAtomFactoryWatchedRetirementStopsBeforeBooleanAndFactorRelease(t *testing.T) {
	atomDebugFixture(t)
	e, f := atomFactoryFixture(t)
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1})
	for _, root := range []bool{false, true} {
		var b sat.BoolVar
		var err error
		if root {
			b, err = f.mkRoot(AtomRootGT, 1, 1, p)
		} else {
			b, err = f.mkInequality(AtomGT, []polynomial.Polynomial{p}, []bool{false})
		}
		assignmentMust(t, err)
		a := f.bools.atoms[b]
		var watch Clause
		assignmentMust(t, watch.initialize(19, []sat.Literal{sat.MakeLiteral(b, false)}, false, nil, make([]sat.Literal, 1)))
		f.bools.watches[b] = []*Clause{&watch}
		before, err := e.core.Info(p)
		assignmentMust(t, err)
		count := f.bools.count
		err = f.deleteAtom(a)
		want := ErrAtomCacheLive
		if diagnostic.DebugBuild {
			want = diagnostic.ErrDebugAssertion
		}
		if !errors.Is(err, want) || f.retiring != a || f.retireStage != 1 || f.bools.dead[b] || f.bools.atoms[b] != a || f.bools.count != count {
			t.Fatal("watch guard lost reached retirement prefix", root, err)
		}
		after, err := e.core.Info(p)
		assignmentMust(t, err)
		if after.RefCount != before.RefCount {
			t.Fatal("watch guard retired polynomial ownership")
		}
		if _, err = f.mkBoolVar(); err != ErrAtomCachePending {
			t.Fatal("accepted work during stopped retirement", err)
		}
		f.bools.watches[b] = nil // actual watcher is detached by its owner.
		assignmentMust(t, f.cleanup())
		if f.bools.count != count-1 || !f.bools.dead[b] || f.bools.atoms[b] != nil {
			t.Fatal("retirement continuation replayed or omitted Boolean deletion")
		}
	}
}

func TestAtomFactoryRootAllocationFrontiersCleanupAndReuse(t *testing.T) {
	success := false
	postRegistrationFailure := false
	for stop := int64(0); stop < 128; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			e, f := atomFactoryFixtureWithColdAllocator(t, true)
			p := evaluatorPolynomial(t, e, evaluatorTerm{-1, 0, 2}, evaluatorTerm{2, 0, 0})
			f.ineq.budget.FailAfter(stop)
			b, err := f.mkRoot(AtomRootLT, 1, 2, p)
			f.ineq.budget.FailAfter(-1)
			if err != nil && f.pendingRoot != nil && f.pendingRoot.atom.boolVar != sat.NullBoolVar {
				postRegistrationFailure = true
			}
			assignmentMust(t, e.pm.CleanupFlipSignIfLeadingMonomialNegative())
			assignmentMust(t, f.ineq.polynomials.CleanupUnique())
			assignmentMust(t, f.cleanup())
			var physical umemory.Failure
			if err != nil && !errors.Is(err, arena.ErrInjectedFailure) && !(errors.As(err, &physical) && physical.Kind == umemory.FailureMemout && physical.ErrorCode() == 101) {
				t.Fatal("unexpected root factory refusal", err)
			}
			if err != nil && (b != sat.NullBoolVar || len(f.ownedRoots) != 0 || f.bools.count != 0) {
				t.Fatal("failed root retained a published atom or Boolean slot", b, f.bools.count)
			}
			if f.rootFactor != nil || f.pendingRoot != nil || f.allocationFailed || f.failed {
				t.Fatal("root cleanup retained work")
			}
			retry, err := f.mkRoot(AtomRootLT, 1, 2, p)
			assignmentMust(t, err)
			again, err := f.mkRoot(AtomRootLT, 1, 2, p)
			assignmentMust(t, err)
			if again != retry {
				t.Fatal("root canonical reuse after cleanup")
			}
			success = b != sat.NullBoolVar
		})
		if success {
			break
		}
	}
	if !success || !postRegistrationFailure {
		t.Fatal("campaign did not reach success and a real post-registration refusal", success, postRegistrationFailure)
	}
}

func TestAtomFactoryRootHashCollisionAndOwnerGuards(t *testing.T) {
	e, f := atomFactoryFixture(t)
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 2}, evaluatorTerm{-2, 0, 0})
	b, err := f.mkRoot(AtomRootEQ, 1, 1, p)
	assignmentMust(t, err)
	c, err := f.mkRoot(AtomRootEQ, 1, 3, p)
	assignmentMust(t, err)
	a1, a3 := f.bools.atoms[b].(*RootAtom), f.bools.atoms[c].(*RootAtom)
	h1, err := a1.Hash(e.core)
	assignmentMust(t, err)
	h3, err := a3.Hash(e.core)
	assignmentMust(t, err)
	if h1 != h3 || b == c || a1.Equal(a3) || len(f.roots[h1]) != 2 {
		t.Fatal("source overlapping index/kind hash bits did not preserve distinct roots")
	}
	d, err := f.mkRoot(AtomRootEQ, 2, 1, p)
	assignmentMust(t, err)
	if d == b || f.bools.atoms[d].(*RootAtom).MaxVariable() != 2 {
		t.Fatal("root variable omitted from canonical identity")
	}
	copy := *f
	if _, err := copy.mkBoolVar(); err != ErrAtomCacheOwner {
		t.Fatal("copied owner accepted registration", err)
	}
	if err := f.close(); err != ErrAtomCacheLive {
		t.Fatal("closed live atom factory", err)
	}
	var foreign RootAtom
	assignmentMust(t, foreign.initialize(AtomRootEQ, 1, 1, p))
	if err := f.deleteAtom(&foreign); err != ErrAtomCacheOwner || f.retiring != nil || f.failed {
		t.Fatal("foreign atom refusal adopted a retirement frame", err)
	}
}

func TestAtomFactoryCleanupPreservesIndependentAllocatorPrefix(t *testing.T) {
	for _, root := range []bool{false, true} {
		t.Run(fmt.Sprint(root), func(t *testing.T) {
			// The command-context fixture uses non-TLS accounting, whose size
			// refusal precedes physical allocation. Use actual TLS services to
			// qualify the distinct retained-physical-block source branch.
			m := assumptionFixtureWithProfile(t, true)
			budget := m.budget
			q, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Memory: m.allocator.RuntimeOwner(), Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
			assignmentMust(t, err)
			t.Cleanup(func() { assignmentMust(t, q.Close()) })
			pm, err := polynomial.NewMultivariatePolynomialManager(polynomial.MultivariatePolynomialManagerConfig{Core: polynomial.MultivariatePolynomialConfig{Integer: q.Manager.Integers(), Rationals: q.Manager, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
			assignmentMust(t, err)
			t.Cleanup(func() { assignmentMust(t, pm.CloseChecked()) })
			core, err := pm.ArithmeticCore()
			assignmentMust(t, err)
			x, err := core.MkVar()
			assignmentMust(t, err)
			p, err := core.MkVariable(x, 1)
			assignmentMust(t, err)
			assignmentMust(t, core.IncRef(p))
			t.Cleanup(func() { assignmentMust(t, core.DecRef(p)) })
			pc, err := polynomial.NewPolynomialCache(pm)
			assignmentMust(t, err)
			t.Cleanup(func() { assignmentMust(t, pc.Close()) })
			f, err := newAtomFactory(pm, pc, m.allocator)
			assignmentMust(t, err)
			t.Cleanup(func() { assignmentMust(t, f.close()) })
			_, err = f.ineq.polynomials.MkUnique(p)
			assignmentMust(t, err)
			a := f.ineq.allocator
			runtime := a.RuntimeOwner()
			assignmentMust(t, runtime.SetMaxSize(1))
			var older umemory.SmallAllocatorAllocationOutcome
			_, started, err := a.AllocateWithOutcomeStarted(1000001, &older)
			if err == nil || !started || !a.MatchesPendingAllocationOutcome(&older) {
				t.Fatal("independent allocation did not qualify a pending physical prefix", err)
			}
			before, used := runtime.Stats(), f.ineq.budget.Used()
			if root {
				_, err = f.mkRoot(AtomRootLT, 1, 1, p)
			} else {
				_, err = f.mkInequality(AtomLT, []polynomial.Polynomial{p}, []bool{false})
			}
			if !errors.Is(err, umemory.ErrAllocatorPending) {
				t.Fatal("younger factory did not reach pending allocator", err)
			}
			assignmentMust(t, f.cleanup())
			if !a.MatchesPendingAllocationOutcome(&older) || runtime.Stats() != before || f.ineq.budget.Used() != used {
				t.Fatal("factory cleanup changed independently older physical work")
			}
			assignmentMust(t, a.CleanupPending())
			assignmentMust(t, runtime.SetMaxSize(0))
		})
	}
}

func TestBooleanVariablesRealOutOfMemorySuppressesRecycling(t *testing.T) {
	m := assumptionFixtureWithProfile(t, true)
	runtime := m.allocator.RuntimeOwner()
	var vars booleanVariables
	vars.initialize(runtime)
	b, err := vars.mk()
	assignmentMust(t, err)
	assignmentMust(t, runtime.SetMaxSize(1))
	var older umemory.SmallAllocatorAllocationOutcome
	_, started, err := m.allocator.AllocateWithOutcomeStarted(1000001, &older)
	if err == nil || !started || !runtime.State().OutOfMemory || !m.allocator.MatchesPendingAllocationOutcome(&older) {
		t.Fatal("source memory limit did not set the actual out-of-memory flag", err)
	}
	assignmentMust(t, m.allocator.CleanupPending())
	assignmentMust(t, runtime.SetMaxSize(0))
	assignmentMust(t, vars.erase(b))
	if vars.ids.FreeLen() != 0 || !vars.dead[b] || vars.count != 0 {
		t.Fatal("source out-of-memory recycle suppression changed deletion")
	}
	next, err := vars.mk()
	assignmentMust(t, err)
	if next != b+1 {
		t.Fatal("out-of-memory path recycled an ID", next, b)
	}
}

func TestAtomFactoryRootReachedAssertionsAndConstructorRefusals(t *testing.T) {
	atomDebugFixture(t)
	e, f := atomFactoryFixture(t)
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1})
	for _, test := range []struct {
		kind     AtomKind
		x, index uint32
	}{{AtomRootLT, 1, 0}, {AtomRootLT, 0, 1}, {AtomGT, 1, 1}} {
		used, count := f.ineq.budget.Used(), f.bools.count
		_, err := f.mkRoot(test.kind, test.x, test.index, p)
		want := arena.ErrConfig
		if diagnostic.DebugBuild {
			want = diagnostic.ErrDebugAssertion
		}
		if !errors.Is(err, want) || f.rootFactor != nil || f.pendingRoot != nil || f.failed || f.bools.count != count || f.ineq.budget.Used() != used {
			t.Fatal("root precondition progressed into construction", test, err)
		}
	}
	for stop := int64(0); stop <= 2; stop++ {
		used := f.ineq.budget.Used()
		f.ineq.budget.FailAfter(stop)
		partial, err := newAtomFactory(e.pm, f.ineq.polynomials, f.ineq.allocator)
		f.ineq.budget.FailAfter(-1)
		if stop < 2 && !errors.Is(err, arena.ErrInjectedFailure) {
			t.Fatal("constructor frontier did not refuse", stop, err)
		}
		if stop == 2 {
			assignmentMust(t, err)
		}
		if partial != nil {
			assignmentMust(t, partial.close())
			if _, err := partial.mkBoolVar(); err != arena.ErrClosed {
				t.Fatal("closed partial constructor accepted work", err)
			}
		}
		if f.ineq.budget.Used() != used {
			t.Fatal("constructor cleanup retained its accepted prefix", stop)
		}
	}
}
