// Source: src/tactic/aig/aig.cpp:1261,1284; src/util/trace.h:40-46.
package aig

import (
	"io"
	"strconv"
)

func (m *Manager) traceSharingProcessing(n *node, index uint8) {
	tags := m.ast.BorrowMemoryTraceTags()
	if !tags.ShouldTrace("max_sharing") {
		return
	}
	_ = tags.EmitConditional("max_sharing", "process", "src/tactic/aig/aig.cpp", 1261,
		func() bool { return true }, func(w io.Writer) error {
			_, _ = io.WriteString(w, "processing ")
			writeAIGRef(w, Literal{node: n})
			_, _ = io.WriteString(w, " idx: ")
			_, _ = io.WriteString(w, strconv.FormatUint(uint64(index), 10))
			_, _ = io.WriteString(w, "\n")
			return nil
		})
}

func (m *Manager) traceSharingNull(null bool) {
	tags := m.ast.BorrowMemoryTraceTags()
	if !tags.ShouldTrace("max_sharing") {
		return
	}
	_ = tags.EmitConditional("max_sharing", "operator()", "src/tactic/aig/aig.cpp", 1284,
		func() bool { return true }, func(w io.Writer) error {
			value := "0"
			if null {
				value = "1"
			}
			_, _ = io.WriteString(w, "r.is_null(): "+value+"\n")
			return nil
		})
}
