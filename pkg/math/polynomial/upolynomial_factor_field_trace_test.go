package polynomial

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type factorFieldOutcomeFailure struct {
	t          *testing.T
	z          *unumeral.IntegerManager
	target     string
	seenTarget int
	held       []unumeral.Int
	heldCount  int
	armed      bool
	text       strings.Builder
}

func (w *factorFieldOutcomeFailure) Write(p []byte) (int, error) {
	if string(p) == w.target {
		w.seenTarget++
		if w.seenTarget == 2 {
			w.armed = true
			for {
				stats := w.z.Cells.Stats()
				if uint64(stats.Live)+uint64(stats.Retired) >= uint64(len(w.held)) {
					break
				}
				if e := w.z.Copy(&w.held[w.heldCount], &w.z.Two64); e != nil {
					w.t.Fatal("caller descriptor padding", e)
				}
				w.heldCount++
			}
		}
	}
	return w.text.Write(p)
}

func testFactorFieldOutcomeRetainsOlderSource(t *testing.T, whole bool) {
	q, s := preparationPhysicalFixtureWithSlots(t, 256)
	m, z := s.manager, q.Integers()
	var big unumeral.Int
	serviceMust(t, z.SetLimbs(&big, []uint32{1, 0, 1}))
	f := upolySourceNew(t, m, nil)
	serviceMust(t, m.SetInts(f, []unumeral.Int{big, unumeral.NewInt(0), unumeral.NewInt(1)}))
	serviceMust(t, z.Del(&big))
	serviceMust(t, m.SetZPUint64(5))
	fs, e := m.NewFactors()
	serviceMust(t, e)
	target := "polynomial::square_free_factor("
	if whole {
		target = "polynomial::factor("
	}
	w := factorFieldOutcomeFailure{t: t, z: z, target: target, held: make([]unumeral.Int, 256)}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization")
	serviceMust(t, m.modular.BindTrace(tags))
	defer func() {
		serviceMust(t, m.modular.BindTrace(nil))
		serviceMust(t, m.CleanupFactorFieldFactorization())
		for i := 0; i < w.heldCount; i++ {
			serviceMust(t, z.Del(&w.held[i]))
		}
		serviceMust(t, fs.Close())
	}()
	if !tags.ShouldTrace("polynomial::factorization") {
		t.Skip("source TRACE erased")
	}
	if whole {
		_, e = m.FactorZP(mustUPolyView(t, m, f), fs)
	} else {
		e = m.SquareFreeFactorZP(mustUPolyView(t, m, f), fs)
	}
	if !w.armed || !errors.Is(e, arena.ErrSlotLimit) || m.factorField == nil || m.factorField.executing || !m.displayReceiptPending(&m.factorField.polynomial) {
		t.Fatal("final finite-field TRACE did not retain exact child/source frame", whole, e, w.armed)
	}
	retained := m.factorField
	if whole {
		if retained.kind != factorFieldWhole || retained.sq == nil || retained.parent != nil {
			t.Fatal("outer factor frame/local factor collection", retained.kind, retained.sq != nil, retained.parent)
		}
	} else if retained.kind != factorFieldSquare || retained.constructed != 8 || retained.sourceStage != 0 {
		t.Fatal("square-free vector declaration frontier", retained.kind, retained.constructed, retained.sourceStage)
	}
	if e = m.SetZ(); !errors.Is(e, ErrUPolynomialCleanupPending) {
		t.Fatal("retained finite-field source allowed manager reuse", e)
	}
	if e = m.SourceFinalize(); !errors.Is(e, ErrUPolynomialCleanupPending) {
		t.Fatal("source finalization ignored retained finite-field display", e)
	}
	serviceMust(t, m.CleanupFactorFieldFactorization())
	if m.factorField != nil || m.displayReceiptPending(&retained.polynomial) || retained.active {
		t.Fatal("exact finite-field owning cleanup incomplete")
	}
}

func TestUPolynomialFiniteFieldSquareFreeOutcomeRetainsOlderSource(t *testing.T) {
	testFactorFieldOutcomeRetainsOlderSource(t, false)
}

func TestUPolynomialFiniteFieldFactorOutcomeRetainsLocalFactors(t *testing.T) {
	testFactorFieldOutcomeRetainsOlderSource(t, true)
}

