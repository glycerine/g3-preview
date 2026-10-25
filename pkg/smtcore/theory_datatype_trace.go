// Source: src/smt/theory_datatype.cpp retained internalization/equality TRACE.
package smtcore

import (
	"errors"
	"io"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func (t *SMTDatatypeTheory) emitDatatype(tag, function string, line int, body func(io.Writer) error) error {
	tags := t.trace
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	var operandErr error
	err := tags.EmitConditional(tag, function, "src/smt/theory_datatype.cpp", line, func() bool { return true }, func(out io.Writer) error {
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

func (t *SMTDatatypeTheory) displayDatatypeExpression(out io.Writer, expression ast.ASTHandle) (result error) {
	s, err := t.State()
	if err != nil {
		return err
	}
	format, err := s.manager.SMT2PPExprTree(expression, nil)
	if err != nil {
		return err
	}
	stream, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: s.manager.Budget()})
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, stream.Close()) }()
	if err = s.manager.PrintFormat(stream, format, nil, nil, nil); err != nil {
		return err
	}
	body, err := stream.String()
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, body)
	return err
}

func (t *SMTDatatypeTheory) traceDatatypeExpression(function string, line int, heading string, expression ast.ASTHandle) error {
	return t.emitDatatype("datatype", function, line, func(out io.Writer) error {
		_, _ = io.WriteString(out, heading)
		if err := t.displayDatatypeExpression(out, expression); err != nil {
			return err
		}
		_, err := io.WriteString(out, "\n")
		return err
	})
}
