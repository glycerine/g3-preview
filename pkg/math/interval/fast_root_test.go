package interval

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestFastRootRetainedNewtonCertifiedBoundsAndCancellation(t *testing.T) {
	budget := arena.NewBudget(1 << 28)
	cfg := numeral.DefaultFastFloatConfig(arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: budget}})
	cfg.InitialCapacity = 4
	m, err := numeral.NewFastFloatManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	q, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{PageSize: 2, Budget: budget},
		Words: arena.VectorConfig{PageSize: 32, Descriptors: arena.Config{PageSize: 2, Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	owners := make([]*numeral.ScopedFastFloat, 4)
	for i := range owners {
		owners[i], err = numeral.NewScopedFastFloat(m)
		if err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		for i := len(owners) - 1; i >= 0; i-- {
			if e := owners[i].Close(); e != nil {
				t.Error(e)
			}
		}
		q.Close()
		if e := m.CloseChecked(); e != nil {
			t.Error(e)
		}
		if budget.Used() != 0 {
			t.Error("retained fast root owners", budget.Used())
		}
	}()
	values := make([]*numeral.FastFloat, len(owners))
	for i, owner := range owners {
		values[i], err = owner.Get()
		if err != nil {
			t.Fatal(err)
		}
	}
	a, p, lo, hi := values[0], values[1], values[2], values[3]
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	toRat := func(v *numeral.FastFloat) *big.Rat {
		t.Helper()
		raw := numeral.NewRational()
		defer q.Del(&raw)
		must(m.ToRational(v, q, &raw))
		text, e := q.String(&raw)
		must(e)
		r, ok := new(big.Rat).SetString(text)
		if !ok {
			t.Fatalf("invalid rational %q", text)
		}
		return r
	}
	must(m.SetFraction32(p, 1, 8192))
	limit := &control.Reslimit{Limit: 1000}
	root := FastRoot{Numbers: m, Limit: limit}
	for _, tc := range []struct {
		num    int32
		den    uint32
		degree uint32
	}{
		{0, 1, 3}, {1, 1, 2}, {-1, 1, 3}, {4, 1, 2}, {8, 1, 3},
		{32, 1, 5}, {2, 1, 2}, {2, 1, 3}, {-2, 1, 3},
		{1, 2, 2}, {1, 2, 3}, {3, 2, 3}, {-1, 2, 3},
	} {
		limit.Steps = 0
		must(m.SetFraction32(a, tc.num, tc.den))
		must(root.Bounds(a, tc.degree, p, lo, hi))
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
	if e := root.Bounds(a, 2, p, lo, hi); !errors.Is(e, ErrExtendedDomain) {
		t.Fatal("even negative root", e)
	}
	must(m.SetInt32(a, 2))
	must(m.Reset(p))
	if e := root.Bounds(a, 2, p, lo, hi); !errors.Is(e, ErrExtendedDomain) {
		t.Fatal("zero precision", e)
	}
	must(m.SetFraction32(p, 1, 8192))
	limit.Limit, limit.Steps = 1, 0
	if e := root.Bounds(a, 2, p, lo, hi); !errors.Is(e, ErrFastRootCanceled) {
		t.Fatal("root checkpoint", e)
	}
}
