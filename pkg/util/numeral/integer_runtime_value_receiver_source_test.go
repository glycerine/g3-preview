package numeral

// Source: src/util/mpz.cpp:big_set; src/util/mpf.cpp:327-333.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// Returning Go transport is a concrete adapter profile. It observes the real
// vector row at the reached lower allocation; it is not a native C++ callback.
func TestIntegerPhysicalCopyPublishesSignBeforeActualAllocationCallback(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var r *umemory.Runtime
	var row umemory.Block
	active, calls := false, 0
	var seen uint32
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if active {
			calls++
			var read error
			seen, read = r.ReadUint32(row, 0)
			if read != nil {
				t.Fatal(read)
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e := NewIntegerManager(IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	row, e = r.Allocate(16)
	serviceMust(t, e)
	out := NewInt(7)
	serviceMust(t, r.WriteUint32(row, 0, 7))
	serviceMust(t, r.WriteUint32(row, 4, 0))
	serviceMust(t, r.WriteUint32(row, 8, 0))
	source := NewInt(0)
	serviceMust(t, m.Copy(&source, &m.Two64))
	serviceMust(t, m.Neg(&source))
	padding, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	receiver := integerRuntimeValueReceiver{runtime: r, block: row, offset: 0}
	var receipt integerCopyAllocationReceipt
	active = true
	call := m.copyFromOwnerWithPhysicalReceipt(&out, m, &source, &receipt, &receiver)
	active = false
	if call == nil || calls == 0 || int32(seen) != -1 || !m.MatchesCopyAllocationInvocation(&receipt) {
		t.Fatal("actual sign before retained lower allocation", call, calls, seen, receipt)
	}
	val, e := r.ReadUint32(row, 0)
	serviceMust(t, e)
	low, e := r.ReadUint32(row, 4)
	serviceMust(t, e)
	high, e := r.ReadUint32(row, 8)
	serviceMust(t, e)
	if int32(val) != out.Val || NumericCellRef(uint64(low)|uint64(high)<<32) != out.Cell {
		t.Fatal("exact failed receiver prefix", out, val, low, high)
	}
	if !errors.Is(call, umemory.Failure{Kind: umemory.FailureMemout}) {
		t.Fatal("original source failure", call)
	}
	integerPhysicalAuditWithRaw(t, m, []umemory.Block{row, padding}, &out, &source)
	if b.Used() != integerActualOwnedBytes(m) {
		t.Fatal("immediate physical/cold graph", b.Used())
	}
	serviceMust(t, r.SetMaxSize(0))
	serviceMust(t, m.CleanupCopyAllocationInvocation(&receipt))
	serviceMust(t, m.Del(&out))
	serviceMust(t, m.Del(&source))
	serviceMust(t, r.Deallocate(&row))
	serviceMust(t, r.Deallocate(&padding))
	serviceMust(t, m.CloseChecked())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("exact caller/manager end", b.Used())
	}
}

func TestIntegerPhysicalCopyReceiverIdentityAndExtentGuards(t *testing.T) {
	m, b := integerManager(t)
	row, e := m.Memory.Allocate(16)
	serviceMust(t, e)
	out, source := NewInt(7), NewInt(11)
	serviceMust(t, m.Memory.WriteUint32(row, 0, 7))
	serviceMust(t, m.Memory.WriteUint32(row, 4, 0))
	serviceMust(t, m.Memory.WriteUint32(row, 8, 0))
	var receipt integerCopyAllocationReceipt
	beforeUsed, beforeState := b.Used(), m.Memory.State()
	bad := integerRuntimeValueReceiver{runtime: m.Memory, block: row, offset: ^uint64(0)}
	if e = m.copyFromOwnerWithPhysicalReceipt(&out, m, &source, &receipt, &bad); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("reached address overflow guard", e)
	}
	if out != NewInt(7) || receipt.invocation != 0 || b.Used() != beforeUsed || m.Memory.State() != beforeState {
		t.Fatal("guard acquired or mutated receiver")
	}
	good := integerRuntimeValueReceiver{runtime: m.Memory, block: row}
	mismatched := NewInt(13)
	if e = m.copyFromOwnerWithPhysicalReceipt(&mismatched, m, &source, &receipt, &good); !errors.Is(e, umemory.ErrRuntimeOwner) {
		t.Fatal("actual captured row mismatch", e)
	}
	serviceMust(t, m.copyFromOwnerWithPhysicalReceipt(&out, m, &source, &receipt, &good))
	word, e := m.Memory.ReadUint32(row, 0)
	serviceMust(t, e)
	if out != source || word != 11 || receipt.invocation != 0 || b.Used() != beforeUsed || m.Memory.State() != beforeState {
		t.Fatal("inline SET actual publication without acquisition", out, word)
	}
	old := row
	serviceMust(t, m.Memory.Deallocate(&row))
	replacement, e := m.Memory.Allocate(16)
	serviceMust(t, e)
	serviceMust(t, m.Memory.WriteUint32(replacement, 0, 19))
	serviceMust(t, m.Memory.WriteUint32(replacement, 4, 0))
	serviceMust(t, m.Memory.WriteUint32(replacement, 8, 0))
	good.block = old
	out = NewInt(11)
	if e = m.copyFromOwnerWithPhysicalReceipt(&out, m, &source, &receipt, &good); e == nil {
		t.Fatal("ended row accepted")
	}
	word, e = m.Memory.ReadUint32(replacement, 0)
	serviceMust(t, e)
	if word != 19 || out != NewInt(11) || receipt.invocation != 0 {
		t.Fatal("stale borrowed row mutated replacement")
	}
	integerPhysicalAuditWithRaw(t, m, []umemory.Block{replacement}, &out, &source)
	serviceMust(t, m.Memory.Deallocate(&replacement))
}
