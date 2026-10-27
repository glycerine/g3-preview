package smtcore

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	uheap "github.com/glycerine/g3/pkg/util/heap"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func TestSMTKernelQueueClassFrozenAcrossAllParameterModes(t *testing.T) {
	for initial := uint32(0); initial <= 6; initial++ {
		for current := uint32(0); current <= 6; current++ {
			t.Run(fmt.Sprintf("%d-to-%d", initial, current), func(t *testing.T) {
				m, cfg := smtModelGeneratorFixture(t)
				p, err := cfg.Context.NewValue()
				astTestMust(t, err)
				t.Cleanup(func() { astTestMust(t, cfg.Context.Del(&p)) })
				key := "auto_config"
				astTestMust(t, cfg.Context.SetBoolName(&p, &key, false))
				key = "case_split"
				astTestMust(t, cfg.Context.SetUIntName(&p, &key, initial))
				key = "phase_selection"
				astTestMust(t, cfg.Context.SetUIntName(&p, &key, 4))
				k := smtTestKernel(t, m)
				astTestMust(t, k.ConfigurePreprocessing(cfg.Context, cfg.Globals, &p))
				q := k.caseSplitQueue
				s := k.internalizer
				if q == nil || q.strategy != initial {
					t.Fatal("initial constructor class", q, initial)
				}
				heaps := func() [5]*uheap.Int {
					var aware *uheap.Int
					if s.theoryAwareBranching != nil {
						aware = s.theoryAwareBranching.heap
					}
					return [5]*uheap.Int{s.basicActivity, s.delayedActivity, s.relevanceActivity.delayed, s.relevanceGoal.heap, aware}
				}
				before := heaps()
				astTestMust(t, q.RecordConflict())
				s.random.Next()
				rng := s.random
				key = "case_split"
				astTestMust(t, cfg.Context.SetUIntName(&p, &key, current))
				key = "phase_selection"
				astTestMust(t, cfg.Context.SetUIntName(&p, &key, 1))
				key = "random_seed"
				astTestMust(t, cfg.Context.SetUIntName(&p, &key, 99))
				astTestMust(t, k.UpdateParams(&p))
				if k.caseSplitQueue != q || q.strategy != initial || s.caseSplitStrategy != initial || before != heaps() || q.phaseCounter != 1 || !q.cacheEnabled || q.phaseSelection != 1 || s.random != rng {
					t.Fatal("live parameters replaced queue, counters, heaps or RNG", initial, current, q.strategy, s.caseSplitStrategy, q.phaseCounter)
				}
			})
		}
	}
}

func TestSMTKernelPhaseCachePersistsAcrossSearchesAndScopes(t *testing.T) {
	m, cfg := smtModelGeneratorFixture(t)
	p, err := cfg.Context.NewValue()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, cfg.Context.Del(&p)) })
	key := "auto_config"
	astTestMust(t, cfg.Context.SetBoolName(&p, &key, false))
	key = "phase_selection"
	astTestMust(t, cfg.Context.SetUIntName(&p, &key, 4))
	k := smtTestKernel(t, m)
	astTestMust(t, k.ConfigurePreprocessing(cfg.Context, cfg.Globals, &p))
	q := k.caseSplitQueue
	for i := 0; i < 400; i++ {
		astTestMust(t, q.RecordConflict())
	}
	astTestMust(t, q.RecordConflict())
	if q.cacheEnabled || q.phaseCounter != 1 || !q.phaseDefault {
		t.Fatal("source on-to-off transition", q.cacheEnabled, q.phaseCounter, q.phaseDefault)
	}
	for _, wantCounter := range []uint32{1, 2} {
		status, err := k.Check()
		astTestMust(t, err)
		if status != lbool.True || k.caseSplitQueue != q || q.phaseCounter != wantCounter || q.cacheEnabled || q.phaseDefault {
			t.Fatal("new search must reset default only", status, q.phaseCounter, q.cacheEnabled, q.phaseDefault)
		}
		astTestMust(t, q.RecordConflict())
	}
	astTestMust(t, k.Push())
	astTestMust(t, q.RecordConflict())
	astTestMust(t, k.Pop(1))
	if q.phaseCounter != 4 || q.cacheEnabled {
		t.Fatal("user pop rewound context phase cache", q.phaseCounter)
	}
}

