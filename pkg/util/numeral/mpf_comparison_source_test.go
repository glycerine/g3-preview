package numeral

// Source: src/util/mpf.cpp:416-469; src/util/mpf.h:116-123.

import (
	"errors"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatComparisonIndependentIEEEPairs(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	defer func() {
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	bits := []uint32{0xff800000, 0xc0800000, 0xbf800001, 0xbf800000, 0x80800000, 0x80000001, 0x80000000, 0, 1, 0x00800000, 0x3f800000, 0x3f800001, 0x40800000, 0x7f800000, 0x7fc00000, 0xffc00000}
	numeral := func(bits uint32) ArbitraryFloat {
		exponent := int64((bits>>23)&255) - 127
		if (bits>>23)&255 == 0 {
			exponent = -127
		}
		return ArbitraryFloat{format: arbitraryFloatFormat(8, 24, bits>>31 != 0), exponent: exponent, significand: NewInt(int32(bits & 0x7fffff))}
	}
	for _, xb := range bits {
		for _, yb := range bits {
			x, y := numeral(xb), numeral(yb)
			fx, fy := math.Float32frombits(xb), math.Float32frombits(yb)
			for _, op := range []struct {
				name string
				call func(*ArbitraryFloat, *ArbitraryFloat) (bool, error)
				want bool
			}{
				{"eq", m.Eq, fx == fy}, {"lt", m.Lt, fx < fy}, {"lte", m.Lte, fx <= fy}, {"le", m.Le, fx <= fy}, {"gt", m.Gt, fx > fy}, {"gte", m.Gte, fx >= fy}, {"ge", m.Ge, fx >= fy},
			} {
				got, e := op.call(&x, &y)
				serviceMust(t, e)
				if got != op.want {
					t.Fatalf("%s(%08x,%08x): got %v want %v", op.name, xb, yb, got, op.want)
				}
			}
		}
	}
}

func TestArbitraryFloatComparisonColdCachePrefixes(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 80; prefix++ {
		b := arena.NewBudget(1 << 26)
		m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
		serviceMust(t, e)
		x := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, false), exponent: 0, significand: NewInt(1)}
		y := x
		y.significand = NewInt(2)
		before := m.valueInvocation
		b.FailAfter(prefix)
		got, e := m.Lte(&x, &y)
		if e == nil && !got {
			t.Fatal("finite order", prefix)
		}
		integerPhysicalAuditWithRaw(t, m.integer, nil, &x.significand, &y.significand)
		var maps uint64
		want := m.header + m.powerWrapperBytes + m.rational.HeaderBytes + integerActualOwnedBytes(m.integer) + maps
		if b.Used() != want {
			t.Fatal("cold cache before-lift equation", prefix, b.Used(), want)
		}
		if m.valueInvocation != before || m.ValueCopyHistoryBytes() != 0 {
			t.Fatal("comparison acquired a caller NUM/history")
		}
		b.FailAfter(-1)
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal("comparison cache leak", prefix, b.Used())
		}
		if e == nil {
			success = true
			break
		}
	}
	if !success {
		t.Fatal("no successful actual cache prefix")
	}
}

func TestArbitraryFloatComparisonLiteralShortCircuits(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	var old Int
	serviceMust(t, m.integer.SetUint64(&old, 1<<40))
	stale := old
	serviceMust(t, m.integer.Del(&old))
	x := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, false), exponent: 0, significand: NewInt(1)}
	y := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, false), exponent: 1, significand: stale}
	// Neither has top/bottom exponent, and unequal exponents erase significand reads.
	if got, e := m.Eq(&x, &y); e != nil || got {
		t.Fatal("eq read erased stale suffix", got, e)
	}
	if got, e := m.Lt(&x, &y); e != nil || !got {
		t.Fatal("lt read erased stale suffix", got, e)
	}
	y.exponent = 0
	if _, e := m.Eq(&x, &y); e == nil {
		t.Fatal("equal exponent did not reach stale significand")
	}
	nan := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, false), exponent: 128, significand: NewInt(1)}
	if got, e := m.Gte(&nan, &y); e != nil || got {
		t.Fatal("first NaN did not erase right operand", got, e)
	}
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestArbitraryFloatComparisonSourceFormatAssertion(t *testing.T) {
	d, log := bindMPNDebugActions(t, "TTTTT")
	b := arena.NewBudget(1 << 26)
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	x := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, false), significand: NewInt(17)}
	y := x
	y.assignSbits(25)
	_, e = m.IsNaN(&x)
	serviceMust(t, e)
	_, e = m.IsZero(&x)
	serviceMust(t, e)
	before := b.Used()
	b.FailAfter(0)
	for _, tc := range []struct {
		line int
		call func(*ArbitraryFloat, *ArbitraryFloat) (bool, error)
	}{{417, m.Eq}, {429, m.Lt}, {451, m.Lte}, {456, m.Gt}, {466, m.Gte}} {
		_, e := tc.call(&x, &y)
		if modularSourceDebug {
			if !isMPNVerificationThrow(e) {
				t.Fatal("source format SASSERT", tc.line, e)
			}
		} else {
			serviceMust(t, e)
		}
		if modularSourceDebug {
			contents, readErr := os.ReadFile(log.Name())
			serviceMust(t, readErr)
			want := "File: src/util/mpf.cpp\nLine: " + strconv.Itoa(tc.line) + "\nx.sbits == y.sbits && x.ebits == y.ebits\n"
			if n := strings.Count(string(contents), want); n != 1 {
				t.Fatal("source comparison assertion site", tc.line, n, string(contents))
			}
		}
	}
	if b.Used() != before {
		t.Fatal("format assertion acquired storage")
	}
	b.FailAfter(-1)
	if modularSourceDebug {
		serviceMust(t, d.Close())
		if _, e := m.Eq(&x, &x); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("valid comparison reads closed assertion owner", e)
		}
	}
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestArbitraryFloatComparisonSourceFormatContinue(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertion only")
	}
	d, log := bindMPNDebugActions(t, "CCC")
	m, _, _ := mpfNumeralHelpersFixture(t)
	var x, y ArbitraryFloat
	serviceMust(t, m.MkZero(8, 24, false, &x))
	serviceMust(t, m.MkZero(8, 25, false, &y))
	if got, e := m.Lte(&x, &y); e != nil || !got {
		t.Fatal("Continue follows nested lte/lt/eq source calls", got, e)
	}
	contents, e := os.ReadFile(log.Name())
	serviceMust(t, e)
	for _, line := range []string{"451", "429", "417"} {
		want := "File: src/util/mpf.cpp\nLine: " + line + "\nx.sbits == y.sbits && x.ebits == y.ebits\n"
		if n := strings.Count(string(contents), want); n != 1 {
			t.Fatal("nested comparison assertion site", line, n, string(contents))
		}
	}
	serviceMust(t, d.EnableAssertions(false))
	if got, e := m.Lte(&x, &y); e != nil || !got {
		t.Fatal("disabled assertions retain source comparison", got, e)
	}
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.Del(&y))
}
