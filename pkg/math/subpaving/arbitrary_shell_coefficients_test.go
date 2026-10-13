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
	if _, err := w.MkInequality(x+1, &q, false, true); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("unknown context variable: %v", err)
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
	if _, err := context.NewVariable(false); !errors.Is(err, arena.ErrClosed) {
		t.Fatalf("closed MPF context: %v", err)
	}
}
