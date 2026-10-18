package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strings"
	"testing"
)

// Concrete charged caller adapter for a direct retained checker invocation.
// Its C++ parent-local construction count stays zero throughout the test.
func quadraticFailureTraceFixture(t *testing.T, slots uint32) (*unumeral.RationalManager, *UPolynomialFactorizationService, *UPolynomialManager, *upolyHenselCallerScope) {
	t.Helper()
	q, s := preparationPhysicalFixtureWithSlots(t, slots)
	zpe, e := NewRelatedUPolynomialManager(s.manager)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, zpe.Close()) })
	parent, e := s.newSquareFreeScope()
	serviceMust(t, e)
	serviceMust(t, s.beginHenselCaller(parent))
	c := &parent.hensel
	c.lift = zpe
	t.Cleanup(func() { c.executing = false; serviceMust(t, s.CleanupRetained()) })
	return q, s, zpe, c
}
func TestUPolynomialHenselQuadraticFailureTraceBranchesAndModularOwner(t *testing.T) {
	for _, kind := range []string{"nonconstant", "constant-not-one", "one"} {
		t.Run(kind, func(t *testing.T) {
			_, s, zpe, c := quadraticFailureTraceFixture(t, 0)
			m := s.manager
			seven := unumeral.NewInt(7)
			serviceMust(t, zpe.SetZP(&seven))
			w := factorDegreeTraceProbe{t: t, service: s, checked: map[*UPolynomialManager]bool{}}
			tags := diagnostic.NewTraceTagsWithWriter(&w)
			tags.Enable("polynomial::factorization::bughunt")
			serviceMust(t, m.modular.BindTrace(tags))
			t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
			av, bv, uv, vv := []int32{2, 1}, []int32{1, 1}, []int32{1}, []int32{1}
			if kind != "nonconstant" {
				av, bv, vv = []int32{2}, []int32{1}, nil
				if kind == "one" {
					av = []int32{1}
				}
			}
			a, b, u, v := upolySourceNew(t, m, av), upolySourceNew(t, m, bv), upolySourceNew(t, m, uv), upolySourceNew(t, m, vv)
			ok, e := s.checkQuadraticHenselWithCaller(zpe, mustUPolyView(t, m, u), mustUPolyView(t, m, a), mustUPolyView(t, m, v), mustUPolyView(t, m, b), c)
			serviceMust(t, e)
			if ok != (kind == "one") {
				t.Fatal("checker result", kind, ok)
			}
			if kind == "one" || !tags.ShouldTrace("polynomial::factorization::bughunt") {
				if w.seen.Len() != 0 || w.headerChecked {
					t.Fatal("unreached failure trace evaluated output")
				}
			} else {
				text := w.seen.String()
				expected := "sage: R.<x> = Zmod(7)['x']\nsage: A = x + 2\nsage: B = x + 1\nsage: U = 1\nsage: V = 1\nsage: print  (1 - UA - VB)\nsage: print 'expected 0'\n"
				if kind == "constant-not-one" {
					expected = "sage: R.<x> = Zmod(7)['x']\nsage: A = 2\nsage: B = 1\nsage: U = 1\nsage: V = 0\nsage: print  (1 - UA - VB)\nsage: print 'expected 0'\n"
				}
				if !strings.Contains(text, expected) {
					t.Fatal("exact source diagnostic body", text)
				}

				for _, part := range []string{":688 ", "sage: R.<x> = Zmod(7)['x']\n", "sage: A = ", "sage: B = ", "sage: U = ", "sage: V = ", "sage: print  (1 - UA - VB)\nsage: print 'expected 0'\n"} {
					if !strings.Contains(text, part) {
						t.Fatal("whole failure record", part, text)
					}
				}
				if !w.headerChecked || !w.flushChecked || !w.polynomialChecked[zpe] || w.polynomialChecked[m] {
					t.Fatal("wrong diagnostic display owner or missing borrow")
				}
			}
			if c.quadraticCheckSource != 0 {
				t.Fatal("normal diagnostic left source checker vectors")
			}
		})
	}
}

