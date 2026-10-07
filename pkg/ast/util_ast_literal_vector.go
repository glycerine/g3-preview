// Source: src/util/vector.h:43-125,215-240,303-310,350-365,389-433,449-455,531-546.
// Source: src/test/sorting_network.cpp:150-181.
// Source author: Leonardo de Moura; Nikolaj Bjorner.
// Licensed under the MIT license; see LICENSE.txt.
package ast

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// ASTHandle is the actual pointer-free16-byte pointer representation. Payload
// holds only raw handles, never owning AST references or a shadow Go slice.
type RuntimeASTLiteralVector struct {
	self                                                *RuntimeASTLiteralVector
	raw                                                 util.RuntimeVectorColumn
	budget                                              *arena.Budget
	debug                                               *util.DebugContext
	header                                              uint64
	constructed, sourceDone, closed                     bool
	destructorCause                                     error
	constructorCause, assignmentCause, copyCleanupCause error
	copyAcquisitions                                    *runtimeASTLiteralCopyAcquisition
	copySerial                                          uint64
	assignmentDestroyed, copyExecuting                  bool
	growthAcquisitions                                  *runtimeASTLiteralGrowthAcquisition
	growthSerial, growthInvocation                      uint64
	growthCause, growthPendingCause                     error
	growthExecuting                                     bool
	constructionFrame                                   *psortASTVectorFrame
	constructionIndex                                   uint32
}

func NewRuntimeASTLiteralVector(r *util.Runtime, b *arena.Budget, d *util.DebugContext) (*RuntimeASTLiteralVector, error) {
	if r == nil || b == nil || r.Budget() != b || (d != nil && d.Budget() != b) {
		return nil, arena.ErrConfig
	}
	if e := r.CheckOpen(); e != nil {
		return nil, e
	}
	header := uint64(reflect.TypeOf(RuntimeASTLiteralVector{}).Size())
	if e := b.Reserve(header); e != nil {
		return nil, e
	}
	v := &RuntimeASTLiteralVector{budget: b, debug: d, header: header, constructed: true}
	v.self = v
	v.raw.Bind(r)
	return v, nil
}
func (v *RuntimeASTLiteralVector) open() error {
	if v == nil || v.self != v {
		return util.ErrRuntimeVectorOwner
	}
	if v.growthExecuting || v.copyExecuting {
		return util.ErrRuntimeUnsignedGrowthExecuting
	}
	if v.closed || !v.constructed || v.sourceDone || v.destructorCause != nil {
		return arena.ErrClosed
	}
	return v.raw.Open()
}
func (v *RuntimeASTLiteralVector) Size() (uint32, error) {
	if e := v.open(); e != nil {
		return 0, e
	}
	return v.raw.Size()
}
func (v *RuntimeASTLiteralVector) Capacity() (uint32, error) {
	if e := v.open(); e != nil {
		return 0, e
	}
	return v.raw.Capacity()
}
func (v *RuntimeASTLiteralVector) index(i uint32, line int32) error {
	if e := v.open(); e != nil {
		return e
	}
	if util.RuntimeVectorSourceDebug {
		enabled := true
		if v.debug != nil {
			var e error
			enabled, e = v.debug.AssertionsEnabled()
			if e != nil {
				return e
			}
		}
		if enabled {
			size, e := v.raw.Size()
			if e != nil {
				return e
			}
			if i >= size {
				return util.DebugAssertionFailure(v.debug, "src/util/vector.h", line, "idx < size()")
			}
		}
	}
	return nil
}
func (v *RuntimeASTLiteralVector) At(i uint32) (ASTHandle, error) {
	if e := v.index(i, 309); e != nil {
		return ASTHandle{}, e
	}
	return (ASTLiteralVectorSpan{v.raw.Runtime, v.raw.Block, 8}).At(i)
}
func (v *RuntimeASTLiteralVector) Set(i uint32, h ASTHandle) error {
	if e := v.index(i, 324); e != nil {
		return e
	}
	return putASTLiteral(v.raw.Runtime, v.raw.Block, 8+16*uint64(i), h)
}
func putASTLiteral(r *util.Runtime, b util.Block, offset uint64, h ASTHandle) error {
	for i, x := range [4]uint32{uint32(h.origin), uint32(h.origin >> 32), h.slot, h.generation} {
		if e := util.WriteRuntimeWord(r, b, offset+4*uint64(i), x); e != nil {
			return e
		}
	}
	return nil
}
func (v *RuntimeASTLiteralVector) Push(h ASTHandle) error {
	if e := v.open(); e != nil {
		return e
	}
	return v.pushReached(h)
}
func (v *RuntimeASTLiteralVector) pushReached(h ASTHandle) error {
	size, e := v.raw.Size()
	if e != nil {
		return e
	}
	cap, e := v.raw.Capacity()
	if e != nil {
		return e
	}
	if size == cap {
		if e = v.raw.ExpandStrideOwned(16, v.debug, v); e != nil {
			return e
		}
	}
	if e = putASTLiteral(v.raw.Runtime, v.raw.Block, 8+16*uint64(size), h); e != nil {
		return e
	}
	return util.WriteRuntimeWord(v.raw.Runtime, v.raw.Block, 4, size+1)
}
func (v *RuntimeASTLiteralVector) Reset() error {
	if e := v.open(); e != nil {
		return e
	}
	if v.raw.Block == (util.Block{}) {
		return nil
	}
	return util.WriteRuntimeWord(v.raw.Runtime, v.raw.Block, 4, 0)
}

