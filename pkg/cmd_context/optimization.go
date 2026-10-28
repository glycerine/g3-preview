package cmd_context

// Source: src/opt/opt_context.cpp scoped objectives and optimize;
// src/opt/opt_cmds.cpp maximize/minimize command integration.

import (
	"errors"
	"fmt"
	"io"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/opt"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func (c *Context) AddObjective(expression ast.ASTHandle, maximize bool) error {
	m, err := c.Manager()
	if err != nil {
		return err
	}
	kind, err := m.GetKind(expression)
	if err != nil {
		return err
	}
	if kind != ast.ASTApp {
		return fmt.Errorf("malformed objective term: it cannot be a quantifier or bound variable")
	}
	sort, err := m.GetSort(expression)
	if err != nil {
		return err
	}
	family, err := m.GetFamilyID(sort)
	if err != nil {
		return err
	}
	arithmetic, err := m.ArithmeticFamilyID()
	if err != nil {
		return err
	}
	if family != arithmetic {
		return fmt.Errorf("objective must be integer or real")
	}
	if err = c.logicChecker.CheckExpression(expression); err != nil {
		return err
	}
	c.checked = false
	if err = c.clearCheckSatResult(); err != nil {
		return err
	}
	ref, err := m.NewRef(expression)
	if err != nil {
		return err
	}
	c.objectives = append(c.objectives, optimizationObjective{expression: expression, ref: ref, maximize: maximize})
	return nil
}

func (c *Context) restoreObjectives(size int) error {
	for len(c.objectives) > size {
		last := len(c.objectives) - 1
		if err := c.manager.DelRef(&c.objectives[last].ref); err != nil {
			return err
		}
		c.objectives = c.objectives[:last]
	}
	return nil
}

func (c *Context) checkOptimize(assumptions []ast.ASTHandle) (status lbool.LBool, resultErr error) {
	optimizer, err := opt.NewArithmeticSolver(c.manager, &c.params, c.logic, c.config)
	if err != nil {
		return lbool.Undef, err
	}
	defer func() { resultErr = errors.Join(resultErr, optimizer.Close()) }()
	for _, assertion := range c.assertions {
		if err = optimizer.Assert(assertion.expression); err != nil {
			return lbool.Undef, err
		}
	}
	for _, assumption := range assumptions {
		if err = optimizer.Assert(assumption); err != nil {
			return lbool.Undef, err
		}
	}
	for _, objective := range c.objectives {
		if err = optimizer.AddObjective(objective.expression, objective.maximize); err != nil {
			return lbool.Undef, err
		}
	}
	optimized, err := optimizer.Check()
	if err != nil {
		return lbool.Undef, err
	}
	defer func() { resultErr = errors.Join(resultErr, optimized.Close()) }()
	result, err := solver.NewSimpleCheckSatResult(c.manager)
	if err != nil {
		return lbool.Undef, err
	}
	published := false
	defer func() {
		if !published {
			resultErr = errors.Join(resultErr, result.Close())
		}
	}()
	if err = result.SetStatus(optimized.Status); err != nil {
		return lbool.Undef, err
	}
	if optimized.Status == lbool.Undef {
		if err = result.SetReasonUnknown("optimization returned unknown"); err != nil {
			return lbool.Undef, err
		}
	}
	if err = result.SetModel(optimized.Model); err != nil {
		return lbool.Undef, err
	}
	if optimized.Status == lbool.True && len(c.objectives) == len(optimized.Values) {
		for i := range c.objectives {
			c.objectives[i].value = optimized.Values[i]
			c.objectives[i].valueReady = optimized.Optimal
		}
	}
	c.tacticResult = result
	c.status, c.checked = optimized.Status, true
	published = true
	return optimized.Status, nil
}

func (c *Context) WriteObjectives(out io.Writer) error {
	if !c.checked {
		return fmt.Errorf("check-sat has not been called with latest objectives")
	}
	if _, err := io.WriteString(out, "(objectives\n"); err != nil {
		return err
	}
	for _, objective := range c.objectives {
		if !objective.valueReady {
			return fmt.Errorf("objective value is not available")
		}
		if _, err := io.WriteString(out, " ("); err != nil {
			return err
		}
		if err := c.WriteExpression(out, objective.expression); err != nil {
			return err
		}
		if _, err := io.WriteString(out, " "); err != nil {
			return err
		}
		if err := c.writeObjectiveValue(out, &objective); err != nil {
			return err
		}
		if _, err := io.WriteString(out, ")\n"); err != nil {
			return err
		}
	}
	_, err := io.WriteString(out, ")\n")
	return err
}

func (c *Context) writeObjectiveValue(out io.Writer, objective *optimizationObjective) error {
	if objective.value.Unbounded {
		if objective.maximize {
			_, err := io.WriteString(out, "oo")
			return err
		}
		_, err := io.WriteString(out, "(* (- 1) oo)")
		return err
	}
	value := &objective.value.Finite
	sort, err := c.manager.GetSort(objective.expression)
	if err != nil {
		return err
	}
	kind, err := c.manager.GetDeclKind(sort)
	if err != nil {
		return err
	}
	ref, err := opt.ArithmeticValueExpression(c.manager, value, kind == 1)
	if err != nil {
		return err
	}
	h, err := c.manager.RefGet(ref)
	if err == nil {
		err = c.WriteExpression(out, h)
	}
	return errors.Join(err, c.manager.DelRef(&ref))
}
