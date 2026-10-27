// Source: src/smt/theory_lra.cpp:2222-2223,2470-2502.
package smtcore

import (
	"errors"
	"fmt"
	"io"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (t *SMTLRATheory) emitLRA(function string, line int, body func(io.Writer) error) error {
	tags := t.trace
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	var operandErr error
	err := tags.EmitConditional("arith", function, "src/smt/theory_lra.cpp", line, func() bool { return true }, func(out io.Writer) error {
		// Source ostream badbit does not prevent later operands from running.
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

func (t *SMTLRATheory) displayEvidenceLiteral(out io.Writer, literal SMTLiteral) (result error) {
	m := t.state.manager
	var expression ast.ASTHandle
	var err error
	switch literal {
	case SMTTrueLiteral:
		expression, err = m.MkTrue()
	case SMTFalseLiteral:
		expression, err = m.MkFalse()
	default:
		expression, err = t.state.BoolExpression(literal.Var())
		if err == nil && literal.Sign() {
			expression, err = m.MkNot(expression)
		}
	}
	if err != nil {
		return err
	}
	ref, err := m.NewRef(expression)
	if err != nil {
		return err
	}
	defer func() {
		err := m.DelRef(&ref)
		if err != nil {
			t.pendingTraceRoots = append(t.pendingTraceRoots, ref)
		}
		result = errors.Join(result, err)
	}()
	printer, err := ast.NewSimpleASTPrinterContext(m, out, out)
	if err != nil {
		return err
	}
	return printer.DisplayExpr(out, expression, 0)
}

// Immediate registration has inequality sources only. The equality/definition
// producers are confined to the unreachable constructor-disabled delayed path.
func (t *SMTLRATheory) displayEvidence(out io.Writer, evidence []smtLRAEvidence) error {
	if _, err := t.State(); err != nil {
		return err
	}
	if out == nil {
		return ast.ErrASTHandle
	}
	for _, e := range evidence {
		if diagnostic.DebugBuild {
			d := diagnostic.BorrowProcessDebugContext()
			enabled, err := diagnostic.SourceAssertionsEnabled(d)
			if err != nil {
				return err
			}
			if enabled && e.coefficient.Sign() == 0 {
				if err = diagnostic.DebugAssertionFailure(d, "src/smt/theory_lra.cpp", 2473, "!ev.first.is_zero()"); err != nil {
					return err
				}
			}
		}
		if e.coefficient.Sign() == 0 {
			continue
		}
		if uint64(e.constraint) >= uint64(len(t.numeric.inequalities)) || t.numeric.inequalities[e.constraint] == SMTNullLiteral {
			return ast.ErrASTDomain
		}
		literal := t.numeric.inequalities[e.constraint]
		if err := t.displayEvidenceLiteral(out, literal); err != nil {
			return err
		}
		assignment, err := t.state.Assignment(literal)
		if err != nil {
			return err
		}
		if err = smtWriteTheoryText(out, " "+assignment.String()+"\n"); err != nil {
			return err
		}
	}
	for _, e := range evidence {
		if err := smtWriteTheoryText(out, fmt.Sprintf("%s: ", e.coefficient.RatString())); err != nil {
			return err
		}
		if err := t.numeric.solver.PrintConstraint(e.constraint, out); err != nil {
			return err
		}
	}
	return nil
}