func TestUPolynomialHenselQuadraticFailureTraceExactBIGChild(t *testing.T) {
	q, s, zpe, c := quadraticFailureTraceFixture(t, 256)
	m, z := s.manager, q.Integers()
	var prime, n unumeral.Int
	t.Cleanup(func() { serviceMust(t, z.Del(&n)); serviceMust(t, z.Del(&prime)) })
	serviceMust(t, z.SetLimbs(&prime, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
	one, two := unumeral.NewInt(1), unumeral.NewInt(2)
	serviceMust(t, z.Sub(&prime, &one, &n))
	serviceMust(t, z.Div(&n, &two, &n))
	serviceMust(t, zpe.SetZP(&prime))
	a, e := m.NewPolynomial()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.DelPolynomial(&a)) })
	serviceMust(t, m.SetInts(a, []unumeral.Int{n}))
	u, b, v := upolySourceNew(t, m, []int32{1}), upolySourceNew(t, m, []int32{1}), upolySourceNew(t, m, nil)
	w := factorDegreeTraceFailure{t: t, z: z, limit: 256, held: make([]unumeral.Int, 256), trigger: "sage: A = "}
	t.Cleanup(func() {
		c.executing = false
		serviceMust(t, s.CleanupRetained())
		for i := 0; i < w.count; i++ {
			serviceMust(t, z.Del(&w.held[i]))
		}
	})
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	ok, body := s.checkQuadraticHenselWithCaller(zpe, mustUPolyView(t, m, u), mustUPolyView(t, m, a), mustUPolyView(t, m, v), mustUPolyView(t, m, b), c)
	if !tags.ShouldTrace("polynomial::factorization::bughunt") {
		serviceMust(t, body)
		if ok {
			t.Fatal("invalid invariant accepted")
		}
		return
	}
	if ok || !errors.Is(body, arena.ErrSlotLimit) || !w.armed || c.quadraticCheckSource != 3 || !zpe.displayReceiptPending(&c.owner.tracePolynomial) || c.owner.constructed != 0 {
		t.Fatal("exact failed checker display/frontier", body, c.quadraticCheckSource)
	}
	roots := []*unumeral.Int{&prime, &n}
	for i := 0; i < w.count; i++ {
		roots = append(roots, &w.held[i])
	}
	upolyHenselAuditPhysical(t, []*UPolynomialManager{m, zpe}, nil, upolyPhysicalExtraOwners{Services: []*UPolynomialFactorizationService{s}, RationalManagers: []*unumeral.RationalManager{q}, Integers: roots})
	c.executing = false
	serviceMust(t, s.CleanupRetained())
	if s.scope != nil {
		t.Fatal("checker child cleanup retained adapter")
	}
}

func TestUPolynomialHenselQuadraticFailureTraceStreamFailure(t *testing.T) {
	_, s, zpe, c := quadraticFailureTraceFixture(t, 0)
	m := s.manager
	seven := unumeral.NewInt(7)
	serviceMust(t, zpe.SetZP(&seven))
	w := factorDegreeRefusedTrace{cause: errors.New("refused quadratic diagnostic output")}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	a, b, u, v := upolySourceNew(t, m, []int32{2, 1}), upolySourceNew(t, m, []int32{1, 1}), upolySourceNew(t, m, []int32{1}), upolySourceNew(t, m, []int32{1})
	before := zpe.displayNonce
	ok, e := s.checkQuadraticHenselWithCaller(zpe, mustUPolyView(t, m, u), mustUPolyView(t, m, a), mustUPolyView(t, m, v), mustUPolyView(t, m, b), c)
	serviceMust(t, e)
	if ok {
		t.Fatal("invalid identity accepted after output refusal")
	}
	if tags.ShouldTrace("polynomial::factorization::bughunt") {
		if w.calls != 1 || w.flushes != 1 || zpe.displayNonce-before != 4 {
			t.Fatal("ostream badbit skipped displays/endl or retried output", w.calls, w.flushes, zpe.displayNonce-before)
		}
	} else if w.calls != 0 || zpe.displayNonce != before {
		t.Fatal("erased trace evaluated display")
	}
	if c.quadraticCheckSource != 0 || c.owner.tracePending() {
		t.Fatal("stream refusal retained numeric source child")
	}
}
