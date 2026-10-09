// Source: src/util/buffer.h:25-48,98-104.
package ast

import (
	"github.com/glycerine/g3/pkg/arena"
)

// ASTHandleBuffer stores borrowed handles in ordinary Go-managed storage.
type ASTHandleBuffer struct{ values []ASTHandle }

func (a *ASTHandleBuffer) Slice() []ASTHandle { return a.values }
func (a *ASTHandleBuffer) Push(_ *arena.Budget, h ASTHandle) error {
	a.values = append(a.values, h)
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
func (a *ASTHandleBuffer) Close(_ *arena.Budget) { a.values = nil }

// Reset clears the logical contents and retains Go slice capacity.
func (a *ASTHandleBuffer) Reset() {
	clear(a.values)
	a.values = a.values[:0]
}

// PopBack decreases the logical size without releasing borrowed expressions
// or backing storage. Source: src/util/buffer.h:160-175.
func (a *ASTHandleBuffer) PopBack() error {
	if len(a.values) == 0 {
		return arena.ErrBounds
	}
	last := len(a.values) - 1
	a.values[last] = ASTHandle{}
	a.values = a.values[:last]
	return nil
}
