// Copyright (c) 2006 Microsoft Corporation
// Source parser assertions: src/test/mpq.cpp set_str_bug, src/test/mpz.cpp
// tst1/bug1/tst_int_min_bug. Added quirk expectations are source-derived;
// independent mathematical expectations are test-only.
package mpnum

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"testing"

	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

func TestSourceStringAssertions(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a, out := NewRational(), NewRational()
		for _, f := range []struct {
			text string
			n    int64
			d    uint64
		}{{"1.0", 1, 1}, {"1.1", 11, 10}, {"1/3", 1, 3}} {
			if e := m.SetString(&a, f.text); e != nil {
				t.Fatal(e)
			}
			m.SetFraction(&out, f.n, f.d)
			if equal, e := m.Equal(&a, &out); e != nil || !equal {
				t.Fatal("source set_str_bug", f, e)
			}
		}
		var v, v2, v3 Int
		if e := m.z.SetString(&v, "1002034040050606089383838288182"); e != nil {
			t.Fatal(e)
		}
		minusTwo := NewInt(-2)
		m.z.Mul(&v, &minusTwo, &v2)
		m.z.Add(&v, &v2, &v3)
		m.z.Neg(&v3)
		for _, f := range []struct {
			a, b *Int
			want int
		}{{&v, &v3, 0}, {&v2, &v, -1}, {&v, &v2, 1}} {
			cmp, e := m.z.Compare(f.a, f.b)
			if e != nil || cmp != f.want {
				t.Fatal("source tst1 equality/order", cmp, e)
			}
		}
		m.z.SetString(&v, "1002043949858757875676767675747473")
		m.z.Sub(&v, &v, &v2)
		if integerString(t, m.z, &v2) != "0" {
			t.Fatal("source bug1")
		}
		m.z.SetString(&v, "18446744075857035263")
		m.z.SetUint64(&v2, ^uint64(0))
		intmin := NewInt(-2147483648)
		m.z.Sub(&v2, &intmin, &v3)
		if cmp, e := m.z.Compare(&v, &v3); e != nil || cmp != 0 {
			t.Fatal("source int_min_bug", cmp, e)
		}
		// Manager projections of active rational.cpp parser identities, not a port of its global wrapper.
		m.SetString(&a, "13/10")
		m.SetString(&out, "7/10")
		m.Add(&a, &out, &a)
		if rationalString(t, m, &a) != "2" {
			t.Fatal("source rational parser sum")
		}
		m.SetString(&a, "100/20")
		if rationalString(t, m, &a) != "5" {
			t.Fatal("source rational reduction")
		}
		m.SetString(&a, "10000000000000000000000000000000000/3")
		m.SetString(&out, "20000000000000000000000000000000000/6")
		if equal, e := m.Equal(&a, &out); e != nil || !equal {
			t.Fatal("source huge fraction", e)
		}
	}
}
func TestIntegerParserByteQuirks(t *testing.T) {
	m, _ := integerManager(t)
	var out Int
	fixtures := []struct{ text, want string }{{"", "0"}, {"junk", "0"}, {"-", "0"}, {" -x1z2", "-12"}, {" \t-12", "12"}, {"\n-12", "12"}, {"+12", "12"}, {"--12", "-12"}, {"1e-2", "12"}, {"1.2/3", "123"}, {"12\x0034", "12"}, {"\x00-12", "0"}, {"１２3", "3"}, {"\xff\xfe-1z2", "12"}, {"-0000", "0"}, {" -9223372036854775808", "-9223372036854775808"}}
	for _, f := range fixtures {
		if e := m.SetString(&out, f.text); e != nil || integerString(t, m, &out) != f.want {
			t.Fatal(f, e)
		}
	}
}
func TestRationalParserByteQuirksAndState(t *testing.T) {
	fixtures := []struct{ text, want string }{{"", "0"}, {"junk", "0"}, {" -x1z2", "-12"}, {" \t-12", "12"}, {"1/2/3", "1/23"}, {"1/-2", "1/2"}, {"-1/-2", "-1/2"}, {"1.2/3", "123/100"}, {"1.2.3", "123/100"}, {"1.", "1"}, {".5", "1/2"}, {"-.5", "-1/2"}, {"1e -2", "100"}, {"1e--2", "1/100"}, {"1e+-2", "100"}, {"1e2e3", "100000000000000000000000"}, {"1e18446744073709551616", "1"}, {"1e18446744073709551617", "10"}, {"1e-18446744073709551617", "1/10"}, {"12\x0034", "12"}, {"1/2\x00e0", "1/2"}, {"1.5e2\x00/3", "150"}, {"１２3.４5", "7/2"}, {"\xff-12", "12"}}
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a := NewRational()
		for _, f := range fixtures {
			m.Reset(&a)
			if e := m.SetString(&a, f.text); e != nil || rationalString(t, m, &a) != f.want {
				t.Fatal(synch, f, e)
			}
		}
		for _, f := range []struct{ seed, text, want string }{{"1/3", "1e2", "100/3"}, {"1/3", "1e0", "1/3"}, {"2/7", "-1E-2", "-1/700"}, {"2/7", "1E", "1/7"}, {"2/7", "1.0E0", "1"}, {"2/7", "1", "1"}, {"2/7", "", "0"}} {
			m.SetString(&a, f.seed)
			if e := m.SetString(&a, f.text); e != nil || rationalString(t, m, &a) != f.want {
				t.Fatal("source previous denominator", synch, f, e)
			}
		}
	}
}
func TestParserExceptionCategoriesAndFailurePublication(t *testing.T) {
	for _, synch := range []bool{false, true} {
		for _, f := range []struct {
			text     string
			reason   ParseErrorReason
			sentinel error
			n, d     int64
		}{
			{"1/2e0", ParseMixedNotation, ErrMixedNotation, 1, 2}, {"1/0e0", ParseMixedNotation, ErrMixedNotation, 1, 0},
			{"1e999999999999/0", ParseMixedNotation, ErrMixedNotation, 1, 13}, {"1/2E", ParseMixedNotation, ErrMixedNotation, 1, 2},
			{"1e2/3", ParseMixedNotation, ErrMixedNotation, 1, 13}, {"1e/3", ParseMixedNotation, ErrMixedNotation, 1, 13},
			{"1/", ParseZeroDenominator, ErrDivideByZero, 1, 0}, {"1/00", ParseZeroDenominator, ErrDivideByZero, 1, 0},
			{"/", ParseZeroDenominator, ErrDivideByZero, 0, 0}, {"1e4294967296", ParseExponentTooLarge, ErrExponentTooLarge, 1, 13},
			{"1e18446744078004518912", ParseExponentTooLarge, ErrExponentTooLarge, 1, 13},
		} {
			m, budget := rationalManager(t, synch)
			a := NewRational()
			qopsMust(t, m.SetFraction(&a, -17, 13))
			e := m.SetString(&a, f.text)
			parserPublicationAudit(t, m, &a.num, &a.den)
			var parsed *ParseError
			if !errors.As(e, &parsed) || parsed.Reason != f.reason || !errors.Is(e, f.sentinel) || roundPublicationDecode(t, m.z, &a.num).Cmp(big.NewInt(f.n)) != 0 || roundPublicationDecode(t, m.z, &a.den).Cmp(big.NewInt(f.d)) != 0 {
				t.Fatal(synch, f, e)
			}
			qopsMust(t, m.Del(&a))
			m.Close()
			if budget.Used() != 0 {
				t.Fatal(budget.Used())
			}
		}
		for _, text := range []string{"1e2147483648", "1e4294967295", "1e2147483647"} {
			m, budget := rationalManager(t, synch)
			a := NewRational()
			qopsMust(t, m.SetFraction(&a, -17, 13))
			var held uint64
			if text == "1e2147483647" {
				held = budget.Limit() - budget.Used() - 4096
				qopsMust(t, budget.Reserve(held))
			} else {
				budget.FailAfter(0)
			}
			e := m.SetString(&a, text)
			parserPublicationAudit(t, m, &a.num, &a.den)
			if roundPublicationDecode(t, m.z, &a.num).Cmp(big.NewInt(1)) != 0 || roundPublicationDecode(t, m.z, &a.den).Cmp(big.NewInt(13)) != 0 {
				t.Fatal("source caller prefix before private Power refusal", text)
			}
			var power *PowerDomainError
			var parsed *ParseError
			if text == "1e2147483647" {
				var failure memory.Failure
				if !errors.Is(e, arena.ErrBudget) && !errors.Is(e, arena.ErrBounds) && (!errors.As(e, &failure) || failure.Kind != memory.FailureMemout || !m.z.memory.State().OutOfMemory) {
					t.Fatal(text, e)
				}
			} else {
				var failure memory.Failure
				if (!errors.Is(e, arena.ErrInjectedFailure) && (!errors.As(e, &failure) || failure.Kind != memory.FailureMemout || !m.z.memory.State().OutOfMemory)) || errors.As(e, &power) || errors.As(e, &parsed) {
					t.Fatal(text, e)
				}
			}
			budget.FailAfter(-1)
			if held != 0 {
				budget.Release(held)
			}
			qopsMust(t, m.Del(&a))
			m.Close()
			if budget.Used() != 0 {
				t.Fatal(budget.Used())
			}
		}
	}
}
func TestParserIndependentFiniteValues(t *testing.T) {
	r := rand.New(rand.NewSource(141))
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a := NewRational()
		var integer Int
		for i := 0; i < 200; i++ {
			n := roundRandomInteger(r, 1+r.Intn(8))
			d := new(big.Int).Abs(roundRandomInteger(r, 1+r.Intn(4)))
			if d.Sign() == 0 {
				d.SetInt64(1)
			}
			if e := m.z.SetString(&integer, n.String()); e != nil {
				t.Fatal(e)
			}
			assertIntegerPowerOracle(t, m.z, &integer, n)
			m.Reset(&a)
			if e := m.SetString(&a, n.String()+"/"+d.String()); e != nil {
				t.Fatal(e)
			}
			want := new(big.Rat).SetFrac(n, d)
			if rationalString(t, m, &a) != want.RatString() {
				t.Fatal(i)
			}
			whole, frac, exp := r.Intn(10000), r.Intn(10000), r.Intn(13)-6
			sign := 1
			if r.Intn(2) == 0 {
				sign = -1
			}
			text := fmt.Sprintf("%d.%04de%d", whole, frac, exp)
			if sign < 0 {
				text = "-" + text
			}
			num := big.NewInt(int64(sign * (whole*10000 + frac)))
			den := big.NewInt(10000)
			scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exp)), nil)
			if exp < 0 {
				scale.Exp(big.NewInt(10), big.NewInt(int64(-exp)), nil)
				den.Mul(den, scale)
			} else {
				num.Mul(num, scale)
			}
			want.SetFrac(num, den)
			if e := m.SetString(&a, text); e != nil || rationalString(t, m, &a) != want.RatString() {
				t.Fatal(i, text, want, e)
			}
		}
	}
}
func TestParserBudgetFailureCleanup(t *testing.T) {
	for _, synch := range []bool{false, true} {
		for _, text := range []string{"-10020340400506060893838382881821234567890123456789012345678901234567890", "123456789012345678901234567890/987654321012345678901234567", "-123456789012345678901.1234567890123456789e-17", "123456789012345678901e17"} {
			for integer := 0; integer < 2; integer++ {
				failures, successes := 0, 0
				ceiling := int64(80)
				if integer == 0 {
					ceiling = 1024
				}
				for after := int64(0); after < ceiling; after++ {
					m, budget := rationalManager(t, synch)
					a := NewRational()
					m.SetFraction(&a, 17, 13)
					out := NewInt(17)
					call := func() error {
						if integer == 1 {
							return m.z.SetString(&out, text)
						}
						return m.SetString(&a, text)
					}
					budget.FailAfter(after)
					e := call()
					qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &out})
					if e != nil && integer == 0 {
						parserPublicationAudit(t, m, &a.num, &a.den, &out)
						var failure memory.Failure
						var parsed *ParseError
						var power *PowerDomainError
						if (!errors.Is(e, arena.ErrInjectedFailure) && (!errors.As(e, &failure) || failure.Kind != memory.FailureMemout || !m.z.memory.State().OutOfMemory)) || errors.As(e, &parsed) || errors.As(e, &power) || !parserPublicationSweepMatches(t, m, &a, text) || roundPublicationDecode(t, m.z, &out).Cmp(big.NewInt(17)) != 0 {
							t.Fatal("source parser failure pair", synch, text, after, e, a)
						}
					}
					budget.FailAfter(-1)
					if e != nil {
						failures++
						var failure memory.Failure
						if integer == 1 && ((!errors.Is(e, arena.ErrInjectedFailure) && (!errors.As(e, &failure) || failure.Kind != memory.FailureMemout || !m.z.memory.State().OutOfMemory)) || rationalString(t, m, &a) != "17/13" || integerString(t, m.z, &out) != "17") {
							t.Fatal("integer parser failure published", synch, text, integer, after, e)
						}
					} else {
						successes++
					}
					if e == nil && integer == 1 {
						var digits strings.Builder
						for i := 0; i < len(text); i++ {
							if text[i] >= '0' && text[i] <= '9' {
								digits.WriteByte(text[i])
							}
						}
						expected, ok := new(big.Int).SetString(digits.String(), 10)
						if !ok {
							t.Fatal("oracle fixture")
						}
						if text[0] == '-' {
							expected.Neg(expected)
						}
						assertIntegerPowerOracle(t, m.z, &out, expected)
					} else if e == nil {
						expected, ok := new(big.Rat).SetString(text)
						if !ok {
							t.Fatal("oracle fixture")
						}
						// Source no-dot scientific parsing retains the seeded denominator13.
						if strings.Contains(text, "e") && !strings.Contains(text, ".") {
							expected.Quo(expected, big.NewRat(13, 1))
						}
						if rationalString(t, m, &a) != expected.RatString() {
							t.Fatal("successful rational mismatch", synch, text, after, expected)
						}
					}
					m.Close()
					if budget.Used() != 0 {
						t.Fatal("parser leaked", budget.Used())
					}
					if integer == 0 && e == nil {
						break
					}
				}
				if successes == 0 || failures == 0 {
					t.Fatal("parser sweep boundary", synch, text, integer, failures, successes)
				}
			}
		}
	}
}
func TestParserWarmLifetimeAndInitialization(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a := NewRational()
		var integer Int
		text := "-123456789012345678901.123456789e-7"
		call := func() error {
			if e := m.z.SetString(&integer, text); e != nil {
				return e
			}
			return m.SetString(&a, text)
		}
		for i := 0; i < 3; i++ {
			if e := call(); e != nil {
				t.Fatal(e)
			}
		}
		var e error
		used := m.budget.Used()
		cells := m.z.cells.Stats().Live
		physical, objects := m.z.memory.Stats(), m.z.allocator.Stats()
		size, err := m.z.memory.AllocationSize()
		qopsMust(t, err)
		count, err := m.z.memory.AllocationCount()
		qopsMust(t, err)
		got := testing.AllocsPerRun(20, func() { e = call() })
		parserPublicationAudit(t, m, &a.num, &a.den, &integer)
		afterSize, err := m.z.memory.AllocationSize()
		qopsMust(t, err)
		afterCount, err := m.z.memory.AllocationCount()
		qopsMust(t, err)
		if e != nil || used != m.budget.Used() || cells != m.z.cells.Stats().Live || physical != m.z.memory.Stats() || objects != m.z.allocator.Stats() || size != afterSize || objects.PendingBlocks != 0 {
			t.Fatal("warmed physical ownership changed", synch, e)
		}
		if objects.DirectObjects == 0 {
			if objects.Chunks == 0 || got != 0 || afterCount != count {
				t.Fatal("pooled warmed allocation", synch, got, count, afterCount)
			}
		} else if objects.Chunks != 0 || afterCount <= count {
			t.Fatal("source Direct allocation events missing", count, afterCount)
		}
		var raw Rational
		if _, e = m.IsInt(&raw); !errors.Is(e, ErrUninitializedRational) {
			t.Fatal(e)
		}
		if e = m.SetString(&raw, "1"); e != nil || rationalString(t, m, &raw) != "1" {
			t.Fatal("explicit initializer", e)
		}
		// Input is borrowed: a replacement string does not become part of either numeral.
		m.SetString(&a, strings.Repeat("0", 50)+"1/3")
		if rationalString(t, m, &a) != "1/3" {
			t.Fatal("borrowed input")
		}
		m.Close()
		if e = m.z.SetString(&integer, "1"); !errors.Is(e, arena.ErrClosed) {
			t.Fatal(e)
		}
		if e = m.SetString(&a, "1"); !errors.Is(e, arena.ErrClosed) {
			t.Fatal(e)
		}
	}
}

