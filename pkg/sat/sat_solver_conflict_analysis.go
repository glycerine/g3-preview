package sat

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// The real conflict-analysis member vectors, borrowed from neither the trail nor
// a clause snapshot. Their source unsigned32 payloads belong to the propagation
// owner's actual Runtime. This cold grouping is not a solver/search constructor.
// Propagation, variable state, Runtime and clause allocator must outlive it.
type satSolverConflictAnalysis struct {
	self                                    *satSolverConflictAnalysis
	propagation                             *satSolverPropagationState
	budget                                  *arena.Budget
	lemma, extAntecedents, unmark, minStack RuntimeUintColumn
	levelSet                                satApproxSet
	minimizedLits                           uint32
	ownerBytes                              uint64
	closing, closed                         bool
}

func newSATSolverConflictAnalysis(p *satSolverPropagationState, b *arena.Budget) (*satSolverConflictAnalysis, error) {
	if e := p.constructionBorrow(); e != nil {
		return nil, e
	}
	if b != p.budget || b != p.memory.Budget() {
		return nil, ErrSATOwner
	}
	n := uint64(reflect.TypeOf(satSolverConflictAnalysis{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	s := &satSolverConflictAnalysis{propagation: p, budget: b, ownerBytes: n}
	s.self = s
	for _, c := range []*RuntimeUintColumn{&s.lemma, &s.extAntecedents, &s.unmark, &s.minStack} {
		c.Bind(p.memory)
	}
	return s, nil
}
func (s *satSolverConflictAnalysis) constructionBorrow() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed || s.closing {
		return arena.ErrClosed
	}
	return s.propagation.constructionBorrow()
}
func (s *satSolverConflictAnalysis) open() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed || s.closing {
		return arena.ErrClosed
	}
	return s.propagation.open()
}
func (s *satSolverConflictAnalysis) Close() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return nil
	}
	s.closing = true
	// Source reverse member declaration order: min-stack, unmark, external
	// antecedents, lemma. The scalar approximate set has no destructor action.
	// Never repair marks left by interrupted source minimization.
	for _, c := range []*RuntimeUintColumn{&s.minStack, &s.unmark, &s.extAntecedents, &s.lemma} {
		if e := c.Finalize(); e != nil {
			return e
		}
	}
	s.budget.Release(s.ownerBytes)
	s.ownerBytes = 0
	s.closed = true
	return nil
}

// Private source member helpers below require an already checked open owner.
// Their algorithm entry points call open before the first source action.
func (s *satSolverConflictAnalysis) marked(v uint32) (bool, error) {
	x, e := s.propagation.variables.mark.Read(v)
	return x != 0, e
}
func (s *satSolverConflictAnalysis) mark(v uint32) error {
	if satStorageSourceDebug {
		yes, e := s.marked(v)
		if e != nil {
			return e
		}
		if yes {
			return ErrSATAssertion
		}
	}
	return s.propagation.variables.mark.Write(v, 1)
}
func (s *satSolverConflictAnalysis) resetMark(v uint32) error {
	if satStorageSourceDebug {
		yes, e := s.marked(v)
		if e != nil {
			return e
		}
		if !yes {
			return ErrSATAssertion
		}
	}
	return s.propagation.variables.mark.Write(v, 0)
}
func (s *satSolverConflictAnalysis) fillExtAntecedents(consequent Literal, j satJustification) error {
	if satStorageSourceDebug && (!j.External() || !s.propagation.extensionPresent) {
		return ErrSATAssertion
	}
	if e := s.extAntecedents.Shrink(0); e != nil {
		return e
	}
	// No actual source extension subclass has been bound. The source call is
	// genuinely missing; retain its completed reset prefix, never fake success.
	return ErrSATMissingSourceExtBinding
}
