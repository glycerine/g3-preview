// Source: src/util/memory_manager.cpp:202-211; src/util/gparams.cpp:633-644.
package params

import (
	"fmt"
	"io"
	"strconv"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// BindMemoryProcess borrows the source process finalization flag. Nil is used
// by standalone parameter owners that do not have a process runtime.
func (s *GlobalParamsService) BindMemoryProcess(p *umemory.ProcessAccounting) error {
	if e := s.open(); e != nil {
		return e
	}
	if p != nil && !p.Valid() {
		return umemory.ErrRuntimeOwner
	}
	s.memoryProcess = p
	return nil
}

func (s *GlobalParamsService) memoryTraceTags() *udiag.TraceTags {
	if s.namespaceTrace != nil {
		return s.namespaceTrace
	}
	return udiag.BorrowProcessTraceTags()
}

func gparamsMemoryParts(w io.Writer, parts ...string) error {
	for _, part := range parts {
		// Source ostream keeps inserting after badbit; TraceTags retains the
		// first transport failure and still attempts the footer and flush.
		_, _ = io.WriteString(w, part)
	}
	return nil
}

func (s *GlobalParamsService) traceMemoryAllocated(g *GlobalParams) {
	if !GlobalParamsSourceDebug {
		return
	}
	tags := s.memoryTraceTags()
	if !tags.ShouldTrace("memory") {
		return
	}
	_ = tags.EmitConditional("memory", "allocate", "src/util/memory_manager.cpp", 209,
		func() bool { return true }, func(w io.Writer) error {
			return gparamsMemoryParts(w, "alloc ", fmt.Sprintf("%p", g), " src/util/gparams.cpp:635 imp ", strconv.FormatUint(g.OwnerBytes, 10), "\n")
		})
}

func (s *GlobalParamsService) traceMemoryDeallocated(g *GlobalParams) {
	if !GlobalParamsSourceDebug {
		return
	}
	tags := s.memoryTraceTags()
	if !tags.ShouldTrace("memory") {
		return
	}
	_ = tags.EmitConditional("memory", "deallocate", "src/util/memory_manager.cpp", 204,
		func() bool { return s.memoryProcess == nil || !s.memoryProcess.Finalizing() }, func(w io.Writer) error {
			return gparamsMemoryParts(w, "dealloc ", fmt.Sprintf("%p", g), " src/util/gparams.cpp:641\n")
		})
}
