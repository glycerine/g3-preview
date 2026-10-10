// Source arithmetic: src/util/mpn.cpp. Source regression projections:
// src/test/mpz.cpp tst2 and bug1
package numeral

import (
	"errors"
	"math/big"
	"math/rand"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func oracle(a []uint32) *big.Int {
	z := new(big.Int)
	for i := len(a) - 1; i >= 0; i-- {
		z.Lsh(z, 32)
		z.Or(z, new(big.Int).SetUint64(uint64(a[i])))
	}
	return z
}
func wantLimbs(z *big.Int, n int) []uint32 {
	c := make([]uint32, n)
	q := new(big.Int).Set(z)
	for i := range c {
		c[i] = uint32(q.Uint64())
		q.Rsh(q, 32)
	}
	return c
}

func TestSourceMPZMagnitudeProjections(t *testing.T) {
	// tst2: UINT_MAX + 1, followed by squaring, projected onto mpn.
	m := naturalManager(t)
	sum := make([]uint32, 2)
	if n := m.Add([]uint32{0xffffffff}, []uint32{1}, sum); n != 2 || !reflect.DeepEqual(sum, []uint32{0, 1}) {
		t.Fatalf("carry: %v size %d", sum, n)
	}
	square := make([]uint32, 4)
	m.Mul(sum, sum, square)
	if !reflect.DeepEqual(square, []uint32{0, 0, 1, 0}) {
		t.Fatal(square)
	}
	// bug1's exact positive input, then v-v == 0. Decimal parsing here is
	// independent oracle setup; signed source parser coverage lives separately.
	z, _ := new(big.Int).SetString("1002043949858757875676767675747473", 10)
	a := wantLimbs(z, 4)
	c := make([]uint32, 4)
	if borrow := m.Sub(a, a, c); borrow != 0 || oracle(c).Sign() != 0 {
		t.Fatalf("self subtraction %v %d", c, borrow)
	}
}

func TestNaturalArithmeticOracle(t *testing.T) {
	// Independent fixed-seed generator, 1000 finite cases, at most eight limbs.
	r := rand.New(rand.NewSource(0x4d504e))
	m := naturalManager(t)
	for trial := 0; trial < 1000; trial++ {
		a := make([]uint32, 1+r.Intn(8))
		b := make([]uint32, 1+r.Intn(8))
		for i := range a {
			a[i] = r.Uint32()
		}
		for i := range b {
			b[i] = r.Uint32()
		}
		if trial%7 == 0 {
			a[len(a)-1] = 0
		}
		if trial%11 == 0 {
			b[len(b)-1] = 0
		}
		aa, bb := oracle(a), oracle(b)
		ac, bc := append([]uint32(nil), a...), append([]uint32(nil), b...)
		if got := m.Compare(a, b); got != aa.Cmp(bb) {
			t.Fatalf("compare trial %d", trial)
		}
		n := max(len(a), len(b))
		c := make([]uint32, n+1)
		size := m.Add(a, b, c)
		expected := new(big.Int).Add(aa, bb)
		if oracle(c).Cmp(expected) != 0 || size < 1 || size > len(c) || (size > 1 && c[size-1] == 0) {
			t.Fatalf("add trial %d", trial)
		}
		c = make([]uint32, n)
		borrow := m.Sub(a, b, c)
		expected.Sub(aa, bb)
		wantBorrow := uint32(0)
		if expected.Sign() < 0 {
			wantBorrow = 1
			expected.Add(expected, new(big.Int).Lsh(big.NewInt(1), uint(32*n)))
		}
		if borrow != wantBorrow || oracle(c).Cmp(expected) != 0 {
			t.Fatalf("sub trial %d", trial)
		}
		c = make([]uint32, len(a)+len(b))
		for i := range c {
			c[i] = 0xdeadbeef
		}
		m.Mul(a, b, c)
		if oracle(c).Cmp(new(big.Int).Mul(aa, bb)) != 0 {
			t.Fatalf("mul trial %d", trial)
		}
		if !reflect.DeepEqual(a, ac) || !reflect.DeepEqual(b, bc) {
			t.Fatal("input modified")
		}
	}
}

func TestNaturalExactAliases(t *testing.T) {
	m := naturalManager(t)
	if m.Compare(nil, nil) != 0 || m.Compare([]uint32{5}, []uint32{5, 0, 0}) != 0 {
		t.Fatal("equal magnitudes with leading zero limbs")
	}
	for _, second := range []bool{false, true} {
		a, b := []uint32{0xffffffff, 0xffffffff, 0}, []uint32{3, 1, 0}
		dst := a
		if second {
			dst = b
		}
		m.Add(a[:2], b[:2], dst)
		if !reflect.DeepEqual(dst, []uint32{2, 1, 1}) {
			t.Fatal(dst)
		}
		a, b = []uint32{0, 2}, []uint32{1, 1}
		dst = a
		if second {
			dst = b
		}
		if m.Sub(a, b, dst) != 0 || !reflect.DeepEqual(dst, []uint32{0xffffffff, 0}) {
			t.Fatal(dst)
		}
	}
}

func TestNaturalDivisionAndDecimal(t *testing.T) {
	m := naturalManager(t)
	r := rand.New(rand.NewSource(0x444956))
	for trial := 0; trial < 2000; trial++ {
		a := make([]uint32, 1+r.Intn(8))
		b := make([]uint32, 1+r.Intn(len(a)))
		for i := range a {
			a[i] = r.Uint32()
		}
		for i := range b {
			b[i] = r.Uint32()
		}
		b[len(b)-1] |= 1
		if trial%9 == 0 {
			b[len(b)-1] = 1
		}
		if trial%17 == 0 {
			b[len(b)-1] = 0xffffffff
		}
		q, rem := make([]uint32, len(a)-len(b)+1), make([]uint32, len(b))
		aa, bb := oracle(a), oracle(b)
		if _, e := m.Div(a, b, q, rem); e != nil {
			t.Fatal(e)
		}
		wq, wr := new(big.Int), new(big.Int)
		wq.QuoRem(aa, bb, wr)
		if oracle(q).Cmp(wq) != 0 || oracle(rem).Cmp(wr) != 0 {
			t.Fatalf("div trial %d: %x / %x = %x rem %x expected %s %s", trial, a, b, q, rem, wq, wr)
		}
		if got, e := m.String(a); e != nil || got != aa.String() {
			t.Fatalf("decimal trial %d: %s != %s", trial, got, aa)
		}
	}
	if got, e := m.String([]uint32{0}); e != nil || got != "0" {
		t.Fatal(got)
	}
}

func TestSourceDivisionReturnBranches(t *testing.T) {
	m := naturalManager(t)
	q, r := []uint32{99}, []uint32{99}
	if branch, e := m.Div([]uint32{7}, []uint32{3}, q, r); e != nil || branch || q[0] != 2 || r[0] != 1 {
		t.Fatal("source single-digit branch returns false")
	}
	q, r = []uint32{99}, make([]uint32, 2)
	if branch, e := m.Div([]uint32{1, 2}, []uint32{0, 3}, q, r); e != nil || branch || q[0] != 0 || !reflect.DeepEqual(r, []uint32{1, 2}) {
		t.Fatal("source smaller branch")
	}
}

func TestNaturalDivisionCorrectionFixtures(t *testing.T) {
	// Fixed operands exercise Algorithm D paths too rare for random limbs:
	// subtract/add-back with equal top two limbs and a smaller low limb;
	// quotient estimate == base and estimate correction using the second limb.
	cases := []struct{ a, b []uint32 }{
		{[]uint32{1, 0, 0x80000000}, []uint32{2, 0, 0x80000000}},
		{[]uint32{1, 0, 1}, []uint32{2, 0, 1}},
		{[]uint32{0, 0, 0x80000000}, []uint32{0xffffffff, 0x80000000}},
		{[]uint32{0xffffffff, 0xffffffff, 0xffffffff}, []uint32{0xffffffff, 0xffffffff}},
	}
	m := naturalManager(t)
	for _, tc := range cases {
		q, r := make([]uint32, len(tc.a)-len(tc.b)+1), make([]uint32, len(tc.b))
		if _, e := m.Div(tc.a, tc.b, q, r); e != nil {
			t.Fatal(e)
		}
		wq, wr := new(big.Int), new(big.Int)
		wq.QuoRem(oracle(tc.a), oracle(tc.b), wr)
		if oracle(q).Cmp(wq) != 0 || oracle(r).Cmp(wr) != 0 {
			t.Fatalf("%x / %x: %x rem %x", tc.a, tc.b, q, r)
		}
	}
}

func TestNaturalInvalidPreconditions(t *testing.T) {
	m := naturalManager(t)
	cases := []func(){
		func() { m.Add(nil, nil, []uint32{0}) },
		func() { m.Add([]uint32{1}, nil, nil) },
		func() { m.Sub([]uint32{1}, nil, nil) },
		func() { m.Mul([]uint32{1}, []uint32{2}, nil) },
		func() { m.Div([]uint32{1}, []uint32{0}, []uint32{0}, []uint32{0}) },
		func() { m.Div([]uint32{1}, []uint32{1, 1}, nil, make([]uint32, 2)) },
	}
	for i, call := range cases {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("case %d did not reject invalid contract", i)
				}
			}()
			call()
		}()
	}
	if value, e := m.String(nil); !errors.Is(e, arena.ErrBounds) || value != "" || m.Formatting.Active || m.Formatting.PendingField {
		t.Fatal("source empty reversal access then lexical unwind", value, e)
	}
	if value, e := m.String([]uint32{0, 0}); e != nil || value != "0" {
		t.Fatal("source multi-limb leading zero is finite", value, e)
	}
}

func naturalManager(t *testing.T) *NaturalManager {
	t.Helper()
	m, e := NewNaturalManager(arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 8, Budget: arena.NewBudget(1 << 22)}})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(m.Close)
	return m
}
