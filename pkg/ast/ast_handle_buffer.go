// Source: src/util/buffer.h:25-48,98-104.
package ast

import (
	"github.com/glycerine/g3/pkg/arena"
	"math"
)

// Lexical source ptr_buffer<expr>: inline16, doubling overflow, full old+new
// charge. Handles are scalar; the helper retains no borrowed caller backing.
type ASTHandleBuffer struct {
	inline  [16]ASTHandle
	count   int
	spill   []ASTHandle
	charged uint64
}

func (a *ASTHandleBuffer) Slice() []ASTHandle {
	if a.spill == nil {
		return a.inline[:a.count]
	}
	return a.spill
}
func (a *ASTHandleBuffer) Push(budget *arena.Budget, h ASTHandle) error {
	if a.spill == nil && a.count < len(a.inline) {
		a.inline[a.count] = h
		a.count++
		return nil
	}
	if a.spill == nil || len(a.spill) == cap(a.spill) {
		n := uint64(32)
		if a.spill != nil {
			n = uint64(cap(a.spill)) * 2
		}
		width := astHandleStorageBytes
		if n > math.MaxUint32 || n > uint64(^uint(0)>>1)/width {
			return arena.ErrBounds
		}
		bytes := n * width
		if e := budget.Reserve(bytes); e != nil {
			return e
		}
		next := make([]ASTHandle, a.count, int(n))
		copy(next, a.Slice())
		budget.Release(a.charged)
		a.charged = bytes
		a.spill = next
	}
	a.spill = append(a.spill, h)
	a.count++
	return nil
}
func (a *ASTHandleBuffer) Append(budget *arena.Budget, hs []ASTHandle) error {
	for _, h := range hs {
		if e := a.Push(budget, h); e != nil {
			return e
		}
	}
	return nil
}
func (a *ASTHandleBuffer) Close(budget *arena.Budget) {
	budget.Release(a.charged)
	a.spill = nil
	a.count = 0
	a.charged = 0
}

// Reset follows buffer::reset: clear the logical size and retain spill capacity.
func (a *ASTHandleBuffer) Reset() {
	a.count = 0
	if a.spill != nil {
		a.spill = a.spill[:0]
	}
}

// PopBack decreases the logical size without releasing borrowed expressions
// or backing storage. Source: src/util/buffer.h:160-175.
func (a *ASTHandleBuffer) PopBack() error {
	if a.count == 0 {
		return arena.ErrBounds
	}
	a.count--
	if a.spill != nil {
		a.spill = a.spill[:a.count]
	}
	return nil
}
