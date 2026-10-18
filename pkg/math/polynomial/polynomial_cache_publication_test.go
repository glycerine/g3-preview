package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestPolynomialHashZeroReadsOnlySize(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	p := polynomialQueryTerms(t, m.core, []int32{1}, [][]MonomialPower{nil})
	l, e := m.core.polynomialLocation(p)
	serviceMust(t, e)
	a := m.core.monomials.allocator
	raw, e := a.Allocate(12)
	serviceMust(t, e)
	original := l.raw
	l.raw = raw
	defer func() { l.raw = original; serviceMust(t, a.Deallocate(12, raw)) }()
	// Query a raw scalar prefix, without claiming construction or ownership.
	// The genuine owning cell is restored before any destructor runs.
	serviceMust(t, a.WriteUint32(raw, 8, 0))
	h, e := m.Hash(p)
	serviceMust(t, e)
	if h != 31 {
		t.Fatal("source zero-polynomial hash", h)
	}
	for _, offset := range []uint64{0, 4} {
		if _, e := a.ReadUint32(raw, offset); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
			t.Fatal("hash touched unrelated header field", offset, e)
		}
	}
}

func TestPolynomialHashNonzeroDoesNotReadReferenceCount(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	p := polynomialQueryTerms(t, m.core, []int32{7, 3, 5}, [][]MonomialPower{nil, {{0, 2}}, {{1, 1}}})
	want, e := m.Hash(p)
	serviceMust(t, e)
	l, e := m.core.polynomialLocation(p)
	serviceMust(t, e)
	a := m.core.monomials.allocator
	n := uint64(polynomialCodecBytes(3))
	raw, e := a.Allocate(n)
	serviceMust(t, e)
	original := l.raw
	// Borrow genuine coefficients/monomials into a query-only scalar prefix;
	// leave refcount unreadable and require actual lexical sorting again.
	for offset := uint64(4); offset < n; offset += 4 {
		word, e := a.ReadUint32(original, offset)
		serviceMust(t, e)
		if offset == 4 {
			word &^= 0x80000000
		}
		serviceMust(t, a.WriteUint32(raw, offset, word))
	}
	l.raw = raw
	defer func() { l.raw = original; serviceMust(t, a.Deallocate(n, raw)) }()
	got, e := m.Hash(p)
	serviceMust(t, e)
	if got != want {
		t.Fatal("reference-independent hash changed", got, want)
	}
	if _, e := a.ReadUint32(raw, 0); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
		t.Fatal("nonzero hashing initialized refcount", e)
	}
}

func TestPolynomialHashChildReadsOnlyMonomialHash(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	p := polynomialQueryTerms(t, m.core, []int32{7}, [][]MonomialPower{{{0, 2}}})
	want, e := m.Hash(p)
	serviceMust(t, e)
	mono, e := m.core.monomialCore(p, 0)
	serviceMust(t, e)
	l, e := m.core.monomials.location(mono)
	serviceMust(t, e)
	a := m.core.monomials.allocator
	h, e := a.ReadUint32(l.raw, 16)
	serviceMust(t, e)
	raw, e := a.Allocate(20)
	serviceMust(t, e)
	original := l.raw
	l.raw = raw
	defer func() { l.raw = original; serviceMust(t, a.Deallocate(20, raw)) }()
	serviceMust(t, a.WriteUint32(raw, 16, h))
	got, e := m.Hash(p)
	serviceMust(t, e)
	if got != want {
		t.Fatal("cached monomial hash changed polynomial hash", got, want)
	}
	for offset := uint64(0); offset < 16; offset += 4 {
		if _, e := a.ReadUint32(raw, offset); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
			t.Fatal("child hasher touched unrelated monomial header", offset, e)
		}
	}
}

func TestPolynomialHashCompositeVisitsLastChildBeforeFirst(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	p := polynomialQueryTerms(t, m.core, []int32{7, 3, 5, 11}, [][]MonomialPower{nil, {{0, 2}}, {{1, 1}}, {{0, 1}}})
	_, e := m.Hash(p) // Complete real lexical sorting before raw-prefix queries.
	serviceMust(t, e)
	pl, e := m.core.polynomialLocation(p)
	serviceMust(t, e)
	last, e := m.core.monomialCore(p, 3)
	serviceMust(t, e)
	ml, e := m.core.monomials.location(last)
	serviceMust(t, e)
	a := m.core.monomials.allocator
	n := uint64(polynomialCodecBytes(4))
	polyRaw, e := a.Allocate(n)
	serviceMust(t, e)
	monoRaw, e := a.Allocate(16) // Hash word at offset16 is out of bounds.
	serviceMust(t, e)
	oldPoly, oldMono := pl.raw, ml.raw
	defer func() {
		pl.raw, ml.raw = oldPoly, oldMono
		serviceMust(t, a.Deallocate(n, polyRaw))
		serviceMust(t, a.Deallocate(16, monoRaw))
	}()
	for offset := uint64(0); offset < n; offset += 4 {
		if offset == 12 { // First coefficient's scalar remains uninitialized.
			continue
		}
		word, e := a.ReadUint32(oldPoly, offset)
		serviceMust(t, e)
		serviceMust(t, a.WriteUint32(polyRaw, offset, word))
	}
	pl.raw, ml.raw = polyRaw, monoRaw
	// get_composite_hash visits3 first for size4. Ascending precomputation
	// instead reaches coefficient0 and reports uninitialized bytes.
	h, e := m.Hash(p)
	if !errors.Is(e, arena.ErrBounds) || h != 0 {
		t.Fatal("composite child source access order", h, e)
	}
}

func TestPolynomialCacheTablePublicationPrecedesReferenceAcquisition(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	cache, e := NewPolynomialCache(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, cache.Close()) })
	p := polynomialQueryTerms(t, m.core, []int32{1}, [][]MonomialPower{nil})
	flags, e := m.core.readWord(p, 4)
	serviceMust(t, e)
	l, e := m.core.polynomialLocation(p)
	serviceMust(t, e)
	a := m.core.monomials.allocator
	raw, e := a.Allocate(12)
	serviceMust(t, e)
	original := l.raw
	l.raw = raw
	defer func() { l.raw = original; serviceMust(t, a.Deallocate(12, raw)) }()
	serviceMust(t, a.WriteUint32(raw, 4, flags))
	serviceMust(t, a.WriteUint32(raw, 8, 0))
	// Hashing and table insertion precede the first refcount access. The
	// guarded INC read refuses; neither owning-vector nor ID publication ran.
	if _, e := cache.MkUnique(p); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
		t.Fatal("missing guarded reference acquisition refusal", e)
	}
	if len(cache.buckets[31]) != 1 || cache.buckets[31][0] != p || len(cache.refs.values) != 0 || cache.ids[flags&0x7fffffff] {
		t.Fatal("cache table/vector/ID publication order differs from source")
	}
	// The source table hit returns the inserted pointer without retrying INC.
	got, e := cache.MkUnique(p)
	serviceMust(t, e)
	if got != p || len(cache.refs.values) != 0 || cache.ids[flags&0x7fffffff] {
		t.Fatal("retry acquired a reference absent from the stopped source prefix")
	}
}
