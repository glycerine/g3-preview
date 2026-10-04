// Licensed under the MIT license.
// Source action fixtures: src/util/mpz.h allocate_if_needed/set_i64 and
// src/util/mpz.cpp set_big_i64/set_big_ui64/ensure_capacity/normalize/power.
package mpnum

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func TestIntegerBaseTwoPowerDestructiveCapacityPrefix(t *testing.T) {
	m, b := mpzPublicationOwner(t)
	var out Int
	if e := m.SetInt64(&out, -(1 << 40)); e != nil {
		t.Fatal(e)
	}
	old := out.cell
	before := mpzPublicationSnapshot(m, b)
	two := NewInt(2)
	b.FailAfter(0)
	e := m.Power(&two, 1024, &out)
	mpzPublicationAudit(t, m, &out)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal(e)
	}
	if out.val != 1 || out.cell != m.failedRef {
		t.Fatalf("source val1 and freed replacement identity: %+v", out)
	}
	mpzPublicationExpectFreed(t, m, b, old, before)
	if e = m.check(&out); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("failed output read", e)
	}
	if e = m.Del(&out); e != nil {
		t.Fatal(e)
	}
}

func TestIntegerAllocateIfNeededAndPreservingCapacityDistinct(t *testing.T) {
	for _, preserve := range []bool{false, true} {
		m, b := mpzPublicationOwner(t)
		var out Int
		if e := m.SetInt64(&out, -(1 << 40)); e != nil {
			t.Fatal(e)
		}
		old := out.cell
		before := mpzPublicationSnapshot(m, b)
		b.FailAfter(0)
		var e error
		if preserve {
			e = m.ensureOwnedCapacity(&out, 33)
		} else {
			e = m.allocateIfNeeded(&out, 33)
		}
		mpzPublicationAudit(t, m, &out)
		if !numericPhysicalReservationRefusal(e) {
			t.Fatal(preserve, e)
		}
		if preserve {
			if out.cell != old || out.val != -1 || mpzPublicationSnapshot(m, b) != before {
				t.Fatal("preserving capacity must retain old ownership", out)
			}
		} else {
			if out.cell != m.failedRef || out.val != 1 {
				t.Fatal("destructive allocation publishes sign", out)
			}
			mpzPublicationExpectFreed(t, m, b, old, before)
		}
		if e = m.Del(&out); e != nil {
			t.Fatal(e)
		}
	}
}

func TestIntegerAllocateIfNeededSmallAndReuseBranches(t *testing.T) {
	m, b := mpzPublicationOwner(t)
	out := NewInt(-19)
	b.FailAfter(0)
	e := m.allocateIfNeeded(&out, 33)
	mpzPublicationAudit(t, m, &out)
	if !numericPhysicalReservationRefusal(e) || out.val != 1 || out.cell != m.failedNullRef {
		t.Fatal("small sign before allocation", out, e)
	}
	if e = m.Reset(&out); e != nil {
		t.Fatal(e)
	}
	b.FailAfter(-1)
	if e = m.SetInt64(&out, -(1 << 40)); e != nil {
		t.Fatal(e)
	}
	old := out.cell
	before := mpzPublicationSnapshot(m, b)
	b.FailAfter(0)
	if e = m.allocateIfNeeded(&out, 1); e != nil {
		t.Fatal(e)
	}
	mpzPublicationAudit(t, m, &out)
	if out.cell != old || out.val != -1 || m.cell(old).capacity != 6 || mpzPublicationSnapshot(m, b) != before {
		t.Fatal("sufficient source cell unchanged", out)
	}
	if e = m.Del(&out); e != nil {
		t.Fatal(e)
	}
}

func TestIntegerBigScalarSetterAllocationBeforeSign(t *testing.T) {
	// These ordinary refusal branches were source-correct before this packet;
	// this is independent confirmation, not a claimed behavioral RED.
	for _, unsigned := range []bool{false, true} {
		m, b := mpzPublicationDirectOwner(t)
		out := NewInt(-17)
		before := mpzPublicationSnapshot(m, b)
		b.FailAfter(0)
		var e error
		if unsigned {
			e = m.SetUint64(&out, ^uint64(0))
		} else {
			e = m.SetInt64(&out, -(1 << 40))
		}
		mpzPublicationAudit(t, m, &out)
		if !numericPhysicalReservationRefusal(e) || out != NewInt(-17) || mpzPublicationSnapshot(m, b) != before {
			t.Fatal("scalar allocation precedes sign", out, e)
		}
	}
}

