// Source: src/util/buffer.h:25-75,261-265;
// src/ast/rewriter/poly_rewriter_def.h:334-352; src/util/util.cpp:112-119.
package rewriter

import "github.com/glycerine/g3/pkg/ast"

type bvPolynomialUnsignedBuffer struct{ values []uint32 }

func (b *bvPolynomialUnsignedBuffer) data() []uint32 { return b.values }
func (b *bvPolynomialUnsignedBuffer) push(_ *ast.ASTManager, v uint32) error {
	b.values = append(b.values, v)
	return nil
}
func (b *bvPolynomialUnsignedBuffer) close(_ *ast.ASTManager) {
	clear(b.values)
	b.values = nil
}

type bvPolynomialSumViews struct{ values []ast.AppArguments }

func (b *bvPolynomialSumViews) data() []ast.AppArguments { return b.values }
func (b *bvPolynomialSumViews) push(_ *ast.ASTManager, v ast.AppArguments) error {
	b.values = append(b.values, v)
	return nil
}
func (b *bvPolynomialSumViews) close(_ *ast.ASTManager) {
	clear(b.values)
	b.values = nil
}

// First coordinate advances fastest, with original unsigned wrap and carry.
func bvPolynomialProductNext(sz, it []uint32) bool {
	for i := range sz {
		it[i]++
		if it[i] < sz[i] {
			return true
		}
		it[i] = 0
	}
	return false
}
