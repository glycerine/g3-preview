// Source: src/ast/ast.cpp:2195; retained util/trace TRACE call site.
package ast

import (
	"io"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

func (m *ASTManager) traceChainableApp() {
	tags := m.traceTags
	if tags == nil {
		tags = udiag.BorrowProcessTraceTags()
	}
	_ = tags.EmitConditional("chainable", "mk_app", "src/ast/ast.cpp", 2195,
		func() bool { return true }, func(w io.Writer) error {
			_, _ = io.WriteString(w, "chainable...\n")
			return nil
		})
}
