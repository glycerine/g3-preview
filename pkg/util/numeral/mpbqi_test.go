package numeral

// Source: old_interval.cpp tst2.

import (
	"errors"
	"math/big"
	"testing"
)

func TestBinaryIntervalQueriesAndScalarRoutes(t *testing.T) {
	m := intervalManager(t)
	var a, b BinaryInterval
	defer m.Del(&a)
	defer m.Del(&b)
	for _, tc := range []struct {
		l, u                  int32
		empty, pos, neg, zero bool
	}{{0, 0, true, true, true, false}, {0, 3, false, true, false, false}, {-3, 0, false, false, true, false}, {-3, 3, false, false, false, true}, {1, 1, true, true, false, false}} {
		intervalSet(t, m, &a, tc.l, tc.u)
		for i, f := range []func(*BinaryInterval) (bool, error){m.IsEmpty, m.IsPos, m.IsNeg, m.ContainsZero} {
			ok, e := f(&a)
			serviceMust(t, e)
			want := [4]bool{tc.empty, tc.pos, tc.neg, tc.zero}
			if ok != want[i] {
				t.Fatal(tc, i, ok)
			}
		}
	}
	n := NewInt(17)
	serviceMust(t, m.SetInteger(&a, &n))
	intervalText(t, m, &a, "(17, 17)")
	bound := NewBinaryIntPower(3, 1)
	serviceMust(t, m.SetPoint(&a, &bound))
	intervalText(t, m, &a, "(3/2, 3/2)")
	intervalSet(t, m, &a, 1, 3)
	lo, hi := NewBinaryInt(0), NewBinaryInt(4)
	serviceMust(t, m.SetLower(&a, &lo))
	serviceMust(t, m.SetUpper(&a, &hi))
	serviceMust(t, m.Copy(&b, &a))
	serviceMust(t, m.Copy(&a, &a))
	ok, e := m.IsSubset(&a, &b)
	serviceMust(t, e)
	if !ok {
		t.Fatal("subset")
	}
	var out BinaryRational
	serviceMust(t, m.Lower(&a, &out))
	binaryText(t, m.bounds, &out, "0")
	serviceMust(t, m.Upper(&a, &out))
	binaryText(t, m.bounds, &out, "4")
	serviceMust(t, m.bounds.Del(&out))
	serviceMust(t, m.Neg(&a))
	intervalText(t, m, &a, "(-4, 0)")
	ok, e = m.Precedes(&a, &b)
	serviceMust(t, e)
	if !ok {
		t.Fatal("precedes")
	}
	ok, e = m.PrecedesBound(&a, &lo)
	serviceMust(t, e)
	if !ok {
		t.Fatal("scalar upper")
	}
	ok, e = m.BoundPrecedes(&hi, &b)
	serviceMust(t, e)
	if ok {
		t.Fatal("scalar lower")
	}
	serviceMust(t, m.AddInteger(&a, &n, &a))
	intervalText(t, m, &a, "(13, 17)")
	serviceMust(t, m.Swap(&a, &b))
	intervalText(t, m, &a, "(0, 4)")
	intervalText(t, m, &b, "(13, 17)")
	serviceMust(t, m.Reset(&a))
	intervalText(t, m, &a, "(0, 0)")
	owner, e := m.Bounds()
	serviceMust(t, e)
	if owner != m.bounds {
		t.Fatal("borrowed manager")
	}
	intervalSet(t, m, &a, 1, 3)
	bad := NewBinaryInt(4)
	e = m.SetLower(&a, &bad)
	if SourceExtendedGCDDebug {
		if !errors.Is(e, ErrBinaryInvariant) {
			t.Fatal(e)
		}
		intervalText(t, m, &a, "(1, 3)")
	} else {
		serviceMust(t, e)
		intervalText(t, m, &a, "(4, 3)")
	}
	intervalAudit(t, m, &a, &b)
}

