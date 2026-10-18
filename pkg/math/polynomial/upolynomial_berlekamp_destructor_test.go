package polynomial

import (
	"errors"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialBerlekampDestructorRetainsAscendingGuardPrefix(t *testing.T) {
	m, q, _ := upolySourceOwner(t)
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	// Qualified integer-mode constructor execution supplies genuine BIG matrix
	// members; this is not claimed as a finite-field input or result.
	serviceMust(t, m.SetZ())
	var big unumeral.Int
	serviceMust(t, q.Integers().SetLimbs(&big, []uint32{1, 0, 1}))
	t.Cleanup(func() { serviceMust(t, q.Integers().Del(&big)) })
	f := upolySourceNew(t, m, nil)
	serviceMust(t, m.SetInts(f, []unumeral.Int{big, big, unumeral.NewInt(1)}))
	matrix := upolyBerlekampMatrix{m: m}
	serviceMust(t, matrix.init(m, mustUPolyView(t, m, f)))
	values := m.values(&matrix.values)
	if !matrix.constructed || len(values) < 4 || values[2].Cell == 0 || values[3].Cell == 0 {
		t.Fatal("actual BIG constructor matrix members", values)
	}
	stoppedRef := values[3].Cell
	cell := q.Integers().Cell(stoppedRef)
	original := cell.Backing
	allocator := q.Integers().SourceAllocator()
	query, e := allocator.Allocate(4096)
	serviceMust(t, e)
	queryRaw, queryOffset, e := allocator.RawLocation(query)
	serviceMust(t, e)
	if _, e := allocator.RuntimeOwner().ReadUint32(queryRaw, queryOffset+4); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
		t.Fatal("fresh direct capacity fixture", e)
	}
	// Query-only access fixture: fresh direct storage leaves the source
	// capacity word unreadable (pooled cells can retain initialized bytes). Genuine owning storage is retained
	// separately and restored before owning destruction. Numerals are unchanged.
	cell.Backing.Object, cell.Backing.Raw, cell.Backing.Offset = query, queryRaw, queryOffset
	q.Integers().Store(stoppedRef, cell)
	restored := false
	restore := func() {
		if !restored {
			cell.Backing = original
			q.Integers().Store(stoppedRef, cell)
			restored = true
		}
	}
	t.Cleanup(func() {
		restore()
		serviceMust(t, m.CleanupBerlekampMatrix())
		matrix.close()
		serviceMust(t, allocator.Deallocate(4096, query))
	})
	backing := matrix.values
	matrix.close()
	if !errors.Is(matrix.closeCause, umemory.ErrRuntimeUninitialized) || !matrix.sourceGuard || matrix.deleteIndex != 3 || matrix.values != backing || values[2].Cell != 0 || values[3].Cell != stoppedRef || len(matrix.column) != 2 || len(matrix.row) != 2 || m.pendingBerlekamp != &matrix {
		t.Fatal("stopped source destructor lost prefix or members", matrix.closeCause, matrix.deleteIndex, matrix.values)
	}
	before := q.Budget().Used()
	first := matrix.closeCause
	matrix.close()
	if matrix.closeCause != first || matrix.deleteIndex != 3 || q.Budget().Used() != before {
		t.Fatal("source guard retry advanced prefix")
	}
	if e := m.SourceFinalize(); e != first {
		t.Fatal("core destruction crossed retained lexical matrix", e)
	}
	if _, e := m.NewPolynomial(); !errors.Is(e, ErrUPolynomialCleanupPending) {
		t.Fatal("new work adopted retained matrix", e)
	}
	restore()
	// Restoring the genuine query bound does not restart a stopped source DT.
	if e := matrix.closeSource(); e != first || matrix.deleteIndex != 3 {
		t.Fatal("source destructor resumed after a guard", e)
	}
	serviceMust(t, m.CleanupBerlekampMatrix())
	if m.pendingBerlekamp != nil || matrix.values.backing != 0 || matrix.column != nil || matrix.row != nil || matrix.deleteIndex != 3 || matrix.closeCause != first {
		t.Fatal("owning abort lost source history or retained members")
	}
	serviceMust(t, m.CleanupBerlekampMatrix())
	matrix.close()
	if m.pendingBerlekamp != nil {
		t.Fatal("completed owning cleanup reacquired old matrix")
	}
}

func TestUPolynomialBerlekampDestructorCompleteThenReuse(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	f := upolySourceNew(t, m, []int32{1, 0, 1})
	matrix := upolyBerlekampMatrix{m: m}
	t.Cleanup(matrix.close)
	for i := 0; i < 2; i++ {
		serviceMust(t, matrix.init(m, mustUPolyView(t, m, f)))
		size := matrix.values.size
		matrix.close()
		if matrix.values.backing != 0 || matrix.column != nil || matrix.row != nil || matrix.deleteIndex != size || m.pendingBerlekamp != nil {
			t.Fatal("completed matrix source destruction", i, matrix.deleteIndex, size)
		}
	}
}

func TestUPolynomialBerlekampFailedConstructorHasOnlyOwningMemberAbort(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	// The default integer-mode manager remembers modulus zero. The constructor
	// reaches the actual modulo-zero domain adapter after its initial row, so
	// it has real member storage but grants no C++ class destructor.
	f := upolySourceNew(t, m, []int32{1, 0, 1})
	matrix := upolyBerlekampMatrix{m: m}
	t.Cleanup(matrix.close)
	e := matrix.init(m, mustUPolyView(t, m, f))
	if !errors.Is(e, ErrUPolynomialDomain) || matrix.constructed || matrix.values.size != 3 || matrix.values.backing == 0 {
		t.Fatal("actual failed-constructor prefix", e, matrix.values)
	}
	matrix.close()
	if matrix.deleteIndex != 0 || !matrix.ownStarted || matrix.ownIndex != 3 || matrix.values.backing != 0 || matrix.column != nil || matrix.row != nil || m.pendingBerlekamp != nil {
		t.Fatal("failed construction ran class destructor or lost actual members")
	}
}
