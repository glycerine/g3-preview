// Source: src/smt/smt_quantifier.cpp quantifier_manager propagation, queue
// instantiation, quick checking, final-check arbitration, and scope lifecycle.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
	"github.com/glycerine/g3/pkg/util/lbool"
)

// SMTQuantifierManager composes the concrete quantifier components owned by
// the Go SMT context. Pattern matching and model-based checking add instances
// through Registry; this manager schedules and internalizes those instances.
type SMTQuantifierManager struct {
	state        *SMTInternalizerState
	registry     *SMTQuantifierRegistry
	rewriter     *rewriter.TheoryRewriter
	checker      *SMTChecker
	quick        *SMTQuickChecker
	substitute   *SMTCachedVarSubstitution
	builder      *SMTQIInstanceBuilder
	promoteUnsat bool
	quickUnsat   bool
	quickNotSat  bool
	lazyQuick    bool
	closed       bool
}

type SMTQuantifierManagerConfig struct {
	PromoteUnsat bool
	QuickUnsat   bool
	QuickNotSat  bool
	LazyQuick    bool
}

func NewSMTQuantifierManager(state *SMTInternalizerState, rw *rewriter.TheoryRewriter, config SMTQuantifierManagerConfig) (*SMTQuantifierManager, error) {
	if state == nil || state.manager == nil || state.quantifiers == nil || rw == nil || rw.Manager() != state.manager {
		return nil, ast.ErrASTHandle
	}
	if state.quantifierManager != nil {
		return nil, ast.ErrASTDomain
	}
	checker, err := NewSMTChecker(state.context)
	if err != nil {
		return nil, err
	}
	quick, err := NewSMTQuickChecker(state.context, rw, state.quantifiers)
	if err != nil {
		return nil, err
	}
	substitute, err := NewSMTCachedVarSubstitution(state.manager)
	if err != nil {
		return nil, err
	}
	builder, err := NewSMTQIInstanceBuilder(state.manager, checker, substitute, state.quantifiers.costs)
	if err != nil {
		return nil, errors.Join(err, substitute.Close())
	}
	result := &SMTQuantifierManager{
		state: state, registry: state.quantifiers, rewriter: rw, checker: checker, quick: quick,
		substitute: substitute, builder: builder, promoteUnsat: config.PromoteUnsat,
		quickUnsat: config.QuickUnsat, quickNotSat: config.QuickNotSat, lazyQuick: config.LazyQuick,
	}
	state.quantifierManager = result
	return result, nil
}

func (q *SMTQuantifierManager) open() error {
	if q == nil || q.state == nil || q.registry == nil || q.builder == nil || q.closed {
		return ast.ErrASTHandle
	}
	return nil
}

func (q *SMTQuantifierManager) InitSearch() error {
	if err := q.open(); err != nil {
		return err
	}
	if err := q.substitute.Reset(); err != nil {
		return err
	}
	return q.registry.InitSearch()
}

func (q *SMTQuantifierManager) CanPropagate() bool {
	return q != nil && !q.closed && q.registry != nil && q.registry.schedule.HasWork()
}

func (q *SMTQuantifierManager) promotedEntries() (map[*SMTFingerprint]bool, error) {
	promoted := make(map[*SMTFingerprint]bool)
	if !q.promoteUnsat {
		return promoted, nil
	}
	for _, entry := range q.registry.schedule.newEntries {
		if entry.Fingerprint == nil {
			return nil, ast.ErrASTDomain
		}
		body, err := q.state.manager.GetQuantifierBody(entry.Fingerprint.data)
		if err != nil {
			return nil, err
		}
		unsat, err := q.checker.IsUnsatisfiable(body, entry.Fingerprint.args)
		if err != nil {
			return nil, err
		}
		if unsat {
			promoted[entry.Fingerprint] = true
		}
	}
	return promoted, nil
}

func (q *SMTQuantifierManager) instantiate(entries []SMTQIQueueEntry) (resultErr error) {
	for _, entry := range entries {
		prepared, err := q.builder.Prepare(entry)
		if err != nil {
			return err
		}
		instance, err := q.builder.MaterializeAndInternalize(prepared, q.rewriter, q.registry.statistics[entry.Fingerprint.data], q.state.scopeLevel, q.state)
		err = errors.Join(err, prepared.Close())
		if err != nil {
			return err
		}
		if instance == nil {
			continue
		}
		if !instance.ReducedTrue() {
			lemma, lemmaErr := instance.Lemma()
			if lemmaErr == nil {
				lemmaErr = q.registry.schedule.RecordInstance(lemma)
			}
			if lemmaErr != nil {
				return errors.Join(lemmaErr, instance.Close())
			}
		}
		if err = instance.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (q *SMTQuantifierManager) Propagate() error {
	if err := q.open(); err != nil {
		return err
	}
	promoted, err := q.promotedEntries()
	if err != nil {
		return err
	}
	entries, err := q.registry.schedule.ScheduleNew(promoted)
	if err != nil {
		return err
	}
	return q.instantiate(entries)
}

func (q *SMTQuantifierManager) active(quantifier ast.ASTHandle) bool {
	return q.state.context.relevant[quantifier] && q.state.context.assignments[quantifier] == lbool.True
}

func (q *SMTQuantifierManager) quickCheck() (bool, error) {
	if !q.quickUnsat && !q.quickNotSat {
		return true, nil
	}
	result := true
	for _, quantifier := range q.registry.quantifiers {
		if !q.active(quantifier) {
			continue
		}
		generated, err := q.quick.InstantiateUnsat(quantifier)
		if err != nil {
			return false, err
		}
		if generated {
			result = false
		}
	}
	if !result || !q.quickNotSat {
		if err := q.Propagate(); err != nil {
			return false, err
		}
		return result, nil
	}
	for _, quantifier := range q.registry.quantifiers {
		if !q.active(quantifier) {
			continue
		}
		generated, err := q.quick.InstantiateNotSat(quantifier)
		if err != nil {
			return false, err
		}
		if generated {
			result = false
		}
	}
	if err := q.Propagate(); err != nil {
		return false, err
	}
	return result, nil
}

func (q *SMTQuantifierManager) FinalCheck(full bool) (SMTFinalCheckStatus, error) {
	if err := q.open(); err != nil {
		return SMTFinalCheckGiveUp, err
	}
	if !full {
		return SMTFinalCheckDone, nil
	}
	done, entries, err := q.registry.schedule.FinalCheck()
	if err != nil {
		return SMTFinalCheckGiveUp, err
	}
	if err = q.instantiate(entries); err != nil {
		return SMTFinalCheckGiveUp, err
	}
	if !done {
		return SMTFinalCheckContinue, nil
	}
	if !q.lazyQuick {
		done, err = q.quickCheck()
		if err != nil {
			return SMTFinalCheckGiveUp, err
		}
		if !done {
			return SMTFinalCheckContinue, nil
		}
	}
	return SMTFinalCheckDone, nil
}

func (q *SMTQuantifierManager) Close() error {
	if q == nil || q.state == nil {
		return ast.ErrASTHandle
	}
	if q.closed {
		return nil
	}
	err := q.substitute.Close()
	if err == nil {
		if q.state.quantifierManager == q {
			q.state.quantifierManager = nil
		}
		q.closed = true
		q.builder = nil
		q.quick = nil
		q.checker = nil
		q.rewriter = nil
		q.registry = nil
		q.state = nil
	}
	return err
}
