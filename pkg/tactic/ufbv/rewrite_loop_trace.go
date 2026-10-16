// Source: src/tactic/ufbv/ufbv_rewriter.cpp:359,462; src/util/trace.h:40-46.
package ufbv

import (
	"io"

	"github.com/glycerine/g3/pkg/ast"
)

func (r *rewriteTraversal) traceRewrite(index *rewriteRuleIndex, line int, heading string, expr ast.ASTHandle) error {
	tags := index.traceTags()
	if !tags.ShouldTrace("demodulator") {
		return nil
	}
	var operandErr error
	_ = tags.EmitConditional("demodulator", "rewrite", "src/tactic/ufbv/ufbv_rewriter.cpp", line,
		func() bool { return true }, func(w io.Writer) error {
			_, _ = io.WriteString(w, heading)
			body, e := formulaTraceText(r.manager, expr)
			if e != nil {
				operandErr = e
				return e
			}
			_, _ = io.WriteString(w, body+"\n")
			return nil
		})
	return operandErr
}
