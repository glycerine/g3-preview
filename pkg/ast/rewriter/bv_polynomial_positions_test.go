// Source: src/util/obj_hashtable.h:58-159; src/util/hashtable.h:286-308,377-396,494-510;
// src/ast/rewriter/poly_rewriter_def.h:570-586.
package rewriter

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

func bvPositionKeys(t *testing.T, m *ast.ASTManager, n int) []ast.ASTHandle {
	t.Helper()
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	keys := make([]ast.ASTHandle, n)
	for i := range keys {
		keys[i], e = m.MkVar(uint32(i), s)
		astTestMust(t, e)
		astTestPin(t, m, keys[i])
	}
	return keys
}

func TestBVPolynomialPositionsGrowthBeforeDuplicateAndBorrowedKeys(t *testing.T) {
	m, _ := bvAddFixture(t)
	keys := bvPositionKeys(t, m, 30)
	before := m.Budget().Used()
	p, e := newBVPolynomialPositions(m)
	astTestMust(t, e)
	defer func() {
		p.close()
		if m.Budget().Used() != before {
			t.Fatal("table storage leaked")
		}
	}()
	for i := 0; i < 7; i++ {
		rc, e := m.GetRefCount(keys[i])
		astTestMust(t, e)
		astTestMust(t, p.insert(keys[i], uint32(i)))
		after, e := m.GetRefCount(keys[i])
		astTestMust(t, e)
		if rc != after {
			t.Fatal("table acquired key")
		}
	}
	if p.capacity != 8 {
		t.Fatal("premature growth")
	}
	used := m.Budget().Used()
	m.Budget().FailAfter(0)
	e = p.insert(keys[0], 99)
	m.Budget().FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || p.capacity != 8 || p.size != 7 || m.Budget().Used() != used {
		t.Fatal("growth refusal changed table", e)
	}
	v, ok, e := p.find(keys[0])
	astTestMust(t, e)
	if !ok || v != 0 {
		t.Fatal("duplicate updated before growth")
	}
	astTestMust(t, p.insert(keys[0], 99))
	if p.capacity != 16 || p.size != 7 {
		t.Fatal("duplicate did not grow first")
	}
	for i := 7; i < 30; i++ {
		astTestMust(t, p.insert(keys[i], uint32(i)))
	}
	for i, key := range keys {
		want := uint32(i)
		if i == 0 {
			want = 99
		}
		got, ok, e := p.find(key)
		astTestMust(t, e)
		if !ok || got != want {
			t.Fatal("rehash lost position", i, got)
		}
	}
	capacity := p.capacity
	astTestMust(t, p.reset())
	if p.capacity != capacity || p.size != 0 {
		t.Fatal("dense reset shrank")
	}
	astTestMust(t, p.insert(keys[0], 8))
	astTestMust(t, p.reset())
	if p.capacity != capacity/2 {
		t.Fatal("sparse reset did not halve capacity")
	}
	used = m.Budget().Used()
	m.Budget().FailAfter(0)
	astTestMust(t, p.reset())
	astTestMust(t, p.insert(keys[1], 42))
	v, ok, e = p.find(keys[1])
	astTestMust(t, e)
	m.Budget().FailAfter(-1)
	if !ok || v != 42 || m.Budget().Used() != used {
		t.Fatal("retained storage not reused")
	}
}

func TestBVPolynomialPositionsCollisionWrapAndStaleKeyReset(t *testing.T) {
	m, _ := bvAddFixture(t)
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	astTestPin(t, m, s)
	var keys []ast.ASTHandle
	var pins []ast.ASTRef
	for i := uint32(0); len(keys) < 4 && i < 256; i++ {
		h, e := m.MkVar(i, s)
		astTestMust(t, e)
		r, e := m.NewRef(h)
		astTestMust(t, e)
		hash, e := m.GetHash(h)
		astTestMust(t, e)
		if hash&7 == 7 {
			keys = append(keys, h)
			pins = append(pins, r)
		} else {
			astTestMust(t, m.DelRef(&r))
		}
	}
	t.Cleanup(func() {
		for i := range pins {
			if pins[i] != (ast.ASTRef{}) {
				astTestMust(t, m.DelRef(&pins[i]))
			}
		}
	})
	if len(keys) != 4 {
		t.Fatal("missing collision fixture")
	}
	p, e := newBVPolynomialPositions(m)
	astTestMust(t, e)
	defer p.close()
	for i, h := range keys {
		astTestMust(t, p.insert(h, uint32(i+1)))
	}
	for i, index := range []int{7, 0, 1, 2} {
		if p.entries[index].key != keys[i] {
			t.Fatal("linear probing did not wrap")
		}
	}
	for i, h := range keys {
		v, ok, e := p.find(h)
		astTestMust(t, e)
		if !ok || v != uint32(i+1) {
			t.Fatal("collision lookup")
		}
	}
	// Borrowed keys may die before reset/destruction, neither of which dereferences them.
	for i := range pins {
		astTestMust(t, m.DelRef(&pins[i]))
	}
	if _, e := m.GetHash(keys[0]); e == nil {
		t.Fatal("table retained deleted key")
	}
	astTestMust(t, p.reset())
	if p.size != 0 || p.entries[7].position != 1 {
		t.Fatal("reset must clear keys only")
	}
}

func TestBVPolynomialPositionsEveryConstructionAndSparseResetReservation(t *testing.T) {
	m, _ := bvAddFixture(t)
	keys := bvPositionKeys(t, m, 14)
	before := m.Budget().Used()
	for ordinal := int64(0); ordinal < 3; ordinal++ {
		m.Budget().FailAfter(ordinal)
		p, e := newBVPolynomialPositions(m)
		m.Budget().FailAfter(-1)
		if ordinal < 2 {
			if p != nil || !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal("construction refusal", ordinal, e)
			}
		} else {
			astTestMust(t, e)
			p.close()
		}
		if m.Budget().Used() != before {
			t.Fatal("constructor leaked")
		}
	}
	p, e := newBVPolynomialPositions(m)
	astTestMust(t, e)
	defer p.close()
	for i, h := range keys {
		astTestMust(t, p.insert(h, uint32(i)))
	}
	astTestMust(t, p.reset())
	astTestMust(t, p.insert(keys[0], 0))
	used, bytes := m.Budget().Used(), p.bytes
	m.Budget().FailAfter(0)
	e = p.reset()
	m.Budget().FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || p.capacity != 16 || p.entries != nil || p.bytes != 0 || m.Budget().Used() != used-bytes {
		t.Fatal("sparse reset did not release before allocation", e)
	}
	if e := p.insert(keys[0], 1); !errors.Is(e, ast.ErrASTPending) {
		t.Fatal("stopped reset retried", e)
	}
	p.close()
	p.close()
	if m.Budget().Used() != before {
		t.Fatal("stopped reset cleanup leaked")
	}
}
