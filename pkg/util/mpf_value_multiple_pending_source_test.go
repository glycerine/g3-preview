// Source: src/util/mpf.cpp:316-333; src/util/mpz.cpp:big_set.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatFreshAllocatingRowWhileOlderCopyRetainsRoots(t *testing.T) {
	for _, copyValue := range []bool{false, true} {
		b := arena.NewBudget(1 << 28)
		var r *Runtime
		var m *ArbitraryFloatManager
		var row Block
		active, calls := false, 0
		var seen [6]uint32
		var closeRefusal, sourceRefusal error
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
		serviceMust(t, m.integer.Copy(&source, &m.integer.Two64))
		var input ArbitraryFloat
		serviceMust(t, m.SetComponents(&input, 11, 113, true, -42, &source))
		padding, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
		serviceMust(t, e)
		serviceMust(t, r.SetMaxSize(1))
		active = true
		var body error
		if copyValue {
			body = m.CopyToRuntime(row, 0, &input)
		} else {
			body = m.SetComponentsToRuntime(row, 0, 11, 113, true, -42, &source)
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
		if observed.EBits() != 11 || observed.SBits() != 113 || !observed.Sign() || observed.Exponent() != wantExponent || observed.significand.Val != 1 {
			t.Fatal("source fields were delayed past actual allocation callback", copyValue, observed)
		}
		if closeRefusal != ErrNumericPending || sourceRefusal != ErrNumericPending || m.closed || m.sourceEnded || m.ownCause != nil || m.sourceCause != nil {
			t.Fatal("executing owner was changed", closeRefusal, sourceRefusal, m)
		}
		out, e := m.CaptureRuntimeValue(row, 0)
		serviceMust(t, e)
		if out.Exponent() != wantExponent || !m.integer.MatchesCopyAllocationInvocation(&m.valueCopy) {
			t.Fatal("actual failed row/child tuple", copyValue, out, m.valueCopy)
		}
		integerPhysicalAuditWithRaw(t, m.integer, []Block{row, padding}, &source, &input.significand, &out.significand)
		var maps uint64
		for i := range m.powers {
			maps += m.powers[i].StorageBytes()
		}
		if want := m.header + m.ValueCopyHistoryBytes() + m.rational.HeaderBytes + integerActualOwnedBytes(m.integer) + maps; b.Used() != want {
			t.Fatal("immediate cold/physical graph", b.Used(), want)
		}
		serviceMust(t, r.SetMaxSize(0))
		retained := m.valueCopy
		older, read := m.CaptureRuntimeValue(row, 0)
		serviceMust(t, read)
		olderCall := m.ValueCalls
		var next error
		if copyValue {
			next = m.CopyToRuntime(row, 32, &input)
		} else {
			next = m.SetComponentsToRuntime(row, 32, 11, 113, true, -42, &source)
		}
		if next != nil {
			t.Error("source-valid allocating SET refused because an older row retains allocation roots", copyValue, next)
		} else {
			fresh, read := m.CaptureRuntimeValue(row, 32)
			serviceMust(t, read)
			same, compare := m.integer.Compare(&fresh.significand, &input.significand)
			if compare != nil || same != 0 || fresh.Exponent() != -42 {
				t.Error("fresh allocating row", fresh, compare)
			}
		}
		unchanged, read := m.CaptureRuntimeValue(row, 0)
		serviceMust(t, read)
		if unchanged != older || olderCall.Retired || olderCall.Issued.Child != retained || !m.integer.MatchesCopyAllocationInvocation(&retained) {
			t.Error("allocating SET changed the older row or its exact retained child", unchanged, older, m.valueCopy, retained)
		}
		serviceMust(t, m.CleanupRuntimeValue(row, 32))
		serviceMust(t, m.CleanupRuntimeValue(row, 0))
		serviceMust(t, m.Del(&input))
		serviceMust(t, m.integer.Del(&source))
		serviceMust(t, r.Deallocate(&row))
		serviceMust(t, r.Deallocate(&padding))
		serviceMust(t, m.Close())
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(copyValue, b.Used())
		}
	}
}

