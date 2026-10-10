package lp

import (
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

type lpPairHarness struct {
	t      *testing.T
	m      *numeral.RationalManager
	budget *arena.Budget
	pairs  []*RationalPair
}

func newLPPairHarness(t *testing.T) *lpPairHarness {
	t.Helper()
	budget := arena.NewBudget(8 << 20)
	m, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{PageSize: 4, Budget: budget},
		Words: arena.VectorConfig{PageSize: 32, Descriptors: arena.Config{PageSize: 4, Budget: budget}},
	}, Synchronized: true})
	if err != nil {
		t.Fatal(err)
	}
	h := &lpPairHarness{t: t, m: m, budget: budget}
	t.Cleanup(func() {
		budget.FailAfter(-1)
		for i := len(h.pairs) - 1; i >= 0; i-- {
			if err := h.pairs[i].Del(m); err != nil {
				t.Error(err)
			}
		}
		if err := m.CloseChecked(); err != nil {
			t.Error(err)
		}
		if budget.Used() != 0 {
			t.Errorf("retained %d bytes", budget.Used())
		}
	})
	return h
}
func (h *lpPairHarness) must(err error) {
	h.t.Helper()
	if err != nil {
		h.t.Fatal(err)
	}
}
func (h *lpPairHarness) pair(x, y string) *RationalPair {
	h.t.Helper()
	p := NewRationalPair()
	h.pairs = append(h.pairs, &p)
	h.must(h.m.SetString(&p.X, x))
	h.must(h.m.SetString(&p.Y, y))
	return &p
}
func (h *lpPairHarness) check(p *RationalPair, x, y *big.Rat) {
	h.t.Helper()
	for i, coordinate := range []*numeral.Rational{&p.X, &p.Y} {
		got, err := h.m.String(coordinate)
		h.must(err)
		parsed, ok := new(big.Rat).SetString(got)
		want := x
		if i == 1 {
			want = y
		}
		if !ok || parsed.Cmp(want) != 0 {
			h.t.Fatalf("coordinate %d: got %s want %s", i, got, want)
		}
	}
}

func TestLPRationalPairArithmeticOracle(t *testing.T) {
	h := newLPPairHarness(t)
	a, b, out := h.pair("0", "0"), h.pair("0", "0"), h.pair("0", "0")
	r := rand.New(rand.NewPCG(112, 1987))
	for step := 0; step < 400; step++ {
		// Above int32/uint64 storage ranges; arithmetic must use real numerals.
		oracle := make([]*big.Rat, 4)
		for i, c := range []*numeral.Rational{&a.X, &a.Y, &b.X, &b.Y} {
			n := new(big.Int).Lsh(big.NewInt(int64(r.IntN(200)-100)), uint(r.IntN(90)))
			d := big.NewInt(int64(r.IntN(99) + 1))
			oracle[i] = new(big.Rat).SetFrac(n, d)
			h.must(h.m.SetString(c, oracle[i].RatString()))
		}
		ax, ay, bx, by := oracle[0], oracle[1], oracle[2], oracle[3]
		h.must(out.Add(h.m, a, b))
		h.check(out, new(big.Rat).Add(ax, bx), new(big.Rat).Add(ay, by))
		h.must(out.Sub(h.m, a, b))
		h.check(out, new(big.Rat).Sub(ax, bx), new(big.Rat).Sub(ay, by))
		h.must(out.Scale(h.m, a, &b.X))
		h.check(out, new(big.Rat).Mul(ax, bx), new(big.Rat).Mul(ay, bx))
		if bx.Sign() != 0 {
			h.must(out.Divide(h.m, a, &b.X))
			h.check(out, new(big.Rat).Quo(ax, bx), new(big.Rat).Quo(ay, bx))
		}
		h.must(out.Copy(h.m, a))
		h.must(out.Negate(h.m))
		h.check(out, new(big.Rat).Neg(ax), new(big.Rat).Neg(ay))
		// Both destination alias choices, including self copy and self subtraction.
		h.must(out.Copy(h.m, a))
		h.must(out.Add(h.m, out, b))
		h.check(out, new(big.Rat).Add(ax, bx), new(big.Rat).Add(ay, by))
		h.must(out.Copy(h.m, b))
		h.must(out.Sub(h.m, a, out))
		h.check(out, new(big.Rat).Sub(ax, bx), new(big.Rat).Sub(ay, by))
		h.must(out.Copy(h.m, out))
		h.must(out.Sub(h.m, out, out))
		h.check(out, new(big.Rat), new(big.Rat))
	}
}

