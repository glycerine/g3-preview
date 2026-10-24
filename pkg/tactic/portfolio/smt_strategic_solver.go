// Source: src/tactic/portfolio/smt_strategic_solver.cpp retained QF_AUFLIA and
// QF_LRA/QF_UF/QF_IDL/QF_LIA branches and smt_strategic_solver_factory::operator().
package portfolio

import (
	"errors"
	"io"
	"os"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/tactic/smtlogics"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

// NewQFAUFLIAStrategicSolver connects the source tactic solver and incremental
// SMT solver through combined_solver's ordinary mode/fallback policy. Other
// logic strategies remain separate ports of the source factory dispatcher.
func NewQFAUFLIAStrategicSolver(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig) (result solver.Solver, err error) {
	strategy, err := smtlogics.NewQFAUFLIATactic(m, p, config)
	if err != nil {
		return nil, err
	}
	return newStrategicSolver(m, strategy, "QF_AUFLIA", p, proofs, models, cores, config)
}

func NewQFLRAStrategicSolver(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig) (solver.Solver, error) {
	strategy, err := smtlogics.NewQFLRATactic(m, p, config)
	if err != nil {
		return nil, err
	}
	return newStrategicSolver(m, strategy, "QF_LRA", p, proofs, models, cores, config)
}

func NewQFUFStrategicSolver(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig) (solver.Solver, error) {
	strategy, err := smtlogics.NewQFUFTactic(m, p, config)
	if err != nil {
		return nil, err
	}
	return newStrategicSolver(m, strategy, "QF_UF", p, proofs, models, cores, config)
}

func NewQFIDLStrategicSolver(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig) (solver.Solver, error) {
	return NewQFIDLStrategicSolverWithOutput(m, p, proofs, models, cores, config, os.Stdout)
}

func NewQFIDLStrategicSolverWithOutput(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig, output io.Writer) (solver.Solver, error) {
	strategy, err := smtlogics.NewQFIDLTacticWithOutput(m, p, config, output)
	if err != nil {
		return nil, err
	}
	return newStrategicSolver(m, strategy, "QF_IDL", p, proofs, models, cores, config)
}

func NewQFLIAStrategicSolver(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig) (solver.Solver, error) {
	return NewQFLIAStrategicSolverWithOutput(m, p, proofs, models, cores, config, os.Stdout)
}

func NewQFLIAStrategicSolverWithOutput(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig, output io.Writer) (solver.Solver, error) {
	strategy, err := smtlogics.NewQFLIATacticWithOutput(m, p, config, output)
	if err != nil {
		return nil, err
	}
	return newStrategicSolver(m, strategy, "QF_LIA", p, proofs, models, cores, config)
}

func newStrategicSolver(m *ast.ASTManager, strategy tactic.Tactic, logic string, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig) (result solver.Solver, err error) {
	if err = strategy.IncRef(); err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, strategy.DecRef()) }()
	first, err := solver.NewTacticSolver(m, strategy, p, proofs, models, cores, logic, solver.TacticSolverConfig{
		Params: config.Params,
		EvaluatorFactory: func(md *model.ModelCore) (solver.ConsequenceEvaluator, error) {
			return model.NewModelEvaluator(md, config.ModelEvaluator)
		},
	})
	if err != nil {
		return nil, err
	}
	if err = first.IncRef(); err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, first.DecRef()) }()
	second, err := smtcore.NewSMTSolver(m, p, logic, config)
	if err != nil {
		return nil, err
	}
	if err = second.IncRef(); err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, second.DecRef()) }()
	combined, err := solver.NewCombinedSolver(first, second, p, solver.CombinedSolverConfig{Params: config.Params, Globals: config.Globals})
	if err != nil {
		return nil, err
	}
	return combined, nil
}
