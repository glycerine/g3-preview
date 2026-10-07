// Source: src/ast/ast.h:934-938; src/util/vector.h:59-100,349-363.
package ast

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

type ASTBuiltinName struct {
	Kind int32
	Name util.Symbol
}

// Caller-owned concrete svector<builtin_name> counterpart. Items borrows the
// backing; symbols borrow the manager's symbol service. Close releases backing.
// Use by pointer; do not copy an active vector.
type ASTBuiltinNames struct {
	items  []ASTBuiltinName
	budget *arena.Budget
}

func (v *ASTBuiltinNames) Items() []ASTBuiltinName { return v.items }
func (v *ASTBuiltinNames) Reset()                  { v.items = v.items[:0] }
func (v *ASTBuiltinNames) Close() {
	if v.budget != nil {
		v.budget.Release(uint64(cap(v.items)) * uint64(reflect.TypeOf(ASTBuiltinName{}).Size()))
	}
	v.items, v.budget = nil, nil
}
func (v *ASTBuiltinNames) push(b *arena.Budget, name util.Symbol, kind int32) error {
	if v.budget != nil && v.budget != b {
		return ErrASTDomain
	}
	if len(v.items) == cap(v.items) {
		n := (uint64(cap(v.items))*3 + 1) / 2
		if n < 2 {
			n = 2
		}
		width := uint64(reflect.TypeOf(ASTBuiltinName{}).Size())
		if n > math.MaxUint32 || n > uint64(^uint(0)>>1)/width {
			return arena.ErrBounds
		}
		if e := b.Reserve(n * width); e != nil {
			return e
		}
		items := make([]ASTBuiltinName, len(v.items), int(n))
		copy(items, v.items)
		b.Release(uint64(cap(v.items)) * width)
		v.items, v.budget = items, b
	}
	v.items = append(v.items, ASTBuiltinName{Kind: kind, Name: name})
	return nil
}
