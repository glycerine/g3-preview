// Source: src/sat/sat_solver.cpp:34–74,2763–2771; src/sat/sat_solver.h:74–483; src/sat/sat_mus.cpp:28–30.
// Licensed under the MIT license. Source authors: Leonardo de Moura, 2011; Nikolaj Bjorner, 2014.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

type satSolverConfig struct {
	ParamsContext *ParamsContext
	Params        ParamsRef
	RLimit        *utilReslimit
	Budget        *arena.Budget
	Runtime       *satRuntime
}

// Named source members borrow the one actual variable/lifecycle/propagation
// graph. Cold contexts break constructor cycles without supplying completed
// source members, duplicated counters, or an alternative solver.
type satSolver struct {
	blockedCorrectionSets        uint32 // actual m_stats field; source has only reset/read sites.
	self                         *satSolver
	runtime                      *satRuntime
	context                      *ParamsContext
	limit                        *utilReslimit
	budget                       *arena.Budget
	config                       *satMainConfig
	clauses                      *satClauseAllocator
	variables                    *satSolverVariableState
	propagation                  *satSolverPropagationState
	lifecycle                    *satSolverClauseLifecycle
	analysis                     *satSolverConflictAnalysis
	dynamic                      *satSolverDynamicSubsumption
	conflict                     *satSolverConflictResolution
	decision                     *satSolverDecision
	cleaner                      *satCleaner
	converter                    *satModelConverter
	simplifier                   *satSimplifier
	scc                          *satSCC
	asymm                        *satAsymmBranch
	probing                      *satSolverProbing
	gc                           *satSolverGC
	model                        satSolverRuntimeUintColumn
	mus                          satSolverMUSMembers
	params                       ParamsRef
	elimTodo                     *satSolverUintSet
	antecedents                  *satSolverIndexMap
	auxLiterals, todoAntecedents satSolverRuntimeUintColumn
	userBinClauses               *satSolverBinPairVector
	binaryClauseGraph            []satSolverRuntimeUintColumn
	closeSourceCause             error
	closeGuardAcknowledged       bool
	userGC                       satSolverUserGC
	nextSimplify                 uint32
	mutexFrames                  *satMutexFrame
	mutexRunning                 bool
	consequenceFrames            *satConsequenceFrame
	consequenceRunning           bool
	ownerBytes                   uint64
	closeStage                   uint32
	constructed, closing, closed bool
}

// Actual source MUS value members and constructor fields. Algorithms and
// named interrupted lexical ownership live in sat_solver_mus.go; optional
// report/TRACE producers remain separate source obligations.
type satSolverMUSMembers struct {
	solver           *satSolver
	frames           *satMUSFrame
	core, mus, model satSolverRuntimeUintColumn
	isActive         bool
	maxNumRestarts   uint32
	constructed      bool
}

