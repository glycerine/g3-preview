package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strings"
	"testing"
)

func TestUPolynomialHenselPairTraceActualBIGChildren(t *testing.T) {
	for _, entry := range []bool{false, true} {
		name := "residual"
		if entry {
			name = "entry"
		}
		t.Run(name, func(t *testing.T) { henselPairTraceActualBIGChild(t, entry) })
	}
}
func henselPairTraceActualBIGChild(t *testing.T, entry bool) {
	q, s := preparationPhysicalFixtureWithSlots(t, 256)
	m, z := s.manager, q.Integers()
	var prime, n, negativeN, constant, term unumeral.Int
	args := []*unumeral.Int{&prime, &n, &negativeN, &constant, &term}
	t.Cleanup(func() {
		for _, v := range args {
			serviceMust(t, z.Del(v))
		}
	})
	serviceMust(t, z.SetLimbs(&prime, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
	one, two := unumeral.NewInt(1), unumeral.NewInt(2)
	serviceMust(t, z.Sub(&prime, &one, &n))
	serviceMust(t, z.Div(&n, &two, &n))
	serviceMust(t, z.Copy(&negativeN, &n))
	serviceMust(t, z.Neg(&negativeN))
	serviceMust(t, z.Mul(&n, &n, &constant))
	serviceMust(t, z.Mul(&prime, &n, &term))
	serviceMust(t, z.Add(&constant, &term, &constant))
	serviceMust(t, z.Neg(&constant))
	// A=x-n,B=x+n,U=1,V=-1; C=AB-p*n gives residual -n (mod p).
	values := [5][]unumeral.Int{{one}, {negativeN, one}, {unumeral.NewInt(-1)}, {n, one}, {constant, unumeral.NewInt(0), one}}
	var inputs [5]UPolynomialView
	for i, vs := range values {
		p, e := m.NewPolynomial()
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, m.DelPolynomial(&p)) })
		serviceMust(t, m.SetInts(p, vs))
		inputs[i] = mustUPolyView(t, m, p)
	}
	outA, e := m.NewPolynomial()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.DelPolynomial(&outA)) })
	outB, e := m.NewPolynomial()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.DelPolynomial(&outB)) })
	trigger, tag := "f = ", "polynomial::factorization"
	if entry {
		trigger, tag = "A = ", "polynomial::factorization::bughunt"
	}
	w := factorDegreeTraceFailure{t: t, z: z, limit: 256, held: make([]unumeral.Int, 256), trigger: trigger}
	t.Cleanup(func() {
		serviceMust(t, s.CleanupRetained())
		for i := 0; i < w.count; i++ {
			serviceMust(t, z.Del(&w.held[i]))
		}
	})
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable(tag)
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	body := s.HenselLiftPair(&prime, &prime, &prime, inputs[0], inputs[1], inputs[2], inputs[3], inputs[4], outA, outB)
	if !tags.ShouldTrace(tag) {
		serviceMust(t, body)
		return
	}
	h := s.henselPairs
	if !errors.Is(body, arena.ErrSlotLimit) || !w.armed || h == nil || !h.active || !m.displayReceiptPending(&h.trace) {
		t.Fatal("actual pair residual display child/frontier", body, h)
	}
	line := ":614 "
	if entry {
		line = ":590 "
		if h.constructed != 0 || h.sourceRemaining != 0 || h.rm != nil {
			t.Fatal("entry constructed younger related core")
		}
	} else if h.constructed != 2 || h.sourceRemaining != 2 || h.f.size != 1 {
		t.Fatal("older residual vector lost")
	}
	if s.henselPairInvocationSourceCause(&h.receipt) == nil || !strings.Contains(w.seen.String(), line) {
		t.Fatal("exact pair receipt/source record lost")
	}
	roots := append([]*unumeral.Int(nil), args...)
	for i := 0; i < w.count; i++ {
		roots = append(roots, &w.held[i])
	}
	upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{Services: []*UPolynomialFactorizationService{s}, RationalManagers: []*unumeral.RationalManager{q}, Integers: roots})
	serviceMust(t, s.cleanupHenselPairInvocation(&h.receipt, false))
	serviceMust(t, s.cleanupHenselPairInvocation(&h.receipt, true))
	if h.active || m.displayReceiptPending(&h.trace) {
		t.Fatal("exact pair child cleanup retained older source")
	}
}

