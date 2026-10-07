// Source: src/util/vector.h, ptr_vector.
package ast

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"reflect"
)

var astHandleStorageBytes = uint64(reflect.TypeOf(ASTHandle{}).Size())

// ASTHandleVector stores borrowed handles; callers manage AST references.
type ASTHandleVector struct{ values []ASTHandle }

func (v *ASTHandleVector) Push(budget *arena.Budget, d ASTHandle) error {
	if len(v.values) == cap(v.values) {
		stride := astHandleStorageBytes
		n, e := util.SourceVectorCapacity(cap(v.values), stride)
		if e != nil {
			return e
		}
		if e = budget.Reserve(uint64(n) * stride); e != nil {
			return e
		}
		next := make([]ASTHandle, len(v.values), n)
		copy(next, v.values)
		budget.Release(uint64(cap(v.values)) * stride)
		v.values = next
	}
	v.values = append(v.values, d)
	return nil
}
func (v *ASTHandleVector) Erase(d ASTHandle) {
	for i, h := range v.values {
		if h == d {
			copy(v.values[i:], v.values[i+1:])
			v.values[len(v.values)-1] = ASTHandle{}
			v.values = v.values[:len(v.values)-1]
			return
		}
	}
}
func (v *ASTHandleVector) Close(budget *arena.Budget) {
	budget.Release(uint64(cap(v.values)) * astHandleStorageBytes)
	v.values = nil
}

func (v *ASTHandleVector) Slice() []ASTHandle { return v.values }
