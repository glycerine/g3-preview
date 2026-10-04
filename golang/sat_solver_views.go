// Source: src/sat/sat_solver.h:245-286,318-323.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2011-05-21.
package g3

import "github.com/glycerine/g3/golang/arena"

// These copyable nonowning values borrow the actual named solver member, not
// its payload. Each access resolves current backing, including after growth.
// The caller must keep the solver open; no allocation or retain is introduced.
type satSolverModelView struct{ owner *satSolver }
type satSolverCoreView struct{ owner *satSolver }

func (s *satSolver) borrowModel() (satSolverModelView, error) {
	if e := s.open(); e != nil {
		return satSolverModelView{}, e
	}
	return satSolverModelView{s}, nil
}
func (s *satSolver) borrowCore() (satSolverCoreView, error) {
	if e := s.open(); e != nil {
		return satSolverCoreView{}, e
	}
	return satSolverCoreView{s}, nil
}
func (v satSolverModelView) Size() (uint32, error) {
	if e := v.owner.open(); e != nil {
		return 0, e
	}
	return v.owner.model.size()
}
func (v satSolverModelView) Read(i satBoolVar) (lbool, error) {
	if e := v.owner.open(); e != nil {
		return lUndef, e
	}
	x, e := v.owner.model.read(uint32(i))
	return lbool(int32(x)), e
}
func (v satSolverCoreView) Size() (uint32, error) {
	if e := v.owner.open(); e != nil {
		return 0, e
	}
	return v.owner.conflict.core.size()
}
func (v satSolverCoreView) Read(i uint32) (satLiteral, error) {
	if e := v.owner.open(); e != nil {
		return 0, e
	}
	x, e := v.owner.conflict.core.read(i)
	return satLiteral(x), e
}
func (s *satSolver) sourceInconsistent() (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	return s.propagation.inconsistent, nil
}
func (s *satSolver) sourceNumVars() (uint32, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	return s.variables.level.size()
}
func (s *satSolver) sourceNumRestarts() (uint32, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	return s.gc.restarts, nil
}
func (s *satSolver) sourceIsExternal(v satBoolVar) (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	return s.variables.isExternal(v)
}
func (s *satSolver) sourceSetExternal(v satBoolVar) error {
	if e := s.open(); e != nil {
		return e
	}
	return s.variables.external.write(uint32(v), 1)
}
func (s *satSolver) sourceWasEliminated(v satBoolVar) (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	return s.variables.wasEliminated(v)
}
func (s *satSolver) sourceScopeLevel() (uint32, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	return s.variables.scopeLevel, nil
}
func (s *satSolver) sourceValueLiteral(l satLiteral) (lbool, error) {
	if e := s.open(); e != nil {
		return lUndef, e
	}
	return s.variables.valueLiteral(l)
}
func (s *satSolver) sourceValueVar(v satBoolVar) (lbool, error) {
	if e := s.open(); e != nil {
		return lUndef, e
	}
	return s.variables.valueVar(v)
}
func (s *satSolver) sourceLevelVar(v satBoolVar) (uint32, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	return s.variables.levelVar(v)
}
func (s *satSolver) sourceLevelLiteral(l satLiteral) (uint32, error) {
	return s.sourceLevelVar(l.variable())
}
func (s *satSolver) sourceInitTrailSize() (uint32, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	if s.variables.scopeLevel == 0 {
		return s.variables.trail.Size(s.variables.trailRef)
	}
	if len(s.lifecycle.scopes) == 0 {
		return 0, arena.ErrBounds
	}
	return s.lifecycle.scopes[0].trailLimit, nil
}
func (s *satSolver) sourceModelIsCurrent() (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	return s.variables.modelIsCurrent, nil
}
func (s *satSolver) sourceCanceled() (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	return !s.limit.inc(), nil
}

// These private concrete pointer borrows preserve member identity. Go has no
// const reference type; mutation through these pointers is outside this facade.
func (s *satSolver) borrowConfig() (*satMainConfig, error) {
	if e := s.open(); e != nil {
		return nil, e
	}
	return s.config, nil
}
func (s *satSolver) borrowModelConverter() (*satModelConverter, error) {
	if e := s.open(); e != nil {
		return nil, e
	}
	return s.converter, nil
}
