// Source: src/util/mpf.cpp:316-333; src/util/mpz.cpp:big_set.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatValueCopyTypedOwningOrder(t *testing.T) {
	for _, copyValue := range []bool{false, true} {
		b := arena.NewBudget(1 << 28)
		var r *Runtime
		var m *ArbitraryFloatManager
		var row Block
		active, calls := false, 0
		var seen [6]uint32
		var closeRefusal, sourceRefusal, falseRefusal, trueRefusal error
		var receipt ArbitraryFloatValueCopyReceipt
		r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f Failure) error {
			if active {
				calls++
				for i := range seen {
					var read error
					seen[i], read = r.ReadUint32(row, uint64(i)*4)
					if read != nil {
						t.Fatal(read)
					}
				}
				closeRefusal = m.Close()
				sourceRefusal = m.SourceFinalize()
				falseRefusal = m.CleanupValueCopy(&receipt, false)
				trueRefusal = m.CleanupValueCopy(&receipt, true)
			}
			return f
		}})
		serviceMust(t, e)
		m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
		serviceMust(t, e)
		row, e = r.Allocate(64)
		serviceMust(t, e)
		for i := uint64(0); i < 64; i += 4 {
			serviceMust(t, r.WriteUint32(row, i, 0))
		}
		seven := NewInt(7)
		serviceMust(t, m.SetComponentsToRuntime(row, 0, 8, 24, false, 99, &seven))
		var source Int
		digits := make([]uint32, 1024)
		for i := range digits {
			digits[i] = 0xffffffff
		}
		digits[len(digits)-1] = 0x7fffffff
		serviceMust(t, m.integer.SetLimbs(&source, digits))
		var input ArbitraryFloat
		serviceMust(t, m.SetComponents(&input, 11, 40000, true, -42, &source))
		padding, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
		serviceMust(t, e)
		serviceMust(t, r.SetMaxSize(1))
		active = true
		var body error
		if copyValue {
			body = m.CopyToRuntimeWithReceipt(row, 0, &input, &receipt)
		} else {
			body = m.SetComponentsToRuntimeWithReceipt(row, 0, 11, 40000, true, -42, &source, &receipt)
		}
		active = false
		if !errors.Is(body, Failure{Kind: FailureMemout}) || calls == 0 {
			t.Fatal(copyValue, body, calls)
		}
		wantExponent := int64(99)
		if copyValue {
			wantExponent = -42
		}
		observed := CaptureArbitraryFloatWords(seen)
		if observed.EBits() != 11 || observed.SBits() != 40000 || !observed.Sign() || observed.Exponent() != wantExponent || observed.significand.Val != 1 {
			t.Fatal("source fields were delayed past actual allocation callback", copyValue, observed)
		}
		if falseRefusal != ErrNumericPending || trueRefusal != ErrNumericPending || closeRefusal != ErrNumericPending || sourceRefusal != ErrNumericPending || m.closed || m.sourceEnded || m.ownCause != nil || m.sourceCause != nil {
			t.Fatal("executing owner was changed", closeRefusal, sourceRefusal, m)
		}
		out, e := m.CaptureRuntimeValue(row, 0)
		serviceMust(t, e)
		if out.Exponent() != wantExponent || !m.integer.MatchesCopyAllocationInvocation(&m.valueCopy) {
			t.Fatal("actual failed row/child tuple", copyValue, out, m.valueCopy)
		}
		integerPhysicalAuditWithRaw(t, m.integer, []Block{row, padding}, &source, &input.significand, &out.significand)
		var maps uint64
		if want := m.header + m.ValueCopyHistoryBytes() + m.rational.HeaderBytes + integerActualOwnedBytes(m.integer) + maps; b.Used() != want {
			t.Fatal("immediate cold/physical graph", b.Used(), want)
		}
		used := b.Used()
		child := m.valueCopy
		serviceMust(t, m.CleanupValueCopy(&receipt, false))
		if b.Used() != used || m.valueCopy != child || !m.integer.MatchesCopyAllocationInvocation(&child) {
			t.Fatal("completed-child ACK aborted the real unjoined child")
		}
		serviceMust(t, r.SetMaxSize(0))
		serviceMust(t, m.CleanupRuntimeValue(row, 0))
		serviceMust(t, m.CleanupValueCopy(&receipt, true))
		if !receipt.frame.Retired || receipt.frame.bodyCause != body || receipt.frame.header == 0 {
			t.Fatal("retired call lost charged diagnostic tuple or original cause")
		}
		retryPadding, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
		serviceMust(t, e)
		var newer ArbitraryFloatValueCopyReceipt
		serviceMust(t, r.SetMaxSize(1))
		active = true
		body2 := m.CopyToRuntimeWithReceipt(row, 32, &input, &newer)
		active = false
		if !errors.Is(body2, Failure{Kind: FailureMemout}) || newer.Child.invocation <= receipt.Child.invocation || !m.integer.MatchesCopyAllocationInvocation(&newer.Child) {
			t.Fatal("second actual failed acquisition did not issue a fresh frame", body2, newer, receipt)
		}
		fresh, read := m.CaptureRuntimeValue(row, 32)
		serviceMust(t, read)
		integerPhysicalAuditWithRaw(t, m.integer, []Block{row, padding, retryPadding}, &source, &input.significand, &fresh.significand)
		used = b.Used()
		serviceMust(t, m.CleanupValueCopy(&receipt, false))
		serviceMust(t, m.CleanupValueCopy(&receipt, true))
		spliced := receipt
		spliced.Child = newer.Child
		if e := m.CleanupValueCopy(&spliced, true); e != ErrRuntimeOwner {
			t.Fatal("spliced issued tuple accepted", e)
		}
		stale := newer
		stale.invocation--
		if e := m.CleanupValueCopy(&stale, true); e != ErrRuntimeOwner {
			t.Fatal("stale caller nonce accepted", e)
		}
		copied := *m
		if e := copied.CleanupValueCopy(&newer, true); e != ErrRuntimeOwner {
			t.Fatal("copied owner accepted", e)
		}
		foreignBudget := arena.NewBudget(1 << 26)
		foreign, e := NewArbitraryFloatManager(serviceConfig(foreignBudget).Integers)
		serviceMust(t, e)
		if e := foreign.CleanupValueCopy(&newer, true); e != ErrRuntimeOwner {
			t.Fatal("foreign owner accepted", e)
		}
		serviceMust(t, foreign.Close())
		if foreignBudget.Used() != 0 || b.Used() != used || !m.integer.MatchesCopyAllocationInvocation(&newer.Child) || newer.frame.Retired {
			t.Fatal("retired or forged receipt changed the new live child", b.Used(), used)
		}
		serviceMust(t, m.CleanupValueCopy(&newer, false))
		serviceMust(t, r.SetMaxSize(0))
		serviceMust(t, m.CleanupRuntimeValue(row, 32))
		serviceMust(t, m.CleanupValueCopy(&newer, true))
		serviceMust(t, m.Del(&input))
		serviceMust(t, m.integer.Del(&source))
		serviceMust(t, r.Deallocate(&row))
		serviceMust(t, r.Deallocate(&padding))
		serviceMust(t, r.Deallocate(&retryPadding))
		serviceMust(t, m.CleanupValueCopy(&receipt, true))
		historyBytes := m.ValueCopyHistoryBytes()
		serviceMust(t, m.SourceFinalize())
		if historyBytes == 0 || m.ValueCopyHistoryBytes() != historyBytes || receipt.frame.header == 0 || receipt.frame.bodyCause != body || !receipt.frame.Retired || newer.frame.header == 0 {
			t.Fatal("source DT erased owning-context diagnostic history")
		}
		serviceMust(t, m.Close())
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(copyValue, b.Used())
		}
	}
}

