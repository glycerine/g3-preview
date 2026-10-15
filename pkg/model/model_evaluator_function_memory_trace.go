// Source: src/model/model_evaluator.cpp:250; src/model/model_core.cpp:29,81,105;
// src/util/memory_manager.cpp:202-211.
package model

import (
	"fmt"
	"io"
	"strconv"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func (f *FunctionInterpretation) traceEvaluatorFunctionAllocated(tags *diagnostic.TraceTags) {
	f.memoryTrace = tags
	f.memoryTraceLive = true
	if !diagnostic.DebugBuild || !tags.ShouldTrace("memory") {
		return
	}
	_ = tags.EmitConditional("memory", "allocate", "src/util/memory_manager.cpp", 209,
		func() bool { return true }, func(w io.Writer) error {
			for _, part := range [...]string{
				"alloc ", fmt.Sprintf("%p", f), " src/model/model_evaluator.cpp:250 func_interp ",
				strconv.FormatUint(f.bytes, 10), "\n",
			} {
				_, _ = io.WriteString(w, part)
			}
			return nil
		})
}

func (f *FunctionInterpretation) traceEvaluatorFunctionDeallocated(line int) {
	if !f.memoryTraceLive {
		return
	}
	f.memoryTraceLive = false
	tags := f.memoryTrace
	if !diagnostic.DebugBuild || !tags.ShouldTrace("memory") {
		return
	}
	_ = tags.EmitConditional("memory", "deallocate", "src/util/memory_manager.cpp", 204,
		func() bool { return !tags.MemoryFinalizing() }, func(w io.Writer) error {
			_, _ = io.WriteString(w, "dealloc "+fmt.Sprintf("%p", f)+" src/model/model_core.cpp:"+strconv.Itoa(line)+"\n")
			return nil
		})
}