func TestIntegerBigScalarSetterSourceCapacityAssertion(t *testing.T) {
	for _, unsigned := range []bool{false, true} {
		m, b := mpzPublicationOwner(t)
		var out Int
		if e := m.SetInt64(&out, -(1 << 40)); e != nil {
			t.Fatal(e)
		}
		ref := out.cell
		original := m.cell(ref)
		altered := original
		altered.capacity = 5 // actual payload still6: observe only the source assertion.
		m.store(ref, altered)
		before := mpzPublicationSnapshot(m, b)
		b.FailAfter(0)
		var e error
		if unsigned {
			e = m.SetUint64(&out, ^uint64(0))
		} else {
			e = m.SetInt64(&out, 1<<40)
		}
		mpzPublicationAudit(t, m, &out)
		if modularSourceDebug {
			if !errors.Is(e, ErrIntegerAssertion) || out.val != -1 || m.digit(ref, 0) != 0 || m.digit(ref, 1) != 256 {
				t.Fatal("capacity assertion precedes sign/digits", unsigned, out, e)
			}
		} else if e != nil || out.val != 1 {
			t.Fatal("release finite actual payload write", unsigned, out, e)
		}
		if out.cell != ref || mpzPublicationSnapshot(m, b) != before {
			t.Fatal("existing cell reuse", out)
		}
		m.store(ref, original)
		if e = m.Del(&out); e != nil {
			t.Fatal(e)
		}
	}
}

func TestIntegerScalarSetterBoundariesAndReuse(t *testing.T) {
	m, b := mpzPublicationOwner(t)
	var out Int
	signed := []int64{-9223372036854775808, -2147483649, -2147483648, -1, 0, 2147483647, 2147483648, 9223372036854775807}
	for _, v := range signed {
		if e := m.SetInt64(&out, v); e != nil {
			t.Fatal(e)
		}
		assertIntegerPowerOracle(t, m, &out, big.NewInt(v))
		mpzPublicationAudit(t, m, &out)
	}
	for _, v := range []uint64{0, 2147483646, 2147483647, 2147483648, 4294967295, 9223372036854775808, ^uint64(0)} {
		if e := m.SetUint64(&out, v); e != nil {
			t.Fatal(e)
		}
		assertIntegerPowerOracle(t, m, &out, new(big.Int).SetUint64(v))
		if out.IsSmall() != (v < 2147483647) {
			t.Fatal("strict unsigned64 threshold", v)
		}
	}
	if e := m.SetInt64(&out, 1<<40); e != nil {
		t.Fatal(e)
	}
	ref := out.cell
	b.FailAfter(0)
	for _, v := range []int64{-(1 << 40), 9223372036854775807, -9223372036854775808} {
		if e := m.SetInt64(&out, v); e != nil {
			t.Fatal(e)
		}
		if out.cell != ref {
			t.Fatal("reuse failed", v)
		}
		assertIntegerPowerOracle(t, m, &out, big.NewInt(v))
	}
	mpzPublicationAudit(t, m, &out)
	if e := m.Del(&out); e != nil {
		t.Fatal(e)
	}
}

func TestIntegerNormalizeOwnedSourceBranches(t *testing.T) {
	for _, tc := range []struct {
		digits []uint32
		sign   int32
		want   string
		small  bool
	}{
		{[]uint32{0, 0, 0}, 1, "0", true},
		{[]uint32{2147483647, 0, 0}, 1, "2147483647", true},
		{[]uint32{2147483647, 0, 0}, -1, "-2147483647", true},
		{[]uint32{2147483648, 0, 0}, -1, "-2147483648", false},
		{[]uint32{7, 1, 0}, -1, "-4294967303", false},
	} {
		m, b := mpzPublicationOwner(t)
		ref, e := m.allocate(6, uint32(len(tc.digits)))
		if e != nil {
			t.Fatal(e)
		}
		out := Int{val: tc.sign, cell: ref}
		// mpz.h:124–128 initializes capacity only. The fixture is constructing
		// a numeral, so publish its pointer and source m_size before digits.
		m.publishCell(ref)
		m.resizeCell(ref, uint32(len(tc.digits)))
		for i, d := range tc.digits {
			m.writeDigit(ref, uint32(i), d)
		}
		b.FailAfter(0)
		if e = m.normalizeOwned(&out); e != nil {
			t.Fatal(e)
		}
		mpzPublicationAudit(t, m, &out)
		want, _ := new(big.Int).SetString(tc.want, 10)
		assertIntegerPowerOracle(t, m, &out, want)
		if out.IsSmall() != tc.small || (!tc.small && out.cell != ref) {
			t.Fatal("normalize representation", tc.want, out)
		}
		if e = m.Del(&out); e != nil {
			t.Fatal(e)
		}
	}
}

