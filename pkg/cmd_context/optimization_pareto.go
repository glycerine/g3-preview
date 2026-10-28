package cmd_context

// Source: src/opt/opt_pareto.cpp GIA dominance loop and
// src/opt/opt_context.cpp Pareto comparison/model callbacks.

import (
	"errors"
	"math/big"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/math/simplex"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/opt"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func (c *Context) paretoArithmeticOnly() bool {
	for i := range c.objectives {
		if c.objectives[i].soft || c.objectives[i].bitVector {
			return false
		}
	}
	return true
}

func (c *Context) evaluateParetoModel(md *model.ModelCore) (values []big.Rat, resultErr error) {
	if md == nil {
		return nil, ast.ErrASTHandle
	}
	evaluator, err := model.NewModelEvaluator(md, c.config.ModelEvaluator)
	if err != nil {
		if evaluator != nil {
			err = errors.Join(err, evaluator.Close())
		}
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, evaluator.Close()) }()
	if err = evaluator.SetModelCompletion(true); err != nil {
		return nil, err
	}
	values = make([]big.Rat, len(c.objectives))
	for i := range c.objectives {
		ref, e := evaluator.EvaluateValue(c.objectives[i].expression)
		if e != nil {
			return nil, e
		}
		h, e := c.manager.RefGet(ref)
		if e == nil {
			var text string
			var matched bool
			text, _, matched, e = c.manager.ArithmeticNumeralText(h)
			if e == nil && !matched {
				e = ast.ErrASTDomain
			}
			if e == nil {
				if _, matched = values[i].SetString(text); !matched {
					e = ast.ErrASTDomain
				}
			}
		}
		e = errors.Join(e, c.manager.DelRef(&ref))
		if e != nil {
			return nil, e
		}
	}
	return values, nil
}

func (c *Context) assertParetoFormula(backend *smtcore.SMTSolver, values []big.Rat, dominates bool) (resultErr error) {
	if backend == nil || len(values) != len(c.objectives) {
		return ast.ErrASTHandle
	}
	var refs []ast.ASTRef
	hold := func(h ast.ASTHandle, err error) (ast.ASTHandle, error) {
		if err != nil {
			return ast.ASTHandle{}, err
		}
		ref, err := c.manager.NewRef(h)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		refs = append(refs, ref)
		return h, nil
	}
	defer func() {
		for i := len(refs) - 1; i >= 0; i-- {
			resultErr = errors.Join(resultErr, c.manager.DelRef(&refs[i]))
		}
	}()
	weak := make([]ast.ASTHandle, len(c.objectives))
	strict := make([]ast.ASTHandle, len(c.objectives))
	notBetter := make([]ast.ASTHandle, len(c.objectives))
	for i := range c.objectives {
		sort, err := c.manager.GetSort(c.objectives[i].expression)
		if err != nil {
			return err
		}
		kind, err := c.manager.GetDeclKind(sort)
		if err != nil {
			return err
		}
		var numeral ast.ASTHandle
		if kind == 1 {
			numeral, err = hold(c.manager.ArithmeticIntegerNumeralText(values[i].RatString()))
		} else {
			numeral, err = hold(c.manager.ArithmeticRealNumeralText(values[i].RatString()))
		}
		if err != nil {
			return err
		}
		term := c.objectives[i].expression
		if c.objectives[i].maximize {
			weak[i], err = hold(c.manager.MkArithmeticGE(term, numeral))
			if err == nil {
				notBetter[i], err = hold(c.manager.MkArithmeticLE(term, numeral))
				if err == nil {
					strict[i], err = hold(c.manager.MkNot(notBetter[i]))
				}
			}
		} else {
			weak[i], err = hold(c.manager.MkArithmeticLE(term, numeral))
			if err == nil {
				notBetter[i], err = hold(c.manager.MkArithmeticGE(term, numeral))
				if err == nil {
					strict[i], err = hold(c.manager.MkNot(notBetter[i]))
				}
			}
		}
		if err != nil {
			return err
		}
	}
	if !dominates {
		conjunction, err := hold(c.manager.MkAndArgs(notBetter))
		if err != nil {
			return err
		}
		formula, err := hold(c.manager.MkNot(conjunction))
		if err != nil {
			return err
		}
		return backend.Assert(formula)
	}
	conjunction, err := hold(c.manager.MkAndArgs(weak))
	if err != nil {
		return err
	}
	disjunction, err := hold(c.manager.MkOrArgs(strict))
	if err != nil {
		return err
	}
	formula, err := hold(c.manager.MkAnd(conjunction, disjunction))
	if err != nil {
		return err
	}
	return backend.Assert(formula)
}

