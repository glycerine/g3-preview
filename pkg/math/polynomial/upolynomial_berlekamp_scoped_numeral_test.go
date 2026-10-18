package polynomial

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type berlekampScopedDestructorProbe struct {
	t           *testing.T
	matrix      *upolyBerlekampMatrix
	diagonal    bool
	query       umemory.ObjectRef
	queryRaw    umemory.Block
	queryOffset uint64
	ref         unumeral.NumericCellRef
	original    unumeral.NumericCell
	armed       bool
	text        strings.Builder
}

func (w *berlekampScopedDestructorProbe) Write(p []byte) (int, error) {
	caption := "polynomial::berlekamp_matrix():"
	if w.diagonal {
		caption = "polynomial::diagonalize():"
	}
	if !w.armed && string(p) == caption {
		local := &w.matrix.constructorTemp
		if w.diagonal {
			local = &w.matrix.diagonalTemp
		}
		if !local.declared || local.sourceEnded || local.value.Cell == 0 {
			w.t.Fatal("actual scoped BIG NUM missing at source record", w.diagonal, local)
		}
		z := w.matrix.m.config.Integer
		w.ref = local.value.Cell
		w.original = z.Cell(w.ref)
		cell := w.original
		cell.Backing.Object, cell.Backing.Raw, cell.Backing.Offset = w.query, w.queryRaw, w.queryOffset
		z.Store(w.ref, cell)
		w.armed = true
	}
	return w.text.Write(p)
}
func TestUPolynomialBerlekampScopedNumeralDestructorRefusal(t *testing.T) {
	for _, diagonal := range []bool{false, true} {
		t.Run(map[bool]string{false: "constructor-tmp", true: "diagonal-multiplier"}[diagonal], func(t *testing.T) {
			m, q, _ := upolySourceOwner(t)
			matrix := upolyBerlekampMatrix{m: m}
			w := berlekampScopedDestructorProbe{t: t, matrix: &matrix, diagonal: diagonal}
			tags := diagnostic.NewTraceTagsWithWriter(&w)
			tags.Enable("polynomial::factorization::bughunt")
			if !tags.ShouldTrace("polynomial::factorization::bughunt") {
				t.Skip("source trace compiled out")
			}
			if diagonal && !polynomialSourceDebug {
				t.Skip("integer-mode inverse continuation requires DEBUG debugger")
			}
			choices := ""
			if diagonal {
				choices = "CCCC"
			}
			polynomialAssertionActionsFixture(t, choices)
			if polynomialSourceDebug {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			three := unumeral.NewInt(3)
			serviceMust(t, m.SetZP(&three))
			// Qualified integer-mode execution produces actual BIG scratch numerals.
			// No valid finite-field BIG factorization is claimed by this profile.
			// Diagonalization uses literal DEBUG Continue at integer-mode
			// inv() UNREACHABLE; that source branch performs no inversion.
			serviceMust(t, m.SetZ())
			var big unumeral.Int
			serviceMust(t, q.Integers().SetLimbs(&big, []uint32{1, 0, 1}))
			t.Cleanup(func() { serviceMust(t, q.Integers().Del(&big)) })
			f := upolySourceNew(t, m, nil)
			serviceMust(t, m.SetInts(f, []unumeral.Int{big, big, unumeral.NewInt(1)}))
			a := q.Integers().SourceAllocator()
			var e error
			w.query, e = a.Allocate(4096)
			serviceMust(t, e)
			w.queryRaw, w.queryOffset, e = a.RawLocation(w.query)
			serviceMust(t, e)
			if _, e = a.RuntimeOwner().ReadUint32(w.queryRaw, w.queryOffset+4); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
				t.Fatal("fresh uninitialized capacity query", e)
			}
			restored := false
			restore := func() {
				if w.armed && !restored {
					q.Integers().Store(w.ref, w.original)
					restored = true
				}
			}
			t.Cleanup(func() {
				restore()
				serviceMust(t, m.CleanupBerlekampMatrix())
				matrix.close()
				serviceMust(t, a.Deallocate(4096, w.query))
			})
			serviceMust(t, m.modular.BindTrace(tags))
			t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
			body := matrix.init(m, mustUPolyView(t, m, f))
			if diagonal {
				serviceMust(t, body)
				_, body = matrix.diagonalize()
			}
			local := &matrix.constructorTemp
			if diagonal {
				local = &matrix.diagonalTemp
			}
			if !w.armed || !errors.Is(body, umemory.ErrRuntimeUninitialized) || local.sourceEnded || local.cause == nil || local.value.Cell != w.ref || m.pendingBerlekamp != &matrix || matrix.constructed != diagonal {
				t.Fatal("source scoped NUM destructor stop/frontier", body, local, matrix.constructed)
			}
			backing := matrix.values
			matrix.close()
			if matrix.values != backing || matrix.deleteIndex != 0 || len(matrix.row) != 2 || len(matrix.column) != 2 {
				t.Fatal("older matrix members crossed unfinished scoped NUM")
			}
			var untouched upolyVectorState
			_, nextErr := matrix.next(&untouched)
			if !errors.Is(nextErr, ErrUPolynomialCleanupPending) || untouched.backing != 0 {
				t.Fatal("pending scratch changed null-vector output", nextErr)
			}
			if e := matrix.traceMatrix(142); !errors.Is(e, ErrUPolynomialCleanupPending) {
				t.Fatal("pending scratch entered a later trace", e)
			}
			first := local.cause
			if e := m.SourceFinalize(); e != first {
				t.Fatal("core crossed scoped NUM source stop", e)
			}
			restore()
			serviceMust(t, m.CleanupBerlekampMatrix())
			if local.sourceEnded || !local.ownEnded || local.cause != first || local.value.Cell != 0 || matrix.deleteIndex != 0 || matrix.values.backing != 0 || m.pendingBerlekamp != nil {
				t.Fatal("owning cleanup lost source scoped NUM history", local)
			}
		})
	}
}
