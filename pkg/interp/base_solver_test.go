package interp

import (
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

type interpolationScriptSolver struct {
	*solver.SimpleCheckSatResult
	manager    *ast.ASTManager
	status     lbool.LBool
	script     []lbool.LBool
	checks     int
	assertions []ast.ASTHandle
	params     *uparams.ParamsRef
}

func newInterpolationScriptSolver(m *ast.ASTManager, status lbool.LBool) (*interpolationScriptSolver, error) {
	result, err := solver.NewSimpleCheckSatResult(m)
	if err != nil {
		return nil, err
	}
	return &interpolationScriptSolver{SimpleCheckSatResult: result, manager: m, status: status}, nil
}

func (s *interpolationScriptSolver) Close() error { return s.SimpleCheckSatResult.Close() }
func (s *interpolationScriptSolver) Translate(*ast.ASTManager, *uparams.ParamsRef) (solver.Solver, error) {
	return s, nil
}
func (s *interpolationScriptSolver) UpdateParams(p *uparams.ParamsRef) error {
	s.params = p
	return nil
}
func (s *interpolationScriptSolver) Params() (*uparams.ParamsRef, error) { return s.params, nil }
func (s *interpolationScriptSolver) CollectParamDescrs(*uparams.ParamDescrs) error {
	return nil
}
func (s *interpolationScriptSolver) SetProduceModels(bool) error { return nil }
func (s *interpolationScriptSolver) Assert(h ast.ASTHandle) error {
	s.assertions = append(s.assertions, h)
	return nil
}
func (s *interpolationScriptSolver) AssertTracked(h, _ ast.ASTHandle) error { return s.Assert(h) }
func (s *interpolationScriptSolver) Push() error                            { return nil }
func (s *interpolationScriptSolver) Pop(uint32) error                       { return nil }
func (s *interpolationScriptSolver) ScopeLevel() (uint32, error)            { return 0, nil }
func (s *interpolationScriptSolver) CheckSat([]ast.ASTHandle) (lbool.LBool, error) {
	status := s.status
	if s.checks < len(s.script) {
		status = s.script[s.checks]
	}
	s.checks++
	if err := s.SetStatus(status); err != nil {
		return lbool.Undef, err
	}
	return status, nil
}
func (s *interpolationScriptSolver) SetProgressCallback(solver.ProgressCallback) error { return nil }
func (s *interpolationScriptSolver) NumAssertions() (uint32, error) {
	return uint32(len(s.assertions)), nil
}
func (s *interpolationScriptSolver) Assertion(i uint32) (ast.ASTHandle, error) {
	if int(i) >= len(s.assertions) {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	return s.assertions[i], nil
}
func (s *interpolationScriptSolver) NumAssumptions() (uint32, error) { return 0, nil }
func (s *interpolationScriptSolver) Assumption(uint32) (ast.ASTHandle, error) {
	return ast.ASTHandle{}, ast.ErrASTDomain
}
func (s *interpolationScriptSolver) GetConsequences([]ast.ASTHandle, []ast.ASTHandle) (lbool.LBool, []ast.ASTHandle, error) {
	return s.status, nil, nil
}
func (s *interpolationScriptSolver) FindMutexes([]ast.ASTHandle) (lbool.LBool, [][]ast.ASTHandle, error) {
	return lbool.True, nil, nil
}
func (s *interpolationScriptSolver) PreferredSAT([]ast.ASTHandle) (lbool.LBool, [][]ast.ASTHandle, error) {
	return s.status, nil, nil
}
func (s *interpolationScriptSolver) Display(io.Writer, []ast.ASTHandle) error { return nil }

type interpolationValueEvaluator struct{ manager *ast.ASTManager }

func (e *interpolationValueEvaluator) EvaluateValue(h ast.ASTHandle) (ast.ASTRef, error) {
	return e.manager.NewRef(h)
}
func (*interpolationValueEvaluator) Close() error { return nil }

func TestInterpolationBaseSolverUnsatProof(t *testing.T) {
	m, manager := interpProofManagerFixture(t)
	boolSort, err := manager.BoolType()
	interpMust(t, err)
	p, err := manager.MakeVariable("base-unsat-p", boolSort)
	interpMust(t, err)
	p = interpPin(t, m, p)
	proof, err := m.MkProofAsserted(p)
	interpMust(t, err)
	proof = interpPin(t, m, proof)
	b, err := NewBase(manager, nil, nil, nil)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, b.Close()) })
	var used *interpolationScriptSolver
	factory := solver.FactoryFunc(func(got *ast.ASTManager, _ *uparams.ParamsRef, proofs, models, cores bool, logic string) (solver.Solver, error) {
		if got != m || !proofs || !models || !cores || logic != "" {
			t.Fatal("solver factory arguments", got == m, proofs, models, cores, logic)
		}
		used, err = newInterpolationScriptSolver(got, lbool.False)
		if err != nil {
			return nil, err
		}
		if err = used.SetProof(proof); err != nil {
			return nil, err
		}
		return used, nil
	})
	interpMust(t, b.ConfigureSolver(factory, nil))
	result, err := b.IsSAT([]ast.ASTHandle{p}, nil)
	interpMust(t, err)
	if result.Satisfiable() || result.Status != lbool.False || len(used.assertions) != 1 || used.assertions[0] != p {
		t.Fatal("unsat result", result.Status, used.assertions)
	}
	if got, e := m.RefGet(result.Proof); e != nil || got != proof {
		t.Fatal("proof ownership", got, e)
	}
	interpMust(t, result.Close())
}

func TestInterpolationBaseSolverModelValuesAndGuards(t *testing.T) {
	m, manager := interpManagerFixture(t)
	boolSort, err := manager.BoolType()
	interpMust(t, err)
	p, err := manager.MakeVariable("base-sat-p", boolSort)
	interpMust(t, err)
	p = interpPin(t, m, p)
	b, err := NewBase(manager, nil, nil, nil)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, b.Close()) })
	if _, err = b.IsSAT(nil, nil); err == nil || err.Error() != "interpolation base has no solver factory" {
		t.Fatal("missing solver", err)
	}
	factory := solver.FactoryFunc(func(got *ast.ASTManager, _ *uparams.ParamsRef, _, _, _ bool, _ string) (solver.Solver, error) {
		s, e := newInterpolationScriptSolver(got, lbool.True)
		if e != nil {
			return nil, e
		}
		md, e := model.NewModelCore(got)
		if e != nil {
			return nil, e
		}
		if e = s.SetModel(md); e != nil {
			return nil, e
		}
		return s, nil
	})
	evaluator := func(md *model.ModelCore) (solver.ConsequenceEvaluator, error) {
		return &interpolationValueEvaluator{manager: m}, nil
	}
	interpMust(t, b.ConfigureSolver(factory, evaluator))
	result, err := b.IsSAT(nil, []ast.ASTHandle{p})
	interpMust(t, err)
	if !result.Satisfiable() || len(result.Values) != 1 {
		t.Fatal("sat result", result.Status, len(result.Values))
	}
	if got, e := m.RefGet(result.Values[0]); e != nil || got != p {
		t.Fatal("evaluated value", got, e)
	}
	interpMust(t, result.Close())
}

var _ solver.Solver = (*interpolationScriptSolver)(nil)
var _ solver.ConsequenceEvaluator = (*interpolationValueEvaluator)(nil)