func NewSATSolver(c satSolverConfig) (s *satSolver, err error) {
	if c.Runtime == nil || c.ParamsContext == nil || c.Budget == nil || c.RLimit == nil || c.RLimit.closed {
		return nil, ErrSATOwner
	}
	if err = c.Runtime.open(); err != nil {
		return nil, err
	}
	if c.Runtime.context != c.ParamsContext || c.Runtime.budget != c.Budget || c.Runtime.memory.Budget() != c.Budget {
		return nil, ErrSATOwner
	}
	if err = c.ParamsContext.check(&c.Params); err != nil {
		return nil, err
	}
	if c.Runtime.liveSolvers == ^uint32(0) {
		return nil, arena.ErrSlotLimit
	}
	n := uint64(reflect.TypeOf(satSolver{}).Size())
	if err = c.Budget.Reserve(n); err != nil {
		return nil, err
	}
	s = &satSolver{runtime: c.Runtime, context: c.ParamsContext, limit: c.RLimit, budget: c.Budget, ownerBytes: n}
	s.self = s
	c.Runtime.liveSolvers++
	defer func() {
		if err != nil {
			if e := s.Close(); e != nil {
				err = errors.Join(err, e)
			} else {
				s = nil
			}
		}
	}()
	// Source m_config is the first owning member. Its symbol/parameter body is
	// unchanged, followed by source stats reset and default m_rand construction.
	if s.config, err = newSATMainConfig(c.Runtime.globals, &c.Params, c.Budget); err != nil {
		return s, err
	}
	if s.decision, err = newSATColdDecision(c.Budget); err != nil {
		return s, err
	}
	if s.clauses, err = NewSATClauseAllocator(satClauseAllocatorConfig{Storage: arena.SmallBytesConfig{Descriptors: arena.Config{Budget: c.Budget}}, IDRuntime: c.Runtime.memory}); err != nil {
		return s, err
	}
	// The following actual charged contexts contain empty bound vector headers.
	// They are Go ownership metadata, not prematurely constructed source queue,
	// MC payloads, probing parameters, or additional malloc accounting events.
	if s.elimTodo, err = newSATSolverUintSet(c.Budget); err != nil {
		return s, err
	}
	vc := arena.VectorConfig{Descriptors: arena.Config{Budget: c.Budget}}
	if s.variables, err = newSATColdVariableState(s.elimTodo, c.Budget, vc, c.Runtime.memory); err != nil {
		return s, err
	}
	if s.converter, err = newSATColdModelConverter(c.Budget); err != nil {
		return s, err
	}
	if s.propagation, err = newSATSolverPropagationState(s.variables, s.clauses, s.converter, c.RLimit, c.Runtime.memory, s.config, c.Budget); err != nil {
		return s, err
	}
	if s.lifecycle, err = newSATSolverClauseLifecycle(s.propagation, c.Budget); err != nil {
		return s, err
	}
	if s.analysis, err = newSATSolverConflictAnalysis(s.propagation, c.Budget); err != nil {
		return s, err
	}
	if s.probing, err = newSATColdProbing(s.lifecycle, c.Budget); err != nil {
		return s, err
	}
	if s.dynamic, err = newSATSolverDynamicSubsumption(s.analysis, s.probing, c.Budget); err != nil {
		return s, err
	}
	if s.conflict, err = newSATSolverConflictResolution(s.analysis, s.dynamic); err != nil {
		return s, err
	}
	s.decision.conflict = s.conflict
	// Declared source member construction resumes: cleaner, model, MC,
	// simplifier, SCC, asymmetric branching, probing, MUS.
	if s.cleaner, err = newSATCleaner(s.lifecycle, c.Budget); err != nil {
		return s, err
	}
	s.model.bind(c.Runtime.memory)
	if err = s.converter.initializeSourceStore(vc); err != nil {
		return s, err
	}
	if s.simplifier, err = newSATSimplifier(s.lifecycle, s.cleaner, s.decision, &c.Params); err != nil {
		return s, err
	}
	if s.scc, err = newSATSCC(s.lifecycle, &c.Params); err != nil {
		return s, err
	}
	if s.asymm, err = newSATAsymmBranch(s.lifecycle, &c.Params); err != nil {
		return s, err
	}
	if err = s.probing.initializeSource(c.ParamsContext, &c.Params); err != nil {
		return s, err
	}
	s.mus.core.bind(c.Runtime.memory)
	s.mus.mus.bind(c.Runtime.memory)
	s.mus.model.bind(c.Runtime.memory)
	s.conflict.solver = s
	s.mus.solver = s
	s.mus.isActive = false
	s.mus.maxNumRestarts = ^uint32(0)
	s.mus.constructed = true
	s.propagation.inconsistent = false
	// Empty clause headers and num_frozen precede watches/columns; the actual
	// m_case_split_queue heap initialization occurs only at its member site.
	if s.variables.watches, err = newSATSolverWatchLists(c.Budget, vc); err != nil {
		return s, err
	}
	if s.variables.activity, err = newSATSolverRuntimeActivity(c.Budget, c.Runtime.memory); err != nil {
		return s, err
	}
	s.variables.activityInc = 128
	if s.variables.queue, err = newSATSolverRuntimeVarQueue(s.variables.activity, c.Budget); err != nil {
		return s, err
	}
	s.variables.qhead = 0
	s.variables.scopeLevel = 0
	if s.variables.trail, s.variables.trailRef, err = newSATSolverRuntimeTrail(c.Budget, c.Runtime.memory); err != nil {
		return s, err
	}
	if s.params, err = c.ParamsContext.NewValue(); err != nil {
		return s, err
	}
	if err = c.ParamsContext.Assign(&s.params, &c.Params); err != nil {
		return s, err
	}
	// Remaining declared vectors are empty until reached by their algorithms.
	// The consequence antecedent map is nevertheless eagerly constructed.
	s.auxLiterals.bind(c.Runtime.memory)
	if s.userBinClauses, err = newSATSolverBinPairVector(s.runtime.memory, c.Budget); err != nil {
		return s, err
	}
	if s.antecedents, err = newSATSolverIndexMap(c.Runtime.memory, c.Budget); err != nil {
		return s, err
	}
	s.todoAntecedents.bind(c.Runtime.memory)
	// GC is charged concrete context for the actual already declared scalar
	// fields and invariant binding, not a second set of conflict counters.
	if s.gc, err = newSATSolverGC(s.conflict, s.cleaner); err != nil {
		return s, err
	}
	if err = s.updateParams(&c.Params); err != nil {
		return s, err
	}
	s.conflict.conflictsSinceGC = 0
	s.conflict.conflicts = 0
	s.nextSimplify = 0
	s.propagation.numCheckpoints = 0
	s.constructed = true
	return s, nil
}
func (s *satSolver) identity() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return arena.ErrClosed
	}
	return nil
}
func (s *satSolver) open() error {
	if e := s.identity(); e != nil {
		return e
	}
	if s.closing || !s.constructed || (s.mus.frames != nil && !s.mus.isActive) || (s.consequenceFrames != nil && !s.consequenceRunning) || (s.mutexFrames != nil && !s.mutexRunning) {
		return arena.ErrClosed
	}
	if e := s.runtime.open(); e != nil {
		return e
	}
	return s.lifecycle.open()
}
func (s *satSolver) updateParams(p *ParamsRef) error {
	// This helper is also the actual constructor body; no fabricated completed
	// solver gate precedes its reached parameter actions.
	if e := s.identity(); e != nil {
		return e
	}
	if s.closing {
		return arena.ErrClosed
	}
	if e := s.context.Assign(&s.params, p); e != nil {
		return e
	}
	if e := s.config.Update(p); e != nil {
		return e
	}
	if e := s.simplifier.Update(p); e != nil {
		return e
	}
	if e := s.asymm.Update(p); e != nil {
		return e
	}
	if e := s.probing.updateParams(s.context, p); e != nil {
		return e
	}
	if e := s.scc.Update(p); e != nil {
		return e
	}
	cfg, e := s.config.State()
	if e != nil {
		return e
	}
	s.decision.rng.SetSeed(cfg.RandomSeed)
	return nil
}
func (s *satSolver) UpdateParams(p *ParamsRef) error {
	if e := s.open(); e != nil {
		return e
	}
	return s.updateParams(p)
}

