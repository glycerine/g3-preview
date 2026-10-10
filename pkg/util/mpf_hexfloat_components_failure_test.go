// Source: src/util/mpf.cpp:1608-1619; actual scoped-value and child prefixes.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	utext "github.com/glycerine/g3/pkg/util/text"
	"math/big"
	"testing"
)

func TestArbitraryFloatHexfloatComponentsEveryColdReservation(t *testing.T) {
	for _, tc := range []struct {
		eb, sb, scale uint32
		exp           int64
		sig           *big.Int
	}{
		{8, 4, 0, 0, big.NewInt(12)}, {8, 4, 2, -126, big.NewInt(3)},
		{8, 120, 3, 17, new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 122), big.NewInt(121))},
		{8, 120, 0, -126, big.NewInt(1)},
		{8, 120, 3, 0, new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 130))},
	} {
		complete := false
		for ordinal := int64(0); ordinal < 256; ordinal++ {
			m, r, b := mpfNumeralHelpersFixture(t)
			sig := NewInt(0)
			serviceMust(t, m.integer.SetString(&sig, tc.sig.String()))
			before := sig
			text, e := NewArbitraryFloatFormatText(b)
			serviceMust(t, e)
			b.FailAfter(ordinal)
			_, body := m.ToStringHexfloatComponentsInto(true, tc.exp, &sig, tc.eb, tc.sb, tc.scale, text)
			s, f := &m.hexfloatComponents, &text.Numeric
			integerPhysicalAudit(t, m.integer, &sig, &s.sig, &s.q.significand, &m.rational.NTmp, &f.Num, &f.Denom, &f.Q.Num, &f.Q.Den, &f.Decimal.Values[0], &f.Decimal.Values[1], &f.Decimal.Values[2], &f.Decimal.Values[3])
			if sig != before || b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes+text.StorageBytes() {
				t.Fatal("component immediate graph", tc.sb, ordinal, body, b.Used(), m.OwnedStorageBytes(), r.Stats(), text.StorageBytes())
			}
			if body != nil {
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected component failure", tc.sb, ordinal, body)
				}
				if _, e := m.ToStringHexfloatComponentsInto(false, 0, &sig, tc.eb, tc.sb, 0, text); !errors.Is(e, ErrNumericPending) {
					t.Fatal("failed component replay", e)
				}
			}
			b.FailAfter(-1)
			serviceMust(t, m.CleanupHexfloatComponents())
			serviceMust(t, text.Close())
			serviceMust(t, m.integer.CleanupRawNumeral(&sig))
			if e := m.Close(); e != nil {
				z := m.integer
				if !errors.Is(e, arena.ErrInvalidSlot) || !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
					t.Fatal("component class stop", e)
				}
				index := int(z.SourceFinalizeStage-2) / 2
				ref := z.Tmp[index]
				if z.SourceFinalizeStage%2 == 1 {
					ref = z.Arg[index]
				}
				if ref != 0 || s.active {
					t.Fatal("unproved component source scratch hole", ref)
				}
				serviceMust(t, m.Close())
			}
			serviceMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal("component refunds", tc.sb, ordinal, b.Used())
			}
			if body == nil {
				complete = true
				break
			}
		}
		if !complete {
			t.Fatal("component reservation horizon", tc.sb)
		}
	}
}

func TestArbitraryFloatHexfloatComponentsWarmAndRefusedText(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	sig := NewInt(2)
	warm, e := NewArbitraryFloatFormatText(b)
	serviceMust(t, e)
	_, e = m.ToStringHexfloatComponentsInto(true, 2, &sig, 2, 2, 0, warm)
	serviceMust(t, e)
	serviceMust(t, warm.Close())
	text, e := NewArbitraryFloatFormatText(b)
	serviceMust(t, e)
	used, stats := b.Used(), r.Stats()
	b.FailAfter(0)
	view, e := m.ToStringHexfloatComponentsInto(true, 2, &sig, 2, 2, 0, text)
	serviceMust(t, e)
	if formatTestString(t, view) != "-oo" || b.Used() != used || r.Stats() != stats {
		t.Fatal("warm special component allocated")
	}
	b.FailAfter(-1)
	// A reused target is refused only at the actual ordinary-format child.
	oldFrame, inv, used := text.Numeric, m.formatInvocation, b.Used()
	_, e = m.ToStringHexfloatComponentsInto(false, 2, &sig, 2, 2, 0, text)
	if !errors.Is(e, ErrNumericPending) || m.hexfloatComponents.format != nil {
		t.Fatal("refused text captured", e)
	}
	serviceMust(t, m.CleanupHexfloatComponents())
	if text.Numeric != oldFrame || m.formatInvocation != inv || b.Used() != used || formatTestString(t, view) != "-oo" {
		t.Fatal("component adopted older target")
	}
	serviceMust(t, text.Close())
}

