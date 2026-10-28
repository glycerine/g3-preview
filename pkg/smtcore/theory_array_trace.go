package smtcore

// Source: src/smt/theory_array_base.cpp:944-957.

import (
	"fmt"
	"io"

	"github.com/glycerine/g3/pkg/ast"
)

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
