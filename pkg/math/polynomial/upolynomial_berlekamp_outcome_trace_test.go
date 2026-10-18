package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strings"
	"testing"
)

type berlekampOutcomeProbe struct {
	t       *testing.T
	m       *UPolynomialManager
	fs      *UPolynomialFactors
	checked bool
	text    strings.Builder
}

func (w *berlekampOutcomeProbe) Write(p []byte) (int, error) {
	if string(p) == ") => " {
		w.checked = true
		if w.fs.count != 1 || w.fs.totalFactors != 1 {
			w.t.Fatal("outcome preceded source factor publication")
		}
		for _, e := range []error{w.m.Close(), w.m.SourceFinalize(), w.m.CleanupBerlekampMatrix(), w.fs.Close(), w.fs.CleanupDisplay()} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				w.t.Fatal("outcome callback acquired borrowed owner", e)
			}
		}
	}
	return w.text.Write(p)
}
func TestUPolynomialBerlekampIrreducibleOutcomeTrace(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	f := upolySourceNew(t, m, []int32{1, 0, 1})
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupBerlekampMatrix()); serviceMust(t, fs.Close()) })
	w := berlekampOutcomeProbe{t: t, m: m, fs: fs}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	split, e := m.FactorZPSquareFreeBerlekamp(mustUPolyView(t, m, f), fs, false)
	serviceMust(t, e)
	if split || fs.count != 1 || m.pendingBerlekamp != nil {
		t.Fatal("source rank-one return/publication", split, fs.count)
	}
	if tags.ShouldTrace("polynomial::factorization") {
		if !w.checked || !strings.Contains(w.text.String(), "upolynomial::factor_square_free_berlekamp(x^2 + 1, 3) => 1 * (x^2 + 1)^1\n") || strings.Index(w.text.String(), ":414 ") >= strings.Index(w.text.String(), ":428 ") {
			t.Fatal("whole irreducible outcome record/order", w.text.String())
		}
	} else if w.checked || w.text.Len() != 0 {
		t.Fatal("erased outcome trace executed")
	}
	product := upolySourceNew(t, m, nil)
	serviceMust(t, fs.Multiply(product))
	same, e := m.Equal(f, product)
	serviceMust(t, e)
	if !same {
		t.Fatal("outcome trace changed public product")
	}
}

func TestUPolynomialBerlekampOutcomeExactFactorDisplayBeforeMatrixDT(t *testing.T) {
	q, s := preparationPhysicalFixtureWithSlots(t, 256)
	m, z := s.manager, q.Integers()
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	var big unumeral.Int
	serviceMust(t, z.SetLimbs(&big, []uint32{1, 0, 1}))
	t.Cleanup(func() { serviceMust(t, z.Del(&big)) })
	prefix := upolySourceNew(t, m, nil)
	serviceMust(t, m.SetInts(prefix, []unumeral.Int{big, unumeral.NewInt(1)}))
	serviceMust(t, fs.PushCopy(mustUPolyView(t, m, prefix), 1))
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	// The older caller factor is deliberately not field-normalized. The actual
	// rank-one finite-field matrix and newly appended factor remain genuine;
	// this qualifies only the old-prefix display access, not its field validity.
	f := upolySourceNew(t, m, []int32{1, 0, 1})
	w := factorDegreeTraceFailure{t: t, z: z, limit: 256, held: make([]unumeral.Int, 256), trigger: ") => "}
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
		t.Skip("source trace compiled out")
	}
	polynomialAssertionActionsFixture(t, "")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	_, body := m.FactorZPSquareFreeBerlekamp(mustUPolyView(t, m, f), fs, false)
	matrix := m.pendingBerlekamp
	if !errors.Is(body, arena.ErrSlotLimit) || !w.armed || matrix == nil || !matrix.constructed || matrix.values.size != 5 || matrix.deleteIndex != 0 || !matrix.constructorTemp.sourceEnded || !matrix.diagonalTemp.sourceEnded || !matrix.outcomeFactors.pending() || fs.count != 2 {
		t.Fatal("outcome exact display and older class frontier", body)
	}
	saved := matrix.outcomeFactors
	serviceMust(t, m.CleanupBerlekampMatrix())
	if m.pendingBerlekamp != nil || saved.pending() || matrix.values.backing != 0 || fs.count != 2 {
		t.Fatal("owning cleanup lost published factors or exact class/display ownership")
	}
}
