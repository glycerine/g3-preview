// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Source: src/test/mpz.cpp::bug4 and src/util/mpz.cpp bitwise preconditions.
package mpnum

import (
	"errors"
	"math/big"
	"math/rand"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func bitwiseMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func TestIntegerSourceBitwiseBug4(t *testing.T) {
	m, _ := integerManager(t)
	var x, y, result1, result2 Int
	bitwiseMust(t, m.SetUint64(&y, 4294967295))
	bitwiseMust(t, m.SetUint64(&x, 4026531839))
	bitwiseMust(t, m.BitwiseOr(&x, &y, &result1))
	bitwiseMust(t, m.Copy(&result2, &x))
	bitwiseMust(t, m.BitwiseOr(&result2, &y, &result2))
	cmp, e := m.Compare(&result1, &result2)
	bitwiseMust(t, e)
	if cmp != 0 {
		t.Fatal("source bug4 assertion")
	}
	text := integerString(t, m, &result1) + " " + integerString(t, m, &result2) + "\n"
	if text != "4294967295 4294967295\n" {
		t.Fatal("source diagnostic with independent expected result", text)
	}
	for _, v := range []*Int{&x, &y, &result1, &result2} {
		bitwiseMust(t, m.Del(v))
	}
}

func TestIntegerBitwiseIndependentOracleAndAliases(t *testing.T) {
	m, _ := integerManager(t)
	rng := rand.New(rand.NewSource(61019))
	values := []string{"0", "1", "2147483647", "2147483648", "4294967295", "18446744073709551615", "18446744073709551616", "340282366920938463463374607431768211455", "6277101735386680763835789423207666416102355444464034512896"}
	for i := 0; i < 30; i++ {
		raw := make([]byte, 24)
		rng.Read(raw)
		values = append(values, new(big.Int).SetBytes(raw).String())
	}
	for i, s := range values {
		bs, _ := new(big.Int).SetString(s, 10)
		for j := 0; j < 4; j++ {
			other := values[(i*7+j*11)%len(values)]
			bt, _ := new(big.Int).SetString(other, 10)
			for op := 0; op < 3; op++ {
				for alias := 0; alias < 4; alias++ {
					var a, b, c Int
					bitwiseMust(t, m.SetString(&a, s))
					bitwiseMust(t, m.SetString(&b, other))
					bitwiseMust(t, m.SetInt64(&c, -93))
					left, right, out := &a, &b, &c
					if alias == 1 {
						out = left
					}
					if alias == 2 {
						out = right
					}
					if alias == 3 {
						right = left
						out = left
					}
					want := new(big.Int)
					rhs := bt
					if alias == 3 {
						rhs = bs
					}
					switch op {
					case 0:
						want.Or(bs, rhs)
						bitwiseMust(t, m.BitwiseOr(left, right, out))
					case 1:
						want.And(bs, rhs)
						bitwiseMust(t, m.BitwiseAnd(left, right, out))
					case 2:
						want.Xor(bs, rhs)
						bitwiseMust(t, m.BitwiseXor(left, right, out))
					}
					assertIntegerPowerOracle(t, m, out, want)
					for _, v := range []*Int{&a, &b, &c} {
						bitwiseMust(t, m.Del(v))
					}
				}
			}
		}
		for _, width := range []uint32{0, 1, 7, 30, 31, 32, 33, 62, 63, 64, 65, 127, 128, 129, 192, 257} {
			for _, alias := range []bool{false, true} {
				var a, out Int
				bitwiseMust(t, m.SetString(&a, s))
				bitwiseMust(t, m.SetInt64(&out, -93))
				dst := &out
				if alias {
					dst = &a
				}
				mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(width)), big.NewInt(1))
				want := new(big.Int).And(new(big.Int).Not(bs), mask)
				bitwiseMust(t, m.BitwiseNot(width, &a, dst))
				assertIntegerPowerOracle(t, m, dst, want)
				bitwiseMust(t, m.Del(&a))
				bitwiseMust(t, m.Del(&out))
			}
		}
	}
}

