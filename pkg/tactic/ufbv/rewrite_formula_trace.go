// Source: src/tactic/ufbv/ufbv_rewriter.cpp:668,736,748,753;
// src/util/trace.h:40-46.
package ufbv

import (
	"errors"
	"io"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func (r *formulaProcessor) formulaTraceTags() *diagnostic.TraceTags {
	if r.trace != nil {
		return r.trace
	}
	return diagnostic.BorrowProcessTraceTags()
}

func (r *formulaProcessor) traceFormulaText(h ast.ASTHandle) (body string, result error) {
	format, e := r.manager.SMT2PPExprTree(h, nil)
	if e != nil {
		return "", e
	}
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: r.manager.Budget()})
	if e != nil {
		return "", e
	}
	defer func() { result = errors.Join(result, out.Close()) }()
	if e := r.manager.PrintFormat(out, format, nil, nil, nil); e != nil {
		return "", e
	}
	return out.String()
}

func (r *formulaProcessor) traceFormulas(line int, heading string, formulas []ast.ASTHandle) error {
	tags := r.formulaTraceTags()
	if !tags.ShouldTrace("demodulator") {
		return nil
	}
	var operandErr error
	_ = tags.EmitConditional("demodulator", "operator()", "src/tactic/ufbv/ufbv_rewriter.cpp", line,
		func() bool { return true }, func(w io.Writer) error {
			_, _ = io.WriteString(w, heading)
			for _, h := range formulas {
				body, e := r.traceFormulaText(h)
				if e != nil {
					operandErr = e
					return e
				}
				_, _ = io.WriteString(w, body+"\n")
			}
			return nil
		})
	return operandErr
}
