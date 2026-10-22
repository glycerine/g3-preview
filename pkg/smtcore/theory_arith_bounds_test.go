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

func TestSMTArithmeticEqualityBoundAndAccumulation(t *testing.T) {
	root := &SMTEnode{}
	left, right := &SMTEnode{root: root}, &SMTEnode{root: root}
	value := newSMTArithmeticValue(big.NewRat(3, 1), nil)
	equality, err := newSMTArithmeticEqualityBound(2, &value, smtArithmeticLowerBound, left, right)
	if err != nil {
		t.Fatal(err)
	}
	var out smtArithmeticAntecedents
	if err = equality.pushJustification(&out, big.NewRat(4, 1), true); err != nil {
		t.Fatal(err)
	}
	if len(out.equalities) != 1 || out.equalityCoefficients[0].Cmp(big.NewRat(4, 1)) != 0 {
		t.Fatal("equality antecedent")
	}
	wrong := &SMTEnode{}
	wrong.root = wrong
	if _, err = newSMTArithmeticEqualityBound(2, &value, smtArithmeticLowerBound, left, wrong); err == nil {
		t.Fatal("different equality roots accepted")
	}
	atom, err := newSMTArithmeticAtom(8, 2, &value, smtArithmeticLowerAtom)
	if err != nil {
		t.Fatal(err)
	}
	if err = atom.assign(true, newSMTArithmeticValuePtr(0, 1)); err != nil {
		t.Fatal(err)
	}
	target := &smtArithmeticDerivedBound{}
	seenLiterals := make(map[int32]bool)
	seenEqualities := make(map[SMTEnodePair]bool)
	if err = smtAccumulateArithmeticJustification(atom, target, big.NewRat(2, 1), false, seenLiterals, seenEqualities); err != nil {
		t.Fatal(err)
	}
	if err = smtAccumulateArithmeticJustification(atom, target, big.NewRat(3, 1), false, seenLiterals, seenEqualities); err != nil {
		t.Fatal(err)
	}
	if len(target.literals) != 1 {
		t.Fatal("non-proof accumulation did not suppress duplicate")
	}
}

func TestSMTNormalizeArithmeticIntegerBound(t *testing.T) {
	for _, row := range []struct {
		rational *big.Rat
		epsilon  int64
		kind     smtArithmeticBoundKind
		want     int64
	}{
		{big.NewRat(3, 2), 0, smtArithmeticLowerBound, 2},
		{big.NewRat(3, 2), 0, smtArithmeticUpperBound, 1},
		{big.NewRat(-3, 2), 0, smtArithmeticLowerBound, -1},
		{big.NewRat(-3, 2), 0, smtArithmeticUpperBound, -2},
		{big.NewRat(2, 1), 1, smtArithmeticLowerBound, 3},
		{big.NewRat(2, 1), -1, smtArithmeticUpperBound, 1},
	} {
		value := newSMTArithmeticValue(row.rational, big.NewRat(row.epsilon, 1))
		got, err := smtNormalizeArithmeticIntegerBound(&value, row.kind)
		if err != nil {
			t.Fatal(err)
		}
		if got.rational.Cmp(big.NewRat(row.want, 1)) != 0 || got.infinitesimal.Sign() != 0 {
			t.Fatalf("normalize %s eps %d kind %d = %s", row.rational, row.epsilon, row.kind, &got.rational)
		}
	}
}
