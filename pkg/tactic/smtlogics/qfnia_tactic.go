package smtlogics

// Source: src/tactic/smtlogics/qfnia_tactic.cpp/.h retained preamble prefix,
// bounded-NIA bit-vector route and final SMT fallback.

import (
	"errors"
	"io"
	"os"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/tactic/arith"
	"github.com/glycerine/g3/pkg/tactic/core"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func NewQFNIATactic(m *ast.ASTManager, p *uparams.ParamsRef, config smtcore.SMTSolverConfig) (tactic.Tactic, error) {
	return NewQFNIATacticWithOutput(m, p, config, os.Stdout)
}

// NewQFNIATacticWithOutput retains the source's first bounded-NIA solver and
// final complete SMT fallback. The intervening timed SMT/NLSAT attempts remain
// pending with the NLSAT tactic layer.
func NewQFNIATacticWithOutput(m *ast.ASTManager, p *uparams.ParamsRef, config smtcore.SMTSolverConfig, output io.Writer) (result tactic.Tactic, err error) {
	c, g := config.Params, config.Globals
	if m == nil || c == nil || g == nil || g.Context != c {
		return nil, uparams.ErrParamsOwner
	}
	if err = c.CheckServices(m.Symbols, m.NumeralService()); err != nil {
		return nil, err
	}
	b := qfliaConstruction{c: c, m: m}
	defer func() {
		if cleanup := b.close(); cleanup != nil {
			err = errors.Join(err, cleanup)
		}
		if err != nil && result != nil {
			if acquire := result.IncRef(); acquire != nil {
				err = errors.Join(err, acquire)
			} else {
				err = errors.Join(err, result.DecRef())
			}
			result = nil
		}
	}()
	copyParams := func() *uparams.ParamsRef {
		value := b.params()
		if b.err == nil && p != nil {
			b.err = c.Assign(value, p)
		}
		return value
	}
	main, pullParams, contextParams := copyParams(), copyParams(), copyParams()
	simplifyParams, nlaParams, smtParams := copyParams(), copyParams(), copyParams()
	b.boolean(pullParams, "pull_cheap_ite", true)
	b.boolean(pullParams, "local_ctx", true)
	b.uint(pullParams, "local_ctx_limit", 10000000)
	b.uint(contextParams, "max_depth", 30)
	b.uint(contextParams, "max_steps", 5000000)
	b.boolean(simplifyParams, "hoist_mul", true)
	b.uint(nlaParams, "nla2bv_max_bv_size", 64)
	b.boolean(smtParams, "som", true)
	if b.err != nil {
		return nil, b.err
	}
	simplify := func(params *uparams.ParamsRef) tactic.Tactic {
		return b.using(b.clean(b.keep(core.NewSimplifyTactic(m, c, g, nil))), params)
	}
	preamble := b.then(
		simplify(main),
		b.keep(core.NewPropagateValuesTactic(m, c, g, nil)),
		b.using(b.clean(b.keep(core.NewContextSimplifyTactic(m, c, g, nil, config.ModelEvaluator))), contextParams),
		simplify(pullParams),
		b.clean(b.keep(core.NewEliminateUnconstrainedTactic(m, c, g, nil, config.ModelEvaluator))),
	)
	nla := b.keep(arith.NewNLA2BVTactic(m, c, g, nlaParams, config.ModelEvaluator))
	bvSolver := b.keep(NewQFBVTacticWithOutput(m, p, config, output))
	bounded := b.then(simplify(simplifyParams), nla, bvSolver, b.keep(tactic.NewFailIfUndecidedTactic(), nil))
	fallback := b.then(simplify(smtParams), b.keep(smtcore.NewSMTTactic(m, nil, config)))
	body := b.then(preamble, b.otherwise(bounded, fallback))
	if b.err != nil {
		return nil, b.err
	}
	result, err = tactic.NewUsingParamsTactic(c, body, main)
	if err != nil {
		return nil, err
	}
	if p == nil {
		empty := c.EmptyValue()
		p = &empty
	}
	err = result.UpdateParams(p)
	return result, err
}