func TestArbitraryFloatTwoFailedAllocatingRowsRetainIndependentRoots(t *testing.T) {
	for _, copyValue := range []bool{false, true} {
		b := arena.NewBudget(1 << 28)
		var r *Runtime
		var m *ArbitraryFloatManager
		var row Block
		active, calls := false, 0
		var seen [6]uint32
		var closeRefusal, sourceRefusal error
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
		serviceMust(t, m.SetComponents(&input, 11, 113, true, -42, &source))
		padding, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
		serviceMust(t, e)
		serviceMust(t, r.SetMaxSize(1))
		active = true
		var body error
		if copyValue {
			body = m.CopyToRuntime(row, 0, &input)
		} else {
			body = m.SetComponentsToRuntime(row, 0, 11, 113, true, -42, &source)
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
		if observed.EBits() != 11 || observed.SBits() != 113 || !observed.Sign() || observed.Exponent() != wantExponent || observed.significand.Val != 1 {
			t.Fatal("source fields were delayed past actual allocation callback", copyValue, observed)
		}
		if closeRefusal != ErrNumericPending || sourceRefusal != ErrNumericPending || m.closed || m.sourceEnded || m.ownCause != nil || m.sourceCause != nil {
			t.Fatal("executing owner was changed", closeRefusal, sourceRefusal, m)
		}
		out, e := m.CaptureRuntimeValue(row, 0)
		serviceMust(t, e)
		if out.Exponent() != wantExponent || !m.integer.MatchesCopyAllocationInvocation(&m.valueCopy) {
			t.Fatal("actual failed row/child tuple", copyValue, out, m.valueCopy)
		}
		integerPhysicalAuditWithRaw(t, m.integer, []Block{row, padding}, &source, &input.significand, &out.significand)
		var maps uint64
		for i := range m.powers {
			maps += m.powers[i].StorageBytes()
		}
		if want := m.header + m.ValueCopyHistoryBytes() + m.rational.HeaderBytes + integerActualOwnedBytes(m.integer) + maps; b.Used() != want {
			t.Fatal("immediate cold/physical graph", b.Used(), want)
		}
		serviceMust(t, r.SetMaxSize(0))
		retained := m.valueCopy
		older, read := m.CaptureRuntimeValue(row, 0)
		serviceMust(t, read)
		olderCall := m.ValueCalls
		retryPadding, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
		serviceMust(t, e)
		serviceMust(t, r.SetMaxSize(1))
		var newer ArbitraryFloatValueCopyReceipt
		active = true
		var next error
		if copyValue {
			next = m.CopyToRuntimeWithReceipt(row, 32, &input, &newer)
		} else {
			next = m.SetComponentsToRuntimeWithReceipt(row, 32, 11, 113, true, -42, &source, &newer)
		}
		active = false
		if !errors.Is(next, Failure{Kind: FailureMemout}) || newer.frame == olderCall || newer.Child.Frame == retained.Frame || !m.integer.MatchesCopyAllocationInvocation(&newer.Child) {
			t.Fatal("second actual failed allocation", next, newer)
		}
		fresh, read := m.CaptureRuntimeValue(row, 32)
		serviceMust(t, read)
		integerPhysicalAuditWithRaw(t, m.integer, []Block{row, padding, retryPadding}, &source, &input.significand, &older.significand, &fresh.significand)
		serviceMust(t, r.SetMaxSize(0))
		unchanged, read := m.CaptureRuntimeValue(row, 0)
		serviceMust(t, read)
		if unchanged != older || olderCall.Retired || olderCall.Issued.Child != retained || !m.integer.MatchesCopyAllocationInvocation(&retained) {
			t.Error("allocating SET changed the older row or its exact retained child", unchanged, older, m.valueCopy, retained)
		}
		serviceMust(t, m.CleanupRuntimeValue(row, 32))
		serviceMust(t, m.CleanupRuntimeValue(row, 0))
		used := b.Used()
		serviceMust(t, m.CleanupValueCopy(&olderCall.Issued, false))
		serviceMust(t, m.CleanupValueCopy(&newer, false))
		if b.Used() != used {
			t.Fatal("false ACK disposed active raw allocations")
		}
		newerBlock := newer.Child.Frame.Transferred.Block()
		serviceMust(t, m.CleanupValueCopy(&olderCall.Issued, true))
		if !olderCall.Retired || newer.frame.Retired || newer.Child.Frame.Transferred.Block() != newerBlock || !m.integer.MatchesCopyAllocationInvocation(&newer.Child) {
			t.Fatal("older owning cleanup changed younger frame")
		}
		serviceMust(t, m.CleanupValueCopy(&newer, true))
		serviceMust(t, m.Del(&input))
		serviceMust(t, m.integer.Del(&source))
		serviceMust(t, r.Deallocate(&row))
		serviceMust(t, r.Deallocate(&padding))
		serviceMust(t, r.Deallocate(&retryPadding))
		serviceMust(t, m.Close())
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(copyValue, b.Used())
		}
	}
}
