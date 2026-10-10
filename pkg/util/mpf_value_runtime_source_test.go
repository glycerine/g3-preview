// Source: src/util/mpf.cpp:316-333; src/util/mpz.cpp:big_set.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatPhysicalRowPublishesAtActualCopyCallback(t *testing.T) {
	for _, copyValue := range []bool{false, true} {
		b := arena.NewBudget(1 << 28)
		var r *umemory.Runtime
		var m *ArbitraryFloatManager
		var row umemory.Block
		active, calls := false, 0
		var seen [6]uint32
		var closeRefusal, sourceRefusal error
		r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
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
		m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
		serviceMust(t, e)
		row, e = r.Allocate(32)
		serviceMust(t, e)
		for i := uint64(0); i < 32; i += 4 {
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
		if !errors.Is(body, umemory.Failure{Kind: umemory.FailureMemout}) || calls == 0 {
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
		integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{row, padding}, &source, &input.significand, &out.significand)
		var maps uint64
		if want := m.header + m.ValueCopyHistoryBytes() + m.rational.HeaderBytes + integerActualOwnedBytes(m.integer) + maps; b.Used() != want {
			t.Fatal("immediate cold/physical graph", b.Used(), want)
		}
		serviceMust(t, r.SetMaxSize(0))
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

func TestArbitraryFloatFreshInlineRowWhileOlderCopyRetainsRoots(t *testing.T) {
	for _, copyValue := range []bool{false, true} {
		b := arena.NewBudget(1 << 28)
		var r *umemory.Runtime
		var m *ArbitraryFloatManager
		var row umemory.Block
		active, calls := false, 0
		var seen [6]uint32
		var closeRefusal, sourceRefusal error
		r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
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
		m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
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
		if !errors.Is(body, umemory.Failure{Kind: umemory.FailureMemout}) || calls == 0 {
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
		integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{row, padding}, &source, &input.significand, &out.significand)
		var maps uint64
		if want := m.header + m.ValueCopyHistoryBytes() + m.rational.HeaderBytes + integerActualOwnedBytes(m.integer) + maps; b.Used() != want {
			t.Fatal("immediate cold/physical graph", b.Used(), want)
		}
		serviceMust(t, r.SetMaxSize(0))
		retained := m.valueCopy
		older, read := m.CaptureRuntimeValue(row, 0)
		serviceMust(t, read)
		inline := NewInt(17)
		var next error
		if copyValue {
			fresh := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, false), exponent: 59, significand: inline}
			next = m.CopyToRuntime(row, 32, &fresh)
		} else {
			next = m.SetComponentsToRuntime(row, 32, 8, 24, false, 59, &inline)
		}
		if next != nil {
			t.Error("fresh inline SET refused because an older row retains allocation roots", copyValue, next)
		} else {
			fresh, read := m.CaptureRuntimeValue(row, 32)
			serviceMust(t, read)
			if fresh.Exponent() != 59 || fresh.significand != inline {
				t.Error("fresh inline row", fresh)
			}
		}
		unchanged, read := m.CaptureRuntimeValue(row, 0)
		serviceMust(t, read)
		if unchanged != older || m.valueCopy != retained || m.valueCopyAcknowledged || !m.integer.MatchesCopyAllocationInvocation(&retained) {
			t.Error("inline SET changed the older row or its exact retained child", unchanged, older, m.valueCopy, retained)
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
