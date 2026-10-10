// Source: src/math/polynomial/polynomial.cpp:1821-1830,3344-3359,3405-3430; src/util/mpzzp.h.
package polynomial

import (
	"fmt"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialContentModeSwitchSourceSet(t *testing.T) {
	for _, scalar := range []bool{false, true} {
		t.Run(map[bool]string{false: "content", true: "scalar"}[scalar], func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			p := polynomialQueryTerms(t, c, []int32{7}, [][]MonomialPower{nil})
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			var i unumeral.Int
			t.Cleanup(func() { serviceMust(t, c.modular.SetZ()); serviceMust(t, c.integer.Del(&i)); serviceMust(t, r.Close()) })
			serviceMust(t, c.modular.SetZPUint64(5))
			if scalar {
				serviceMust(t, full.IntegerScalarContent(p, &i))
			} else {
				serviceMust(t, c.IntegerContent(p, &i, r))
			}
			if i != unumeral.NewInt(2) {
				t.Fatalf("source m_manager is mpzzp; SET(7) at prime5 must normalize to2; got %v", i)
			}
		})
	}
}

func TestPolynomialIntegerContentFieldEveryActualReservationPrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 128 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			serviceMust(t, c.modular.SetZPUint64(23))
			p := polynomialQueryTerms(t, c, []int32{6, 10}, [][]MonomialPower{{{0, 1}}, nil})
			out, e := NewPolynomialRef(c)
			serviceMust(t, e)
			var i unumeral.Int
			t.Cleanup(func() {
				c.integer.Budget().FailAfter(-1)
				serviceMust(t, c.CleanupIntegerContent())
				serviceMust(t, c.integer.Del(&i))
				serviceMust(t, out.Close())
			})
			c.integer.Budget().FailAfter(ordinal)
			e = c.IntegerContent(p, &i, out)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual field content refusal", e)
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, out.root.value, out.root.incoming}, []*unumeral.Int{&i}, out.header)
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, c.CleanupIntegerContent())
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, out.root.value, out.root.incoming}, []*unumeral.Int{&i}, out.header)
			if e == nil {
				if i != unumeral.NewInt(2) {
					t.Fatal("source field integer GCD", i)
				}
				polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{1, 0, 0}: 3, {}: 5})
				if c.modular.DivTmp != unumeral.NewInt(-11) || c.modular.InvTmp3 != unumeral.NewInt(1) {
					t.Fatal("actual source div/inverse scratch", c.modular.DivTmp, c.modular.InvTmp3)
				}
				success = true
				t.Log("first actual success horizon", ordinal)
			}
		})
	}
	if !success {
		t.Fatal("field content did not reach first success")
	}
}
func TestPolynomialICCPZPXModeSwitchSourceSet(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{7}, [][]MonomialPower{nil})
	cr, e := NewPolynomialRef(c)
	serviceMust(t, e)
	pr, e := NewPolynomialRef(c)
	serviceMust(t, e)
	var i unumeral.Int
	t.Cleanup(func() {
		serviceMust(t, c.modular.SetZ())
		serviceMust(t, c.integer.Del(&i))
		serviceMust(t, cr.Close())
		serviceMust(t, pr.Close())
	})
	serviceMust(t, c.modular.SetZPUint64(5))
	serviceMust(t, full.iccpZPX(p, 0, &i, &cr.root, &pr.root))
	if i != unumeral.NewInt(2) || cr.root.value != c.unit || pr.root.value != c.unit {
		t.Fatal("source modular constant shortcut", i, cr.root.value, pr.root.value)
	}
}
