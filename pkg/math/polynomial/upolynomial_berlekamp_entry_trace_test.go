package polynomial

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type berlekampEntryProbe struct {
	t       *testing.T
	m       *UPolynomialManager
	fs      *UPolynomialFactors
	checked bool
	text    strings.Builder
}

func (w *berlekampEntryProbe) Write(p []byte) (int, error) {
	if string(p) == "upolynomial::factor_square_free_berlekamp(" {
		w.checked = true
		if w.m.pendingBerlekamp != nil || w.fs.count != 0 {
			w.t.Fatal("entry acquired matrix members or published factors")
		}
		for _, e := range []error{w.m.Close(), w.m.SourceFinalize(), w.m.CleanupBerlekampMatrix(), w.fs.Close()} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				w.t.Fatal("entry callback acquired borrowed owner", e)
			}
		}
	}
	return w.text.Write(p)
}
func TestUPolynomialBerlekampEntryTraceBeforeMonicAssertion(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("monic assertion compiled out")
	}
	m, _, _ := upolySourceOwner(t)
	five := unumeral.NewInt(5)
	serviceMust(t, m.SetZP(&five))
	f := upolySourceNew(t, m, []int32{1, 0, 2})
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupBerlekampMatrix()); serviceMust(t, fs.Close()) })
	w := berlekampEntryProbe{t: t, m: m, fs: fs}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	polynomialAssertionActionsFixture(t, "T")
	_, body := m.FactorZPSquareFreeBerlekamp(mustUPolyView(t, m, f), fs, false)
	if !errors.Is(body, ErrUPolynomialAssertion) || !linearThrownAssertion(body) {
		t.Fatal("monic source Throw", body)
	}
	if tags.ShouldTrace("polynomial::factorization") {
		if !w.checked || !strings.Contains(w.text.String(), "upolynomial::factor_square_free_berlekamp(2 x^2 + 1, 5)\n") || strings.Contains(w.text.String(), ":97 ") {
			t.Fatal("entry trace/source monic frontier", w.text.String())
		}
	} else if w.checked || w.text.Len() != 0 {
		t.Fatal("erased entry trace executed")
	}
	if fs.count != 0 || m.pendingBerlekamp != nil || m.factorTrace != nil {
		t.Fatal("monic Throw retained class members/borrow/output")
	}
}

func TestUPolynomialBerlekampPublicEntryTraceRetainsExactDisplay(t *testing.T) {
	q, s := preparationPhysicalFixtureWithSlots(t, 256)
	m, z := s.manager, q.Integers()
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	serviceMust(t, m.SetZ())
	// Qualified integer-mode prefix: the failure is before monic access and
	// before all finite-field arithmetic or matrix member construction.
	var big unumeral.Int
	serviceMust(t, z.SetLimbs(&big, []uint32{1, 0, 1}))
	t.Cleanup(func() { serviceMust(t, z.Del(&big)) })
	f := upolySourceNew(t, m, nil)
	serviceMust(t, m.SetInts(f, []unumeral.Int{big, unumeral.NewInt(0), unumeral.NewInt(1)}))
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	w := factorDegreeTraceFailure{t: t, z: z, limit: 256, held: make([]unumeral.Int, 256), trigger: "upolynomial::factor_square_free_berlekamp("}
	t.Cleanup(func() {
		serviceMust(t, m.CleanupBerlekampMatrix())
		for i := 0; i < w.count; i++ {
			serviceMust(t, z.Del(&w.held[i]))
		}
	})
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	if !tags.ShouldTrace("polynomial::factorization") {
		t.Skip("entry trace compiled out")
	}
	_, body := m.FactorZPSquareFreeBerlekamp(mustUPolyView(t, m, f), fs, false)
	matrix := m.pendingBerlekamp
	if !errors.Is(body, arena.ErrSlotLimit) || !w.armed || matrix == nil || matrix.constructed || matrix.constructorTemp.declared || matrix.values.backing != 0 || len(matrix.column) != 0 || len(matrix.row) != 0 || fs.count != 0 || !m.displayReceiptPending(&matrix.entryTrace) {
		t.Fatal("public entry exact display/class frontier", body)
	}
	saved := matrix.entryTrace
	serviceMust(t, m.CleanupBerlekampMatrix())
	if m.pendingBerlekamp != nil || m.displayReceiptPending(&saved) || matrix.constructorTemp.declared || matrix.constructed {
		t.Fatal("entry owning cleanup invented source class construction")
	}
}

func TestUPolynomialBerlekampPublicEntryTraceStickyStreamFailure(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	f := upolySourceNew(t, m, []int32{1, 0, 1})
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupBerlekampMatrix()); serviceMust(t, fs.Close()) })
	w := factorDegreeRefusedTrace{cause: errors.New("entry stream refusal")}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	split, e := m.FactorZPSquareFreeBerlekamp(mustUPolyView(t, m, f), fs, false)
	serviceMust(t, e)
	if split || fs.count != 1 || m.pendingBerlekamp != nil {
		t.Fatal("sticky entry stream changed irreducible result", split, fs.count)
	}
	product := upolySourceNew(t, m, nil)
	serviceMust(t, fs.Multiply(product))
	same, e := m.Equal(f, product)
	serviceMust(t, e)
	if !same {
		t.Fatal("entry stream refusal changed product")
	}
	if tags.ShouldTrace("polynomial::factorization") {
		if w.calls != 1 || w.flushes < 1 {
			t.Fatal("sticky entry stream retried physical output or skipped endl", w.calls, w.flushes)
		}
	} else if w.calls != 0 {
		t.Fatal("erased entry stream executed")
	}
}
