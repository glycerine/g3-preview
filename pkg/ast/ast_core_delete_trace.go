// Source: src/ast/ast.cpp:1790-1792; src/util/trace.h.
package ast

import (
	"fmt"
	"io"
	"strconv"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

func (m *ASTManager) traceDeletingNode(n *astNodeRecord) {
	tags := m.traceTags
	if tags == nil {
		tags = udiag.BorrowProcessTraceTags()
	}
	if !tags.ShouldTrace("ast") && !tags.ShouldTrace("del_quantifier") && !tags.ShouldTrace("mk_var_bug") {
		return
	}
	id := strconv.FormatUint(uint64(n.id), 10)
	pointer := func(w io.Writer) { _, _ = fmt.Fprintf(w, "%p", n) }
	if tags.ShouldTrace("ast") {
		_ = tags.EmitConditional("ast", "delete_node", "src/ast/ast.cpp", 1790,
			func() bool { return true }, func(w io.Writer) error {
				_, _ = io.WriteString(w, "Deleting object ")
				_, _ = io.WriteString(w, id)
				_, _ = io.WriteString(w, " ")
				pointer(w)
				_, _ = io.WriteString(w, "\n")
				return nil
			})
	}
	if tags.ShouldTrace("del_quantifier") {
		_ = tags.EmitConditional("del_quantifier", "delete_node", "src/ast/ast.cpp", 1791,
			func() bool { return n.kind == ASTQuantifier }, func(w io.Writer) error {
				_, _ = io.WriteString(w, "deleting quantifier ")
				_, _ = io.WriteString(w, id)
				_, _ = io.WriteString(w, " ")
				pointer(w)
				_, _ = io.WriteString(w, "\n")
				return nil
			})
	}
	if tags.ShouldTrace("mk_var_bug") {
		_ = tags.EmitConditional("mk_var_bug", "delete_node", "src/ast/ast.cpp", 1792,
			func() bool { return true }, func(w io.Writer) error {
				_, _ = io.WriteString(w, "del_ast: ")
				_, _ = io.WriteString(w, id)
				_, _ = io.WriteString(w, "\n")
				return nil
			})
	}
}