func (c *Context) closeParetoBackend() error {
	c.optimizationParetoActive = false
	if c.optimizationParetoBackend == nil {
		return nil
	}
	err := c.optimizationParetoBackend.Close()
	if err == nil {
		c.optimizationParetoBackend = nil
	}
	return err
}

func (c *Context) newParetoBackend(assumptions []ast.ASTHandle) (backend *smtcore.SMTSolver, resultErr error) {
	backend, err := smtcore.NewSMTSolver(c.manager, &c.params, c.logic, c.config)
	if err != nil {
		return nil, err
	}
	published := false
	defer func() {
		if !published {
			resultErr = errors.Join(resultErr, backend.Close())
		}
	}()
	if err = backend.SetProduceModels(true); err != nil {
		return nil, err
	}
	for _, assertion := range c.assertions {
		if err = backend.Assert(assertion.expression); err != nil {
			return nil, err
		}
	}
	for _, assumption := range assumptions {
		if err = backend.Assert(assumption); err != nil {
			return nil, err
		}
	}
	published = true
	return backend, nil
}

func (c *Context) checkOptimizePareto(assumptions []ast.ASTHandle, continuation bool) (result *opt.ArithmeticResult, resultErr error) {
	if !c.paretoArithmeticOnly() {
		return nil, ast.ErrASTDomain
	}
	if !continuation || !c.optimizationParetoActive {
		if err := c.closeParetoBackend(); err != nil {
			return nil, err
		}
		backend, err := c.newParetoBackend(assumptions)
		if err != nil {
			return nil, err
		}
		c.optimizationParetoBackend = backend
	}
	c.optimizationParetoActive = true
	backend := c.optimizationParetoBackend
	status, err := backend.CheckSat(nil)
	if err != nil {
		return nil, err
	}
	if status != lbool.True {
		if closeErr := c.closeParetoBackend(); closeErr != nil {
			return nil, closeErr
		}
		return &opt.ArithmeticResult{Status: status, Values: make([]opt.ObjectiveValue, len(c.objectives)), Optimal: status == lbool.False}, nil
	}
	var best *model.ModelCore
	var values []big.Rat
	defer func() {
		if resultErr != nil && best != nil {
			resultErr = errors.Join(resultErr, best.DecRef())
		}
	}()
	candidate, err := backend.AcquireModel()
	if err != nil {
		return nil, err
	}
	if candidate == nil {
		return nil, ast.ErrASTHandle
	}
	values, err = c.evaluateParetoModel(candidate)
	if err != nil {
		return nil, errors.Join(err, candidate.DecRef())
	}
	best = candidate
	if err = backend.Push(); err != nil {
		return nil, err
	}
	pushed := true
	defer func() {
		if pushed {
			resultErr = errors.Join(resultErr, backend.Pop(1))
		}
	}()
	for status == lbool.True {
		if err = c.assertParetoFormula(backend, values, true); err != nil {
			return nil, err
		}
		status, err = backend.CheckSat(nil)
		if err != nil {
			return nil, err
		}
		if status != lbool.True {
			break
		}
		candidate, err = backend.AcquireModel()
		if err != nil {
			return nil, err
		}
		if candidate == nil {
			return nil, ast.ErrASTHandle
		}
		candidateValues, err := c.evaluateParetoModel(candidate)
		if err != nil {
			return nil, errors.Join(err, candidate.DecRef())
		}
		if best != nil {
			if err = best.DecRef(); err != nil {
				return nil, errors.Join(err, candidate.DecRef())
			}
		}
		best, values = candidate, candidateValues
	}
	if err = backend.Pop(1); err != nil {
		return nil, err
	}
	pushed = false
	if status != lbool.False {
		if closeErr := c.closeParetoBackend(); closeErr != nil {
			return nil, closeErr
		}
		result = &opt.ArithmeticResult{Status: status, Values: make([]opt.ObjectiveValue, len(c.objectives)), Model: best}
		best = nil
		return result, nil
	}
	if err = c.assertParetoFormula(backend, values, false); err != nil {
		return nil, err
	}
	result = &opt.ArithmeticResult{Status: lbool.True, Values: make([]opt.ObjectiveValue, len(c.objectives)), Model: best, Optimal: true}
	best = nil
	for i := range values {
		finite := simplex.RationalInfinitesimal(&values[i])
		result.Values[i] = opt.ObjectiveValue{
			Arithmetic:           smtcore.SMTArithmeticObjectiveValue{Finite: finite},
			ArithmeticModel:      smtcore.SMTArithmeticObjectiveValue{Finite: finite.Clone()},
			ArithmeticModelValid: true,
		}
	}
	return result, nil
}
