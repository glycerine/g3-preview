// Source: src/util/buffer.h:25-75,261-265;
// src/ast/rewriter/poly_rewriter_def.h:334-352; src/util/util.cpp:112-119.
package rewriter

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"reflect"
)

type bvPolynomialUnsignedBuffer struct {
	inline [16]uint32
	spill  []uint32
	count  uint32
	bytes  uint64
}

func (b *bvPolynomialUnsignedBuffer) data() []uint32 {
	if b.spill != nil {
		return b.spill
	}
	return b.inline[:b.count]
}
func (b *bvPolynomialUnsignedBuffer) push(m *ast.ASTManager, v uint32) error {
	if b.spill == nil && b.count < 16 {
		b.inline[b.count] = v
		b.count++
		return nil
	}
	if b.spill == nil || len(b.spill) == cap(b.spill) {
		capacity := uint64(32)
		if b.spill != nil {
			capacity = uint64(cap(b.spill)) * 2
		}
		if capacity > uint64(^uint32(0)) || capacity > uint64(^uint(0)>>1)/4 {
			return arena.ErrBounds
		}
		bytes := capacity * 4
		if e := m.Budget().Reserve(bytes); e != nil {
			return e
		}
		data := make([]uint32, b.count, int(capacity))
		copy(data, b.data())
		m.Budget().Release(b.bytes)
		b.spill, b.bytes = data, bytes
	}
	b.spill = append(b.spill, v)
	b.count++
	return nil
}
func (b *bvPolynomialUnsignedBuffer) close(m *ast.ASTManager) {
	m.Budget().Release(b.bytes)
	b.spill = nil
	b.count = 0
	b.bytes = 0
}

type bvPolynomialSumViews struct {
	inline [16]ast.AppArguments
	spill  []ast.AppArguments
	count  uint32
	bytes  uint64
}

func (b *bvPolynomialSumViews) data() []ast.AppArguments {
	if b.spill != nil {
		return b.spill
	}
	return b.inline[:b.count]
}
func (b *bvPolynomialSumViews) push(m *ast.ASTManager, v ast.AppArguments) error {
	if b.spill == nil && b.count < 16 {
		b.inline[b.count] = v
		b.count++
		return nil
	}
	if b.spill == nil || len(b.spill) == cap(b.spill) {
		capacity := uint64(32)
		if b.spill != nil {
			capacity = uint64(cap(b.spill)) * 2
		}
		width := uint64(reflect.TypeOf(ast.AppArguments{}).Size())
		if capacity > uint64(^uint32(0)) || capacity > uint64(^uint(0)>>1)/width {
			return arena.ErrBounds
		}
		bytes := capacity * width
		if e := m.Budget().Reserve(bytes); e != nil {
			return e
		}
		data := make([]ast.AppArguments, b.count, int(capacity))
		copy(data, b.data())
		m.Budget().Release(b.bytes)
		b.spill, b.bytes = data, bytes
	}
	b.spill = append(b.spill, v)
	b.count++
	return nil
}
func (b *bvPolynomialSumViews) close(m *ast.ASTManager) {
	m.Budget().Release(b.bytes)
	b.spill = nil
	b.count = 0
	b.bytes = 0
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
