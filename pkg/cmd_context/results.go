// Source: src/cmd_context/basic_cmds.cpp get-value, get-proof and
// get-unsat-core; cmd_context.cpp result availability checks.
package cmd_context

import (
	"errors"
	"fmt"
	"io"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/lbool"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func (c *Context) WriteExpression(out io.Writer, expression ast.ASTHandle) (result error) {
	format, err := c.manager.SMT2PPExprTree(expression, nil)
	if err != nil {
		return err
	}
	stream, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: c.manager.Budget()})
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, stream.Close()) }()
	empty := c.config.Params.EmptyValue()
	if err = c.manager.PrintFormat(stream, format, c.config.Params, &empty, c.config.Globals); err != nil {
		return err
	}
	text, err := stream.String()
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, text)
	return err
}

func (c *Context) WriteValues(out io.Writer, expressions []ast.ASTHandle) (result error) {
	md, err := c.Model()
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, md.DecRef()) }()
	evaluator, err := model.NewModelEvaluator(md, c.config.ModelEvaluator)
	if evaluator != nil {
		defer func() { result = errors.Join(result, evaluator.Close()) }()
	}
	if err != nil {
		return err
	}
	if err = evaluator.SetModelCompletion(true); err != nil {
		return err
	}
	if _, err = io.WriteString(out, "("); err != nil {
		return err
	}
	for i, expression := range expressions {
		if i > 0 {
			if _, err = io.WriteString(out, "\n "); err != nil {
				return err
			}
		}
		value, err := evaluator.EvaluateValue(expression)
		if err != nil {
			return err
		}
		err = c.writeValuePair(out, expression, value)
		err = errors.Join(err, c.manager.DelRef(&value))
		if err != nil {
			return err
		}
	}
	_, err = io.WriteString(out, ")\n")
	return err
}

func (c *Context) writeValuePair(out io.Writer, expression ast.ASTHandle, value ast.ASTRef) error {
	h, err := c.manager.RefGet(value)
	if err != nil {
		return err
	}
	if _, err = io.WriteString(out, "("); err != nil {
		return err
	}
	if err = c.WriteExpression(out, expression); err != nil {
		return err
	}
	if _, err = io.WriteString(out, " "); err != nil {
		return err
	}
	if err = c.WriteExpression(out, h); err != nil {
		return err
	}
	_, err = io.WriteString(out, ")")
	return err
}

func (c *Context) WriteUnsatCore(out io.Writer) error {
	state, err := c.parameters.State()
	if err != nil {
		return err
	}
	if !c.checked || c.status != lbool.False || !state.UnsatCore {
		return fmt.Errorf("unsat core is not available")
	}
	core, err := c.solver.UnsatCore()
	if err != nil {
		return err
	}
	if _, err = io.WriteString(out, "("); err != nil {
		return err
	}
	for i, expression := range core {
		if i > 0 {
			if _, err = io.WriteString(out, " "); err != nil {
				return err
			}
		}
		if err = c.WriteExpression(out, expression); err != nil {
			return err
		}
	}
	_, err = io.WriteString(out, ")\n")
	return err
}

func (c *Context) WriteProof(out io.Writer) error {
	state, err := c.parameters.State()
	if err != nil {
		return err
	}
	if !c.checked || c.status != lbool.False || !state.Proof {
		return fmt.Errorf("proof is not available")
	}
	proof, err := c.solver.Proof()
	if err != nil {
		return err
	}
	if proof.IsNull() {
		return fmt.Errorf("proof is not available")
	}
	if err = c.WriteExpression(out, proof); err != nil {
		return err
	}
	_, err = io.WriteString(out, "\n")
	return err
}
