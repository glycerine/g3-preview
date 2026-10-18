package polynomial

import unumeral "github.com/glycerine/g3/pkg/util/numeral"

// Source: upolynomial.cpp:2833,2844,2845,2849. Gate before actual mixed
// binary/rational comparisons, including the separate lower/upper predicates.
func (m *UPolynomialManager) convertBoundAssertion(line int32, bqm *unumeral.BinaryRationalManager, x *unumeral.BinaryRational, qm *unumeral.RationalManager, y *unumeral.Rational) error {
	enabled, e := refinementSourceAssertionsEnabled()
	if e != nil || !enabled {
		return e
	}
	var valid bool
	var expression string
	switch line {
	case 2833:
		valid, e = bqm.EqualRational(x, qm, y)
		expression = "bqm.eq(c, a)"
	case 2844:
		valid, e = bqm.LessRational(x, qm, y)
		expression = "bqm.lt(lower, a)"
	case 2845:
		valid, e = bqm.GreaterRational(x, qm, y)
		expression = "bqm.gt(upper, a)"
	case 2849:
		valid, e = bqm.LessRational(x, qm, y)
		expression = "bqm.lt(upper, b)"
	}
	if e != nil {
		return e
	}
	if !valid {
		return m.scalarDivisionAssertionFailure(line, expression)
	}
	return nil
}
