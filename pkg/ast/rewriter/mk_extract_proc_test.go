// Source: src/ast/rewriter/mk_extract_proc.cpp:19-52.
package rewriter

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

func TestMkExtractProcOriginalCacheIdentityAndReferenceOrder(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	satOK(t, e)
	_, e = m.RegisterBitVectorPlugin()
	satOK(t, e)
	p, e := NewMkExtractProc(m)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, p.Close()) })
	if p.high != 0 || p.low != ^uint32(0) || !p.domain.IsNull() || !p.cached.IsNull() {
		t.Fatal("source initial cache")
	}
	s, e := m.BitVectorSort(8)
	satOK(t, e)
	x, e := m.MkVar(0, s)
	satOK(t, e)
	astTestPin(t, m, x)
	y, e := m.MkVar(1, s)
	satOK(t, e)
	astTestPin(t, m, y)
	beforeSort, e := m.GetRefCount(s)
	satOK(t, e)
	h, e := p.Apply(6, 2, x)
	satOK(t, e)
	astTestPin(t, m, h)
	d, e := m.GetDecl(h)
	satOK(t, e)
	if p.cached != d || p.domain != s || !p.acquired {
		t.Fatal("source cache publication")
	}
	rc, e := m.GetRefCount(d)
	satOK(t, e)
	// One application owns the declaration; the cache owns the other reference.
	if rc != 2 {
		t.Fatal("cache declaration refcount", rc)
	}
	afterSort, e := m.GetRefCount(s)
	satOK(t, e)
	if afterSort != beforeSort+1 {
		t.Fatal("cache separately acquired domain sort", beforeSort, afterSort)
	}
	beforeHit := m.Stats()
	// Native allocation pools reuse the candidate block; g3debug uses
	// direct allocations even when hash-consing returns the existing node.
	if !ast.ASTSourceDebug {
		m.Budget().FailAfter(0)
	}
	again, e := p.Apply(6, 2, x)
	m.Budget().FailAfter(-1)
	satOK(t, e)
	if again != h || m.Stats() != beforeHit {
		t.Fatal("cache hit must bypass declaration construction")
	}
	other, e := p.Apply(6, 2, y)
	satOK(t, e)
	astTestPin(t, m, other)
	otherDecl, e := m.GetDecl(other)
	satOK(t, e)
	if otherDecl != d {
		t.Fatal("cache key included expression")
	}
	changed, e := p.Apply(5, 1, x)
	satOK(t, e)
	astTestPin(t, m, changed)
	oldRC, e := m.GetRefCount(d)
	satOK(t, e)
	if oldRC != 2 {
		t.Fatal("replacement did not release previous cached declaration", oldRC)
	}
	newer, e := m.GetDecl(changed)
	satOK(t, e)
	if p.cached != newer || newer == d {
		t.Fatal("bounds cache key")
	}
	s16, e := m.BitVectorSort(16)
	satOK(t, e)
	z, e := m.MkVar(0, s16)
	satOK(t, e)
	astTestPin(t, m, z)
	_, e = p.Apply(5, 1, z)
	satOK(t, e)
	if p.domain != s16 || p.cached == newer {
		t.Fatal("domain sort cache key")
	}
	copyOwner := *p
	if _, e = copyOwner.Apply(5, 1, z); e != ast.ErrASTHandle {
		t.Fatal("copied cache owner", e)
	}
	satOK(t, p.Close())
	satOK(t, p.Close())
	if _, e = p.Apply(5, 1, z); e != ast.ErrASTHandle {
		t.Fatal("closed cache owner", e)
	}
}

func TestMkExtractProcReplacementEveryColdReservation(t *testing.T) {
	success := false
	failures := 0
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			m := astTestManager(t)
			_, e := m.RegisterArithmeticPlugin()
			satOK(t, e)
			_, e = m.RegisterBitVectorPlugin()
			satOK(t, e)
			p, e := NewMkExtractProc(m)
			satOK(t, e)
			t.Cleanup(func() { satOK(t, p.Close()) })
			s, e := m.BitVectorSort(65)
			satOK(t, e)
			x, e := m.MkVar(0, s)
			satOK(t, e)
			astTestPin(t, m, x)
			old, e := p.Apply(63, 1, x)
			satOK(t, e)
			astTestPin(t, m, old)
			oldDecl, e := m.GetDecl(old)
			satOK(t, e)
			m.Budget().FailAfter(ordinal)
			_, body := p.Apply(62, 2, x)
			m.Budget().FailAfter(-1)
			rc, e := m.GetRefCount(oldDecl)
			satOK(t, e)
			if rc != 1 {
				t.Fatal("old cache release must precede replacement factory", ordinal, rc)
			}
			if body != nil {
				failures++
				if !errors.Is(body, arena.ErrInjectedFailure) {
					t.Fatal("unexpected refusal", body)
				}
				if p.acquired {
					t.Fatal("failed replacement still owns released cache")
				}
				if _, e = p.Apply(63, 1, x); e != ast.ErrASTPending {
					t.Fatal("unsafe retry after source prefix", e)
				}
			} else {
				success = true
				if !p.acquired {
					t.Fatal("successful cache not acquired")
				}
			}
			// Owning Close releases only actually acquired cache refs. Fixture closure
			// checks the full shared budget, including factory failure roots, is zero.
		})
		if success {
			break
		}
	}
	if !success || failures == 0 {
		t.Fatal("reservation horizon", success, failures)
	}
}