func TestLPRationalPairOrderSignsAndFormatting(t *testing.T) {
	h := newLPPairHarness(t)
	var pairs []*RationalPair
	var coords [][2]int64
	for x := int64(-2); x <= 2; x++ {
		for y := int64(-2); y <= 2; y++ {
			pairs = append(pairs, h.pair(big.NewInt(x).String(), big.NewInt(y).String()))
			coords = append(coords, [2]int64{x, y})
		}
	}
	for i, a := range pairs {
		sign, err := a.Sign(h.m)
		h.must(err)
		want := coords[i][0]
		if want == 0 {
			want = coords[i][1]
		}
		if want < 0 {
			want = -1
		}
		if want > 0 {
			want = 1
		}
		if (sign < 0) != (want < 0) || (sign > 0) != (want > 0) {
			t.Fatal(i, sign, want)
		}
		zero, err := a.IsZero(h.m)
		h.must(err)
		if zero != (coords[i] == [2]int64{}) {
			t.Fatal(i, zero)
		}
		f, err := a.Float64(h.m)
		h.must(err)
		if f != float64(coords[i][0]) {
			t.Fatal(i, f)
		}
		text, err := a.String(h.m)
		h.must(err)
		if text != "("+big.NewInt(coords[i][0]).String()+", "+big.NewInt(coords[i][1]).String()+")" {
			t.Fatal(text)
		}
		for j, b := range pairs {
			got, err := a.Compare(h.m, b)
			h.must(err)
			want := 0
			if i < j {
				want = -1
			}
			if i > j {
				want = 1
			}
			if got != want {
				t.Fatal(i, j, got, want)
			}
			eq, err := a.Equal(h.m, b)
			h.must(err)
			if eq != (i == j) {
				t.Fatal(i, j, eq)
			}
		}
	}
	p := h.pair("1/3", "-5/7")
	text, err := p.String(h.m)
	h.must(err)
	if text != "(1/3, -5/7)" {
		t.Fatal(text)
	}
	// Compound scalar references are reread after the first coordinate action.
	p = h.pair("2", "3")
	h.must(p.Scale(h.m, p, &p.X))
	h.check(p, big.NewRat(4, 1), big.NewRat(12, 1))
	h.must(h.m.SetInt32(&p.X, 2))
	h.must(h.m.SetInt32(&p.Y, 3))
	h.must(p.Divide(h.m, p, &p.X))
	h.check(p, big.NewRat(1, 1), big.NewRat(3, 1))
	h.must(p.Del(h.m))
	h.check(p, new(big.Rat), new(big.Rat))
}

func TestLPRationalPairCoordinateErrorPrefixes(t *testing.T) {
	h := newLPPairHarness(t)
	foreign := newLPPairHarness(t)
	a, b, out := h.pair("2", "3"), h.pair("1", "1"), h.pair("9", "9")
	// These big references have no live cell in h.m; operations that dereference
	// them must fail before that coordinate is consumed. Raw numeral ownership
	// is a caller contract: this fixture does not claim general provenance checks
	// when unrelated managers happen to contain matching reference IDs.
	for _, coordinate := range []int{0, 1} {
		var bad *RationalPair
		if coordinate == 0 {
			bad = foreign.pair("123456789012345678901234567890", "0")
		} else {
			bad = foreign.pair("0", "123456789012345678901234567890")
		}
		for _, op := range []struct {
			name  string
			run   func() error
			wantX string
		}{
			{"copy", func() error { return out.Copy(h.m, bad) }, "0"},
			{"add", func() error { return out.Add(h.m, bad, b) }, "1"},
			{"sub", func() error { return out.Sub(h.m, bad, b) }, "-1"},
			{"scale", func() error { return out.Scale(h.m, bad, &b.X) }, "0"},
			{"divide", func() error { return out.Divide(h.m, bad, &b.X) }, "0"},
		} {
			h.must(h.m.SetInt32(&out.X, 9))
			h.must(h.m.SetInt32(&out.Y, 9))
			if err := op.run(); err == nil {
				t.Fatal(op.name, coordinate, err)
			}
			wantX := "9"
			if coordinate == 1 {
				wantX = op.wantX
			}
			h.check(out, mustLPRat(t, wantX), big.NewRat(9, 1))
		}
		if _, err := bad.Compare(h.m, bad); err == nil {
			t.Fatal(coordinate, err)
		}
		if _, err := bad.Equal(h.m, bad); err == nil {
			t.Fatal(coordinate, err)
		}
		if _, err := bad.String(h.m); err == nil {
			t.Fatal(coordinate, err)
		}
		if err := bad.Del(h.m); err == nil {
			t.Fatal(coordinate, err)
		}

	}
	closed := newLPPairHarness(t)
	closed.must(closed.m.CloseChecked())
	if _, err := a.Sign(closed.m); err != arena.ErrClosed {
		t.Fatal(err)
	}
	if _, err := a.IsZero(closed.m); err != arena.ErrClosed {
		t.Fatal(err)
	}
	if err := a.Negate(closed.m); err != arena.ErrClosed {
		t.Fatal(err)
	}

	zero := h.pair("0", "0")
	h.must(out.Divide(h.m, a, &zero.X))
	// The existing source rational profile retains a zero denominator here;
	// numeric_pair adds no division guard or invented exception.
	if out.X.Den.Val != 0 || out.Y.Den.Val != 0 {
		t.Fatal(out.X, out.Y)
	}
}
func mustLPRat(t *testing.T, text string) *big.Rat {
	t.Helper()
	x, ok := new(big.Rat).SetString(text)
	if !ok {
		t.Fatal(text)
	}
	return x
}

// Source: test_rationals (--test_mpq). Keep its ten-term accumulation, with
// deterministic inputs and an exact expected result in place of timing/output.
func TestLPRationalPairSourceAccumulation(t *testing.T) {
	h := newLPPairHarness(t)
	var coefficients, values []*RationalPair
	expected := new(big.Rat)
	for i := int64(1); i <= 10; i++ {
		c := big.NewRat(i, i+1)
		x := big.NewRat(11-i, i+2)
		coefficients = append(coefficients, h.pair(c.RatString(), "0"))
		values = append(values, h.pair(x.RatString(), "0"))
		expected.Add(expected, new(big.Rat).Mul(c, x))
	}
	sum, product := h.pair("0", "0"), h.pair("0", "0")
	for k := 0; k < 40; k++ {
		for i, c := range coefficients {
			h.must(product.Scale(h.m, values[i], &c.X))
			h.must(sum.Add(h.m, sum, product))
		}
	}
	expected.Mul(expected, big.NewRat(40, 1))
	h.check(sum, expected, new(big.Rat))
}
