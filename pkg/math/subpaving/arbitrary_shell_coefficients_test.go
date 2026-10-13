package subpaving

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	umm "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestArbitraryShellCoefficientSourceConversionAndCleanup(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	r, err := umm.NewRuntime(umm.RuntimeConfig{Budget: b, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	config := numeral.IntegerConfig{
		Memory: r, Profile: umm.SmallAllocatorProfile{PointerBits: 64, Debug: true},
		Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}},
	}
	core, err := numeral.NewArbitraryFloatManager(config)
	if err != nil {
		t.Fatal(err)
	}
	numbers, err := numeral.NewArbitraryFloatNumeralManager(core, 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	qm, err := core.BorrowRationalManager()
	if err != nil {
		t.Fatal(err)
	}
	context, err := NewArbitraryContext(numbers, &control.Reslimit{Limit: 1000}, 0, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewArbitraryShellCoefficients(context)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewArbitraryShellCoefficients(nil); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("missing source context: %v", err)
	}
	copyOwner := *w
	if _, _, err := copyOwner.PrepareSum(&numeral.Int{Val: 1}, nil); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("copied coefficient owner: %v", err)
	}
	if err := copyOwner.Close(); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("copied owner cleanup: %v", err)
	}
	one, big, three := numeral.NewInt(1), numeral.Int{}, numeral.NewInt(3)
	q := numeral.NewRational()
	t.Cleanup(func() {
		for _, e := range []error{w.Close(), context.Close(), qm.Del(&q), qm.Z.Del(&big), numbers.Close(), core.Close(), r.Close()} {
			if e != nil {
				t.Error(e)
			}
		}
		if b.Used() != 0 {
			t.Error("MPF coefficient owners", b.Used())
		}
	})
	if err := qm.Z.Power(&numeral.Int{Val: 2}, 53, &big); err != nil {
		t.Fatal(err)
	}
	if err := qm.Z.Add(&big, &one, &big); err != nil {
		t.Fatal(err)
	}
	if _, _, err := w.PrepareSum(&three, []*numeral.Int{&one, &big}); !errors.Is(err, ErrArbitraryShellInexactInteger) {
		t.Fatalf("rounded large coefficient: %v", err)
	}
	if err := qm.Z.Sub(&big, &one, &big); err != nil {
		t.Fatal(err)
	}
	constant, coefficients, err := w.PrepareSum(&three, []*numeral.Int{&one, &big})
	if err != nil {
		t.Fatal(err)
	}
	if len(coefficients) != 2 {
		t.Fatal("coefficient order and count", len(coefficients))
	}
	for i, tc := range []struct {
		value *numeral.ArbitraryFloat
		want  float64
	}{{&coefficients[0], 1}, {&coefficients[1], math.Exp2(53)}, {constant, 3}} {
		got, err := core.ToDouble(tc.value)
		if err != nil || got != tc.want {
			t.Fatalf("coefficient %d: %v, %v; want %v", i, got, err, tc.want)
		}
	}
	_, coefficients, err = w.PrepareSum(&three, []*numeral.Int{&one})
	if err != nil || len(coefficients) != 1 || len(w.coefficients) != 2 {
		t.Fatalf("source reserve keeps older slot: %d, %d, %v", len(coefficients), len(w.coefficients), err)
	}
	if err := qm.SetFractionSigned(&q, 1, 3); err != nil {
		t.Fatal(err)
	}
	x, err := w.MkVar(false)
	if err != nil || x != 0 {
		t.Fatalf("first context variable: %d, %v", x, err)
	}
	y, err := w.MkVar(true)
	if err != nil || y != 1 {
		t.Fatalf("second context variable: %d, %v", y, err)
	}
	z, err := w.MkSum(&three, []*numeral.Int{&one, &big}, []uint32{y, x})
	if err != nil || z != 2 || context.variables[z] {
		t.Fatalf("sorted MPF sum variable: %d, %v", z, err)
	}
	p := context.polynomials[z]
	if p == nil || len(p.terms) != 2 || p.terms[0].variable != x || p.terms[1].variable != y {
		t.Fatalf("sorted polynomial terms: %+v", p)
	}
	for i, tc := range []struct {
		value *numeral.ArbitraryFloat
		want  float64
	}{{&p.terms[0].coefficient, math.Exp2(53)}, {&p.terms[1].coefficient, 1}, {&p.constant, 3}} {
		got, e := core.ToDouble(tc.value)
		if e != nil || got != tc.want {
			t.Fatalf("owned sorted sum %d: %v, %v", i, got, e)
		}
	}
	if len(context.watchers[x]) != 1 || context.watchers[x][0] != z || len(context.watchers[y]) != 1 || context.watchers[y][0] != z {
		t.Fatalf("sum watcher registration: %v, %v", context.watchers[x], context.watchers[y])
	}
	duplicate, err := w.MkSum(&three, []*numeral.Int{&one, &big}, []uint32{x, x})
	if err != nil || duplicate != 3 {
		t.Fatalf("duplicate-variable sum: %d, %v", duplicate, err)
	}
	duplicateTerms := context.polynomials[duplicate].terms
	first, err := core.ToDouble(&duplicateTerms[0].coefficient)
	if err != nil || first != math.Exp2(53) || duplicateTerms[1].coefficient != (numeral.ArbitraryFloat{}) {
		t.Fatalf("source duplicate overwrite/swap: %v, %v, %+v", first, err, duplicateTerms[1].coefficient)
	}
	if len(context.watchers[x]) != 3 || context.watchers[x][1] != duplicate || context.watchers[x][2] != duplicate {
		t.Fatalf("duplicate watcher registrations: %v", context.watchers[x])
	}
	if _, err := w.MkSum(&three, []*numeral.Int{&one}, []uint32{x, y}); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("mismatched sum inputs: %v", err)
	}
	preparedConstant, preparedCoefficients, err := w.PrepareSum(&three, []*numeral.Int{&big})
	if err != nil {
		t.Fatal(err)
	}
	beforeStoppedSum := len(context.variables)
	b.FailAfter(0)
	_, stoppedSum := context.DefinePolynomial(preparedConstant, preparedCoefficients, []uint32{x})
	b.FailAfter(-1)
	if stoppedSum == nil || len(context.variables) != beforeStoppedSum {
		t.Fatalf("stopped MPF sum: %v, %d", stoppedSum, len(context.variables))
	}
	inputPowers := []ArbitraryPower{{Variable: y, Degree: 2}, {Variable: x, Degree: 1}, {Variable: y, Degree: 3}}
	monomial, err := w.MkMonomial(inputPowers)
	if err != nil || monomial != 4 || !context.variables[monomial] {
		t.Fatalf("merged MPF monomial: %d, %v", monomial, err)
	}
	merged := context.monomials[monomial].powers
	if len(merged) != 2 || merged[0] != (ArbitraryPower{x, 1}) || merged[1] != (ArbitraryPower{y, 5}) || inputPowers[0].Variable != y {
		t.Fatalf("sorted/merged monomial and borrowed input: %v, %v", merged, inputPowers)
	}
	if len(context.watchers[x]) != 4 || context.watchers[x][3] != monomial || len(context.watchers[y]) != 2 || context.watchers[y][1] != monomial {
		t.Fatalf("merged monomial watchers: %v, %v", context.watchers[x], context.watchers[y])
	}
	secondMonomial, err := w.MkMonomial([]ArbitraryPower{{Variable: x, Degree: 2}})
	if err != nil || secondMonomial != 5 || context.variables[secondMonomial] || len(context.monomials[monomial].powers) != 2 {
		t.Fatalf("scratch reuse and noninteger monomial: %d, %v", secondMonomial, err)
	}
	if _, err := w.MkMonomial([]ArbitraryPower{{Variable: x, Degree: math.MaxUint32}, {Variable: x, Degree: 1}}); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("degree overflow: %v", err)
	}
	if _, err := w.MkMonomial([]ArbitraryPower{{Variable: uint32(len(context.variables)), Degree: 1}}); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("unknown monomial factor: %v", err)
	}
	lower, err := w.PrepareInequality(&q, true)
	if err != nil {
		t.Fatal(err)
	}
	down, err := core.ToDouble(lower)
	if err != nil || down != 1.0/3.0 {
		t.Fatalf("lower direction: %v, %v", down, err)
	}
	upper, err := w.PrepareInequality(&q, false)
	if err != nil {
		t.Fatal(err)
	}
	up, err := core.ToDouble(upper)
	if err != nil || up != math.Nextafter(1.0/3.0, math.Inf(1)) {
		t.Fatalf("upper direction: %v, %v", up, err)
	}
	a, err := w.MkInequality(x, &q, true, false)
	if err != nil || a == nil || a.variable != x || !a.lower || a.open {
		t.Fatalf("context inequality registration: %v, %v", a, err)
	}
	stored, err := core.ToDouble(&a.value)
	if err != nil || stored != down {
		t.Fatalf("context owns directed scalar: %v, %v", stored, err)
	}
	if _, err := w.MkInequality(uint32(len(context.variables)), &q, false, true); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("unknown context variable: %v", err)
	}
	bAtom, err := w.MkInequality(y, &q, false, true)
	if err != nil {
		t.Fatal(err)
	}
	cAtom, err := w.MkInequality(x, &q, false, false)
	if err != nil {
		t.Fatal(err)
	}
	clause, err := w.AddClause([]*ArbitraryInequality{bAtom, a, cAtom})
	if err != nil || clause == nil || len(clause.atoms) != 3 || clause.atoms[0] != a || clause.atoms[1] != cAtom || clause.atoms[2] != bAtom {
		t.Fatalf("stable MPF clause sort: %+v, %v", clause, err)
	}
	if a.refs != 1 || bAtom.refs != 1 || cAtom.refs != 1 || len(context.clauseWatchers[x]) != 1 || context.clauseWatchers[x][0] != clause || len(context.clauseWatchers[y]) != 1 || context.clauseWatchers[y][0] != clause {
		t.Fatalf("MPF clause references and unique watches: %d, %d, %d, %v, %v", a.refs, bAtom.refs, cAtom.refs, context.clauseWatchers[x], context.clauseWatchers[y])
	}
	if singleton, err := w.AddClause([]*ArbitraryInequality{a}); err != nil || singleton != nil || len(context.unitClauses) != 1 || !context.unitClauses[0].axiom || a.refs != 2 {
		t.Fatalf("singleton MPF axiom unit: %v, %v", singleton, err)
	}
	if _, err := w.AddClause(nil); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("empty MPF clause: %v", err)
	}
	if _, err := w.AddClause([]*ArbitraryInequality{a, nil}); !errors.Is(err, arena.ErrInvalidSlot) || a.refs != 2 {
		t.Fatalf("invalid MPF atom before any ref publication: %v, %d", err, a.refs)
	}
	beforeStopped := len(context.inequalities)
	b.FailAfter(0)
	_, stopped := context.NewInequality(x, &a.value, true, false)
	b.FailAfter(-1)
	if stopped == nil || len(context.inequalities) != beforeStopped+1 {
		t.Fatalf("stopped copied numeral ownership: %v, %d", stopped, len(context.inequalities))
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := w.PrepareSum(&three, nil); !errors.Is(err, arena.ErrClosed) {
		t.Fatalf("closed coefficient owner: %v", err)
	}
	if err := context.Close(); err != nil {
		t.Fatal(err)
	}
	if a.refs != 0 || bAtom.refs != 0 || cAtom.refs != 0 {
		t.Fatalf("MPF clause/unit references after context cleanup: %d, %d, %d", a.refs, bAtom.refs, cAtom.refs)
	}
	if _, err := context.NewVariable(false); !errors.Is(err, arena.ErrClosed) {
		t.Fatalf("closed MPF context: %v", err)
	}
}
