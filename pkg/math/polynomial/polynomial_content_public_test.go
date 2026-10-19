package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func polynomialPublicContentInput(t *testing.T, c *MultivariatePolynomialCore) Polynomial {
	return polynomialQueryTerms(t, c, []int32{6, 12, 9, 18, 3, 6}, [][]MonomialPower{{{0, 2}, {1, 1}}, {{0, 2}}, {{0, 1}, {1, 1}}, {{0, 1}}, {{1, 1}}, nil})
}
func TestPolynomialPublicContentIntegerAndPolynomialFactors(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialPublicContentInput(t, c)
	want := polynomialQueryTerms(t, c, []int32{3, 6}, [][]MonomialPower{{{1, 1}}, nil})
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	defer func() { serviceMust(t, out.Close()) }()
	serviceMust(t, m.Content(p, 0, out))
	equal, e := c.Equal(want, out.root.value)
	serviceMust(t, e)
	if !equal {
		t.Fatal("content must retain integer factor 3 and polynomial factor y+2")
	}
	for _, n := range []int32{0, -6, 1} {
		value := unumeral.NewInt(n)
		constant, e := c.MkConstantTake(&value)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(constant))
		serviceMust(t, m.Content(constant, 0, out))
		equal, e = c.Equal(constant, out.root.value)
		serviceMust(t, e)
		if !equal {
			t.Fatal("constant content", n)
		}
		serviceMust(t, c.DecRef(constant))
	}
}
func TestPolynomialPublicContentEveryReservationAndAlias(t *testing.T) {
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		m := polynomialPSCFullFixture(t)
		c := m.core
		p := polynomialPublicContentInput(t, c)
		want := polynomialQueryTerms(t, c, []int32{3, 6}, [][]MonomialPower{{{1, 1}}, nil})
		out, e := NewPolynomialRef(c)
		serviceMust(t, e)
		serviceMust(t, out.root.assign(c, p))
		c.integer.Budget().FailAfter(ordinal)
		body := m.Content(p, 0, out)
		c.integer.Budget().FailAfter(-1)
		if body != nil && !numericPhysicalReservationRefusal(body) {
			t.Fatal("content source prefix", ordinal, body)
		}
		serviceMust(t, m.CleanupGCD())
		if body == nil {
			equal, e := c.Equal(want, out.root.value)
			serviceMust(t, e)
			if !equal {
				t.Fatal("aliased content independent polynomial result")
			}
			serviceMust(t, out.Close())
			t.Log("completed reservation ordinal", ordinal)
			return
		}
		serviceMust(t, out.Close())
	}
	t.Fatal("content never completed")
}

func TestPolynomialPublicContentInnerReferenceEndsBeforeOuterAssignment(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialPublicContentInput(t, c)
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	defer func() { serviceMust(t, out.Close()) }()
	hits := 0
	h, e := NewPolynomialDeletionHandler(c.integer.Budget(), func(_ *PolynomialDeletionHandler, c *MultivariatePolynomialCore, deleted Polynomial) error {
		if !out.root.assigning || out.root.value != deleted || m.gcdDepth != 1 || m.gcdFrames[0].nums[0] != unumeral.NewInt(3) {
			return nil
		}
		hits++
		if len(m.gcdFrames) < 2 || m.gcdFrames[1].depth != 0 || m.gcdFrames[1].refs[0].root.value != (Polynomial{}) {
			t.Fatal("inner pp survives into outer scalar-content assignment")
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	serviceMust(t, m.Content(p, 0, out))
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
	if hits == 0 {
		t.Fatal("actual outer assignment deletion was not observed")
	}
}
