package smtcore

// Source: retained TRACE/CTRACE sites in src/smt/theory_fpa.cpp.

import (
	"fmt"
	"io"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (t *SMTFloatingPointTheory) fpaTraceTags() (*diagnostic.TraceTags, error) {
	state, err := t.State()
	if err != nil {
		return nil, err
	}
	return state.manager.BorrowSourceTraceTags(), nil
}

func (t *SMTFloatingPointTheory) emitFPATrace(tag, function string, line int, condition func() bool, body func(io.Writer) error) error {
	tags, err := t.fpaTraceTags()
	if err != nil || !tags.ShouldTrace(tag) {
		return err
	}
	if condition == nil {
		condition = func() bool { return true }
	}
	var operandErr error
	err = tags.EmitConditional(tag, function, "src/smt/theory_fpa.cpp", line, condition, func(out io.Writer) error {
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

func (t *SMTFloatingPointTheory) displayFPATraceExpression(out io.Writer, expression ast.ASTHandle) error {
	state, err := t.State()
	if err != nil {
		return err
	}
	printer, err := ast.NewSimpleASTPrinterContext(state.manager, out, out)
	if err != nil {
		return err
	}
	return printer.DisplayExpr(out, expression, 0)
}

func (t *SMTFloatingPointTheory) traceFPAExpression(tag, function string, line int, prefix string, expression ast.ASTHandle) error {
	return t.emitFPATrace(tag, function, line, nil, func(out io.Writer) error {
		if _, err := io.WriteString(out, prefix); err != nil {
			return err
		}
		if err := t.displayFPATraceExpression(out, expression); err != nil {
			return err
		}
		return diagnostic.WriteTraceEndl(out)
	})
}

func (t *SMTFloatingPointTheory) traceFPAConversion(line int, heading string, input, output ast.ASTHandle) error {
	return t.emitFPATrace("t_fpa_detail", "convert", line, nil, func(out io.Writer) error {
		if _, err := io.WriteString(out, heading+"\n"); err != nil {
			return err
		}
		if err := t.displayFPATraceExpression(out, input); err != nil {
			return err
		}
		if _, err := io.WriteString(out, "\n -> \n"); err != nil {
			return err
		}
		if err := t.displayFPATraceExpression(out, output); err != nil {
			return err
		}
		return diagnostic.WriteTraceEndl(out)
	})
}

func (t *SMTFloatingPointTheory) createFPAVariable(node *SMTEnode) (SMTTheoryVar, error) {
	variable, err := t.CreateVariable(node)
	if err != nil {
		return SMTNullTheoryVar, err
	}
	err = t.emitFPATrace("t_fpa", "attach_new_th_var", 422, nil, func(out io.Writer) error {
		if _, writeErr := io.WriteString(out, "new theory var: "); writeErr != nil {
			return writeErr
		}
		if displayErr := t.displayFPATraceExpression(out, node.Owner()); displayErr != nil {
			return displayErr
		}
		_, writeErr := fmt.Fprintf(out, " := %d\n", variable)
		return writeErr
	})
	if err != nil {
		return SMTNullTheoryVar, err
	}
	return variable, nil
}

func (t *SMTFloatingPointTheory) traceFPAPop(count uint32, scopes int) error {
	return t.emitFPATrace("t_fpa", "pop_scope_eh", 625, nil, func(out io.Writer) error {
		_, err := fmt.Fprintf(out, "pop %d; now %d\n", count, scopes)
		return err
	})
}