func TestIntegerBitwiseFailurePublicationAndWarm(t *testing.T) {
	for op := 0; op < 4; op++ {
		failed, success := false, false
		for prefix := int64(0); prefix < 80; prefix++ {
			m, budget := integerManager(t)
			var a, b, out Int
			bitwiseMust(t, m.SetString(&a, "1461501637330902918203684832716283019655932542981"))
			bitwiseMust(t, m.SetString(&b, "18446744073709551617"))
			bitwiseMust(t, m.SetInt64(&out, -19))
			live := m.cells.Stats().Live
			budget.FailAfter(prefix)
			var e error
			switch op {
			case 0:
				e = m.BitwiseOr(&a, &b, &out)
			case 1:
				e = m.BitwiseAnd(&a, &b, &out)
			case 2:
				e = m.BitwiseXor(&a, &b, &out)
			case 3:
				e = m.BitwiseNot(191, &a, &out)
			}
			budget.FailAfter(-1)
			if e != nil {
				failed = true
				if m.cells.Stats().Live != live {
					t.Fatal("failed operation lost a temporary cell before Close", op, prefix, live, m.cells.Stats().Live)
				}
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal(e)
				}
				if integerString(t, m, &out) != "-19" || integerString(t, m, &a) != "1461501637330902918203684832716283019655932542981" || integerString(t, m, &b) != "18446744073709551617" {
					t.Fatal("failed operation published output")
				}
			} else {
				success = true
			}
			m.Close()
			if budget.Used() != 0 {
				t.Fatal("failed temporary cleanup")
			}
			if success {
				break
			}
		}
		if !failed || !success {
			t.Fatal("fault sweep endpoints", op, failed, success)
		}
	}
	m, budget := integerManager(t)
	var a, b, out Int
	bitwiseMust(t, m.SetString(&a, "340282366920938463463374607431768211455"))
	bitwiseMust(t, m.SetString(&b, "18446744073709551616"))
	for i := 0; i < 5; i++ {
		bitwiseMust(t, m.BitwiseOr(&a, &b, &out))
		bitwiseMust(t, m.BitwiseAnd(&a, &b, &out))
		bitwiseMust(t, m.BitwiseXor(&a, &b, &out))
		bitwiseMust(t, m.BitwiseNot(129, &a, &out))
	}
	budget.FailAfter(0)
	if n := testing.AllocsPerRun(100, func() {
		if e := m.BitwiseOr(&a, &b, &out); e != nil {
			panic(e)
		}
		if e := m.BitwiseAnd(&a, &b, &out); e != nil {
			panic(e)
		}
		if e := m.BitwiseXor(&a, &b, &out); e != nil {
			panic(e)
		}
		if e := m.BitwiseNot(129, &a, &out); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatal("warm owned operations allocate", n)
	}
	budget.FailAfter(-1)
}

func TestIntegerBitwiseSourcePreconditions(t *testing.T) {
	m, _ := integerManager(t)
	a, b, out := NewInt(-1), NewInt(3), NewInt(77)
	for _, f := range []func() error{func() error { return m.BitwiseOr(&a, &b, &out) }, func() error { return m.BitwiseAnd(&b, &a, &out) }, func() error { return m.BitwiseXor(&a, &a, &out) }, func() error { return m.BitwiseNot(32, &a, &out) }} {
		if e := f(); !errors.Is(e, ErrBitwiseDomain) || out != NewInt(77) {
			t.Fatal("negative source precondition", e)
		}
	}
	m.Close()
	if e := m.BitwiseOr(&b, &b, &out); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestIntegerBitwiseAliasedFailurePublication(t *testing.T) {
	for op := 0; op < 4; op++ {
		for alias := 0; alias < 2; alias++ {
			failed, success := false, false
			for prefix := int64(0); prefix < 80; prefix++ {
				m, budget := integerManager(t)
				var a, b Int
				const left = "1461501637330902918203684832716283019655932542981"
				const right = "18446744073709551617"
				bitwiseMust(t, m.SetString(&a, left))
				bitwiseMust(t, m.SetString(&b, right))
				dst := &a
				if alias == 1 {
					dst = &b
				}
				input := &a
				if op == 3 && alias == 1 {
					input = &b
				}
				live := m.cells.Stats().Live
				budget.FailAfter(prefix)
				var e error
				switch op {
				case 0:
					e = m.BitwiseOr(&a, &b, dst)
				case 1:
					e = m.BitwiseAnd(&a, &b, dst)
				case 2:
					e = m.BitwiseXor(&a, &b, dst)
				case 3:
					e = m.BitwiseNot(191, input, dst)
				}
				budget.FailAfter(-1)
				if e != nil {
					failed = true
					if m.cells.Stats().Live != live {
						t.Fatal("failed alias operation lost a temporary cell before Close", op, alias, prefix)
					}
					if !errors.Is(e, arena.ErrInjectedFailure) {
						t.Fatal(e)
					}
					if integerString(t, m, &a) != left || integerString(t, m, &b) != right {
						t.Fatal("failure changed aliased input", op, alias, prefix)
					}
				} else {
					success = true
				}
				m.Close()
				if budget.Used() != 0 {
					t.Fatal("alias failure leaked")
				}
				if success {
					break
				}
			}
			if !failed || !success {
				t.Fatal("alias fault sweep endpoints", op, alias)
			}
		}
	}
}

func TestIntegerBitwiseSmallPathsReleaseBigOutput(t *testing.T) {
	for op := 0; op < 4; op++ {
		m, budget := integerManager(t)
		a, b := NewInt(5), NewInt(6)
		var out Int
		bitwiseMust(t, m.SetUint64(&out, 18446744073709551615))
		live := m.cells.Stats().Live
		budget.FailAfter(0)
		var e error
		var want int32
		switch op {
		case 0:
			e = m.BitwiseOr(&a, &b, &out)
			want = 7
		case 1:
			e = m.BitwiseAnd(&a, &b, &out)
			want = 4
		case 2:
			e = m.BitwiseXor(&a, &b, &out)
			want = 3
		case 3:
			e = m.BitwiseNot(3, &a, &out)
			want = 2
		}
		budget.FailAfter(-1)
		bitwiseMust(t, e)
		if out != NewInt(want) || m.cells.Stats().Live != live-1 || a != NewInt(5) || b != NewInt(6) {
			t.Fatal("small operation retained old big output", op, out, m.cells.Stats().Live, live)
		}
	}
}