func TestBinaryIntervalArithmeticAliasesAndPowers(t *testing.T) {
	m := intervalManager(t)
	for op := 0; op < 3; op++ {
		for alias := 0; alias < 3; alias++ {
			var a, b, c BinaryInterval
			intervalSet(t, m, &a, -2, 3)
			intervalSet(t, m, &b, -4, 5)
			out := &c
			if alias == 1 {
				out = &a
			}
			if alias == 2 {
				out = &b
			}
			switch op {
			case 0:
				serviceMust(t, m.Add(&a, &b, out))
				intervalText(t, m, out, "(-6, 8)")
			case 1:
				serviceMust(t, m.Sub(&a, &b, out))
				want := "(-7, 7)"
				if alias == 2 {
					want = "(-7, 10)"
				}
				intervalText(t, m, out, want)
			case 2:
				serviceMust(t, m.Mul(&a, &b, out))
				intervalText(t, m, out, "(-12, 15)")
			}
			intervalAudit(t, m, &a, &b, &c)
			serviceMust(t, m.Del(&a))
			serviceMust(t, m.Del(&b))
			serviceMust(t, m.Del(&c))
		}
	}
	for l := int32(-3); l <= 2; l++ {
		for u := l; u <= 3; u++ {
			for p := uint32(1); p <= 5; p++ {
				var a, c BinaryInterval
				intervalSet(t, m, &a, l, u)
				serviceMust(t, m.Power(&a, p, &c))
				lp := new(big.Int).Exp(big.NewInt(int64(l)), big.NewInt(int64(p)), nil)
				up := new(big.Int).Exp(big.NewInt(int64(u)), big.NewInt(int64(p)), nil)
				low, high := new(big.Int).Set(lp), new(big.Int).Set(up)
				if p%2 == 0 {
					if low.Cmp(high) > 0 {
						low, high = high, low
					}
					if l <= 0 && u >= 0 {
						low.SetInt64(0)
					}
				}
				if binaryRat(t, m.bounds, &c.lower).Cmp(new(big.Rat).SetInt(low)) != 0 || binaryRat(t, m.bounds, &c.upper).Cmp(new(big.Rat).SetInt(high)) != 0 {
					t.Fatal(l, u, p)
				}
				serviceMust(t, m.Del(&a))
				serviceMust(t, m.Del(&c))
			}
		}
	}
}

func intervalManager(t *testing.T) *BinaryIntervalManager {
	t.Helper()
	b, _ := binaryManager(t)
	m, e := NewBinaryIntervalManager(b)
	serviceMust(t, e)
	t.Cleanup(m.Close)
	return m
}
func intervalSet(t *testing.T, m *BinaryIntervalManager, a *BinaryInterval, l, u int32) {
	t.Helper()
	lo, hi := NewBinaryInt(l), NewBinaryInt(u)
	serviceMust(t, m.SetBounds(a, &lo, &hi))
}
func intervalText(t *testing.T, m *BinaryIntervalManager, a *BinaryInterval, w string) {
	t.Helper()
	s, e := m.String(a)
	if e != nil || s != w {
		t.Fatalf("got %q %v want %q", s, e, w)
	}
}

