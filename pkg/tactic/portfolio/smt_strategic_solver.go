// Source: src/tactic/portfolio/smt_strategic_solver.cpp retained QF_AUFLIA
// branch of mk_tactic_for_logic and smt_strategic_solver_factory::operator().
package portfolio

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/solver"
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
	if err = strategy.IncRef(); err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, strategy.DecRef()) }()
	first, err := solver.NewTacticSolver(m, strategy, p, proofs, models, cores, "QF_AUFLIA", solver.TacticSolverConfig{
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
	second, err := smtcore.NewSMTSolver(m, p, "QF_AUFLIA", config)
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
