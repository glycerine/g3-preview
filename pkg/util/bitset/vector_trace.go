// Source: src/util/bit_vector.cpp:43-68, the two growth TRACE sites.
package bitset

import (
	"fmt"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// BindTrace borrows an isolated trace stream. Without one, resize uses the
// source process-wide stream. A nil binding restores that process stream.
func (v *Vector) BindTrace(tags *diagnostic.TraceTags) { v.trace = tags }

func (v *Vector) resizeTraceTags() *diagnostic.TraceTags {
	if v.trace != nil {
		return v.trace
	}
	return diagnostic.BorrowProcessTraceTags()
}

func (v *Vector) traceResizeStart(n, words uint32) {
	tags := v.resizeTraceTags()
	if !tags.ShouldTrace("bit_vector") {
		return
	}
	_ = tags.Emit("bit_vector", "resize", "src/util/bit_vector.cpp", 43,
		fmt.Sprintf("expanding: %d capacity: %d num words: %d\n", n, cap(v.words), words))
}

func (v *Vector) traceResizeFields(begin, end, pos, mask uint32, val bool) {
	tags := v.resizeTraceTags()
	if !tags.ShouldTrace("bit_vector") {
		return
	}
	cval := 0
	if val {
		cval = -1
	}
	_ = tags.Emit("bit_vector", "resize", "src/util/bit_vector.cpp", 67,
		fmt.Sprintf("num_bits: %d\nbwidx:    %d\newidx:    %d\npos:      %d\nmask:     %x\ncval:     %d\n",
			v.size, begin, end, pos, mask, cval))
}
