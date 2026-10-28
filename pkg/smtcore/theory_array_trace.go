package smtcore

// Source: src/smt/theory_array_base.cpp:944-957.

import (
	"fmt"
	"io"

	"github.com/glycerine/g3/pkg/ast"
)

func (t *SMTArrayTheory) emitArrayTrace(tag, function string, line int, condition func() bool, body func(io.Writer) error) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	tags := state.manager.BorrowSourceTraceTags()
	if tags == nil || !tags.ShouldTrace(tag) {
		return nil
	}
	if condition == nil {
		condition = func() bool { return true }
	}
	var operandErr error
	err = tags.EmitConditional(tag, function, "src/smt/theory_array_base.cpp", line, condition, func(out io.Writer) error {
		operandErr = body(assertedTraceWriter{out})
		return operandErr
	})
	if operandErr != nil {
		return operandErr
	}
	if tags.IsStreamFailure(err) {
		return nil
	}
	return err
}

func (t *SMTArrayTheory) traceArrayExpressions(tag, function string, line int, prefix string, expressions ...ast.ASTHandle) error {
	return t.emitArrayTrace(tag, function, line, nil, func(out io.Writer) error {
		if _, err := io.WriteString(out, prefix); err != nil {
			return err
		}
		state, err := t.State()
		if err != nil {
			return err
		}
		for index, expression := range expressions {
			if index != 0 {
				if _, err = io.WriteString(out, " "); err != nil {
					return err
				}
			}
			if err = displayArrayTraceExpression(state.manager, out, expression); err != nil {
				return err
			}
		}
		_, err = io.WriteString(out, "\n")
		return err
	})
}

func (t *SMTArrayTheory) assertArrayAxiom(literals []SMTLiteral) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	if err = t.emitArrayTrace("array_axiom", "assert_axiom", 70, nil, func(out io.Writer) error {
		if _, err := io.WriteString(out, "literals:\n"); err != nil {
			return err
		}
		for _, literal := range literals {
			if literal.Sign() {
				if _, err := io.WriteString(out, "not "); err != nil {
					return err
				}
			}
			expression, err := smtLiteralExpression(literal, state.boolExpr)
			if err != nil {
				return err
			}
			if err = displayArrayTraceExpression(state.manager, out, expression); err != nil {
				return err
			}
			if _, err = io.WriteString(out, " \n"); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return state.addTheoryAxiom(t.FamilyID(), literals)
}

func (t *SMTArrayTheory) traceArrayModelValue(generator *SMTModelGenerator, node *SMTEnode, reads []*SMTEnode, fallback *SMTEnode) error {
	if generator == nil || node == nil || node.root == nil {
		return ast.ErrASTDomain
	}
	tags := generator.traceTags()
	err := tags.EmitConditional("array", "mk_value", "src/smt/theory_array_base.cpp", 944, func() bool { return true }, func(raw io.Writer) error {
		out := assertedTraceWriter{raw}
		if err := displayArrayTraceExpression(generator.manager, out, node.root.owner); err != nil {
			return err
		}
		if _, err := io.WriteString(out, "\n"); err != nil {
			return err
		}
		for _, read := range reads {
			if read == nil || read.root == nil {
				return ast.ErrASTDomain
			}
			id, err := read.root.OwnerID()
			if err != nil {
				return err
			}
			if _, err = fmt.Fprintf(out, "#%d ", id); err != nil {
				return err
			}
			if err = displayArrayTraceExpression(generator.manager, out, read.owner); err != nil {
				return err
			}
			if _, err = io.WriteString(out, "\n"); err != nil {
				return err
			}
		}
		if fallback != nil {
			if _, err := io.WriteString(out, "else: "); err != nil {
				return err
			}
			if err := displayArrayTraceExpression(generator.manager, out, fallback.owner); err != nil {
				return err
			}
			_, err := io.WriteString(out, "\n")
			return err
		}
		return nil
	})
	if tags.IsStreamFailure(err) {
		return nil
	}
	return err
}

func displayArrayTraceExpression(manager *ast.ASTManager, out io.Writer, expression ast.ASTHandle) error {
	printer, err := ast.NewSimpleASTPrinterContext(manager, out, out)
	if err != nil {
		return err
	}
	return printer.DisplayExpr(out, expression, 0)
}
