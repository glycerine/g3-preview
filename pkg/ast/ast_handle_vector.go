// Source: src/util/vector.h, ptr_vector.
package ast

import (
	"github.com/glycerine/g3/pkg/arena"
)

// ASTHandleVector stores borrowed handles; callers manage AST references.
type ASTHandleVector struct{ values []ASTHandle }

func (v *ASTHandleVector) Push(_ *arena.Budget, d ASTHandle) error {
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
func (v *ASTHandleVector) Close(_ *arena.Budget) {
	v.values = nil
}

func (v *ASTHandleVector) Slice() []ASTHandle { return v.values }
