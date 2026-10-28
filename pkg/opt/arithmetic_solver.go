package opt

// Source: src/opt/opt_solver.cpp and src/opt/optsmt.cpp retained arithmetic
// objective loop.

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

// ArithmeticResult owns the best model retained by one optimization check.
// Value is expressed in the caller's original maximize/minimize orientation.
type ArithmeticResult struct {
	Status  lbool.LBool
	Value   smtcore.SMTArithmeticObjectiveValue
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
	backend  *smtcore.SMTSolver
	manager  *ast.ASTManager
	maximize bool
	added    bool
}

func NewArithmeticSolver(m *ast.ASTManager, params *uparams.ParamsRef, logic string, config smtcore.SMTSolverConfig) (*ArithmeticSolver, error) {
	backend, err := smtcore.NewSMTSolver(m, params, logic, config)
	if err != nil {
		return nil, err
	}
	if err = backend.SetProduceModels(true); err != nil {
		return nil, errors.Join(err, backend.Close())
	}
	return &ArithmeticSolver{backend: backend, manager: m}, nil
}

func (s *ArithmeticSolver) Assert(formula ast.ASTHandle) error {
	if s == nil || s.backend == nil {
		return ast.ErrASTHandle
	}
	return s.backend.Assert(formula)
}

// AddObjective registers the single-objective end-to-end path. Minimization
// uses the source normalization to maximization of the negated term.
func (s *ArithmeticSolver) AddObjective(term ast.ASTHandle, maximize bool) error {
	if s == nil || s.backend == nil {
		return ast.ErrASTHandle
	}
	if s.added {
		return fmt.Errorf("opt: multiple objectives are not yet supported by the arithmetic shell")
	}
	objective := term
	var err error
	if !maximize {
		objective, err = s.manager.MkArithmeticNeg(term)
		if err != nil {
			return err
		}
	}
	if _, err = s.backend.AddObjective(objective); err != nil {
		return err
	}
	s.maximize, s.added = maximize, true
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

func (s *ArithmeticSolver) orient(value smtcore.SMTArithmeticObjectiveValue) smtcore.SMTArithmeticObjectiveValue {
	if s.maximize || value.Unbounded {
		return value
	}
	value.Finite = value.Finite.Clone()
	value.Finite.Neg()
	return value
}

// Check runs the source blocker loop and preserves the last valid optimized
// model across the final unsatisfiable blocker check.
func (s *ArithmeticSolver) Check() (result *ArithmeticResult, resultErr error) {
	if s == nil || s.backend == nil || !s.added {
		return nil, ast.ErrASTHandle
	}
	result = &ArithmeticResult{Status: lbool.Undef}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, result.Close())
			result = nil
		}
	}()
	status, err := s.backend.CheckSat(nil)
	if err != nil {
		return nil, err
	}
	result.Status = status
	if status != lbool.True {
		return result, nil
	}
	if err = s.replaceModel(result); err != nil {
		return nil, err
	}
	for status == lbool.True {
		value, blocker, shared, err := s.backend.MaximizeObjective(0)
		if err != nil {
			return nil, err
		}
		blockerRef, err := s.manager.NewRef(blocker)
		if err != nil {
			return nil, err
		}
		updated, updateErr := s.backend.UpdateModel(shared)
		if updateErr == nil && updated {
			updateErr = s.replaceModel(result)
		}
		if updateErr != nil {
			return nil, errors.Join(updateErr, s.manager.DelRef(&blockerRef))
		}
		result.Value = s.orient(value)
		if value.Unbounded {
			result.Status, result.Optimal = lbool.True, true
			return result, s.manager.DelRef(&blockerRef)
		}
		blocker, err = s.manager.RefGet(blockerRef)
		if err == nil {
			err = s.backend.Assert(blocker)
		}
		err = errors.Join(err, s.manager.DelRef(&blockerRef))
		if err != nil {
			return nil, err
		}
		status, err = s.backend.CheckSat(nil)
		if err != nil {
			return nil, err
		}
	}
	if status == lbool.False {
		result.Status, result.Optimal = lbool.True, true
	} else {
		result.Status = status
	}
	return result, nil
}

func (s *ArithmeticSolver) Close() error {
	if s == nil || s.backend == nil {
		return nil
	}
	err := s.backend.Close()
	if err == nil {
		s.backend = nil
	}
	return err
}