func TestUPolynomialFiniteFieldFactorRetainsNestedSquareFreeSource(t *testing.T) {
	q, s := preparationPhysicalFixtureWithSlots(t, 256)
	m, z := s.manager, q.Integers()
	var big unumeral.Int
	serviceMust(t, z.SetLimbs(&big, []uint32{1, 0, 1}))
	f := upolySourceNew(t, m, nil)
	serviceMust(t, m.SetInts(f, []unumeral.Int{big, unumeral.NewInt(0), unumeral.NewInt(1)}))
	serviceMust(t, z.Del(&big))
	serviceMust(t, m.SetZPUint64(5))
	fs, e := m.NewFactors()
	serviceMust(t, e)
	w := factorFieldOutcomeFailure{t: t, z: z, target: "polynomial::square_free_factor(", held: make([]unumeral.Int, 256)}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization")
	serviceMust(t, m.modular.BindTrace(tags))
	defer func() {
		serviceMust(t, m.modular.BindTrace(nil))
		serviceMust(t, m.CleanupFactorFieldFactorization())
		for i := 0; i < w.heldCount; i++ {
			serviceMust(t, z.Del(&w.held[i]))
		}
		serviceMust(t, fs.Close())
	}()
	if !tags.ShouldTrace("polynomial::factorization") {
		t.Skip("source TRACE erased")
	}

	_, e = m.FactorZP(mustUPolyView(t, m, f), fs)
	inner := m.factorField
	if !w.armed || !errors.Is(e, arena.ErrSlotLimit) || inner == nil || inner.kind != factorFieldSquare || inner.parent == nil || inner.parent.kind != factorFieldWhole {
		t.Fatal("nested finite-field source stack", e, w.armed, inner)
	}
	outer := inner.parent
	if !m.displayReceiptPending(&inner.polynomial) || inner.constructed != 8 || inner.sourceStage != 0 || outer.sq == nil || outer.executing {
		t.Fatal("nested square-free/outer factor frontier", inner.constructed, inner.sourceStage, outer.sq != nil, outer.executing)
	}
	if e = m.SourceFinalize(); !errors.Is(e, ErrUPolynomialCleanupPending) {
		t.Fatal("source finalization ignored nested retained display", e)
	}
	serviceMust(t, m.CleanupFactorFieldFactorization())
	if m.factorField != nil || inner.active || outer.active || m.displayReceiptPending(&inner.polynomial) {
		t.Fatal("nested finite-field cleanup incomplete")
	}
}

func TestUPolynomialFiniteFieldSquareFreeTraceRecords(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	serviceMust(t, m.SetZPUint64(5))
	f := upolySourceNew(t, m, []int32{-1, 0, 1})
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	var out strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("polynomial::factorization")
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })

	serviceMust(t, m.SquareFreeFactorZP(mustUPolyView(t, m, f), fs))
	if !tags.ShouldTrace("polynomial::factorization") {
		if out.Len() != 0 {
			t.Fatal("erased finite-field TRACE executed", out.String())
		}
		return
	}
	text := out.String()
	ordered := []string{
		"upolynomial_factorization.cpp:256 ",
		"polynomial::square_free_factor(x^2 - 1) over Z_5\n",
		"upolynomial_factorization.cpp:268 ",
		"Initial factors: 1\nR.<x> = GF(5)['x']\nT_0 = x^2 - 1\n",
		"upolynomial_factorization.cpp:285 ",
		"k = 0\n",
		"upolynomial_factorization.cpp:289 ",
		"T_0_d = T_0.derivative(x)\nT_0_d == 2 x\n",
		"upolynomial_factorization.cpp:296 ",
		"T = T_0.gcd(T_0_d)\nT == 1\n",
		"upolynomial_factorization.cpp:303 ",
		"V = T_0.quo_rem(T)[0]\nV == x^2 - 1\n",
		"upolynomial_factorization.cpp:324 ",
		"W = T.gcd(V)\n1\n",
		"upolynomial_factorization.cpp:331 ",
		"A_ek = V.quo_rem(W)[0]\nA_ek == x^2 - 1\n",
		"upolynomial_factorization.cpp:338 ",
		"V = W\nV == 1\n",
		"upolynomial_factorization.cpp:345 ",
		"T = T.quo_rem(V)[0]\nT == 1\n",
		"upolynomial_factorization.cpp:352 ",
		"Factor: (x^2 - 1)^1\n",
		"upolynomial_factorization.cpp:364 ",
		"T_0 = 1\n",
		"upolynomial_factorization.cpp:369 ",
		"polynomial::square_free_factor(x^2 - 1) => 1 * (x^2 - 1)^1\n",
	}
	position := 0
	for _, fragment := range ordered {
		i := strings.Index(text[position:], fragment)
		if i < 0 {
			t.Fatalf("missing ordered TRACE fragment %q in %q", fragment, text)
		}
		position += i + len(fragment)
	}
}

func TestUPolynomialFiniteFieldFactorEntryAndOutcomeTrace(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	serviceMust(t, m.SetZPUint64(5))
	f := upolySourceNew(t, m, []int32{-1, 0, 1})
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	var out strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("polynomial::factorization")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })

	_, e = m.FactorZP(mustUPolyView(t, m, f), fs)
	serviceMust(t, e)
	if !tags.ShouldTrace("polynomial::factorization") {
		if out.Len() != 0 {
			t.Fatal("erased finite-field TRACE executed", out.String())
		}
		return
	}
	text := out.String()
	entry := strings.Index(text, "upolynomial_factorization.cpp:375 ")
	outcome := strings.Index(text, "upolynomial_factorization.cpp:399 ")
	if entry < 0 || outcome <= entry || !strings.Contains(text[entry:], "polynomial::factor(x^2 - 1) over Z_5\n") ||
		!strings.Contains(text[outcome:], "polynomial::factor(x^2 - 1) => 1 * (x + 1)^1 * (x - 1)^1\n") {
		t.Fatal("finite-field factor entry/outcome records", text)
	}
}
