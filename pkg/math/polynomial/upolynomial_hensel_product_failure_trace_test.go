package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strings"
	"testing"
)

func productFailureTraceFixture(t *testing.T, slots uint32) (*unumeral.RationalManager, *UPolynomialFactorizationService, *UPolynomialManager, *UPolynomialManager, *upolyHenselCallerScope, *UPolynomialFactors, *UPolynomialFactors) {
	t.Helper()
	q, s, zpe, c := quadraticFailureTraceFixture(t, slots)
	zp, e := NewRelatedUPolynomialManager(s.manager)
	serviceMust(t, e)
	c.prime = zp
	pf, e := zp.NewFactors()
	serviceMust(t, e)
	lf, e := zpe.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() {
		c.executing = false
		serviceMust(t, s.CleanupRetained())
		serviceMust(t, lf.Close())
		serviceMust(t, pf.Close())
		serviceMust(t, zp.Close())
	})
	return q, s, zp, zpe, c, pf, lf
}
func TestUPolynomialHenselProductFailureTraceSourceBranches(t *testing.T) {
	for _, kind := range []string{"prime", "lifted", "count", "bound", "one"} {
		t.Run(kind, func(t *testing.T) {
			_, s, zp, zpe, c, pf, lf := productFailureTraceFixture(t, 0)
			m := s.manager
			p, pe := unumeral.NewInt(5), unumeral.NewInt(25)
			if kind == "bound" {
				pe = unumeral.NewInt(5)
			}
			serviceMust(t, zp.SetZP(&p))
			serviceMust(t, zpe.SetZP(&pe))
			first := []int32{-1, 1}
			if kind == "prime" {
				first = []int32{-2, 1}
			}
			for _, coeff := range [][]int32{first, {1, 1}} {
				poly := upolySourceNew(t, m, coeff)
				serviceMust(t, pf.PushCopy(mustUPolyView(t, m, poly), 1))
			}
			first = []int32{-1, 1}
			if kind == "lifted" {
				first = []int32{4, 1}
			}
			poly := upolySourceNew(t, m, first)
			serviceMust(t, lf.PushCopy(mustUPolyView(t, m, poly), 1))
			if kind != "count" {
				poly = upolySourceNew(t, m, []int32{1, 1})
				serviceMust(t, lf.PushCopy(mustUPolyView(t, m, poly), 1))
			}
			f := upolySourceNew(t, m, []int32{-1, 0, 1})
			w := factorDegreeTraceProbe{t: t, service: s, checked: map[*UPolynomialManager]bool{}}
			tags := diagnostic.NewTraceTagsWithWriter(&w)
			tags.Enable("polynomial::factorization::bughunt")
			serviceMust(t, m.modular.BindTrace(tags))
			t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
			ok, e := s.checkHenselFactorsWithCaller(mustUPolyView(t, m, f), pf, lf, 2, c)
			serviceMust(t, e)
			if ok != (kind == "one") {
				t.Fatal("checker source branch", kind, ok)
			}
			if (kind != "prime" && kind != "lifted") || !tags.ShouldTrace("polynomial::factorization::bughunt") {
				if w.seen.Len() != 0 {
					t.Fatal("unreached product diagnostic")
				}
			} else {
				text := w.seen.String()
				expected := "f = x^2 - 1\nzp_fs = 1 * (x - 2)^1 * (x + 1)^1\nsage: R.<x> = Zmod(5)['x']\nsage: mult_zp = x^2 - x - 2\nsage: f_zp = x^2 - 1\nsage: mult_zp == f_zp\n"
				site := ":826 "
				if kind == "lifted" {
					site = ":848 "
					expected = "f = x^2 - 1\nzpe_fs = 1 * (x + 4)^1 * (x + 1)^1\nsage: R.<x> = Zmod(25)['x']\nsage: mult_zpe = x^2 + 5 x + 4\nsage: f_zpe = x^2 - 1\nsage: mult_zpe == f_zpe\n"
				}
				if !strings.Contains(text, site) || !strings.Contains(text, expected) || !w.headerChecked || !w.flushChecked || len(w.checked) != 1 {
					t.Fatal("whole product mismatch record/owners", text)
				}
			}
			if c.checkSource != 0 || !c.checkPowerEnded {
				t.Fatal("normal checker source cleanup")
			}
		})
	}
}

