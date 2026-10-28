package opt

// Source: src/opt/opt_solver.cpp and src/opt/optsmt.cpp retained arithmetic
// objective loop.

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

// ArithmeticResult owns the best model retained by one optimization check.
// Values are expressed in the caller's original maximize/minimize orientation.
type ArithmeticResult struct {
	Status  lbool.LBool
	Values  []smtcore.SMTArithmeticObjectiveValue
	Model   *model.ModelCore
	Optimal bool
}

func (r *ArithmeticResult) Close() error {
	if r == nil || r.Model == nil {
		return nil
	}
	err := r.Model.DecRef()
	if err == nil {
		r.Model = nil
	}
	return err
}

// ArithmeticSolver is the first retained opt_solver path. It deliberately
// owns a dedicated SMT backend, as native optimization does, so temporary
// blockers cannot mutate the command context's ordinary incremental solver.
type ArithmeticSolver struct {
	backend    *smtcore.SMTSolver
	manager    *ast.ASTManager
	objectives []arithmeticObjective
	filter     *tactic.FilterModelConverter
	prepared   bool
}

type arithmeticObjective struct {
	term     ast.ASTRef
	maximize bool
}

func NewArithmeticSolver(m *ast.ASTManager, params *uparams.ParamsRef, logic string, config smtcore.SMTSolverConfig) (*ArithmeticSolver, error) {
	backend, err := smtcore.NewSMTSolver(m, params, logic, config)
	if err != nil {
		return nil, err
	}
	if err = backend.SetProduceModels(true); err != nil {
		return nil, errors.Join(err, backend.Close())
	}
	filter, err := tactic.NewFilterModel(m)
	if err != nil {
		return nil, errors.Join(err, backend.Close())
	}
	return &ArithmeticSolver{backend: backend, manager: m, filter: filter}, nil
}

func (s *ArithmeticSolver) Assert(formula ast.ASTHandle) error {
	if s == nil || s.backend == nil {
		return ast.ErrASTHandle
	}
	return s.backend.Assert(formula)
}

// AddObjective registers one lexicographic objective. Minimization uses the
// source normalization to maximization of the negated term.
func (s *ArithmeticSolver) AddObjective(term ast.ASTHandle, maximize bool) error {
	if s == nil || s.backend == nil {
		return ast.ErrASTHandle
	}
	// optsmt::setup forces the solver to its base level with an empty scoped
	// push. Besides committing hard assertions, this installs the logic-selected
	// arithmetic backend before objective internalization.
	if !s.prepared {
		if err := s.backend.Push(); err != nil {
			return err
		}
		if err := s.backend.Pop(1); err != nil {
			return err
		}
		s.prepared = true
	}
	objective := term
	var err error
	if !maximize {
		objective, err = s.manager.MkArithmeticNeg(term)
		if err != nil {
			return err
		}
	}
	ref, err := s.manager.NewRef(term)
	if err != nil {
		return err
	}
	if _, err = s.backend.AddObjective(objective); err != nil {
		return errors.Join(err, s.manager.DelRef(&ref))
	}
	s.objectives = append(s.objectives, arithmeticObjective{term: ref, maximize: maximize})
	return nil
}

func (s *ArithmeticSolver) replaceModel(result *ArithmeticResult) error {
	md, err := s.backend.AcquireModel()
	if err != nil || md == nil {
		return err
	}
	if result.Model != nil {
		if err = result.Model.DecRef(); err != nil {
			_ = md.DecRef()
			return err
		}
	}
	result.Model = md
	return nil
}

func (s *ArithmeticSolver) orient(index int, value smtcore.SMTArithmeticObjectiveValue) smtcore.SMTArithmeticObjectiveValue {
	if s.objectives[index].maximize || value.Unbounded {
		return value
	}
	value.Finite = value.Finite.Clone()
	value.Finite.Neg()
	return value
}

