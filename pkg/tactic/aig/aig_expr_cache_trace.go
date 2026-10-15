// Source: src/tactic/aig/aig.cpp:435-443,1548-1559; src/util/trace.h:40-46.
package aig

import (
	"io"
	"strconv"
)

func writeAIGRef(w io.Writer, l Literal) {
	if l.inverted {
		_, _ = io.WriteString(w, "-")
	}
	prefix, number := "#", l.node.id
	if l.node.and {
		prefix, number = "@", l.node.id-firstNodeID
	}
	_, _ = io.WriteString(w, prefix)
	_, _ = io.WriteString(w, strconv.FormatUint(uint64(number), 10))
}

func (m *Manager) traceExprCacheDecRef(l Literal) {
	tags := m.ast.BorrowMemoryTraceTags()
	if !tags.ShouldTrace("expr2aig") {
		return
	}
	_ = tags.EmitConditional("expr2aig", "~expr2aig", "src/tactic/aig/aig.cpp", 439,
		func() bool { return true }, func(w io.Writer) error {
			_, _ = io.WriteString(w, "dec-ref: ")
			writeAIGRef(w, l)
			_, _ = io.WriteString(w, " ref-count: ")
			_, _ = io.WriteString(w, strconv.FormatUint(uint64(l.node.refs), 10))
			_, _ = io.WriteString(w, "\n")
			return nil
		})
}
