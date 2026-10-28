package numeral

// Source: src/util/mpf.cpp:194-231; src/api/api_numeral.cpp:50-88.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatFormattedLiteralSourceValues(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	var out ArbitraryFloat
	for _, tc := range []struct {
		text string
		want float64
		sign bool
	}{
		{"1.5p2", 6, false},
		{"+1.25P3", 10, false},
		{"-2.5p-1", -1.25, true},
		{"-0.0P5", 0, true},
		{"3", 3, false},
		{"1.5p2\x00ignored", 6, false},
		{"1P2p3", 96, false},
		{"3/2p2", 6, false},
		{"-1.25e2P-1", -62.5, true},
		{"1p+2", 4, false},
	} {
		serviceMust(t, m.SetFormattedLiteral(&out, 11, 53, ArbitraryFloatTowardZero, tc.text))
		got, e := m.ToDouble(&out)
		serviceMust(t, e)
		if got != tc.want || out.Sign() != tc.sign || out.EBits() != 11 || out.SBits() != 53 {
			t.Fatal("formatted source literal", tc, got, out)
		}
	}
	serviceMust(t, m.Del(&out))
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal("formatted literal owner refund", b.Used())
	}
}

func TestArbitraryFloatFormattedLiteralAllocationPrefixes(t *testing.T) {
	completed := false
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		b := arena.NewBudget(1 << 27)
		m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
		serviceMust(t, e)
		var out ArbitraryFloat
		b.FailAfter(ordinal)
		body := m.SetFormattedLiteral(&out, 11, 120, ArbitraryFloatNearestEven, "-123456789.25p17")
		if out.EBits() != 11 || out.SBits() != 120 || body == nil && !out.Sign() {
			t.Fatal("source format precedes parsing; sign follows conversion", ordinal, body, out)
		}
		b.FailAfter(-1)
		serviceMust(t, m.CleanupFormattedLiteral())
		serviceMust(t, m.CleanupValue(&out))
		if e := m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
			t.Fatal("formatted source finalize", ordinal, body, e)
		}
		if e := m.Close(); e != nil {
			t.Fatal("formatted owner close", ordinal, body, e)
		}
		if b.Used() != 0 {
			t.Fatal("formatted failed prefix retained ownership", ordinal, body, b.Used())
		}
		if body == nil {
			completed = true
			break
		}
	}
	if !completed {
		t.Fatal("formatted path never completed")
	}
}

func TestArbitraryFloatFormattedLiteralRetainedChildThenLaterCall(t *testing.T) {
	found := false
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		b := arena.NewBudget(1 << 27)
		m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
		serviceMust(t, e)
		var out ArbitraryFloat
		b.FailAfter(ordinal)
		body := m.SetFormattedLiteral(&out, 11, 4000, ArbitraryFloatNearestEven, "-123456789.25p17")
		h := m.formattedLiteral.child.frame
		b.FailAfter(-1)
		if body != nil && h != nil {
			oldBody, oldIssued := h.bodyCause, h.issued
			serviceMust(t, m.CleanupFormattedLiteral())
			serviceMust(t, m.CleanupValue(&out))
			serviceMust(t, m.SetFormattedLiteral(&out, 11, 53, ArbitraryFloatNearestEven, "2p1"))
			got, e := m.ToDouble(&out)
			serviceMust(t, e)
			if got != 4 || h.bodyCause != oldBody || h.issued != oldIssued {
				t.Fatal("later formatted call adopted older conversion history", ordinal, got)
			}
			found = true
		}
		serviceMust(t, m.CleanupFormattedLiteral())
		serviceMust(t, m.CleanupValue(&out))
		if e := m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
			t.Fatal("formatted retained source end", ordinal, e)
		}
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal("formatted retained owner refund", ordinal, b.Used())
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("no retained formatted conversion child")
	}
}

func TestArbitraryFloatFormattedLiteralEntryAuthority(t *testing.T) {
	b := arena.NewBudget(1 << 25)
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	var out ArbitraryFloat
	before, used := out, b.Used()
	if e := m.SetFormattedLiteral(nil, 8, 24, ArbitraryFloatNearestEven, "1p0"); !errors.Is(e, umemory.ErrRuntimeOwner) {
		t.Fatal("nil output", e)
	}
	copyManager := *m
	if e := copyManager.SetFormattedLiteral(&out, 8, 24, ArbitraryFloatNearestEven, "1p0"); !errors.Is(e, umemory.ErrRuntimeOwner) {
		t.Fatal("copied manager", e)
	}
	m.formattedLiteral.invocation = ^uint64(0)
	if e := m.SetFormattedLiteral(&out, 8, 24, ArbitraryFloatNearestEven, "1p0"); !errors.Is(e, ErrNumericPending) || out != before || b.Used() != used {
		t.Fatal("counter refusal before publication", e, out)
	}
	m.formattedLiteral.invocation = 0
	serviceMust(t, m.SetFormattedLiteral(&out, 8, 24, ArbitraryFloatNearestEven, "0p0"))
	serviceMust(t, m.Del(&out))
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal("entry authority owner refund", b.Used())
	}
}
