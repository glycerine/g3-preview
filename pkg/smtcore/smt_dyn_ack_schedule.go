// Source: src/smt/dyn_ack.cpp:118-138,315-340; src/smt/dyn_ack.h:49-69.
package smtcore

import (
	"math"

	"github.com/glycerine/g3/pkg/ast"
)

// dynAckCandidate preserves the source pair-before-triple propagation order.
// The enclosing SMT context still constructs each selected clause.
type dynAckCandidate struct {
	pair   dynAckPair
	triple dynAckTriple
	isPair bool
}

type dynAckSchedule struct {
	pairs        *dynAckPairState
	triples      *dynAckTripleState
	gcEvery      uint32
	factor       float64
	propagations uint32
	instances    uint32
}

func newDynAckSchedule(pairs *dynAckPairState, triples *dynAckTripleState, gcEvery uint32, factor float64) (*dynAckSchedule, error) {
	if pairs == nil || triples == nil || pairs.manager != triples.manager || math.IsNaN(factor) || math.IsInf(factor, 0) || factor < 0 {
		return nil, ast.ErrASTDomain
	}
	if e := pairs.check(); e != nil {
		return nil, e
	}
	if e := triples.check(); e != nil {
		return nil, e
	}
	return &dynAckSchedule{pairs: pairs, triples: triples, gcEvery: gcEvery, factor: factor}, nil
}

// propagate selects reached candidate records using the source cumulative
// conflict budget. The caller must construct clauses and invoke each state's
// markInstantiated transition for selected records in returned order.
func (s *dynAckSchedule) propagate(conflicts uint32, enabled bool) ([]dynAckCandidate, error) {
	if s == nil {
		return nil, ast.ErrASTHandle
	}
	if e := s.pairs.check(); e != nil {
		return nil, e
	}
	if e := s.triples.check(); e != nil {
		return nil, e
	}
	if !enabled {
		return nil, nil
	}
	s.propagations++
	if s.propagations > s.gcEvery {
		if e := s.pairs.gc(); e != nil {
			return nil, e
		}
		s.propagations = 0
	}
	budget := float64(conflicts) * s.factor
	if budget > math.MaxUint32 {
		return nil, ast.ErrASTDomain
	}
	maxInstances := uint32(budget) // source static_cast<unsigned> truncates
	var selected []dynAckCandidate
	for s.instances < maxInstances && s.pairs.queueHead < len(s.pairs.toInstantiate) {
		key := s.pairs.toInstantiate[s.pairs.queueHead]
		s.pairs.queueHead++
		s.instances++
		selected = append(selected, dynAckCandidate{pair: key, isPair: true})
	}
	for s.instances < maxInstances && s.triples.queueHead < len(s.triples.toInstantiate) {
		key := s.triples.toInstantiate[s.triples.queueHead]
		s.triples.queueHead++
		s.instances++
		selected = append(selected, dynAckCandidate{triple: key})
	}
	return selected, nil
}

// initSearch keeps instantiated-clause records, as the source hook does.
func (s *dynAckSchedule) initSearch() error {
	if s == nil {
		return ast.ErrASTHandle
	}
	if e := s.pairs.initSearch(); e != nil {
		return e
	}
	if e := s.triples.initSearch(); e != nil {
		return e
	}
	s.instances = 0
	s.propagations = 0
	return nil
}

func (s *dynAckSchedule) reset() error {
	if e := s.initSearch(); e != nil {
		return e
	}
	clear(s.pairs.instantiated)
	clear(s.triples.instantiated)
	return nil
}