type henselPairTraceProbe struct {
	t             *testing.T
	s             *UPolynomialFactorizationService
	text          strings.Builder
	stages        map[uint8]bool
	checkBorrowed bool
}

func (w *henselPairTraceProbe) Write(p []byte) (int, error) {
	for h := w.s.henselPairs; h != nil; h = h.next {
		if h.traceBorrow.executing && (!w.stages[h.constructed] || (h.check.brCore != nil && !w.checkBorrowed)) {
			w.stages[h.constructed] = true
			if h.check.brCore != nil {
				w.checkBorrowed = true
			}
			for _, m := range [4]*UPolynomialManager{w.s.manager, h.rm, h.check.brCore, h.check.bCore} {
				if m == nil {
					continue
				}
				for _, e := range []error{m.Close(), m.SourceFinalize(), m.CleanupDisplay()} {
					if !errors.Is(e, ErrUPolynomialCleanupPending) {
						w.t.Fatal("pair TRACE callback acquired borrowed core", e)
					}
				}
			}
			for _, e := range []error{w.s.Close(), w.s.CleanupRetained(), w.s.cleanupHenselPairInvocation(&h.receipt, false)} {
				if !errors.Is(e, ErrUPolynomialCleanupPending) {
					w.t.Fatal("pair TRACE callback acquired executing invocation", e)
				}
			}
		}
	}
	return w.text.Write(p)
}
func TestUPolynomialHenselPairTraceRecordsAndBorrowedOwners(t *testing.T) {
	_, s := preparationPhysicalFixture(t)
	m := s.manager
	w := henselPairTraceProbe{t: t, s: s, stages: map[uint8]bool{}}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization")
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	u := upolySourceNew(t, m, []int32{-1})
	a := upolySourceNew(t, m, []int32{0, 1})
	v := upolySourceNew(t, m, []int32{1})
	b := upolySourceNew(t, m, []int32{1, 1})
	c := upolySourceNew(t, m, []int32{3, 1, 1})
	outA := upolySourceNew(t, m, nil)
	outB := upolySourceNew(t, m, nil)
	one, prime := unumeral.NewInt(1), unumeral.NewInt(3)
	serviceMust(t, s.HenselLiftPair(&one, &prime, &prime, mustUPolyView(t, m, u), mustUPolyView(t, m, a), mustUPolyView(t, m, v), mustUPolyView(t, m, b), mustUPolyView(t, m, c), outA, outB))
	if tags.ShouldTrace("polynomial::factorization") {
		text := w.text.String()
		for _, record := range []string{":590 ", "polynomial::hensel_lift(a = 1, b = 3, r = 3, U = -1, A = x, V = 1, B = x + 1, C = x^2 + x + 3)\n", ":614 ", "f = 1\n", ":637 ", "V == 1\n", ":641 ", "Vf = V*f\nVf == 1\n", ":646 ", "[t, S] = Vf.quo_rem(A)\nt == 0\nS == 1\n", ":653 ", "T == U*f\nT == -1\n", ":658 ", "tmp = B*t\ntmp == 0\n", ":663 ", "T = B*tmp\nT == -1\n"} {
			if !strings.Contains(text, record) {
				t.Fatal("actual pair source record", record, text)
			}
		}
		if len(w.stages) != 4 {
			t.Fatal("pair source construction stages", w.stages)
		}
	} else if w.text.Len() != 0 {
		t.Fatal("erased pair trace")
	}
	wantA := upolySourceNew(t, m, []int32{3, 1})
	wantB := upolySourceNew(t, m, []int32{-2, 1})
	for _, pair := range [2][2]UPolynomial{{outA, wantA}, {outB, wantB}} {
		equal, e := m.Equal(pair[0], pair[1])
		serviceMust(t, e)
		if !equal {
			t.Fatal("pair trace changed actual output")
		}
	}
}
