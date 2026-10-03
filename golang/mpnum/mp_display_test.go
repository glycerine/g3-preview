// Copyright (c) 2006 Microsoft Corporation
// Active source outputs: src/test/mpq.cpp tst2. Other exact expectations are
// independent finite projections of mpz/mpq printing branches, not C++ runs.
package mpnum

import (
	"bytes"
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
	"math/big"
	"strings"
	"testing"
)

func TestSourceRationalDecimalDisplays(t *testing.T) {
	m, _ := rationalManager(t, false)
	a := NewRational()
	m.SetFraction(&a, 1, 3)
	first, e := m.DecimalString(&a, 10, false)
	if e != nil {
		t.Fatal(e)
	}
	m.SetFraction(&a, 1, 4)
	second, e := m.DecimalString(&a, 10, false)
	if e != nil {
		t.Fatal(e)
	}
	// Source prints both fractions at precision10; exact bytes are independently
	// derived mathematical expectations, not source assertions or observed C++.
	if got := "1/3: " + first + "\n1/4: " + second + "\n"; got != "1/3: 0.3333333333?\n1/4: 0.25\n" {
		t.Fatal(got)
	}
}
func TestIntegerSMT2DisplayForms(t *testing.T) {
	m, _ := integerManager(t)
	for _, f := range []struct{ input, normal, decimal string }{{"0", "0", "0.0"}, {"1", "1", "1.0"}, {"-1", "(- 1)", "(- 1.0)"}, {"-2147483648", "(- 2147483648)", "(- 2147483648.0)"}, {"18446744073709551616", "18446744073709551616", "18446744073709551616.0"}, {"-18446744073709551616", "(- 18446744073709551616)", "(- 18446744073709551616.0)"}} {
		var a Int
		m.SetString(&a, f.input)
		for _, decimal := range []bool{false, true} {
			want := f.normal
			if decimal {
				want = f.decimal
			}
			got, e := m.SMT2String(&a, decimal)
			if e != nil || got != want {
				t.Fatal(f, decimal, got, e)
			}
			dst := bytes.Repeat([]byte{'!'}, len(want)+1)
			n, e := m.FormatSMT2Into(&a, decimal, dst[:len(want)])
			if e != nil || n != len(want) || string(dst[:n]) != want || dst[n] != '!' {
				t.Fatal(f, n, e)
			}
		}
		if integerString(t, m, &a) != f.input {
			t.Fatal("SMT2 mutated numeral")
		}
		m.Del(&a)
	}
}
func TestRationalSMT2AndThinDisplays(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		for _, f := range []struct{ input, normal, decimal, pretty string }{{"0", "0", "0.0", "0"}, {"-2", "(- 2)", "(- 2.0)", "-2"}, {"1/3", "(/ 1 3)", "(/ 1.0 3.0)", "1/3"}, {"-1/3", "(/ (- 1) 3)", "(/ (- 1.0) 3.0)", "-1/3"}, {"4/2", "2", "2.0", "2"}} {
			a := NewRational()
			m.SetString(&a, f.input)
			for _, decimal := range []bool{false, true} {
				want := f.normal
				if decimal {
					want = f.decimal
				}
				text, e := m.SMT2String(&a, decimal)
				if e != nil || text != want {
					t.Fatal(f, decimal, text, e)
				}
			}
			s, e := m.ToRationalString(&a)
			if e != nil || s != f.pretty {
				t.Fatal(f, s, e)
			}
			dst := make([]byte, len(f.pretty))
			n, e := m.FormatPrettyInto(&a, dst)
			if e != nil || n != len(dst) || string(dst) != f.pretty {
				t.Fatal(f, n, e)
			}
			m.drop(&a)
		}
	}
}
func decimalOracle(a *big.Rat, precision uint32, truncate bool) string {
	n, d := new(big.Int).Abs(new(big.Int).Set(a.Num())), new(big.Int).Set(a.Denom())
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(n, d, remainder)
	out := quotient.String()
	if a.Sign() < 0 {
		out = "-" + out
	}
	if remainder.Sign() == 0 {
		return out
	}
	out += "."
	for i := uint32(0); i < precision; i++ {
		remainder.Mul(remainder, big.NewInt(10))
		quotient.QuoRem(remainder, d, remainder)
		out += quotient.String()
		if remainder.Sign() == 0 {
			return out
		}
	}
	if !truncate {
		out += "?"
	}
	return out
}
func TestDecimalIndependentFiniteAndPrecision(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		for n := int64(-19); n <= 19; n++ {
			for d := uint64(1); d <= 19; d++ {
				for _, p := range []uint32{0, 1, 2, 3, 10} {
					for _, truncate := range []bool{false, true} {
						a := NewRational()
						m.SetFraction(&a, n, d)
						want := decimalOracle(new(big.Rat).SetFrac(big.NewInt(n), new(big.Int).SetUint64(d)), p, truncate)
						text, e := m.DecimalString(&a, p, truncate)
						if e != nil || text != want {
							t.Fatal(synch, n, d, p, truncate, text, want, e)
						}
						dst := make([]byte, len(want))
						size, e := m.FormatDecimalInto(&a, p, truncate, dst)
						if e != nil || size != len(want) || string(dst) != want {
							t.Fatal("caller-buffer decimal", size, e)
						}
						m.drop(&a)
					}
				}
			}
		}
		a := NewRational()
		m.SetFraction(&a, 1, 4)
		text, e := m.DecimalString(&a, math.MaxUint32, false)
		if e != nil || text != "0.25" {
			t.Fatal("finite huge precision", text, e)
		}
	}
}
func TestNumericDisplayShortBufferAndLifecycle(t *testing.T) {
	m, _ := rationalManager(t, false)
	a := NewRational()
	m.SetFraction(&a, -1, 3)
	for _, call := range []func([]byte) (int, error){func(dst []byte) (int, error) { return m.FormatSMT2Into(&a, true, dst) }, func(dst []byte) (int, error) { return m.FormatDecimalInto(&a, 7, false, dst) }, func(dst []byte) (int, error) { return m.FormatPrettyInto(&a, dst) }} {
		dst := []byte("!")
		before := append([]byte(nil), dst...)
		n, e := call(dst)
		if !errors.Is(e, ErrShortBuffer) || n <= len(dst) || !bytes.Equal(before, dst) {
			t.Fatal(n, e)
		}
	}
	var raw Rational
	dst := []byte("unchanged")
	if _, e := m.FormatDecimalInto(&raw, 2, false, dst); !errors.Is(e, ErrUninitializedRational) || string(dst) != "unchanged" {
		t.Fatal(e)
	}
	text, e := m.SMT2String(&a, true)
	if e != nil {
		t.Fatal(e)
	}
	m.Close()
	if text != "(/ (- 1.0) 3.0)" {
		t.Fatal("escaped lifetime")
	}
	if _, e := m.FormatSMT2Into(&a, true, dst); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func displayFormatterRefs(m *NaturalManager) uint32 {
	var count uint32
	for _, ref := range m.formatScratch {
		if ref != (arena.VectorRef{}) {
			count++
		}
	}
	return count
}
func TestNumericDisplayReservationFailureCleanup(t *testing.T) {
	for _, synch := range []bool{false, true} {
		for operation := 0; operation < 3; operation++ {
			for _, warm := range []bool{false, true} {
				completed, failures := false, 0
				for fault := int64(0); fault < 2048; fault++ {
					b := arena.NewBudget(1 << 24)
					cfg := IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 32, Descriptors: arena.Config{PageSize: 2, Budget: b}}}
					m, e := NewRationalManager(RationalConfig{Integers: cfg, Synchronized: synch})
					if e != nil {
						t.Fatal(e)
					}
					a := NewRational()
					// Relatively prime big limbs exercise scoped copies and long formatting.
					numerator := new(big.Int).Lsh(big.NewInt(1), 4096)
					numerator.Add(numerator, big.NewInt(1))
					numerator.Neg(numerator)
					denominator := new(big.Int).Lsh(big.NewInt(1), 4064)
					denominator.Add(denominator, big.NewInt(3))
					if e = m.z.SetString(&a.num, numerator.String()); e != nil {
						t.Fatal(e)
					}
					if e = m.z.SetString(&a.den, denominator.String()); e != nil {
						t.Fatal(e)
					}
					dst := bytes.Repeat([]byte{'!'}, 4000)
					if warm {
						if _, e = m.z.natural.FormatInto([]uint32{123}, dst); e != nil {
							t.Fatal(e)
						}
						for i := range dst {
							dst[i] = '!'
						}
					}
					beforeA := a // Scalar identity snapshot; does not acquire ownership.
					b.FailAfter(fault)
					var n int
					switch operation {
					case 0:
						n, e = m.z.FormatSMT2Into(&a.num, true, dst)
					case 1:
						n, e = m.FormatSMT2Into(&a, true, dst)
					case 2:
						n, e = m.FormatDecimalInto(&a, 20, false, dst)
					}
					// Assert immediate ownership before formatting or Close can mask a leak.
					qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den})
					b.FailAfter(-1)
					if a != beforeA {
						t.Fatal("display changed input", synch, operation, warm, fault)
					}
					if e != nil {
						failures++
						if !numericPhysicalReservationRefusal(e) || !bytes.Equal(dst, bytes.Repeat([]byte{'!'}, len(dst))) {
							t.Fatal("display failure publication", synch, operation, warm, fault, n, e)
						}
					} else {
						completed = true
						if n == 0 || dst[n] != '!' {
							t.Fatal(n)
						}
					}
					m.Close()
					if b.Used() != 0 {
						t.Fatal("display close", b.Used())
					}
					if completed {
						break
					}
				}
				if !completed || failures == 0 {
					t.Fatal("incomplete sweep", synch, operation, warm, failures)
				}
			}
		}
	}
}
func TestNumericDisplayWorkspaceGrowthAndBudget(t *testing.T) {
	m, b := rationalManager(t, false)
	a := NewRational()
	if e := m.SetFraction(&a, -1, 3); e != nil {
		t.Fatal(e)
	}
	dst := bytes.Repeat([]byte{'!'}, 2500)
	if _, e := m.FormatDecimalInto(&a, 2, false, dst); e != nil {
		t.Fatal(e)
	}
	nm := m.z.natural
	oldRef, oldCapacity := nm.textRef, nm.textCapacity
	other := b.Limit() - b.Used() - 2047
	if e := b.Reserve(other); e != nil {
		t.Fatal(e)
	}
	for i := range dst {
		dst[i] = '!'
	}
	if _, e := m.FormatDecimalInto(&a, math.MaxUint32, false, dst); !errors.Is(e, arena.ErrBudget) || !bytes.Equal(dst, bytes.Repeat([]byte{'!'}, len(dst))) || nm.textRef != oldRef || nm.textCapacity != oldCapacity {
		t.Fatal("growth refusal", e)
	}
	b.Release(other)
	want := "-0." + strings.Repeat("3", 1100) + "?"
	n, e := m.FormatDecimalInto(&a, 1100, false, dst)
	if e != nil || string(dst[:n]) != want || dst[n] != '!' || nm.textCapacity != 2048 || nm.text.IsLive(oldRef) {
		t.Fatal("growth/prefix", n, e, nm.textCapacity)
	}
	// A denominator preparation must preserve a large negative numerator prefix.
	numerator := new(big.Int).Lsh(big.NewInt(1), 4096)
	numerator.Neg(numerator)
	denominator := new(big.Int).Lsh(big.NewInt(1), 4097)
	denominator.Add(denominator, big.NewInt(1))
	if e = m.z.SetString(&a.num, numerator.String()); e != nil {
		t.Fatal(e)
	}
	if e = m.z.SetString(&a.den, denominator.String()); e != nil {
		t.Fatal(e)
	}
	want = "(/ (- " + new(big.Int).Abs(numerator).String() + ".0) " + denominator.String() + ".0)"
	text, e := m.SMT2String(&a, true)
	if e != nil || text != want {
		t.Fatal("SMT2 growth/prefix", e)
	}
}
func TestNumericDisplayWarmAllocationBoundaries(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, b := rationalManager(t, synch)
		a := NewRational()
		if e := m.SetString(&a, "-18446744073709551617/3"); e != nil {
			t.Fatal(e)
		}
		dst := make([]byte, 200)
		calls := []func() (int, error){func() (int, error) { return m.z.FormatSMT2Into(&a.num, true, dst) }, func() (int, error) { return m.FormatSMT2Into(&a, true, dst) }, func() (int, error) { return m.FormatDecimalInto(&a, 20, false, dst) }}
		for _, call := range calls {
			for i := 0; i < 3; i++ {
				if _, e := call(); e != nil {
					t.Fatal(e)
				}
			}
			var e error
			if !sourceExtendedGCDDebug {
				b.FailAfter(0)
			}
			before := numericWarmProfileSnapshot(t, m.z, b)
			allocations := testing.AllocsPerRun(30, func() { _, e = call() })
			b.FailAfter(-1)
			if e != nil {
				t.Fatal("warm caller buffer", synch, allocations, e)
			}
			numericWarmProfileAudit(t, m.z, b, before, allocations, sourceExtendedGCDDebug)
		}
		stringsCalls := []func() (string, error){func() (string, error) { return m.z.SMT2String(&a.num, true) }, func() (string, error) { return m.SMT2String(&a, true) }, func() (string, error) { return m.DecimalString(&a, 20, false) }}
		for _, call := range stringsCalls {
			var text string
			var e error
			before := numericWarmProfileSnapshot(t, m.z, b)
			allocations := testing.AllocsPerRun(30, func() { text, e = call() })
			after := numericWarmProfileSnapshot(t, m.z, b)
			// AllocsPerRun includes its one warmup. Each real Runtime allocation
			// owns data and initialized-byte metadata; the result string adds one.
			events := after.count - before.count
			wantAllocs := float64(1) + 2*float64(events)/31
			if events%31 != 0 || allocations != wantAllocs || e != nil || text == "" {
				t.Fatal("final host string", synch, allocations, wantAllocs, e)
			}
			numericWarmProfileAudit(t, m.z, b, before, 0, sourceExtendedGCDDebug)
		}
	}
}

