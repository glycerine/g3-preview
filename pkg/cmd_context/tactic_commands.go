// Source: src/cmd_context/tactic_cmds.cpp check_sat_using_tactict_cmd;
// src/cmd_context/cmd_context_to_goal.cpp assert_exprs_from.
package cmd_context

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/tactic/arith"
	"github.com/glycerine/g3/pkg/tactic/core"
	"github.com/glycerine/g3/pkg/tactic/smtlogics"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/statistics"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

// NewBuiltinTactic returns the uncounted tactic produced by a retained native
// factory. The parser or composition wrapper must acquire its own reference.
func (c *Context) NewBuiltinTactic(name string) (tactic.Tactic, error) {
	if err := c.valid(); err != nil {
		return nil, err
	}
	switch name {
	case "smt", "simplify", "elim-and", "propagate-values", "solve-eqs", "elim-uncnstr", "symmetry-reduce", "fix-dl-var", "normalize-bounds", "diff-neq", "lia2pb", "qfauflia", "qflra", "qfuf", "skip", "fail", "fail-if-undecided":
	default:
		return nil, fmt.Errorf("invalid tactic, unknown tactic %s", name)
	}
	m, err := c.Manager()
	if err != nil {
		return nil, err
	}
	switch name {
	case "smt":
		return smtcore.NewSMTTactic(m, nil, c.config)
	case "simplify":
		return core.MakeSimplifyTactic(m, c.config.Params, c.config.Globals, nil)
	case "elim-and":
		return core.MakeEliminateAndTactic(m, c.config.Params, c.config.Globals, nil)
	case "propagate-values":
		return core.NewPropagateValuesTactic(m, c.config.Params, c.config.Globals, nil)
	case "solve-eqs":
		return core.MakeSolveEquationsTactic(m, c.config.Params, c.config.Globals, nil, c.config.ModelEvaluator)
	case "elim-uncnstr":
		return core.MakeEliminateUnconstrainedTactic(m, c.config.Params, c.config.Globals, nil, c.config.ModelEvaluator)
	case "qfauflia":
		return smtlogics.NewQFAUFLIATactic(m, nil, c.config)
	case "qflra":
		return smtlogics.NewQFLRATactic(m, nil, c.config)
	case "qfuf":
		return smtlogics.NewQFUFTactic(m, nil, c.config)
	case "symmetry-reduce":
		return core.NewSymmetryReduceTactic(m, c.config.Params, c.config.Globals)
	case "fix-dl-var":
		return arith.MakeFixDLVarTactic(m, c.config.Params, c.config.Globals, nil, c.config.ModelEvaluator)
	case "normalize-bounds":
		return arith.MakeNormalizeBoundsTactic(m, c.config.Params, c.config.Globals, nil, c.config.ModelEvaluator)
	case "diff-neq":
		return arith.MakeDiffNeqTactic(m, c.config.Params, c.config.Globals, nil)
	case "lia2pb":
		return arith.MakeLIA2PBTactic(m, c.config.Params, c.config.Globals, nil, c.config.ModelEvaluator)
	case "skip":
		return tactic.NewSkipTactic(), nil
	case "fail":
		return tactic.NewFailTactic(), nil
	case "fail-if-undecided":
		return tactic.NewFailIfUndecidedTactic(), nil
	default:
		return nil, fmt.Errorf("invalid tactic, unknown tactic %s", name)
	}
}

func (c *Context) clearTacticResult() error {
	if c.tacticResult != nil {
		if err := c.tacticResult.Close(); err != nil {
			return err
		}
		c.tacticResult = nil
	}
	return nil
}

func (c *Context) checkSatResult() solver.CheckSatResult {
	if c.tacticResult != nil {
		return c.tacticResult
	}
	return c.solver
}

// AssertGoal copies the current assertions, asserted proofs and original
// assertion-name dependencies into a goal, in command-session order.
func (c *Context) AssertGoal(goal *tactic.Goal) error {
	m, err := c.Manager()
	if err != nil {
		return err
	}
	if goal == nil || goal.Manager() != m {
		return ast.ErrASTHandle
	}
	state, err := c.parameters.State()
	if err != nil {
		return err
	}
	if state.Proof && state.UnsatCore {
		return fmt.Errorf("Frontend does not support simultaneous generation of proofs and unsat cores")
	}
	proofs, err := goal.ProofsEnabled()
	if err != nil {
		return err
	}
	for _, row := range c.assertions {
		var proof ast.ASTHandle
		if proofs {
			proof, err = m.MkProofAsserted(row.expression)
			if err != nil {
				return err
			}
		}
		var dep ast.ASTDependency
		if state.UnsatCore && !row.tag.IsNull() {
			dep, err = m.DependencyLeaf(row.tag)
			if err != nil {
				return err
			}
		}
		if err = goal.AssertWithProof(row.expression, proof, dep); err != nil {
			return err
		}
	}
	return nil
}