func TestIntegerBaseTwoPowerExactValuesAndAliases(t *testing.T) {
	for _, p := range []uint32{0, 1, 30, 31, 32, 63, 64, 191, 192, 1024} {
		for _, alias := range []bool{false, true} {
			m, _ := mpzPublicationOwner(t)
			two := NewInt(2)
			var out Int
			if e := m.SetInt64(&out, -(1 << 40)); e != nil {
				t.Fatal(e)
			}
			target := &out
			if alias {
				target = &two
			}
			if e := m.Power(&two, p, target); e != nil {
				t.Fatal(p, alias, e)
			}
			mpzPublicationAudit(t, m, &two, &out)
			assertIntegerPowerOracle(t, m, target, new(big.Int).Lsh(big.NewInt(1), uint(p)))
			if e := m.Del(&two); e != nil {
				t.Fatal(e)
			}
			if e := m.Del(&out); e != nil {
				t.Fatal(e)
			}
		}
	}
}

func TestIntegerBaseTwoPowerReservationSweepAndRetainedCapacity(t *testing.T) {
	refused, completed := false, false
	for prefix := int64(0); prefix < 32; prefix++ {
		m, b := mpzPublicationOwner(t)
		var out Int
		if e := m.SetInt64(&out, -(1 << 40)); e != nil {
			t.Fatal(e)
		}
		old := out.cell
		two := NewInt(2)
		b.FailAfter(prefix)
		e := m.Power(&two, 1024, &out)
		mpzPublicationAudit(t, m, &out)
		if e != nil {
			refused = true
			if !numericPhysicalReservationRefusal(e) || out.cell != m.failedRef || out.val != 1 || validNumericRef(m.cells, old) {
				t.Fatal("allocation prefix", prefix, out, e)
			}
		} else {
			completed = true
			assertIntegerPowerOracle(t, m, &out, new(big.Int).Lsh(big.NewInt(1), 1024))
			if m.cell(out.cell).capacity != 33 {
				t.Fatal("requested source capacity", m.cell(out.cell).capacity)
			}
		}
		if e = m.Del(&out); e != nil {
			t.Fatal(e)
		}
		if completed {
			break
		}
	}
	if !refused || !completed {
		t.Fatal("reservation sweep failed to cover refusal and success", refused, completed)
	}
	m, b := mpzPublicationOwner(t)
	var out Int
	two := NewInt(2)
	if e := m.Power(&two, 1024, &out); e != nil {
		t.Fatal(e)
	}
	ref := out.cell
	before := mpzPublicationSnapshot(m, b)
	b.FailAfter(0)
	for _, p := range []uint32{191, 192, 1024} {
		if e := m.Power(&two, p, &out); e != nil {
			t.Fatal(e)
		}
		if out.cell != ref || m.cell(ref).capacity != 33 {
			t.Fatal("retained sufficient source capacity")
		}
		assertIntegerPowerOracle(t, m, &out, new(big.Int).Lsh(big.NewInt(1), uint(p)))
	}
	mpzPublicationAudit(t, m, &out)
	if mpzPublicationSnapshot(m, b) != before {
		t.Fatal("unexpected warmed reservation")
	}
	var e error
	allocs := testing.AllocsPerRun(100, func() { e = m.Power(&two, 192, &out) })
	if e != nil || allocs != 0 {
		t.Fatal("warm source fast path", e, allocs)
	}
	mpzPublicationAudit(t, m, &out)
	if e = m.Del(&out); e != nil {
		t.Fatal(e)
	}
}
