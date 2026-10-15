// Source: src/tactic/aig/aig.cpp:863,881,931; src/util/trace.h:40-46.
package aig

import "io"

func (m *Manager) traceGoalConversion(n *node, line int, label string) {
	tags := m.ast.BorrowMemoryTraceTags()
	if !tags.ShouldTrace("aig2expr") {
		return
	}
	_ = tags.EmitConditional("aig2expr", "aig2expr", "src/tactic/aig/aig.cpp", line,
		func() bool { return true }, func(w io.Writer) error {
			_, _ = io.WriteString(w, label)
			writeAIGRef(w, Literal{node: n})
			_, _ = io.WriteString(w, "\n")
			return nil
		})
}