func TestSMTKernelRNGReseedsAtSetupButNotUpdateOrRepeatedCheckSetup(t *testing.T) {
	for _, pushFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(pushFirst), func(t *testing.T) {
			m, cfg := smtModelGeneratorFixture(t)
			p, err := cfg.Context.NewValue()
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, cfg.Context.Del(&p)) })
			key := "auto_config"
			astTestMust(t, cfg.Context.SetBoolName(&p, &key, false))
			k := smtTestKernel(t, m)
			astTestMust(t, k.ConfigurePreprocessing(cfg.Context, cfg.Globals, &p))
			s := k.internalizer
			s.random.Next()
			before := s.random
			key = "random_seed"
			astTestMust(t, cfg.Context.SetUIntName(&p, &key, 77))
			astTestMust(t, k.UpdateParams(&p))
			if s.random != before {
				t.Fatal("parameter update reseeded before component setup")
			}
			if pushFirst {
				astTestMust(t, k.Push())
				astTestMust(t, k.Pop(1))
			} else {
				astTestMust(t, k.configureCheck(false))
			}
			if s.random != *z3rand.New(77) || !k.booleanSetupSeeded {
				t.Fatal("source setup_components did not seed once")
			}
			s.random.Next()
			before = s.random
			key = "random_seed"
			astTestMust(t, cfg.Context.SetUIntName(&p, &key, 123))
			astTestMust(t, k.UpdateParams(&p))
			astTestMust(t, k.configureCheck(false))
			if s.random != before {
				t.Fatal("repeated check setup/update restarted context random stream")
			}
		})
	}
}

func TestSMTKernelFrozenTheoryAwareQueueStillUsesHintsAfterClassUpdate(t *testing.T) {
	m, cfg := smtModelGeneratorFixture(t)
	p, err := cfg.Context.NewValue()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, cfg.Context.Del(&p)) })
	key := "case_split"
	astTestMust(t, cfg.Context.SetUIntName(&p, &key, 6))
	k := smtTestKernel(t, m)
	astTestMust(t, k.ConfigurePreprocessing(cfg.Context, cfg.Globals, &p))
	a := smtRelevancyBool(t, m, "frozen-aware-a")
	b := smtRelevancyBool(t, m, "frozen-aware-b")
	for _, x := range []ast.ASTHandle{a, b} {
		astTestMust(t, k.internalizer.Internalize(x, true))
	}
	av, _ := k.internalizer.BoolVar(b)
	key = "case_split"
	astTestMust(t, cfg.Context.SetUIntName(&p, &key, 0))
	astTestMust(t, k.UpdateParams(&p))
	astTestMust(t, k.internalizer.AddTheoryAwareBranchingInfo(av, 10, lbool.True))
	got, phase, err := k.caseSplitQueue.NextCaseSplit()
	astTestMust(t, err)
	// Source returns successful selections before consulting its phase-hint map.
	// The priority must select b ahead of a; the default phase remains false.
	if got != av || phase != lbool.False || k.caseSplitQueue.strategy != 6 {
		t.Fatal("class update discarded retained theory hint consumer", got, av, phase)
	}
}

func TestSMTKernelQueueReadsBorrowedPhaseOwner(t *testing.T) {
	m, cfg := smtModelGeneratorFixture(t)
	p, err := cfg.Context.NewValue()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, cfg.Context.Del(&p)) })
	key := "case_split"
	astTestMust(t, cfg.Context.SetUIntName(&p, &key, 6))
	k := smtTestKernel(t, m)
	astTestMust(t, k.ConfigurePreprocessing(cfg.Context, cfg.Globals, &p))
	q := k.caseSplitQueue
	a := smtRelevancyBool(t, m, "borrowed-live-phase")
	astTestMust(t, k.internalizer.Internalize(a, true))
	av, _ := k.internalizer.BoolVar(a)
	key = "phase_selection"
	astTestMust(t, cfg.Context.SetUIntName(&p, &key, 1))
	key = "case_split"
	astTestMust(t, cfg.Context.SetUIntName(&p, &key, 0))
	// Update the actual borrowed owner without synchronizing kernel snapshots.
	astTestMust(t, k.smtParams.Update(&p))
	got, phase, err := q.NextCaseSplit()
	astTestMust(t, err)
	if got != av || phase != lbool.True || q.strategy != 6 {
		t.Fatal("queue missed live phase update or changed concrete class", got, phase, q.strategy)
	}
	// A new context selects the current class; the existing context retains 6.
	fresh := smtTestKernel(t, m)
	astTestMust(t, fresh.ConfigurePreprocessing(cfg.Context, cfg.Globals, &p))
	if fresh.caseSplitQueue.strategy != 0 {
		t.Fatal("fresh context used stale queue class")
	}
	k.smtParams.Running = true
	_, _, err = q.NextCaseSplit()
	k.smtParams.Running = false
	if err != uparams.ErrParamsPrecondition {
		t.Fatal("queue ignored stopped parameter owner", err)
	}
}