func TestSourceParserLiteralInventory(t *testing.T) {
	// Additional exact expectations for source bug2/scoped print-only integer setups.
	m, _ := integerManager(t)
	var out Int
	for _, text := range []string{"78160958900072552148646898355155840547214990303507678364419557473408815232466629049311995556059463490539128818602490221544425042127795", "403412298227858394469856607272647132163832860126054679347881638761723785858733108109249157334220127", "102928187172727273", "1029489380487098723984579237"} {
		want, ok := new(big.Int).SetString(text, 10)
		if !ok {
			t.Fatal(text)
		}
		if e := m.SetString(&out, text); e != nil {
			t.Fatal(e)
		}
		assertIntegerPowerOracle(t, m, &out, want)
	}
	// Embedded active rational.cpp textual inputs, projected through the concrete
	// manager. Independent parse expectations do not claim wrapper/hash/root ports.
	fixtures := []string{"13/10", "7/10", "100/20", "10000000000000000000000000000000000", "10000000000000000000000000000000000/3", "20000000000000000000000000000000000/6", "1/3000000000000000000000000", "300000000/5", "300000000/10", "5/600000000", "200000000/5", "10000000000000000000000000000000003/3", "300000000000000", "0000000000000000000", "3000000000000000000/2", "99999999999/2", "23/2", "9223372036854775807", "2147483646", "4294967293", "4294967295", "10203939394995449949494394932929", "41000000000000", "-7000000000000", "-41000000000000", "7000000000000", "10000000000000000000000000000000001"}
	for _, synch := range []bool{false, true} {
		rm, _ := rationalManager(t, synch)
		a := NewRational()
		for _, text := range fixtures {
			want, ok := new(big.Rat).SetString(text)
			if !ok {
				t.Fatal(text)
			}
			if e := rm.SetString(&a, text); e != nil || rationalString(t, rm, &a) != want.RatString() {
				t.Fatal(text, e)
			}
		}
	}
}

func TestParserSourceIntMinRepresentation(t *testing.T) {
	m, _ := integerManager(t)
	var a Int
	if e := m.SetString(&a, "-2147483648"); e != nil {
		t.Fatal(e)
	}
	if a.IsSmall() {
		t.Fatal("source digit arithmetic should preserve big INT_MIN magnitude")
	}
	if e := m.MachineDiv2k(&a, 31, &a); e != nil || integerString(t, m, &a) != "-1" {
		t.Fatal("parsed big source shift", e)
	}
}
