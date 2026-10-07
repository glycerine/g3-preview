// Source: src/math/polynomial/polynomial.cpp:3384-3428.
package polynomial

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialIntegerContentIndependentSignedBranches(t *testing.T) {
	for _, which := range []uint32{0, 1, 2, 3} {
		m := polynomialCellFixture(t)
		p, want := m.zero, m.zero
		expected := int32(0)
		switch which {
		case 1:
			p = polynomialQueryTerms(t, m, []int32{-7}, [][]MonomialPower{nil})
			want = m.unit
			expected = -7
		case 2:
			p = polynomialQueryTerms(t, m, []int32{-6, 10, -14}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
			want = polynomialQueryTerms(t, m, []int32{-3, 5, -7}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
			expected = 2
		case 3:
			p = polynomialQueryTerms(t, m, []int32{-3, 5, -7}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
			want = p
			expected = 1
		}
		i := util.NewInt(99)
		c, e := NewPolynomialRef(m)
		serviceMust(t, e)
		serviceMust(t, c.Assign(m.unit))
		m.config.Checkpoint = func() bool { t.Fatal("integer content has no source checkpoint"); return false }
		serviceMust(t, m.IntegerContent(p, &i, c))
		if i != util.NewInt(expected) {
			t.Fatal("independent integer content", which, i, expected)
		}
		eq, e := m.Equal(c.root.value, want)
		serviceMust(t, e)
		if !eq {
			t.Fatal("primitive polynomial", which)
		}
		if which == 0 || which == 3 {
			if c.root.value != p {
				t.Fatal("source direct identity branch")
			}
		}
		serviceMust(t, c.Close())
		serviceMust(t, m.modular.Del(&i))
	}
}

func TestPolynomialIntegerContentEveryActualReservationPrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 128 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			m := full.core
			p := polynomialQueryTerms(t, m, []int32{6, 10}, [][]MonomialPower{{{0, 1}}, nil})
			want := polynomialQueryTerms(t, m, []int32{3, 5}, [][]MonomialPower{{{0, 1}}, nil})
			out, e := NewPolynomialRef(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, out.Close()) })
			serviceMust(t, out.Assign(m.unit))
			t.Cleanup(func() { m.integer.Budget().FailAfter(-1); serviceMust(t, m.CleanupIntegerContent()) })
			i := util.NewInt(99)
			t.Cleanup(func() { serviceMust(t, m.integer.Del(&i)) })
			m.integer.Budget().FailAfter(ordinal)
			e = m.IntegerContent(p, &i, out)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual content reservation category", e)
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, want, out.root.value}, []*util.Int{&i}, out.header)
			m.integer.Budget().FailAfter(-1)
			serviceMust(t, m.CleanupIntegerContent())
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, want, out.root.value}, []*util.Int{&i}, out.header)
			if e == nil {
				if i != util.NewInt(2) {
					t.Fatal("independent positive gcd", i)
				}
				equal, x := m.Equal(out.root.value, want)
				serviceMust(t, x)
				if !equal {
					t.Fatal("independent primitive part")
				}
				success = true
				t.Log("every actual refusal before source completion", ordinal)
			}
		})
	}
	if !success {
		t.Fatal("no complete content source route")
	}
}
