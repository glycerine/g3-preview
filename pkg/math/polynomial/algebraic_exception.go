// Source: src/math/polynomial/algebraic_numbers.h:41-44.
package polynomial

import "github.com/glycerine/g3/pkg/util/z3exception"

// AlgebraicException preserves the root-construction exception class and its
// default_exception base used by retained AST callers.
type AlgebraicException struct{ z3exception.DefaultException }

func (e AlgebraicException) Unwrap() error { return e.DefaultException }
func newAlgebraicException(message string) AlgebraicException {
	return AlgebraicException{z3exception.NewDefaultException(message)}
}
