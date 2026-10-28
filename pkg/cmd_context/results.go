package cmd_context

// Source: src/cmd_context/basic_cmds.cpp get-value, get-assignment, get-proof
// and get-unsat-core; cmd_context.cpp result availability checks.
// Original get-value term text follows smt2parser.cpp parse_get_value.

import (
	"errors"
	"fmt"
	"io"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/lbool"
	usmt2 "github.com/glycerine/g3/pkg/util/smt2"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
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
	return c.WriteValuesWithText(out, expressions, nil)
}

func (c *Context) WriteAssertions(out io.Writer) error {
	if err := c.valid(); err != nil {
		return err
	}
	if !c.InteractiveMode {
		return fmt.Errorf("command is only available in interactive mode, use command (set-option :interactive-mode true)")
	}
	if _, err := io.WriteString(out, "("); err != nil {
		return err
	}
	for i := range c.assertions {
		if i > 0 {
			if _, err := io.WriteString(out, "\n "); err != nil {
				return err
			}
		}
		if c.assertions[i].displayText != "" {
			if _, err := io.WriteString(out, c.assertions[i].displayText); err != nil {
				return err
			}
		} else if err := c.WriteExpression(out, c.assertions[i].expression); err != nil {
			return err
		}
	}
	_, err := io.WriteString(out, ")\n")
	return err
}

func (c *Context) WriteAssignment(out io.Writer) (result error) {
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
	boolSort, err := c.manager.MkBoolSort()
	if err != nil {
		return err
	}
	if _, err = io.WriteString(out, "("); err != nil {
		return err
	}
	first := true
	seen := make(map[string]bool)
	for _, name := range c.macroTrail {
		if seen[name] {
			continue
		}
		seen[name] = true
		for _, declaration := range c.macros[name] {
			if len(declaration.domain) != 0 {
				continue
			}
			sort, e := c.manager.GetSort(declaration.body)
			if e != nil {
				return e
			}
			if sort != boolSort {
				continue
			}
			value, e := evaluator.EvaluateValue(declaration.body)
			if e != nil {
				return e
			}
			h, e := c.manager.RefGet(value)
			if e == nil && (c.manager.IsTrue(h) || c.manager.IsFalse(h)) {
				e = c.writeAssignmentPair(out, name, c.manager.IsTrue(h), &first)
			}
			e = errors.Join(e, c.manager.DelRef(&value))
			if e != nil {
				return e
			}
		}
	}
	_, err = io.WriteString(out, ")\n")
	return err
}

func (c *Context) writeAssignmentPair(out io.Writer, name string, value bool, first *bool) error {
	symbol, err := c.manager.Symbols.Intern(name)
	if err != nil {
		return err
	}
	quoted, err := usmt2.IsSMT2QuotedSymbol(c.manager.Symbols, symbol)
	if err != nil {
		return err
	}
	if quoted {
		name, err = usmt2.MkSMT2QuotedSymbol(c.manager.Symbols, symbol, nil)
		if err != nil {
			return err
		}
	}
	if !*first {
		if _, err = io.WriteString(out, " "); err != nil {
			return err
		}
	}
	*first = false
	_, err = fmt.Fprintf(out, "(%s %t)", name, value)
	return err
}

// WriteValuesWithText evaluates parsed expressions while echoing their original
// SMT-LIB terms. A nil text slice uses the ordinary AST expression printer.
func (c *Context) WriteValuesWithText(out io.Writer, expressions []ast.ASTHandle, text []string) (result error) {
	if text != nil && len(text) != len(expressions) {
		return fmt.Errorf("get-value term text count does not match expression count")
	}
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
		term := ""
		if text != nil {
			term = text[i]
		}
		err = c.writeValuePair(out, expression, value, term)
		err = errors.Join(err, c.manager.DelRef(&value))
		if err != nil {
			return err
		}
	}
	_, err = io.WriteString(out, ")\n")
	return err
}

func (c *Context) writeValuePair(out io.Writer, expression ast.ASTHandle, value ast.ASTRef, text string) error {
	h, err := c.manager.RefGet(value)
	if err != nil {
		return err
	}
	if _, err = io.WriteString(out, "("); err != nil {
		return err
	}
	if text == "" {
		err = c.WriteExpression(out, expression)
	} else {
		_, err = io.WriteString(out, text)
	}
	if err != nil {
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
	core, err := c.checkSatResult().UnsatCore()
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
	if !state.Proof {
		return fmt.Errorf("proof construction is not enabled, use command (set-option :produce-proofs true)")
	}
	if !c.checked || c.status != lbool.False {
		return fmt.Errorf("proof is not available")
	}
	proof, err := c.checkSatResult().Proof()
	if err != nil {
		return err
	}
	if proof.IsNull() {
		return fmt.Errorf("proof is not available")
	}
	if state.WellSortedCheck {
		good, e := c.manager.IsWellSorted(proof, nil)
		if e != nil {
			return e
		}
		if !good {
			return fmt.Errorf("proof is not well sorted")
		}
	}
	pp, err := ast.NewPPParams(c.config.Params, c.config.Globals, nil)
	if err != nil {
		return err
	}
	pretty, err := pp.PrettyProof()
	if err = errors.Join(err, pp.Close()); err != nil {
		return err
	}
	if pretty {
		err = c.WriteExpression(out, proof)
	} else {
		printer, e := ast.NewLegacySMTPrinter(c.manager)
		if e != nil {
			return e
		}
		var logic usymbol.Symbol
		if c.logic != "" {
			logic, e = c.manager.Symbols.Intern(c.logic)
			if e != nil {
				return e
			}
		}
		declaredName := func(h ast.ASTHandle) (string, error) {
			name, e := c.manager.GetASTName(h)
			if e != nil {
				return "", e
			}
			return c.manager.Symbols.String(name)
		}
		config := ast.LegacySMTBenchmarkConfig{Logic: logic,
			IsDeclaredSort: func(h ast.ASTHandle) (bool, error) {
				name, e := declaredName(h)
				_, found := c.sorts[name]
				return found, e
			},
			IsDeclaredFunc: func(h ast.ASTHandle) (bool, error) { name, e := declaredName(h); return len(c.functions[name]) != 0, e },
		}
		err = printer.DisplayBenchmark(out, proof, config)
	}
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, "\n")
	return err
}