// Close first acknowledges explicitly retained unsafe-exception ownership,
// then executes the real destructor body (only for a completed constructor)
// and declaration-reverse member stages. Stopped release retains its stage and
// every later owner. A failed source ctor has no solver destructor body;
// partial cold contexts are separately owning-aborted, never declared complete.
// Close retains the actual source invariant boundary. Owning abort is a
// separate caller action, never a fabricated successful check_invariant.
func (s *satSolver) Close() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if !s.closed && s.closeSourceCause != nil && !s.closeGuardAcknowledged {
		return s.closeSourceCause
	}
	return s.closeMembers()
}

// CleanupStopped acknowledges ONLY a previously observed source destructor
// invariant guard. The immutable cause/history remains visible, and every
// remaining real release still uses its original stopped action cursor.
func (s *satSolver) CleanupStopped() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closeSourceCause == nil {
		return ErrSATOwner
	}
	if !s.closeGuardAcknowledged {
		if s.closeStage != 1 {
			return ErrSATOwner
		}
		s.closeGuardAcknowledged = true
		s.closeStage++
	}
	return s.closeMembers()
}
func (s *satSolver) closeMembers() error {

	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return nil
	}
	s.closing = true
	for s.closeStage < 45 {
		var e error
		switch s.closeStage {
		case 0:
			if e = s.cleanupMutexes(); e != nil {
				break
			}
			if e = s.cleanupConsequences(); e != nil {
				break
			}
			if e = s.mus.cleanup(); e != nil {
				break
			}
			if s.lifecycle != nil {
				e = s.lifecycle.cleanupOwningContinuations()
			}
		case 1:
			if s.constructed && satStorageSourceDebug {
				e = s.lifecycle.checkInvariant()
				if e != nil {
					s.closeSourceCause = e
				}
			}
		case 2:
			if s.constructed {
				e = s.lifecycle.deleteClausesBody()
			}
		case 3:
			for i := len(s.binaryClauseGraph); i > 0; i-- {
				if e = s.binaryClauseGraph[i-1].finalize(); e != nil {
					break
				}
			}
			if e == nil {
				s.binaryClauseGraph = nil
			}
		case 4:
			if s.todoAntecedents.raw.self != nil {
				e = s.todoAntecedents.finalize()
			}
		case 5:
			if s.antecedents != nil {
				e = s.antecedents.Close()
			}
		case 6:
			if s.userBinClauses != nil {
				e = s.userBinClauses.Close()
			}
		case 7:
			if s.auxLiterals.raw.self != nil {
				e = s.auxLiterals.finalize()
			}
		case 8:
			if s.conflict != nil {
				e = s.conflict.userScopeLiterals.finalize()
			}
		case 9:
			if s.analysis != nil {
				e = s.analysis.minStack.finalize()
			}
		case 10:
			if s.analysis != nil {
				e = s.analysis.unmark.finalize()
			}
		case 11:
			if s.propagation != nil {
				s.propagation.diffLevels.finalize(s.budget)
			}
		case 12:
			if s.analysis != nil {
				e = s.analysis.extAntecedents.finalize()
			}
		case 13:
			if s.analysis != nil {
				e = s.analysis.lemma.finalize()
			}
		case 14:
			if s.conflict != nil {
				e = s.conflict.minCore.finalize()
			}
		case 15:
			if s.conflict != nil {
				e = s.conflict.core.finalize()
			}
		case 16:
			if s.conflict != nil {
				e = s.conflict.assumptionSet.Close()
			}
		case 17:
			if s.conflict != nil {
				e = s.conflict.assumptions.finalize()
			}
		case 18:
			if s.params.owner != 0 {
				e = s.context.Del(&s.params)
			}
		case 19:
			// The Go clock profile owns its running measurement; C++'s stopwatch
			// destructor has no source callback. Stop is an explicit clock adapter.
			if s.gc != nil {
				e = s.gc.watch.Stop()
			}
		case 20:
			if s.lifecycle != nil {
				e = s.lifecycle.finalizeScopesReinit()
			}
		case 21:
			if s.variables != nil {
				e = s.variables.finalizeTrail()
			}
		case 22:
			if s.variables != nil {
				e = s.variables.finalizeQueue()
			}
		case 23:
			if s.variables != nil {
				e = s.variables.finalizeColumnsWatches()
			}
		case 24:
			if s.lifecycle != nil {
				e = s.lifecycle.finalizeClauseVectors()
			}
		case 25:
			if s.mus.constructed {
				if e = s.mus.model.finalize(); e == nil {
					e = s.mus.mus.finalize()
				}
				if e == nil {
					e = s.mus.core.finalize()
				}
			}
		case 26:
			if s.probing != nil {
				e = s.probing.Close()
			}
		case 27:
			if s.asymm != nil {
				e = s.asymm.Close()
			}
		case 28:
			if s.scc != nil {
				e = s.scc.Close()
			}
		case 29:
			if s.simplifier != nil {
				e = s.simplifier.Close()
			}
		case 30:
			if s.converter != nil {
				e = s.converter.Close()
			}
		case 31:
			if s.model.raw.self != nil {
				e = s.model.finalize()
			}
		case 32:
			if s.cleaner != nil {
				e = s.cleaner.Close()
			}
		case 33:
			if s.clauses != nil {
				e = s.clauses.Close()
			}
		case 34:
			if s.decision != nil {
				e = s.decision.Close()
			}
		case 35:
			if s.config != nil {
				e = s.config.Close()
			}
		// Only actual Go cold-context headers remain after the source member
		// actions above. Their idempotent Close acknowledges already drained
		// fields and does not replay source body or allocation events.
		case 36:
			if s.gc != nil {
				e = s.gc.Close()
			}
		case 37:
			if s.conflict != nil {
				e = s.conflict.Close()
			}
		case 38:
			if s.dynamic != nil {
				e = s.dynamic.Close()
			}
		case 39:
			if s.analysis != nil {
				e = s.analysis.Close()
			}
		case 40:
			if s.propagation != nil {
				e = s.propagation.Close()
			}
		case 41:
			if s.lifecycle != nil {
				e = s.lifecycle.closeContext()
			}
		case 42:
			if s.variables != nil {
				e = s.variables.Close()
			}
		case 43:
			if s.elimTodo != nil {
				e = s.elimTodo.Close()
			}
		case 44:
			s.runtime.liveSolvers--
			s.budget.Release(s.ownerBytes)
			s.ownerBytes = 0
			s.closed = true
		}
		if e != nil {
			return e
		}
		s.closeStage++
	}
	return nil
}
