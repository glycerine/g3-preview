// Source: src/math/realclosure/realclosure.cpp:2452-2462, 3881-3920.
package realclosure

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRealClosureAlgebraicConstantAndZeroBoundSigns(t *testing.T) {
	budget := arena.NewBudget(1 << 25)
	service, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	bounds, err := numeral.NewBinaryRationalManager(service.Manager.Z, numeral.BinaryRationalConfig{Diagnostics: service})
	if err != nil {
		t.Fatal(err)
	}
	store := &realValueStore{numbers: service.Manager}
	one, err := store.newRational(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	minusOne, err := store.newRational(-1, 1)
	if err != nil {
		t.Fatal(err)
	}
	owner := &algebraicRootOwner{values: store, intervals: bounds}
	makeValue := func(coefficients []*realValue, lo, hi int32) *realValue {
		t.Helper()
		lower, upper := numeral.NewBinaryInt(lo), numeral.NewBinaryInt(hi)
		enclosure, err := NewBinaryInterval(bounds)
		if err != nil {
			t.Fatal(err)
		}
		isolating, err := NewBinaryInterval(bounds)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range []*BinaryInterval{enclosure, isolating} {
			if err := v.SetFinite(&lower, &upper, false, false); err != nil {
				t.Fatal(err)
			}
		}
		root, err := owner.mkAlgebraic(coefficients, enclosure, isolating, nil, -1)
		if err != nil {
			t.Fatal(err)
		}
		value, err := store.newAlgebraicValue(root, one)
		if err != nil {
			t.Fatal(err)
		}
		if err := enclosure.Close(); err != nil {
			t.Fatal(err)
		}
		if err := isolating.Close(); err != nil {
			t.Fatal(err)
		}
		return value
	}
	positive := makeValue([]*realValue{minusOne, one}, 1, 2)
	negative := makeValue([]*realValue{one, one}, -2, -1)
	zero := numeral.NewBinaryInt(0)
	nonzero := numeral.NewBinaryInt(2)
	for _, tc := range []struct {
		name         string
		coefficients []*realValue
		bound        *numeral.BinaryRational
		want         int32
	}{
		{"positive constant", []*realValue{positive}, &nonzero, 1},
		{"negative constant", []*realValue{negative}, &nonzero, -1},
		{"positive zero bound", []*realValue{positive, negative}, &zero, 1},
		{"negative zero bound", []*realValue{negative, positive}, &zero, -1},
	} {
		got, err := store.signAtBinaryBound(tc.coefficients, bounds, tc.bound)
		if err != nil || got != tc.want {
			t.Fatalf("%s: sign %d, error %v", tc.name, got, err)
		}
	}
	if _, err := store.signAtBinaryBound([]*realValue{positive, negative}, bounds, &nonzero); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("unported algebraic Horner branch", err)
	}
	rows := []realPolynomial{
		{coefficients: []*realValue{positive}},
		{coefficients: []*realValue{negative, positive}},
	}
	if got, err := store.signVariationsAtInfinity(rows, true); err != nil || got != 0 {
		t.Fatal("positive infinity signs", got, err)
	}
	if got, err := store.signVariationsAtInfinity(rows, false); err != nil || got != 1 {
		t.Fatal("negative infinity signs", got, err)
	}
	crossLower, crossUpper := numeral.NewBinaryInt(-1), numeral.NewBinaryInt(1)
	if err := positive.functionInterval.SetFinite(&crossLower, &crossUpper, false, false); err != nil {
		t.Fatal(err)
	}
	if _, err := store.signValue(positive); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("ambiguous algebraic interval sign", err)
	}
	for _, value := range []*realValue{positive, negative, one, minusOne} {
		if err := store.release(value); err != nil {
			t.Fatal(err)
		}
	}
	if err := bounds.CloseChecked(); err != nil {
		t.Fatal(err)
	}
	if err := service.Close(); err != nil {
		t.Fatal(err)
	}
	if used := budget.Used(); used != 0 {
		t.Fatal("algebraic sign fixture retained bytes", used)
	}
}
