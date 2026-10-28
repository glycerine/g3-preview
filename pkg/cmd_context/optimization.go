package cmd_context

// Source: src/opt/opt_context.cpp scoped objectives and optimize;
// src/opt/opt_cmds.cpp maximize/minimize command integration.

import (
	"errors"
	"fmt"
	"io"
	"math/big"

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
	bitVector, err := m.RegisterBitVectorPlugin()
	if err != nil {
		return err
	}
	isBitVector := family == bitVector
	if family != arithmetic && !isBitVector {
		return fmt.Errorf("objective must be bit-vector, integer or real")
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
	objective := optimizationObjective{expression: expression, ref: ref, maximize: maximize, bitVector: isBitVector}
	if isBitVector {
		objective.bitWidth, err = m.BitVectorSize(sort)
		if err != nil {
			return errors.Join(err, m.DelRef(&ref))
		}
	}
	c.objectives = append(c.objectives, objective)
	return nil
}

// AddSoftConstraint groups constraints by identifier at the position of the
// group's first occurrence. Terms appended inside a scope are trailed
// separately because the group itself may predate that scope.
func (c *Context) AddSoftConstraint(expression ast.ASTHandle, weight *big.Rat, id string) error {
	if weight == nil {
		return ast.ErrASTHandle
	}
	m, err := c.Manager()
	if err != nil {
		return err
	}
	sort, err := m.GetSort(expression)
	if err != nil {
		return err
	}
	boolean, err := m.MkBoolSort()
	if err != nil {
		return err
	}
	if sort != boolean {
		return fmt.Errorf("invalid type for expression: expected Boolean type")
	}
	if err = c.logicChecker.CheckExpression(expression); err != nil {
		return err
	}
	c.checked = false
	if err = c.clearCheckSatResult(); err != nil {
		return err
	}
	index, found := c.softObjectiveIndex[id]
	if !found {
		index = len(c.objectives)
		c.objectives = append(c.objectives, optimizationObjective{soft: true, id: id})
		c.softObjectiveIndex[id] = index
	}
	if weight.Sign() == 0 {
		return nil
	}
	ref, err := m.NewRef(expression)
	if err != nil {
		return err
	}
	term := optimizationSoftTerm{expression: expression, ref: ref}
	term.weight.Set(weight)
	c.objectives[index].softTerms = append(c.objectives[index].softTerms, term)
	c.softObjectiveTrail = append(c.softObjectiveTrail, index)
	return nil
}

func (c *Context) restoreObjectives(size, softTerms int) error {
	for len(c.softObjectiveTrail) > softTerms {
		last := len(c.softObjectiveTrail) - 1
		index := c.softObjectiveTrail[last]
		if index < 0 || index >= len(c.objectives) || !c.objectives[index].soft || len(c.objectives[index].softTerms) == 0 {
			return ast.ErrASTDomain
		}
		objective := &c.objectives[index]
		termIndex := len(objective.softTerms) - 1
		if err := c.manager.DelRef(&objective.softTerms[termIndex].ref); err != nil {
			return err
		}
		objective.softTerms = objective.softTerms[:termIndex]
		c.softObjectiveTrail = c.softObjectiveTrail[:last]
	}
	for len(c.objectives) > size {
		last := len(c.objectives) - 1
		objective := &c.objectives[last]
		if objective.soft {
			delete(c.softObjectiveIndex, objective.id)
			if len(objective.softTerms) != 0 {
				return ast.ErrASTDomain
			}
		} else if err := c.manager.DelRef(&objective.ref); err != nil {
			return err
		}
		c.objectives = c.objectives[:last]
	}
	return nil
}

func (c *Context) addBitVectorObjective(optimizer *opt.ArithmeticSolver, objective *optimizationObjective) (resultErr error) {
	if optimizer == nil || objective == nil || !objective.bitVector {
		return ast.ErrASTHandle
	}
	var refs []ast.ASTRef
	defer func() {
		for i := len(refs) - 1; i >= 0; i-- {
			resultErr = errors.Join(resultErr, c.manager.DelRef(&refs[i]))
		}
	}()
	bit := uint64(0)
	if objective.maximize {
		bit = 1
	}
	value, err := c.manager.BitVectorNumeralUint64(bit, 1)
	if err != nil {
		return err
	}
	valueRef, err := c.manager.NewRef(value)
	if err != nil {
		return err
	}
	refs = append(refs, valueRef)
	terms := make([]ast.ASTHandle, objective.bitWidth)
	weights := make([]*big.Rat, objective.bitWidth)
	for i := uint32(0); i < objective.bitWidth; i++ {
		extracted, e := c.manager.MkBitVectorExtract(i, i, objective.expression)
		if e != nil {
			return e
		}
		extractedRef, e := c.manager.NewRef(extracted)
		if e != nil {
			return e
		}
		refs = append(refs, extractedRef)
		value, e = c.manager.RefGet(valueRef)
		if e != nil {
			return e
		}
		extracted, e = c.manager.RefGet(extractedRef)
		if e != nil {
			return e
		}
		term, e := c.manager.MkEq(value, extracted)
		if e != nil {
			return e
		}
		termRef, e := c.manager.NewRef(term)
		if e != nil {
			return e
		}
		refs = append(refs, termRef)
		terms[i] = term
		weights[i] = new(big.Rat).SetInt(new(big.Int).Lsh(big.NewInt(1), uint(i)))
	}
	return optimizer.AddSoftObjective(terms, weights)
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
		if objective.soft {
			terms := make([]ast.ASTHandle, len(objective.softTerms))
			weights := make([]*big.Rat, len(objective.softTerms))
			for i := range objective.softTerms {
				terms[i] = objective.softTerms[i].expression
				weights[i] = &objective.softTerms[i].weight
			}
			err = optimizer.AddSoftObjective(terms, weights)
		} else if objective.bitVector {
			err = c.addBitVectorObjective(optimizer, &objective)
		} else {
			err = optimizer.AddObjective(objective.expression, objective.maximize)
		}
		if err != nil {
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
			if c.objectives[i].soft {
				c.objectives[i].softValue.Set(&optimized.Values[i].SoftCost)
			} else if c.objectives[i].bitVector {
				c.objectives[i].softValue.Set(&optimized.Values[i].SoftCost)
				if c.objectives[i].maximize {
					maximum := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(c.objectives[i].bitWidth)), big.NewInt(1))
					c.objectives[i].softValue.Sub(new(big.Rat).SetInt(maximum), &c.objectives[i].softValue)
				}
			} else {
				c.objectives[i].value = optimized.Values[i].Arithmetic
			}
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
		if objective.soft {
			if _, err := io.WriteString(out, objective.id); err != nil {
				return err
			}
		} else {
			if err := c.WriteExpression(out, objective.expression); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, " "); err != nil {
			return err
		}
		if objective.soft || objective.bitVector {
			if err := c.writeSoftObjectiveValue(out, &objective.softValue); err != nil {
				return err
			}
		} else {
			if err := c.writeObjectiveValue(out, &objective); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, ")\n"); err != nil {
			return err
		}
	}
	_, err := io.WriteString(out, ")\n")
	return err
}

func (c *Context) writeSoftObjectiveValue(out io.Writer, value *big.Rat) (resultErr error) {
	service := c.manager.NumeralService()
	if service == nil || value == nil {
		return ast.ErrASTHandle
	}
	numeral, err := service.NewValue()
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, service.Del(&numeral)) }()
	if err = service.SetString(&numeral, value.RatString()); err != nil {
		return err
	}
	expression, err := c.manager.ArithmeticNumeral(&numeral, value.IsInt())
	if err != nil {
		return err
	}
	ref, err := c.manager.NewRef(expression)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, c.manager.DelRef(&ref)) }()
	expression, err = c.manager.RefGet(ref)
	if err != nil {
		return err
	}
	return c.WriteExpression(out, expression)
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
