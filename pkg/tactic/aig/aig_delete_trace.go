// Source: src/tactic/aig/aig.cpp:149; src/util/trace.h:40-46.
package aig

import (
	"io"
	"strconv"
)

func (m *Manager) traceDeleting(n *node) {
	tags := m.ast.BorrowMemoryTraceTags()
	if !tags.ShouldTrace("aig_lit_count") {
		return
	}
	_ = tags.EmitConditional("aig_lit_count", "delete_node", "src/tactic/aig/aig.cpp", 149,
		func() bool { return true }, func(w io.Writer) error {
			_, _ = io.WriteString(w, "deleting: ")
			prefix, number := "#", n.id
			if n.and {
				prefix, number = "@", n.id-firstNodeID
			}
			_, _ = io.WriteString(w, prefix)
			_, _ = io.WriteString(w, strconv.FormatUint(uint64(number), 10))
			_, _ = io.WriteString(w, "\n")
			return nil
		})
}
