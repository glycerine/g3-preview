package polynomial

import (
	"errors"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestUPolynomialBerlekampDivisionAMSourceStopAndOwningCleanup(t *testing.T) {
	q, s := preparationPhysicalFixture(t)
	m, z := s.manager, q.Integers()
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	driver := upolySourceNew(t, m, []int32{-1, 0, 1})
	matrix := upolyBerlekampMatrix{m: m}
	serviceMust(t, matrix.init(m, mustUPolyView(t, m, driver)))
	// The direct owner adapter is genuinely constructed at p=3. Only the
	// general division helper is exercised at the larger actual prime below;
	// no large-prime Berlekamp factorization is claimed.
	serviceMust(t, m.SetZ())
	var big, plus, prime unumeral.Int
	serviceMust(t, z.SetLimbs(&big, []uint32{1, 0, 1}))
	serviceMust(t, z.Copy(&plus, &big))
	serviceMust(t, z.Inc(&plus))
	serviceMust(t, z.SetLimbs(&prime, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
	t.Cleanup(func() { serviceMust(t, z.Del(&prime)); serviceMust(t, z.Del(&plus)); serviceMust(t, z.Del(&big)) })
	a := upolySourceNew(t, m, nil)
	serviceMust(t, m.SetInts(a, []unumeral.Int{unumeral.NewInt(1), plus, big}))
	b := upolySourceNew(t, m, []int32{1, 1})
	out := upolySourceNew(t, m, nil)
	serviceMust(t, m.SetZP(&prime))
	av, bv := mustUPolyView(t, m, a), mustUPolyView(t, m, b)
	target, e := m.descriptor(out)
	serviceMust(t, e)
	allocator := z.SourceAllocator()
	query, e := allocator.Allocate(4096)
	serviceMust(t, e)
	raw, offset, e := allocator.RawLocation(query)
	serviceMust(t, e)
	if _, e := allocator.RuntimeOwner().ReadUint32(raw, offset+4); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
		t.Fatal("fresh direct capacity query", e)
	}
	var ref unumeral.NumericCellRef
	var original unumeral.NumericCell
	restored := false
	restore := func() {
		if ref != 0 && !restored {
			z.Store(ref, original)
			restored = true
		}
	}
	t.Cleanup(func() {
		restore()
		serviceMust(t, m.CleanupBerlekampMatrix())
		matrix.close()
		serviceMust(t, allocator.Deallocate(4096, query))
	})
	checks := 0
	m.config.Checkpoint = func() bool {
		checks++
		if checks != 2 {
			return true
		}
		local := &matrix.numeric.divisionAM
		if !local.declared || local.sourceEnded || local.value.Cell == 0 {
			t.Fatal("actual BIG a_m not live at second division checkpoint", local)
		}
		ref = local.value.Cell
		original = z.Cell(ref)
		// Query-only unreadable capacity metadata; preserve the actual numerical
		// value/allocation and restore genuine backing before owning destruction.
		cell := original
		cell.Backing.Object, cell.Backing.Raw, cell.Backing.Offset = query, raw, offset
		z.Store(ref, cell)
		return false
	}
	body := matrix.vectorNumericCall(true, av, bv, &target.state)
	local := &matrix.numeric.divisionAM
	if !errors.Is(body, ErrUPolynomialCanceled) || !errors.Is(body, umemory.ErrRuntimeUninitialized) || checks != 2 || !local.declared || local.sourceEnded || local.ownEnded || local.value.Cell != ref || local.cause == nil || m.pendingBerlekamp != &matrix || matrix.deleteIndex != 0 {
		t.Fatal("canceled helper lost a_m source destructor stop", body, checks, local)
	}
	// The quotient wrapper has not swapped its output. Its actual kernel has
	// already published the leading quotient B and retains the partial row.
	if target.state.size != 0 || m.div2.size != 2 {
		t.Fatal("quotient publication frontier", target.state, m.div2)
	}
	text, e := z.String(&m.values(&m.div2)[1])
	serviceMust(t, e)
	if text != "18446744073709551617" {
		t.Fatal("actual leading quotient prefix", text)
	}
	first := local.cause
	matrix.close()
	if matrix.values.backing == 0 || matrix.deleteIndex != 0 || local.sourceEnded {
		t.Fatal("older matrix crossed a_m source stop")
	}
	if e := m.SourceFinalize(); e != first {
		t.Fatal("core retirement crossed actual a_m", e)
	}
	restore()
	serviceMust(t, m.CleanupBerlekampMatrix())
	if !local.ownEnded || local.sourceEnded || local.cause != first || local.value.Cell != 0 || matrix.values.backing != 0 || m.pendingBerlekamp != nil {
		t.Fatal("owning cleanup lost a_m source history", local)
	}
}

func TestUPolynomialBerlekampDivisionAMSuccessfulSourceEnd(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	f := upolySourceNew(t, m, []int32{-1, 0, 1})
	b := upolySourceNew(t, m, []int32{1, 1})
	out := upolySourceNew(t, m, nil)
	matrix := upolyBerlekampMatrix{m: m}
	t.Cleanup(matrix.close)
	serviceMust(t, matrix.init(m, mustUPolyView(t, m, f)))
	target, e := m.descriptor(out)
	serviceMust(t, e)
	serviceMust(t, matrix.vectorNumericCall(true, mustUPolyView(t, m, f), mustUPolyView(t, m, b), &target.state))
	local := &matrix.numeric.divisionAM
	if !local.declared || !local.sourceEnded || local.ownEnded || local.value.Cell != 0 || matrix.numeric.pending() {
		t.Fatal("actual successful a_m source end", local)
	}
}