// CPtr captures source m_data only; it does not read SIZE or pointed bytes.
func (v *RuntimeASTLiteralVector) CPtr() (ASTLiteralVectorSpan, error) {
	if v == nil || v.self != v {
		return ASTLiteralVectorSpan{}, util.ErrRuntimeVectorOwner
	}
	if v.closed || !v.constructed || v.sourceDone || v.destructorCause != nil {
		return ASTLiteralVectorSpan{}, arena.ErrClosed
	}
	return ASTLiteralVectorSpan{v.raw.Runtime, v.raw.Block, 8}, nil
}

type ASTLiteralVectorSpan struct {
	runtime *util.Runtime
	block   util.Block
	offset  uint64
}

func (s ASTLiteralVectorSpan) At(i uint32) (ASTHandle, error) {
	var w [4]uint32
	base := s.offset + 16*uint64(i)
	if base < s.offset || base > ^uint64(0)-16 {
		return ASTHandle{}, arena.ErrBounds
	}
	for j := range w {
		x, e := util.ReadRuntimeWord(s.runtime, s.block, base+4*uint64(j))
		if e != nil {
			return ASTHandle{}, e
		}
		w[j] = x
	}
	return ASTHandle{uint64(w[0]) | uint64(w[1])<<32, w[2], w[3]}, nil
}
func (v *RuntimeASTLiteralVector) SourceFinalize() error {
	if v == nil || v.self != v {
		return util.ErrRuntimeVectorOwner
	}
	if v.growthExecuting || v.copyExecuting {
		return util.ErrRuntimeUnsignedGrowthExecuting
	}
	if v.closed || v.sourceDone || !v.constructed {
		return nil
	}
	if v.destructorCause != nil {
		return v.destructorCause
	}
	if v.raw.Block != (util.Block{}) {
		if e := v.raw.Runtime.Deallocate(&v.raw.Block); e != nil {
			v.destructorCause = e
			return e
		}
	}
	v.sourceDone = true
	return nil
}
func (v *RuntimeASTLiteralVector) closeOwning() error {
	if v.assignmentDestroyed {
		v.raw.Block = util.Block{}
		v.assignmentDestroyed = false
	}
	if e := v.raw.Finalize(); e != nil {
		return e
	}
	if e := v.cleanupGrowthAcquisitions(); e != nil {
		return e
	}
	if e := v.cleanupCopyAcquisitions(); e != nil {
		return e
	}
	v.budget.Release(v.header)
	v.header = 0
	v.closed = true
	return nil
}
func (v *RuntimeASTLiteralVector) Close() error {
	if v == nil || v.self != v {
		return util.ErrRuntimeVectorOwner
	}
	if v.closed {
		return nil
	}
	if e := v.SourceFinalize(); e != nil {
		return e
	}
	return v.closeOwning()
}
func (v *RuntimeASTLiteralVector) CleanupStopped() error {
	if v == nil || v.self != v || v.destructorCause == nil {
		return util.ErrRuntimeVectorOwner
	}
	if v.closed {
		return nil
	}
	return v.closeOwning()
}

// Reserve is source vector::reserve: resize to requested SIZE and default-null
// placement, rather than the unrelated capacity-only reserve adapter.
func (v *RuntimeASTLiteralVector) Reserve(n uint32) error { return v.Resize(n, ASTHandle{}) }
func (v *RuntimeASTLiteralVector) Resize(n uint32, fill ASTHandle) error {
	if e := v.open(); e != nil {
		return e
	}
	old, e := v.raw.Size()
	if e != nil {
		return e
	}
	if n <= old {
		if v.raw.Block == (util.Block{}) {
			return nil
		}
		return util.WriteRuntimeWord(v.raw.Runtime, v.raw.Block, 4, n)
	}
	for {
		capacity, e := v.raw.Capacity()
		if e != nil {
			return e
		}
		if n <= capacity {
			break
		}
		if e = v.raw.ExpandStrideOwned(16, v.debug, v); e != nil {
			return e
		}
	}
	if e = util.WriteRuntimeWord(v.raw.Runtime, v.raw.Block, 4, n); e != nil {
		return e
	}
	for i := old; i < n; i++ {
		if e = putASTLiteral(v.raw.Runtime, v.raw.Block, 8+16*uint64(i), fill); e != nil {
			return e
		}
	}
	return nil
}
func (v *RuntimeASTLiteralVector) PopBack() error {
	if e := v.open(); e != nil {
		return e
	}
	if util.RuntimeVectorSourceDebug {
		enabled := true
		if v.debug != nil {
			var e error
			enabled, e = v.debug.AssertionsEnabled()
			if e != nil {
				return e
			}
		}
		if enabled {
			size, e := v.raw.Size()
			if e != nil {
				return e
			}
			if size == 0 {
				if e = util.DebugAssertionFailure(v.debug, "src/util/vector.h", 344, "!empty()"); e != nil {
					return e
				}
			}
		}
	}
	size, e := v.raw.Size()
	if e != nil {
		return e
	}
	return util.WriteRuntimeWord(v.raw.Runtime, v.raw.Block, 4, size-1)
}

// Append preserves source vector::append(vector): live source SIZE and indexed
// read each iteration. Nonempty self-append has no terminating source domain.
func (v *RuntimeASTLiteralVector) Append(src *RuntimeASTLiteralVector) error {
	if e := v.open(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	for i := uint32(0); ; i++ {
		n, e := src.Size()
		if e != nil {
			return e
		}
		if i >= n {
			return nil
		}
		h, e := src.At(i)
		if e != nil {
			return e
		}
		if e = v.Push(h); e != nil {
			return e
		}
	}
}
func (v *RuntimeASTLiteralVector) AppendArray(n uint32, span ASTLiteralVectorSpan) error {
	if e := v.open(); e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		h, e := span.At(i)
		if e != nil {
			return e
		}
		if e = v.Push(h); e != nil {
			return e
		}
	}
	return nil
}
