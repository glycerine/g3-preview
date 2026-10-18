package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strings"
	"testing"
)

type berlekampSplitProbe struct {
	t    *testing.T
	m    *UPolynomialManager
	fs   *UPolynomialFactors
	seen map[string]bool
	text strings.Builder
}

func (w *berlekampSplitProbe) Write(p []byte) (int, error) {
	label := string(p)
	if label == "current_factor = " || label == ") => " {
		w.seen[label] = true
		want := uint32(1)
		if label == ") => " {
			want = 2
		}
		if w.fs.count != want {
			w.t.Fatal("split trace crossed source publication frontier", label, w.fs.count)
		}
		for _, e := range []error{w.m.Close(), w.m.SourceFinalize(), w.m.CleanupBerlekampMatrix(), w.fs.Close()} {
			if !errors.Is(e, ErrUPolynomialCleanupPending) {
				w.t.Fatal("split callback acquired active owner", e)
			}
		}
	}
	return w.text.Write(p)
}
func TestUPolynomialBerlekampCompleteSplitSourceRecords(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	f := upolySourceNew(t, m, []int32{-1, 0, 1})
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupBerlekampMatrix()); serviceMust(t, fs.Close()) })
	w := berlekampSplitProbe{t: t, m: m, fs: fs, seen: map[string]bool{}}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization")
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	split, e := m.FactorZPSquareFreeBerlekamp(mustUPolyView(t, m, f), fs, false)
	serviceMust(t, e)
	if !split || fs.count != 2 || m.pendingBerlekamp != nil {
		t.Fatal("actual two-factor split", split, fs.count)
	}
	if tags.ShouldTrace("polynomial::factorization::bughunt") {
		text := w.text.String()
		for _, part := range []string{"computing factors, expecting 2\n", "null vector: 0 1 \n", "current_factor = x^2 - 1\ngcd_norm = x - 1\ndiv = x + 1\n", "polynomial::factor(x^2 - 1, 3) => 1 * (x + 1)^1 * (x - 1)^1 of degree 2\n"} {
			if !strings.Contains(text, part) {
				t.Fatal("literal split record", part, text)
			}
		}
		previous := -1
		for _, location := range []string{":432 ", ":443 ", ":482 ", ":498 "} {
			pos := strings.Index(text, location)
			if pos <= previous {
				t.Fatal("split source record order", location, text)
			}
			previous = pos
		}
		if len(w.seen) != 2 {
			t.Fatal("actual split record callbacks", w.seen)
		}
	} else if w.text.Len() != 0 {
		t.Fatal("erased split traces executed")
	}
	product := upolySourceNew(t, m, nil)
	serviceMust(t, fs.Multiply(product))
	same, e := m.Equal(f, product)
	serviceMust(t, e)
	if !same {
		t.Fatal("split trace changed product")
	}
}
func TestUPolynomialBerlekampSplitOutcomeStopsBeforeLocalVectorDT(t *testing.T) {
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
	f := upolySourceNew(t, m, []int32{-1, 0, 1})
	// Qualified invalid old caller prefix: normalization assertions are disabled
	// only to reach its real display copy after the genuine new two-factor split.
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
	if !errors.Is(body, arena.ErrSlotLimit) || !w.armed || matrix == nil || !matrix.constructed || !matrix.outcomeFactors.pending() || matrix.split.sourceRemaining != 3 || matrix.split.sourceIndex != 0 || matrix.deleteIndex != 0 || fs.count != 3 {
		t.Fatal("split outcome display/local-vector frontier", body)
	}
	locals := &matrix.split
	if locals.vector.size != 2 || locals.div.size != 3 || locals.gcd.size != 2 || locals.vector.backing == 0 || locals.div.backing == 0 || locals.gcd.backing == 0 {
		t.Fatal("actual retained split vectors", locals)
	}
	originalCause := locals.cause
	serviceMust(t, m.CleanupBerlekampMatrix())
	if m.pendingBerlekamp != nil || !locals.ownEnded || locals.sourceRemaining != 3 || locals.sourceIndex != 0 || locals.cause != originalCause || locals.vector.backing != 0 || locals.div.backing != 0 || locals.gcd.backing != 0 || matrix.values.backing != 0 || fs.count != 3 {
		t.Fatal("owning abort lost split source history or output")
	}
}

func TestUPolynomialBerlekampSplitTraceStickyStreamPreservesResult(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	three := unumeral.NewInt(3)
	serviceMust(t, m.SetZP(&three))
	f := upolySourceNew(t, m, []int32{-1, 0, 1})
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupBerlekampMatrix()); serviceMust(t, fs.Close()) })
	w := factorDegreeRefusedTrace{cause: errors.New("split stream refusal")}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization")
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	split, e := m.FactorZPSquareFreeBerlekamp(mustUPolyView(t, m, f), fs, false)
	serviceMust(t, e)
	if !split || fs.count != 2 || m.pendingBerlekamp != nil {
		t.Fatal("sticky split trace changed source result", split, fs.count)
	}
	product := upolySourceNew(t, m, nil)
	serviceMust(t, fs.Multiply(product))
	same, e := m.Equal(f, product)
	serviceMust(t, e)
	if !same {
		t.Fatal("sticky split trace changed factor product")
	}
	if tags.ShouldTrace("polynomial::factorization") {
		if w.calls != 1 || w.flushes < 7 {
			t.Fatal("sticky split trace retried physical writer or skipped record flushes", w.calls, w.flushes)
		}
	} else if w.calls != 0 {
		t.Fatal("erased split trace executed")
	}
}