func TestArbitraryFloatHexfloatComponentsEntryAuthority(t *testing.T) {
	m, _, b := mpfNumeralHelpersFixture(t)
	sig := NewInt(12)
	if _, e := m.ToStringHexfloatComponentsInto(false, 0, &sig, 8, 4, 0, nil); !errors.Is(e, utext.ErrOwnedStringOwner) {
		t.Fatal("nil text", e)
	}
	text, e := NewArbitraryFloatFormatText(b)
	serviceMust(t, e)
	copyOwner := *m
	if _, e := copyOwner.ToStringHexfloatComponentsInto(false, 0, &sig, 8, 4, 0, text); !errors.Is(e, umemory.ErrRuntimeOwner) {
		t.Fatal("copied manager", e)
	}
	m.hexfloatComponents.invocation = ^uint64(0)
	if _, e := m.ToStringHexfloatComponentsInto(false, 0, &sig, 8, 4, 0, text); !errors.Is(e, ErrNumericPending) || text.Numeric.invocation != 0 {
		t.Fatal("component counter wrap", e)
	}
	m.hexfloatComponents.invocation = 0
	if _, e := m.ToStringHexfloatComponentsInto(false, 0, nil, 8, 4, 0, text); e == nil {
		t.Fatal("nil input accepted")
	}
	serviceMust(t, text.Close())
}

func TestArbitraryFloatHexfloatComponentsRefusedDivisionKeepsOlderRoots(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 80; ordinal++ {
		budget := arena.NewBudget(1 << 28)
		m, e := NewArbitraryFloatManager(IntegerConfig{Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: budget, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
		serviceMust(t, e)
		z := m.integer
		sig := NewInt(-3)
		warm, e := NewArbitraryFloatFormatText(budget)
		serviceMust(t, e)
		_, e = m.ToStringHexfloatComponentsInto(false, 0, &sig, 8, 4, 1, warm)
		serviceMust(t, e)
		serviceMust(t, warm.Close())
		a, b, q := NewInt(-1), NewInt(1), NewInt(0)
		serviceMust(t, z.Mul2kInPlace(&a, 4000))
		serviceMust(t, z.Mul2kInPlace(&b, 2000))
		serviceMust(t, z.Inc(&b))
		budget.FailAfter(ordinal)
		body := z.Div(&a, &b, &q)
		old := &z.DivisionRaw
		matched := body != nil && old.Active && NumericAllocationPending(&old.Numeric)
		budget.FailAfter(-1)
		if matched {
			reached = true
			roots, values, inv, used := old.Numeric, old.Values, old.Invocation, budget.Used()
			out, e := NewArbitraryFloatFormatText(budget)
			serviceMust(t, e)
			callerBytes := out.StorageBytes()
			if _, e := m.ToStringHexfloatComponentsInto(false, 0, &sig, 8, 4, 1, out); !errors.Is(e, ErrNumericPending) {
				t.Fatal("older division refusal", ordinal, e)
			}
			if m.hexfloatComponents.division.Invocation != 0 {
				t.Fatal("refused division adopted")
			}
			serviceMust(t, m.CleanupHexfloatComponents())
			if !old.Active || old.Numeric != roots || old.Values != values || old.Invocation != inv || budget.Used() != used+callerBytes {
				t.Fatal("component cleanup touched older division")
			}
			integerPhysicalAudit(t, z, &a, &b, &q, &sig, &m.hexfloatComponents.sig, &m.hexfloatComponents.q.significand)
			serviceMust(t, out.Close())
		}
		serviceMust(t, z.CleanupDivision())
		for _, v := range []*Int{&a, &b, &q} {
			serviceMust(t, z.CleanupRawNumeral(v))
		}
		if e := m.SourceFinalize(); e != nil {
			if !errors.Is(e, arena.ErrInvalidSlot) || !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
				t.Fatal("older division class stop", e)
			}
			i := int(z.SourceFinalizeStage-2) / 2
			ref := z.Tmp[i]
			if z.SourceFinalizeStage%2 == 1 {
				ref = z.Arg[i]
			}
			if ref != 0 {
				t.Fatal("unproved old division scratch hole", ref)
			}
		}
		serviceMust(t, m.Close())
		if budget.Used() != 0 {
			t.Fatal("old division refunds", budget.Used())
		}
		if matched {
			break
		}
	}
	if !reached {
		t.Fatal("no actual retained RAW division allocation reached")
	}
}
