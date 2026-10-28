package smtcore

// Source: src/opt/opt_solver.cpp objective registration and arithmetic-theory
// dispatch; src/smt/theory_opt.h.

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

type smtArithmeticObjectiveTheory interface {
	AddObjective(ast.ASTHandle) (SMTTheoryVar, error)
	ObjectiveValue(SMTTheoryVar) (SMTArithmeticObjectiveValue, error)
	MaximizeObjective(SMTTheoryVar) (SMTArithmeticObjectiveValue, ast.ASTHandle, bool, error)
}

func (s *SMTSolver) arithmeticObjectiveTheory() (smtArithmeticObjectiveTheory, error) {
	if err := s.open(); err != nil {
		return nil, err
	}
	family, err := s.manager.ArithmeticFamilyID()
	if err != nil {
		return nil, err
	}
	theory, ok := s.kernel.internalizer.theories[family].(smtArithmeticObjectiveTheory)
	if !ok || theory == nil {
		return nil, ast.ErrASTDomain
	}
	return theory, nil
}

// AddObjective registers an arithmetic objective with the concrete arithmetic
// theory selected by this solver. The returned index is stable until
// ResetObjectives. The solver retains the term independently of its caller.
func (s *SMTSolver) AddObjective(term ast.ASTHandle) (uint32, error) {
	theory, err := s.arithmeticObjectiveTheory()
	if err != nil {
		return 0, err
	}
	variable, err := theory.AddObjective(term)
	if err != nil {
		return 0, err
	}
	if variable == SMTNullTheoryVar {
		return 0, ast.ErrASTDomain
	}
	ref, err := s.manager.NewRef(term)
	if err != nil {
		return 0, err
	}
	index := uint32(len(s.objectives))
	s.objectives = append(s.objectives, smtSolverObjective{term: ref, theory: theory, variable: variable})
	return index, nil
}

func (s *SMTSolver) objective(index uint32) (*smtSolverObjective, error) {
	if err := s.open(); err != nil {
		return nil, err
	}
	if uint64(index) >= uint64(len(s.objectives)) {
		return nil, arena.ErrBounds
	}
	return &s.objectives[index], nil
}

// ObjectiveValue returns the concrete theory's current exact assignment.
func (s *SMTSolver) ObjectiveValue(index uint32) (SMTArithmeticObjectiveValue, error) {
	objective, err := s.objective(index)
	if err != nil {
		return SMTArithmeticObjectiveValue{}, err
	}
	return objective.theory.ObjectiveValue(objective.variable)
}

// MaximizeObjective moves the current arithmetic assignment to a local exact
// maximum and returns the source blocker used by the outer optimization loop.
// The blocker is borrowed and must be rooted before another manager operation.
func (s *SMTSolver) MaximizeObjective(index uint32) (SMTArithmeticObjectiveValue, ast.ASTHandle, bool, error) {
	objective, err := s.objective(index)
	if err != nil {
		return SMTArithmeticObjectiveValue{}, ast.ASTHandle{}, false, err
	}
	return objective.theory.MaximizeObjective(objective.variable)
}

// ResetObjectives releases every objective term retained by this solver.
func (s *SMTSolver) ResetObjectives() error {
	if err := s.open(); err != nil {
		return err
	}
	for len(s.objectives) != 0 {
		last := len(s.objectives) - 1
		if err := s.manager.DelRef(&s.objectives[last].term); err != nil {
			return err
		}
		s.objectives = s.objectives[:last]
	}
	return nil
}