func TestSourceNoncanonicalRationalDisplayActions(t *testing.T) {
	// Reachable source RatDiv/Root outputs need not be normalized. These fixtures
	// freeze source actions, independently derived from numerator-only sign and
	// positive-numerator MachineDiv behavior; they are not mathematical decimal
	// representations or observations from a C++ run.
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		for _, f := range []struct {
			num, den     int32
			smt, decimal string
		}{
			{1, -3, "(/ 1 (- 3))", "0.-3-3-3?"},
			{-1, -3, "(/ (- 1) (- 3))", "-0.-3-3-3?"},
			{7, -3, "(/ 7 (- 3))", "-2.-3-3-3?"},
			{-7, -3, "(/ (- 7) (- 3))", "--2.-3-3-3?"},
			{1, -1, "(/ 1 (- 1))", "-1"},
			{-1, -1, "(/ (- 1) (- 1))", "--1"},
			{2, 4, "(/ 2 4)", "0.5"},
		} {
			a := Rational{num: NewInt(f.num), den: NewInt(f.den)}
			before := a
			got, e := m.SMT2String(&a, false)
			if e != nil || got != f.smt {
				t.Fatal(synch, f, got, e)
			}
			got, e = m.DecimalString(&a, 3, false)
			if e != nil || got != f.decimal {
				t.Fatal(synch, f, got, e)
			}
			wantTruncated := strings.TrimSuffix(f.decimal, "?")
			got, e = m.DecimalString(&a, 3, true)
			if e != nil || got != wantTruncated || a != before {
				t.Fatal(synch, f, got, e)
			}
		}
	}
}
