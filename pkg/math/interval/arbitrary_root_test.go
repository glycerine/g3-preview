package interval

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestArbitraryRootRetainedNewtonCertifiedBoundsAndCancellation(t *testing.T) {
	budget := arena.NewBudget(1 << 27)
	runtime, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	core, err := numeral.NewArbitraryFloatManager(numeral.IntegerConfig{
		Memory: runtime, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true},
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := numeral.NewArbitraryFloatNumeralManager(core, 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	var owners [4]*numeral.ScopedArbitraryFloatNumeral
	for i := range owners {
		owners[i], err = numeral.NewScopedArbitraryFloatNumeral(m)
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for i := len(owners) - 1; i >= 0; i-- {
			if err := owners[i].Close(); err != nil {
				t.Error(err)
			}
		}
		if err := m.Close(); err != nil {
			t.Error(err)
		}
		if err := core.Close(); err != nil {
			t.Error(err)
		}
		if err := runtime.Close(); err != nil {
			t.Error(err)
		}
		if budget.Used() != 0 {
			t.Error("MPF root owners", budget.Used())
		}
	})
	var values [4]*numeral.ArbitraryFloat
	for i, owner := range owners {
		values[i], err = owner.Get()
		if err != nil {
			t.Fatal(err)
		}
	}
	a, p, lo, hi := values[0], values[1], values[2], values[3]
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	toRat := func(x *numeral.ArbitraryFloat) *big.Rat {
		t.Helper()
		s, err := m.ToRationalString(x)
		must(err)
		r, ok := new(big.Rat).SetString(s)
		if !ok {
			t.Fatalf("invalid rational %q", s)
		}
		return r
	}
	must(m.SetFractionSigned(p, 1, 8192))
	limit := &control.Reslimit{Limit: 1000}
	root := ArbitraryRoot{Numbers: m, Limit: limit}
	for _, tc := range []struct {
		num, den int32
		degree   uint32
	}{
		{1, 1, 2}, {-1, 1, 3}, {4, 1, 2}, {8, 1, 3}, {32, 1, 5},
		{2, 1, 2}, {2, 1, 3}, {-2, 1, 3}, {1, 2, 2}, {1, 2, 3}, {3, 2, 3}, {-1, 2, 3},
	} {
		limit.Steps = 0
		must(m.SetFractionSigned(a, tc.num, tc.den))
		if err := root.Bounds(a, tc.degree, p, lo, hi); err != nil {
			t.Fatalf("root %d/%d degree %d: %v (budget %d)", tc.num, tc.den, tc.degree, err, budget.Used())
		}
		lower, upper := toRat(lo), toRat(hi)
		if lower.Cmp(upper) > 0 {
			t.Fatalf("reversed %d/%d^(1/%d): [%s,%s]", tc.num, tc.den, tc.degree, lower, upper)
		}
		lowerPower, upperPower := big.NewRat(1, 1), big.NewRat(1, 1)
		for i := uint32(0); i < tc.degree; i++ {
			lowerPower.Mul(lowerPower, lower)
			upperPower.Mul(upperPower, upper)
		}
		target := new(big.Rat).SetFrac64(int64(tc.num), int64(tc.den))
		if lowerPower.Cmp(target) > 0 || upperPower.Cmp(target) < 0 {
			t.Fatalf("uncertified %s^(1/%d): [%s,%s]", target, tc.degree, lower, upper)
		}
	}
	must(m.SetInt32(a, -2))
	if err := root.Bounds(a, 2, p, lo, hi); !errors.Is(err, ErrExtendedDomain) {
		t.Fatal("even negative root", err)
	}
	must(m.SetInt32(a, 2))
	must(m.Reset(p))
	if err := root.Bounds(a, 2, p, lo, hi); !errors.Is(err, ErrExtendedDomain) {
		t.Fatal("zero precision", err)
	}
	must(m.SetFractionSigned(p, 1, 8192))
	limit.Limit, limit.Steps = 1, 0
	if err := root.Bounds(a, 2, p, lo, hi); !errors.Is(err, ErrArbitraryRootCanceled) {
		t.Fatal("root checkpoint", err)
	}
	must(m.Reset(a))
	if err := root.Bounds(a, 3, p, lo, hi); !errors.Is(err, numeral.ErrArbitraryFloatNonRegular) {
		t.Fatal("source checked zero copy", err)
	}
}