func (s *ArithmeticSolver) optimizeOne(index int, result *ArithmeticResult) (status lbool.LBool, optimal bool, resultErr error) {
	if err := s.backend.Push(); err != nil {
		return lbool.Undef, false, err
	}
	active := true
	defer func() {
		if active {
			resultErr = errors.Join(resultErr, s.backend.Pop(1))
		}
	}()
	status, err := s.backend.CheckSat(nil)
	if err != nil || status != lbool.True {
		return status, false, err
	}
	if err = s.replaceModel(result); err != nil {
		return lbool.Undef, false, err
	}
	for status == lbool.True {
		value, blocker, shared, err := s.backend.MaximizeObjective(uint32(index))
		if err != nil {
			return lbool.Undef, false, err
		}
		blockerRef, err := s.manager.NewRef(blocker)
		if err != nil {
			return lbool.Undef, false, err
		}
		updated, updateErr := s.backend.UpdateModel(shared)
		if updateErr == nil && updated {
			updateErr = s.replaceModel(result)
		}
		if updateErr != nil {
			return lbool.Undef, false, errors.Join(updateErr, s.manager.DelRef(&blockerRef))
		}
		result.Values[index] = s.orient(index, value)
		if value.Unbounded {
			return lbool.True, true, s.manager.DelRef(&blockerRef)
		}
		blocker, err = s.manager.RefGet(blockerRef)
		if err == nil {
			err = s.backend.Assert(blocker)
		}
		err = errors.Join(err, s.manager.DelRef(&blockerRef))
		if err != nil {
			return lbool.Undef, false, err
		}
		status, err = s.backend.CheckSat(nil)
		if err != nil {
			return lbool.Undef, false, err
		}
	}
	return status, status == lbool.False, nil
}

func (s *ArithmeticSolver) commitObjective(index int, value smtcore.SMTArithmeticObjectiveValue) error {
	if value.Unbounded || value.Finite.Infinity.Sign() != 0 || value.Finite.Epsilon.Sign() != 0 {
		return fmt.Errorf("opt: lexicographic continuation after an unattained or unbounded objective requires interval bounds")
	}
	if !s.objectives[index].maximize {
		value.Finite = value.Finite.Clone()
		value.Finite.Neg()
	}
	bound, err := s.backend.MakeObjectiveLowerBound(uint32(index), s.filter, value.Finite)
	if err != nil {
		return err
	}
	ref, err := s.manager.NewRef(bound)
	if err != nil {
		return err
	}
	bound, err = s.manager.RefGet(ref)
	if err == nil {
		err = s.backend.Assert(bound)
	}
	return errors.Join(err, s.manager.DelRef(&ref))
}

// Check runs the source blocker loop and preserves the last valid optimized
// model across final unsatisfiable blocker checks. Attained finite objectives
// are committed in order to implement the retained lexicographic priority.
func (s *ArithmeticSolver) Check() (result *ArithmeticResult, resultErr error) {
	if s == nil || s.backend == nil || len(s.objectives) == 0 {
		return nil, ast.ErrASTHandle
	}
	result = &ArithmeticResult{Status: lbool.Undef, Values: make([]smtcore.SMTArithmeticObjectiveValue, len(s.objectives))}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, result.Close())
			result = nil
		}
	}()
	for i := range s.objectives {
		status, optimal, err := s.optimizeOne(i, result)
		if err != nil {
			return nil, err
		}
		if !optimal {
			result.Status = status
			return result, nil
		}
		if i+1 < len(s.objectives) {
			if err = s.commitObjective(i, result.Values[i]); err != nil {
				return nil, err
			}
		}
	}
	result.Status, result.Optimal = lbool.True, true
	return result, nil
}

func (s *ArithmeticSolver) Close() error {
	if s == nil || s.backend == nil {
		return nil
	}
	for len(s.objectives) != 0 {
		last := len(s.objectives) - 1
		if err := s.manager.DelRef(&s.objectives[last].term); err != nil {
			return err
		}
		s.objectives = s.objectives[:last]
	}
	var err error
	if s.filter != nil {
		err = s.filter.Close()
		if err == nil {
			s.filter = nil
		}
	}
	err = errors.Join(err, s.backend.Close())
	if err == nil {
		s.backend = nil
	}
	return err
}