func TestArbitraryFloatInlineCopyHasNoHistoryReservation(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	row, e := m.memory.Allocate(32)
	serviceMust(t, e)
	for i := uint64(0); i < 32; i += 4 {
		serviceMust(t, m.memory.WriteUint32(row, i, 0))
	}
	inline := NewInt(17)
	var a ArbitraryFloat
	var receipt ArbitraryFloatValueCopyReceipt
	before := b.Used()
	b.FailAfter(0)
	serviceMust(t, m.SetComponents(&a, 8, 24, false, 59, &inline))
	serviceMust(t, m.CopyToRuntimeWithReceipt(row, 0, &a, &receipt))
	serviceMust(t, m.SetComponentsToRuntimeWithReceipt(row, 0, 11, 53, true, -42, &inline, &receipt))
	if receipt != (ArbitraryFloatValueCopyReceipt{}) || m.ValueCopyHistoryBytes() != 0 || b.Used() != before {
		t.Fatal("inline path acquired caller metadata", receipt, m.ValueCopyHistoryBytes(), b.Used(), before)
	}
	got, e := m.CaptureRuntimeValue(row, 0)
	serviceMust(t, e)
	if got.Sign() != true || got.Exponent() != -42 || got.significand != inline {
		t.Fatal(got)
	}
	b.FailAfter(-1)
	serviceMust(t, m.CleanupValueCopy(&receipt, false))
	serviceMust(t, m.CleanupValueCopy(&receipt, true))
	serviceMust(t, m.DelRuntimeValue(row, 0))
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.memory.Deallocate(&row))
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
