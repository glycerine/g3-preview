// Source: src/smt/dyn_ack.cpp clause construction, context integration, and
// clause-producing propagation.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
)

type smtDynAckManager struct {
	state      *SMTInternalizerState
	pairs      *dynAckPairState
	triples    *dynAckTripleState
	schedule   *dynAckSchedule
	strategy   dynAckStrategy
	equalities bool
	closed     bool
}

func newSMTDynAckManager(state *SMTInternalizerState, config SMTDynAckState) (*smtDynAckManager, error) {
	if state == nil || state.manager == nil || !validDynAckStrategy(dynAckStrategy(config.Strategy)) {
		return nil, ast.ErrASTDomain
	}
	pairs, err := newDynAckPairState(state.manager, config.Threshold, config.GCInverseDecay)
	if err != nil {
		return nil, err
	}
	triples, err := newDynAckTripleState(state.manager, config.Threshold)
	if err != nil {
		return nil, errors.Join(err, pairs.close())
	}
	schedule, err := newDynAckSchedule(pairs, triples, config.GC, config.Factor)
	if err != nil {
		return nil, errors.Join(err, pairs.close(), triples.close())
	}
	return &smtDynAckManager{state: state, pairs: pairs, triples: triples, schedule: schedule, strategy: dynAckStrategy(config.Strategy), equalities: config.Equalities}, nil
}

func defaultSMTDynAckState() SMTDynAckState {
	return SMTDynAckState{Strategy: uint32(dynAckRoot), Factor: 0.1, Threshold: 10, GC: 2000, GCInverseDecay: 0.8}
}

func (m *smtDynAckManager) equalityLiteral(first, second ast.ASTHandle) (SMTLiteral, error) {
	equality, err := m.state.manager.MkEq(first, second)
	if err != nil {
		return SMTNullLiteral, err
	}
	if err = m.state.Internalize(equality, true); err != nil {
		return SMTNullLiteral, err
	}
	return m.state.Literal(equality)
}

func (m *smtDynAckManager) instantiatePair(key dynAckPair) error {
	leftCount, err := m.state.manager.GetNumArgs(key.first)
	if err != nil {
		return err
	}
	rightCount, err := m.state.manager.GetNumArgs(key.second)
	if err != nil || leftCount == 0 || leftCount != rightCount {
		return ast.ErrASTDomain
	}
	literals := make([]SMTLiteral, 0, leftCount+1)
	for index := uint32(0); index < leftCount; index++ {
		left, err := m.state.manager.GetArg(key.first, index)
		if err != nil {
			return err
		}
		right, err := m.state.manager.GetArg(key.second, index)
		if err != nil {
			return err
		}
		if left == right {
			continue
		}
		literal, err := m.equalityLiteral(left, right)
		if err != nil {
			return err
		}
		literals = append(literals, literal.Negated())
	}
	conclusion, err := m.equalityLiteral(key.first, key.second)
	if err != nil {
		return err
	}
	literals = append(literals, conclusion)
	if len(literals) < 2 {
		return ast.ErrASTDomain
	}
	if err = m.pairs.markInstantiated(key); err != nil {
		return err
	}
	if err = m.state.addClauseKind(literals, false, SMTClauseAux, ast.ASTHandle{}); err != nil {
		return err
	}
	m.state.statistics.NumDynAck++
	return nil
}

func (m *smtDynAckManager) instantiateTriple(key dynAckTriple) error {
	firstRepresentative, err := m.equalityLiteral(key.first, key.representative)
	if err != nil {
		return err
	}
	secondRepresentative, err := m.equalityLiteral(key.second, key.representative)
	if err != nil {
		return err
	}
	conclusion, err := m.equalityLiteral(key.first, key.second)
	if err != nil {
		return err
	}
	if err = m.triples.markInstantiated(key); err != nil {
		return err
	}
	if err = m.state.addClauseKind([]SMTLiteral{firstRepresentative.Negated(), secondRepresentative.Negated(), conclusion}, false, SMTClauseAux, ast.ASTHandle{}); err != nil {
		return err
	}
	m.state.statistics.NumDynAck++
	return nil
}

func (m *smtDynAckManager) propagate() (bool, error) {
	if m == nil || m.closed || m.state == nil {
		return false, ast.ErrASTHandle
	}
	candidates, err := m.schedule.propagate(m.state.statistics.NumConflicts, m.strategy)
	if err != nil {
		return false, err
	}
	for _, candidate := range candidates {
		if candidate.isPair {
			err = m.instantiatePair(candidate.pair)
		} else {
			err = m.instantiateTriple(candidate.triple)
		}
		if err != nil {
			return false, err
		}
	}
	return len(candidates) != 0, nil
}

func (m *smtDynAckManager) recordConflictUses(expander *SMTAntecedentExpander, conflict SMTBooleanJustification) error {
	if m == nil || m.closed || expander == nil {
		return ast.ErrASTHandle
	}
	for _, use := range expander.CongruenceUses() {
		if err := m.schedule.usedCongruence(use.First.owner, use.Second.owner, m.strategy); err != nil {
			return err
		}
	}
	if m.equalities {
		for _, use := range expander.EqualityUses() {
			if err := m.schedule.usedEquality(use.First.owner, use.Second.owner, use.Common.owner, true); err != nil {
				return err
			}
		}
	}
	if object, ok := conflict.Justification(); ok {
		switch value := object.(type) {
		case *SMTDisequalityConflictJustification:
			if value.Equality.Kind() == SMTEqualityJustificationCongruence {
				if err := m.schedule.conflictCongruence(value.First.owner, value.Second.owner, m.strategy); err != nil {
					return err
				}
			}
		case *SMTEqualityConflictJustification:
			if value.Equality.Kind() == SMTEqualityJustificationCongruence {
				if err := m.schedule.conflictCongruence(value.First.owner, value.Second.owner, m.strategy); err != nil {
					return err
				}
			}
		}
	}
	expander.ResetUses()
	return nil
}

func (m *smtDynAckManager) close() error {
	if m == nil || m.closed {
		return nil
	}
	err := errors.Join(m.pairs.close(), m.triples.close())
	if err == nil {
		m.state, m.schedule, m.pairs, m.triples = nil, nil, nil, nil
		m.closed = true
	}
	return err
}
