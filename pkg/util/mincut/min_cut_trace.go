// Source: src/util/min_cut.cpp:35, after edge publication.
package mincut

import (
	"fmt"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (m *MinCut) traceAddEdge(from, to uint32) {
	tags := m.trace
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	if !tags.ShouldTrace("spacer.mincut") {
		return
	}
	_ = tags.Emit("spacer.mincut", "add_edge", "src/util/min_cut.cpp", 35,
		fmt.Sprintf("adding edge (%d,%d)\n", from, to))
}