// Four actual active assertions adapted from the integer specialization of
// basic_interval<mpz,false>; the original Int/SMT registration stays open.
func TestBinaryIntervalSourceSharedTemplate(t *testing.T) {
	m := intervalManager(t)
	x, e := NewScopedBinaryInterval(m)
	serviceMust(t, e)
	y, e := NewScopedBinaryInterval(m)
	serviceMust(t, e)
	z, e := NewScopedBinaryInterval(m)
	serviceMust(t, e)
	defer x.Close()
	defer y.Close()
	defer z.Close()
	xv, e := x.Get()
	serviceMust(t, e)
	yv, e := y.Get()
	serviceMust(t, e)
	zv, e := z.Get()
	serviceMust(t, e)
	intervalSet(t, m, xv, 1, 2)
	intervalSet(t, m, yv, -2, 3)
	serviceMust(t, m.Add(xv, yv, zv))
	for i, w := range []int32{-1, 5} {
		v := &zv.lower.Num
		if i == 1 {
			v = &zv.upper.Num
		}
		n := NewInt(w)
		ok, e := m.bounds.Z.Equal(v, &n)
		serviceMust(t, e)
		if !ok {
			t.Fatal("source add assertion", i)
		}
	}
	intervalText(t, m, zv, "(-1, 5)")
	serviceMust(t, m.Mul(xv, yv, zv))
	for i, w := range []int32{-4, 6} {
		v := &zv.lower.Num
		if i == 1 {
			v = &zv.upper.Num
		}
		n := NewInt(w)
		ok, e := m.bounds.Z.Equal(v, &n)
		serviceMust(t, e)
		if !ok {
			t.Fatal("source mul assertion", i)
		}
	}
	intervalText(t, m, zv, "(-4, 6)")
}
func TestBinaryIntervalSourceDefects(t *testing.T) {
	m := intervalManager(t)
	var a, b, c BinaryInterval
	defer m.Del(&a)
	defer m.Del(&b)
	defer m.Del(&c)
	intervalSet(t, m, &a, 0, 3)
	intervalSet(t, m, &b, 1, 4)
	ok, e := m.Disjoint(&a, &b)
	serviceMust(t, e)
	if !ok {
		t.Fatal("source upper/upper typo")
	}
	ok, e = m.Disjoint(&b, &a)
	serviceMust(t, e)
	if ok {
		t.Fatal("reverse")
	}
	ok, e = m.Disjoint(&a, &a)
	serviceMust(t, e)
	if !ok {
		t.Fatal("self")
	}
	intervalSet(t, m, &a, 2, 4)
	intervalSet(t, m, &b, 1, 3)
	serviceMust(t, m.Sub(&a, &b, &b))
	intervalText(t, m, &b, "(-1, 5)")
	intervalSet(t, m, &a, 1, 3)
	serviceMust(t, m.Sub(&a, &a, &a))
	intervalText(t, m, &a, "(-2, 5)")
	for _, tc := range []struct {
		l, u            int32
		separate, alias string
	}{{-3, -1, "(1, 9)", "(9, 1)"}, {-2, 3, "(0, 9)", "(4, 9)"}} {
		intervalSet(t, m, &a, tc.l, tc.u)
		serviceMust(t, m.Power(&a, 2, &c))
		intervalText(t, m, &c, tc.separate)
		serviceMust(t, m.Power(&a, 2, &a))
		intervalText(t, m, &a, tc.alias)
	}
	intervalSet(t, m, &a, -2, 3)
	serviceMust(t, m.Power(&a, 0, &c))
	intervalText(t, m, &c, "(0, 1)")
	lo, hi := NewBinaryIntPower(2, 1), NewBinaryInt(1)
	serviceMust(t, m.SetBounds(&a, &lo, &hi))
	ok, e = m.IsEmpty(&a)
	serviceMust(t, e)
	if ok {
		t.Fatal("structural Eq")
	}
}
func TestBinaryIntervalIndependentArithmetic(t *testing.T) {
	m := intervalManager(t)
	var a, b, c BinaryInterval
	defer m.Del(&a)
	defer m.Del(&b)
	defer m.Del(&c)
	// Independent exact dyadic endpoints, all four min/max corners and ties.
	for l := int32(-3); l <= 2; l++ {
		for u := l; u <= 3; u++ {
			for v := int32(-2); v <= 1; v++ {
				for w := v; w <= 2; w++ {
					lo, hi := NewBinaryIntPower(l, 1), NewBinaryIntPower(u, 1)
					serviceMust(t, m.SetBounds(&a, &lo, &hi))
					lo, hi = NewBinaryIntPower(v, 2), NewBinaryIntPower(w, 2)
					serviceMust(t, m.SetBounds(&b, &lo, &hi))
					al, au := big.NewRat(int64(l), 2), big.NewRat(int64(u), 2)
					bl, bu := big.NewRat(int64(v), 4), big.NewRat(int64(w), 4)
					for op := 0; op < 3; op++ {
						var low, high *big.Rat
						switch op {
						case 0:
							serviceMust(t, m.Add(&a, &b, &c))
							low = new(big.Rat).Add(al, bl)
							high = new(big.Rat).Add(au, bu)
						case 1:
							serviceMust(t, m.Sub(&a, &b, &c))
							low = new(big.Rat).Sub(al, bu)
							high = new(big.Rat).Sub(au, bl)
						case 2:
							serviceMust(t, m.Mul(&a, &b, &c))
							products := []*big.Rat{new(big.Rat).Mul(al, bl), new(big.Rat).Mul(al, bu), new(big.Rat).Mul(au, bl), new(big.Rat).Mul(au, bu)}
							low, high = products[0], products[0]
							for _, p := range products {
								if p.Cmp(low) < 0 {
									low = p
								}
								if p.Cmp(high) > 0 {
									high = p
								}
							}
						}
						if binaryRat(t, m.bounds, &c.lower).Cmp(low) != 0 || binaryRat(t, m.bounds, &c.upper).Cmp(high) != 0 {
							t.Fatal(l, u, v, w, op)
						}
					}
				}
			}
		}
	}
}