func TestUPolynomialHenselProductFailureTraceExactBIGChildren(t *testing.T) {
	for _, lifted := range []bool{false, true} {
		name := "prime"
		if lifted {
			name = "lifted"
		}
		t.Run(name, func(t *testing.T) {
			q, s, zp, zpe, c, pf, lf := productFailureTraceFixture(t, 256)
			m, z := s.manager, q.Integers()
			var p, pe, n unumeral.Int
			t.Cleanup(func() { serviceMust(t, z.Del(&n)); serviceMust(t, z.Del(&pe)); serviceMust(t, z.Del(&p)) })
			serviceMust(t, z.SetLimbs(&p, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
			serviceMust(t, z.Mul(&p, &p, &pe))
			one, two := unumeral.NewInt(1), unumeral.NewInt(2)
			serviceMust(t, z.Sub(&p, &one, &n))
			serviceMust(t, z.Div(&n, &two, &n))
			serviceMust(t, zp.SetZP(&p))
			serviceMust(t, zpe.SetZP(&pe))
			for i, fs := range [2]*UPolynomialFactors{pf, lf} {
				first := unumeral.NewInt(-1)
				if (i == 0 && !lifted) || (i == 1 && lifted) {
					first = n
				}
				for _, coeff := range [][]unumeral.Int{{first, one}, {one, one}} {
					poly, e := m.NewPolynomial()
					serviceMust(t, e)
					t.Cleanup(func() { serviceMust(t, m.DelPolynomial(&poly)) })
					serviceMust(t, m.SetInts(poly, coeff))
					serviceMust(t, fs.PushCopy(mustUPolyView(t, m, poly), 1))
				}
			}
			f := upolySourceNew(t, m, []int32{-1, 0, 1})
			trigger := "zp_fs = "
			if lifted {
				trigger = "zpe_fs = "
			}
			w := factorDegreeTraceFailure{t: t, z: z, limit: 256, held: make([]unumeral.Int, 256), trigger: trigger}
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
			ok, body := s.checkHenselFactorsWithCaller(mustUPolyView(t, m, f), pf, lf, 2, c)
			if !tags.ShouldTrace("polynomial::factorization::bughunt") {
				serviceMust(t, body)
				if ok {
					t.Fatal("unequal product accepted")
				}
				return
			}
			count := uint8(2)
			if lifted {
				count = 4
			}
			if ok || !errors.Is(body, arena.ErrSlotLimit) || !w.armed || c.checkSource != count || !c.checkPowerConstructed || c.checkPowerEnded || !c.owner.trace.pending() || c.owner.constructed != 0 {
				t.Fatal("product display child before vectors/power", body, c.checkSource)
			}
			roots := []*unumeral.Int{&p, &pe, &n}
			for i := 0; i < w.count; i++ {
				roots = append(roots, &w.held[i])
			}
			upolyHenselAuditPhysical(t, []*UPolynomialManager{m, zp, zpe}, []*UPolynomialFactors{pf, lf}, upolyPhysicalExtraOwners{Services: []*UPolynomialFactorizationService{s}, RationalManagers: []*unumeral.RationalManager{q}, Integers: roots})
			c.executing = false
			serviceMust(t, s.CleanupRetained())
			if s.scope != nil {
				t.Fatal("product child cleanup retained caller adapter")
			}
		})
	}
}

func TestUPolynomialHenselProductFailureTraceStreamFailure(t *testing.T) {
	_, s, zp, zpe, c, pf, lf := productFailureTraceFixture(t, 0)
	m := s.manager
	p, pe := unumeral.NewInt(5), unumeral.NewInt(25)
	serviceMust(t, zp.SetZP(&p))
	serviceMust(t, zpe.SetZP(&pe))
	for _, coeff := range [][]int32{{-2, 1}, {1, 1}} {
		poly := upolySourceNew(t, m, coeff)
		serviceMust(t, pf.PushCopy(mustUPolyView(t, m, poly), 1))
	}
	f := upolySourceNew(t, m, []int32{-1, 0, 1})
	w := factorDegreeRefusedTrace{cause: errors.New("refused product diagnostic output")}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	rootBefore, primeBefore := m.displayNonce, zp.displayNonce
	ok, e := s.checkHenselFactorsWithCaller(mustUPolyView(t, m, f), pf, lf, 2, c)
	serviceMust(t, e)
	if ok {
		t.Fatal("unequal product accepted after output refusal")
	}
	if tags.ShouldTrace("polynomial::factorization::bughunt") {
		if w.calls != 1 || w.flushes != 1 || m.displayNonce-rootBefore != 3 || zp.displayNonce-primeBefore != 2 {
			t.Fatal("badbit skipped actual polynomial/factor displays", w.calls, w.flushes, m.displayNonce-rootBefore, zp.displayNonce-primeBefore)
		}
	} else if w.calls != 0 || m.displayNonce != rootBefore || zp.displayNonce != primeBefore {
		t.Fatal("erased product diagnostic evaluated operands")
	}
	if c.checkSource != 0 || !c.checkPowerEnded || c.owner.tracePending() {
		t.Fatal("output refusal changed checker source cleanup")
	}
}
