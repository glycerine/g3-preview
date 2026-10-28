package portfolio

// Source: src/tactic/portfolio/smt_strategic_solver.cpp retained QF_AUFLIA and
// QF_LRA/QF_UF/QF_IDL/QF_LIA/QF_BV/QF_AUFBV/QF_ABV/QF_UFBV and
// UFNIA/UFLRA/AUFLIA/AUFLIRA/AUFNIRA branches and
// smt_strategic_solver_factory::operator().

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

// NewALLStrategicSolver retains the default strategy's theory-neutral
// simplification, propagation, equation solving and unconstrained-variable
// elimination path. The direct ALL SMT solver remains the incremental backend,
// so combined_solver owns the same nonincremental/incremental transition and
// unknown fallback as the source factory.
func NewALLStrategicSolver(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig) (result solver.Solver, err error) {
	strong, err := smtlogics.NewQFAUFLIATactic(m, p, config)
	if err != nil {
		return nil, err
	}
	if err = strong.IncRef(); err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, strong.DecRef()) }()
	ordinary, err := smtcore.NewSMTTactic(m, p, config)
	if err != nil {
		return nil, err
	}
	if err = ordinary.IncRef(); err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, ordinary.DecRef()) }()
	probe, err := newALLAsArrayProbe(m)
	if err != nil {
		return nil, err
	}
	if err = probe.IncRef(); err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, probe.DecRef()) }()
	strategy, err := tactic.NewCondTactic(probe, strong, ordinary)
	if err != nil {
		return nil, err
	}
	result, err = newStrategicSolver(m, strategy, "ALL", p, proofs, models, cores, config)
	return result, err
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

func NewQFBVStrategicSolver(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig) (solver.Solver, error) {
	return NewQFBVStrategicSolverWithOutput(m, p, proofs, models, cores, config, os.Stdout)
}

func NewQFBVStrategicSolverWithOutput(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig, output io.Writer) (solver.Solver, error) {
	strategy, err := smtlogics.NewQFBVTacticWithOutput(m, p, config, output)
	if err != nil {
		return nil, err
	}
	return newStrategicSolver(m, strategy, "QF_BV", p, proofs, models, cores, config)
}

func NewQFAUFBVStrategicSolver(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig) (solver.Solver, error) {
	strategy, err := smtlogics.NewQFAUFBVTactic(m, p, config)
	if err != nil {
		return nil, err
	}
	return newStrategicSolver(m, strategy, "QF_AUFBV", p, proofs, models, cores, config)
}

func NewQFABVStrategicSolver(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig) (solver.Solver, error) {
	strategy, err := smtlogics.NewQFAUFBVTactic(m, p, config)
	if err != nil {
		return nil, err
	}
	return newStrategicSolver(m, strategy, "QF_ABV", p, proofs, models, cores, config)
}

func NewQFUFBVStrategicSolver(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig) (solver.Solver, error) {
	return NewQFUFBVStrategicSolverWithOutput(m, p, proofs, models, cores, config, os.Stdout)
}

func NewQFUFBVStrategicSolverWithOutput(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig, output io.Writer) (solver.Solver, error) {
	strategy, err := smtlogics.NewQFUFBVTacticWithOutput(m, p, config, output)
	if err != nil {
		return nil, err
	}
	return newStrategicSolver(m, strategy, "QF_UFBV", p, proofs, models, cores, config)
}

func NewUFLRAStrategicSolver(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig) (solver.Solver, error) {
	strategy, err := smtlogics.NewUFLRATactic(m, p, config)
	if err != nil {
		return nil, err
	}
	return newStrategicSolver(m, strategy, "UFLRA", p, proofs, models, cores, config)
}

func NewUFNIAStrategicSolver(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig) (solver.Solver, error) {
	strategy, err := smtlogics.NewUFNIATactic(m, p, config)
	if err != nil {
		return nil, err
	}
	return newStrategicSolver(m, strategy, "UFNIA", p, proofs, models, cores, config)
}

func NewAUFLIAStrategicSolver(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig) (solver.Solver, error) {
	strategy, err := smtlogics.NewAUFLIATactic(m, p, config)
	if err != nil {
		return nil, err
	}
	return newStrategicSolver(m, strategy, "AUFLIA", p, proofs, models, cores, config)
}

func NewAUFLIRAStrategicSolver(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig) (solver.Solver, error) {
	strategy, err := smtlogics.NewAUFLIRATactic(m, p, config)
	if err != nil {
		return nil, err
	}
	return newStrategicSolver(m, strategy, "AUFLIRA", p, proofs, models, cores, config)
}

func NewAUFNIRAStrategicSolver(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, config smtcore.SMTSolverConfig) (solver.Solver, error) {
	strategy, err := smtlogics.NewAUFNIRATactic(m, p, config)
	if err != nil {
		return nil, err
	}
	return newStrategicSolver(m, strategy, "AUFNIRA", p, proofs, models, cores, config)
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
