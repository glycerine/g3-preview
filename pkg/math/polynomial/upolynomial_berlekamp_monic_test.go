package polynomial

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func berlekampLargePrimeMonicFixture(t *testing.T) (*UPolynomialManager, *unumeral.RationalManager, *upolyBerlekampMatrix, UPolynomialView, *upolyVectorState) {
	t.Helper()
	q, s := preparationPhysicalFixture(t)
	m, z := s.manager, q.Integers()
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	driver := upolySourceNew(t, m, []int32{-1, 0, 1})
	matrix := &upolyBerlekampMatrix{m: m}
	serviceMust(t, matrix.init(m, mustUPolyView(t, m, driver)))
	// Genuine concrete matrix owner, followed by the general GCD helper at a
	// real larger field. This is not large-prime Berlekamp factorization evidence.
	serviceMust(t, m.SetZ())
	var big, prime unumeral.Int
	serviceMust(t, z.SetLimbs(&big, []uint32{1, 0, 1}))
	serviceMust(t, z.SetLimbs(&prime, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
	t.Cleanup(func() { serviceMust(t, z.Del(&big)); serviceMust(t, z.Del(&prime)) })
	a := upolySourceNew(t, m, nil)
	serviceMust(t, m.SetInts(a, []unumeral.Int{unumeral.NewInt(1), big}))
	out := upolySourceNew(t, m, nil)
	target, e := m.descriptor(out)
	serviceMust(t, e)
	serviceMust(t, m.SetZP(&prime))
	av := mustUPolyView(t, m, a)
	t.Cleanup(func() { q.Budget().FailAfter(-1); serviceMust(t, m.CleanupBerlekampMatrix()); matrix.close() })
	return m, q, matrix, av, &target.state
}
func TestUPolynomialBerlekampMonicRawBodyFailureAndExplicitEnds(t *testing.T) {
	complete, rawBodyStop := false, false
	for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, q, matrix, a, out := berlekampLargePrimeMonicFixture(t)
			q.Budget().FailAfter(ordinal)
			body := matrix.vectorNumericCall(false, a, a, out)
			q.Budget().FailAfter(-1)
			local := &matrix.numeric.monic
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("actual monic caller reservation prefix", ordinal, body)
			}
			if body != nil && local.declared && !local.endEntered {
				rawBodyStop = true
				if local.sourceIndex != 0 || local.ownEnded || !matrix.numeric.pending() || m.pendingBerlekamp != matrix {
					t.Fatal("failed raw body ran explicit source ends", ordinal, body, local)
				}
			}
			if body == nil {
				complete = true
				if !local.declared || !local.endEntered || local.sourceIndex != 2 || local.ownEnded || local.lc.Cell != 0 || local.inv.Cell != 0 || out.size != 2 {
					t.Fatal("raw monic explicit LC then inverse end", local)
				}
				constant, e := q.Integers().String(&m.values(out)[0])
				serviceMust(t, e)
				if constant != "18446744073709551615" || m.values(out)[1] != unumeral.NewInt(1) {
					t.Fatal("independent inverse of 2^64+1 modulo 2^127-1", constant)
				}
			}
			sourceIndex, endEntered, cause := local.sourceIndex, local.endEntered, local.cause
			serviceMust(t, m.CleanupBerlekampMatrix())
			if local.sourceIndex != sourceIndex || local.endEntered != endEntered || local.cause != cause {
				t.Fatal("owning abort rewrote raw source history")
			}
		})
	}
	if !complete || !rawBodyStop {
		t.Fatal("actual monic completion/raw body failure not reached", complete, rawBodyStop)
	}
}

type berlekampMonicEndProbe struct {
	t        *testing.T
	matrix   *upolyBerlekampMatrix
	z        *unumeral.IntegerManager
	query    umemory.ObjectRef
	raw      umemory.Block
	offset   uint64
	ref      unumeral.NumericCellRef
	original unumeral.NumericCell
	armed    bool
}

func (w *berlekampMonicEndProbe) Write(p []byte) (int, error) {
	local := &w.matrix.numeric.monic
	if !w.armed && string(p) == "a: " && local.declared && local.lc.Cell != 0 {
		if local.endEntered || local.sourceIndex != 0 || local.inv.Cell == 0 {
			w.t.Fatal("actual raw lc/inverse not live before inverse body", local)
		}
		w.ref = local.lc.Cell
		w.original = w.z.Cell(w.ref)
		cell := w.original
		cell.Backing.Object, cell.Backing.Raw, cell.Backing.Offset = w.query, w.raw, w.offset
		w.z.Store(w.ref, cell)
		w.armed = true
	}
	return len(p), nil
}
func TestUPolynomialBerlekampMonicExplicitLCGuardStopsInverseEnd(t *testing.T) {
	m, q, matrix, a, out := berlekampLargePrimeMonicFixture(t)
	z := q.Integers()
	probe := berlekampMonicEndProbe{t: t, matrix: matrix, z: z}
	tags := diagnostic.NewTraceTagsWithWriter(&probe)
	tags.Enable("mpzp_inv_bug")
	if !tags.ShouldTrace("mpzp_inv_bug") {
		t.Skip("actual inverse source trace compiled out")
	}
	allocator := z.SourceAllocator()
	var e error
	probe.query, e = allocator.Allocate(4096)
	serviceMust(t, e)
	probe.raw, probe.offset, e = allocator.RawLocation(probe.query)
	serviceMust(t, e)
	if _, e := allocator.RuntimeOwner().ReadUint32(probe.raw, probe.offset+4); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
		t.Fatal("fresh direct capacity query", e)
	}
	restored := false
	restore := func() {
		if probe.armed && !restored {
			z.Store(probe.ref, probe.original)
			restored = true
		}
	}
	t.Cleanup(func() {
		restore()
		serviceMust(t, m.CleanupBerlekampMatrix())
		matrix.close()
		serviceMust(t, allocator.Deallocate(4096, probe.query))
	})
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	body := matrix.vectorNumericCall(false, a, a, out)
	local := &matrix.numeric.monic
	if !probe.armed || !errors.Is(body, umemory.ErrRuntimeUninitialized) || !local.endEntered || local.sourceIndex != 0 || local.lc.Cell != probe.ref || local.inv.Cell == 0 || local.cause == nil || m.pendingBerlekamp != matrix || matrix.deleteIndex != 0 {
		t.Fatal("explicit LC end guard lost inverse or older matrix", body, local)
	}
	if out.size != 2 || m.values(out)[1] != unumeral.NewInt(1) {
		t.Fatal("monic output was not published before explicit end stop")
	}
	first := local.cause
	matrix.close()
	if matrix.values.backing == 0 || local.sourceIndex != 0 || local.inv.Cell == 0 {
		t.Fatal("source guard retry deleted inverse or matrix")
	}
	restore()
	serviceMust(t, m.CleanupBerlekampMatrix())
	if !local.ownEnded || local.sourceIndex != 0 || local.cause != first || local.lc.Cell != 0 || local.inv.Cell != 0 || matrix.values.backing != 0 || m.pendingBerlekamp != nil {
		t.Fatal("owning raw cleanup lost explicit end history", local)
	}
}
