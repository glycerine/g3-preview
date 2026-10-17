package polynomial

import (
	"errors"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialToNumeralExternalOwnerAndNumeralMode(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c, z := full.core, full.core.integer
	p := polynomialQueryTerms(t, c, []int32{3}, [][]MonomialPower{{{0, 2}}})
	var big, coefficient unumeral.Int
	one := unumeral.NewInt(1)
	serviceMust(t, z.Mul2k(&one, 129, &big))
	serviceMust(t, z.Copy(&coefficient, &big))
	constant, e := c.MkConstantTake(&coefficient)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(constant))
	combined, e := c.Add(p, constant)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(combined))
	external, e := NewUPolynomialManager(UPolynomialConfig{Integer: z, Checkpoint: full.upm.config.Checkpoint, Cooperation: full.upm.config.Cooperation})
	serviceMust(t, e)
	out, e := external.NewPolynomial()
	serviceMust(t, e)
	defer func() {
		serviceMust(t, external.DelPolynomial(&out))
		serviceMust(t, external.Close())
		serviceMust(t, c.DecRef(combined))
		serviceMust(t, c.DecRef(constant))
		serviceMust(t, z.Del(&big))
	}()
	serviceMust(t, full.ToNumeralVectorInto(combined, external, out))
	v, e := external.wholeView(out)
	serviceMust(t, e)
	value, e := external.read(v, 0)
	serviceMust(t, e)
	cmp, e := z.Compare(value, &big)
	serviceMust(t, e)
	if cmp != 0 || full.factorNumeralVector.outputManager != external || len(external.bindings) != 0 {
		t.Fatal("conversion used a copied or internally owned coefficient store")
	}
	// The actual receiver's numeral service, rather than full.upm's service,
	// controls copy normalization. 2^129 is 1 in GF(7).
	serviceMust(t, external.SetZPUint64(7))
	serviceMust(t, full.ToNumeralVectorInto(combined, external, out))
	checkAlgebraicResultantPolynomial(t, external, out, 1, 0, 3)
}

func TestPolynomialToNumeralExternalRejectsOtherIntegerService(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	other := polynomialPSCFullFixture(t)
	p := polynomialQueryTerms(t, full.core, []int32{1, 2}, [][]MonomialPower{nil, {{0, 1}}})
	out, e := other.upm.NewPolynomial()
	serviceMust(t, e)
	defer func() { serviceMust(t, other.upm.DelPolynomial(&out)) }()
	serviceMust(t, other.upm.SetInts(out, []unumeral.Int{unumeral.NewInt(17)}))
	if e = full.ToNumeralVectorInto(p, other.upm, out); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal("accepted a foreign integer service", e)
	}
	checkAlgebraicResultantPolynomial(t, other.upm, out, 17)
	if full.factorNumeralVector.active {
		t.Fatal("refused receiver constructed a conversion scope")
	}
}
