// Source: src/ast/pattern/pattern_inference.cpp:439,455,479;
// src/util/util.h:171; src/util/memory_manager.cpp:202-211.
package pattern

import (
	"fmt"
	"io"
	"reflect"
	"strconv"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (c *patternCandidates) patternMemoryTags() *diagnostic.TraceTags {
	if c.traceTags != nil {
		return c.traceTags
	}
	return diagnostic.BorrowProcessTraceTags()
}

func (c *patternCandidates) tracePrePatternAllocated(p *prePattern, line int) {
	tags := c.patternMemoryTags()
	if !diagnostic.DebugBuild || !tags.ShouldTrace("memory") {
		return
	}
	_ = tags.EmitConditional("memory", "allocate", "src/util/memory_manager.cpp", 209,
		func() bool { return true }, func(w io.Writer) error {
			_, _ = io.WriteString(w, "alloc "+fmt.Sprintf("%p", p)+" src/ast/pattern/pattern_inference.cpp:"+
				strconv.Itoa(line)+" pre_pattern "+strconv.FormatUint(uint64(reflect.TypeOf(prePattern{}).Size()), 10)+"\n")
			return nil
		})
}

func (c *patternCandidates) tracePrePatternDeallocated(p *prePattern) {
	tags := c.patternMemoryTags()
	if !diagnostic.DebugBuild || !tags.ShouldTrace("memory") {
		return
	}
	_ = tags.EmitConditional("memory", "deallocate", "src/util/memory_manager.cpp", 204,
		func() bool { return !tags.MemoryFinalizing() }, func(w io.Writer) error {
			_, _ = io.WriteString(w, "dealloc "+fmt.Sprintf("%p", p)+" src/util/util.h:171\n")
			return nil
		})
}
