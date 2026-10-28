package numeral

// Source: src/util/mpf.cpp:373-391; src/util/mpf.h:92-96.

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatSignSourceBranches(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	defer func() {
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	for _, tc := range []struct {
		name string
		exp  int64
		sig  int32
		nan  bool
	}{
		{"normal", 0, 3, false}, {"denormal", -127, 1, false}, {"zero", -127, 0, false},
		{"infinity", 128, 0, false}, {"nan", 128, 1, true},
	} {
		for _, sign := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{false: "+", true: "-"}[sign], func(t *testing.T) {
				x := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, sign), exponent: tc.exp, significand: NewInt(tc.sig)}
				before := x
				serviceMust(t, m.Neg(&x))
				want := !sign
				if tc.nan {
					want = sign
				}
				if x.Sign() != want || x.EBits() != 8 || x.SBits() != 24 || x.exponent != before.exponent || x.significand != before.significand {
					t.Fatal("neg source fields", x, before)
				}
				serviceMust(t, m.Abs(&x))
				if x.Sign() || x.exponent != before.exponent || x.significand != before.significand {
					t.Fatal("abs source fields", x, before)
				}
				var copy ArbitraryFloat
				serviceMust(t, m.NegCopy(&copy, &before, nil))
				if copy.Sign() != want || copy.exponent != before.exponent || copy.significand != before.significand {
					t.Fatal("copy neg", copy, before)
				}
				serviceMust(t, m.AbsCopy(&copy, &before, nil))
				if copy.Sign() || copy.exponent != before.exponent || copy.significand != before.significand {
					t.Fatal("copy abs", copy, before)
				}
				serviceMust(t, m.Del(&copy))
			})
		}
	}
}

func TestArbitraryFloatSignCopyAllocationPrefixes(t *testing.T) {
	for _, neg := range []bool{false, true} {
		success := false
		for prefix := int64(0); prefix < 40; prefix++ {
			b := arena.NewBudget(1 << 26)
			m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
			serviceMust(t, e)
			source := ArbitraryFloat{format: arbitraryFloatFormat(11, 53, true), exponent: 42}
			serviceMust(t, m.integer.SetUint64(&source.significand, 1<<40))
			// Warm the classification cache so the sweep isolates the source SET child.
			_, e = m.IsNaN(&source)
			serviceMust(t, e)
			var out ArbitraryFloat
			var receipt ArbitraryFloatValueCopyReceipt
			b.FailAfter(prefix)
			if neg {
				e = m.NegCopy(&out, &source, &receipt)
			} else {
				e = m.AbsCopy(&out, &source, &receipt)
			}
			if e != nil && !out.Sign() {
				t.Fatal("failed SET reached later sign mutation", neg, prefix, e, out)
			}
			if e == nil {
				if out.Sign() || out.Exponent() != 42 {
					t.Fatal("successful sign copy", neg, prefix, out)
				}
				v, read := m.integer.Uint64(&out.significand)
				serviceMust(t, read)
				if v != 1<<40 {
					t.Fatal("lost significand", v)
				}
			}
			integerPhysicalAuditWithRaw(t, m.integer, nil, &source.significand, &out.significand)
			var maps uint64
			want := m.header + m.ValueCopyHistoryBytes() + m.powerWrapperBytes + m.rational.HeaderBytes + integerActualOwnedBytes(m.integer) + maps
			if b.Used() != want {
				t.Fatal("before-lift physical/cold graph", neg, prefix, b.Used(), want)
			}
			b.FailAfter(-1)
			serviceMust(t, m.CleanupValueCopy(&receipt, false))
			serviceMust(t, m.CleanupValue(&out))
			serviceMust(t, m.CleanupValueCopy(&receipt, true))
			serviceMust(t, m.Del(&source))
			serviceMust(t, m.Close())
			if b.Used() != 0 {
				t.Fatal("sign copy owner leak", neg, prefix, b.Used())
			}
			if e == nil {
				success = true
				break
			}
		}
		if !success {
			t.Fatal("no successful actual SET prefix", neg)
		}
	}
}

func TestArbitraryFloatSignErasedAllocationPaths(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	x := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, true), exponent: 0, significand: NewInt(17)}
	_, e = m.IsNaN(&x)
	serviceMust(t, e)
	before := b.Used()
	var receipt ArbitraryFloatValueCopyReceipt
	b.FailAfter(0)
	serviceMust(t, m.Abs(&x))
	serviceMust(t, m.Neg(&x))
	serviceMust(t, m.AbsCopy(&x, &x, &receipt))
	serviceMust(t, m.NegCopy(&x, &x, &receipt))
	if b.Used() != before || receipt != (ArbitraryFloatValueCopyReceipt{}) || m.ValueCopyHistoryBytes() != 0 {
		t.Fatal("erased sign operation acquired storage", b.Used(), before, receipt)
	}
	b.FailAfter(-1)
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