// CheckSatUsing keeps the ordinary incremental solver and its scopes intact.
// Results from the selected tactic replace the command session's last result.
func (c *Context) CheckSatUsing(selected tactic.Tactic, options *uparams.ParamsRef) (status lbool.LBool, err error) {
	m, err := c.Manager()
	if err != nil {
		return lbool.Undef, err
	}
	if selected == nil {
		return lbool.Undef, fmt.Errorf("check-sat-using needs a tactic argument")
	}
	state, err := c.parameters.State()
	if err != nil {
		return lbool.Undef, err
	}
	if options == nil {
		empty := c.config.Params.EmptyValue()
		options = &empty
	}
	params, err := c.parameters.MergeDefaultParams(options)
	if err != nil {
		return lbool.Undef, err
	}
	defer func() { err = errors.Join(err, c.config.Params.Del(&params)) }()
	wrapped, err := tactic.NewUsingParamsTactic(c.config.Params, selected, &params)
	if err != nil {
		return lbool.Undef, err
	}
	if err = wrapped.IncRef(); err != nil {
		return lbool.Undef, err
	}
	defer func(owned tactic.Tactic) { err = errors.Join(err, owned.DecRef()) }(wrapped)
	var logic usymbol.Symbol
	if c.logic != "" {
		logic, err = m.Symbols.Intern(c.logic)
		if err != nil {
			return lbool.Undef, err
		}
	}
	if err = wrapped.SetLogic(logic); err != nil {
		return lbool.Undef, err
	}
	goal, err := tactic.NewGoal(m, state.Proof, state.Model, state.UnsatCore)
	if err != nil {
		return lbool.Undef, err
	}
	defer func() { err = errors.Join(err, goal.Close()) }()
	if err = c.AssertGoal(goal); err != nil {
		return lbool.Undef, err
	}
	c.checked = false
	if err = c.clearTacticResult(); err != nil {
		return lbool.Undef, err
	}
	c.tacticResult, err = solver.NewSimpleCheckSatResult(m)
	if err != nil {
		return lbool.Undef, err
	}
	var md tactic.ModelRef
	defer func() { err = errors.Join(err, md.Close()) }()
	labels := tactic.ModelLabels{Budget: m.Budget()}
	defer labels.Close()
	proofRef, err := m.NewRef(ast.ASTHandle{})
	if err != nil {
		return lbool.Undef, err
	}
	defer func() { err = errors.Join(err, m.DelRef(&proofRef)) }()
	var dep tactic.DependencyRef
	defer func() { err = errors.Join(err, dep.Close()) }()
	limitName := "rlimit"
	limit, err := c.config.Params.GetUIntName(&params, &limitName, state.RLimit)
	if err != nil {
		return lbool.Undef, err
	}
	if err = m.ResourceLimit().Push(limit); err != nil {
		return lbool.Undef, err
	}
	defer m.ResourceLimit().Pop()
	timeoutName := "timeout"
	timeout, err := c.config.Params.GetUIntName(&params, &timeoutName, state.Timeout)
	if err != nil {
		return lbool.Undef, err
	}
	if timeout != ^uint32(0) && timeout != 0 {
		timed, e := tactic.NewTryForTactic(wrapped, timeout)
		if e != nil {
			return lbool.Undef, e
		}
		if e = timed.IncRef(); e != nil {
			return lbool.Undef, e
		}
		defer func() { err = errors.Join(err, timed.DecRef()) }()
		wrapped = timed
	}
	status, reason, checkErr := tactic.CheckSat(wrapped, goal, &md, &labels, proofRef, &dep)
	if checkErr != nil {
		return lbool.Undef, checkErr
	}
	r := c.tacticResult
	if err = r.SetStatus(status); err != nil {
		return lbool.Undef, err
	}
	if status == lbool.Undef {
		if reason == "" {
			reason = "unknown"
		}
		if err = r.SetReasonUnknown(reason); err != nil {
			return lbool.Undef, err
		}
	}
	model, err := md.Get()
	if err != nil {
		return lbool.Undef, err
	}
	if err = r.SetModel(model); err != nil {
		return lbool.Undef, err
	}
	proof, err := m.RefGet(proofRef)
	if err != nil {
		return lbool.Undef, err
	}
	if err = r.SetProof(proof); err != nil {
		return lbool.Undef, err
	}
	leaves, err := m.DependencyLinearize(dep.Get(), nil)
	if err != nil {
		return lbool.Undef, err
	}
	if err = r.SetUnsatCore(leaves); err != nil {
		return lbool.Undef, err
	}
	if err = r.SetLabels(labels.Values); err != nil {
		return lbool.Undef, err
	}
	stats, err := r.Statistics()
	if err != nil {
		return lbool.Undef, err
	}
	for key, value := range wrapped.CollectStatistics() {
		stats.UpdateDouble(statistics.Key(key), value)
	}
	c.status, c.checked = status, true
	return status, nil
}
