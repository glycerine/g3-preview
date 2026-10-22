package smtcore

import (
	"math/big"
	"testing"
)

func TestSMTArithmeticValueLexicographicAndAtomComplements(t *testing.T) {
	finite := newSMTArithmeticValue(big.NewRat(4, 1), nil)
	strict := newSMTArithmeticValue(big.NewRat(4, 1), big.NewRat(1, 1))
	infinite := newSMTArithmeticValue(nil, nil)
	infinite.infinity.SetInt64(1)
	if finite.cmp(&strict) >= 0 || strict.cmp(&infinite) >= 0 {
		t.Fatal("arithmetic value lexicographic order")
	}
	epsilon := newSMTArithmeticValue(nil, big.NewRat(1, 1))
	lower, err := newSMTArithmeticAtom(3, 7, &finite, smtArithmeticLowerAtom)
	if err != nil {
		t.Fatal(err)
	}
	if err = lower.assign(false, &epsilon); err != nil {
		t.Fatal(err)
	}
	if lower.kind != smtArithmeticUpperBound || lower.value.rational.Cmp(big.NewRat(4, 1)) != 0 || lower.value.infinitesimal.Cmp(big.NewRat(-1, 1)) != 0 {
		t.Fatal("false lower atom complement")
	}
	upper, err := newSMTArithmeticAtom(4, 8, &finite, smtArithmeticUpperAtom)
	if err != nil {
		t.Fatal(err)
	}
	if err = upper.assign(false, &epsilon); err != nil {
		t.Fatal(err)
	}
	if upper.kind != smtArithmeticLowerBound || upper.value.infinitesimal.Cmp(big.NewRat(1, 1)) != 0 {
		t.Fatal("false upper atom complement")
	}
}

func TestSMTArithmeticAntecedentAndDerivedCoefficients(t *testing.T) {
	derived := &smtArithmeticJustifiedDerivedBound{}
	literal := NewSMTLiteral(5, false)
	if err := derived.pushLiteral(literal, big.NewRat(2, 1)); err != nil {
		t.Fatal(err)
	}
	if err := derived.pushLiteral(literal, big.NewRat(3, 1)); err != nil {
		t.Fatal(err)
	}
	var out smtArithmeticAntecedents
	if err := derived.pushJustification(&out, big.NewRat(-2, 1), true); err != nil {
		t.Fatal(err)
	}
	if len(out.literals) != 1 || out.literals[0] != literal || out.literalCoefficients[0].Cmp(big.NewRat(-10, 1)) != 0 || out.numParameters() != 2 {
		t.Fatal("scaled derived antecedent")
	}
	out.reset()
	if len(out.literals) != 0 || out.numParameters() != 0 {
		t.Fatal("antecedent reset")
	}
	atom, err := newSMTArithmeticAtom(6, 1, newSMTArithmeticValuePtr(0, 0), smtArithmeticLowerAtom)
	if err != nil {
		t.Fatal(err)
	}
	if err = atom.assign(true, newSMTArithmeticValuePtr(0, 1)); err != nil {
		t.Fatal(err)
	}
	if err = atom.pushJustification(&out, big.NewRat(7, 1), false); err != nil {
		t.Fatal(err)
	}
	if len(out.literals) != 1 || out.literals[0] != NewSMTLiteral(6, false) || out.numParameters() != 0 {
		t.Fatal("atom antecedent")
	}
}

func newSMTArithmeticValuePtr(rational, infinitesimal int64) *smtArithmeticValue {
	value := newSMTArithmeticValue(big.NewRat(rational, 1), big.NewRat(infinitesimal, 1))
	return &value
}

func TestSMTArithmeticBoundTrailEncoding(t *testing.T) {
	bound := &smtArithmeticBound{variable: 9}
	trail, err := newSMTArithmeticBoundTrail(9, bound, true)
	if err != nil {
		t.Fatal(err)
	}
	if !trail.upper() || trail.variable() != 9 || trail.old != bound {
		t.Fatal("upper bound trail encoding")
	}
	trail, err = newSMTArithmeticBoundTrail(10, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if trail.upper() || trail.variable() != 10 {
		t.Fatal("lower bound trail encoding")
	}
}
