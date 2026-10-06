// Licensed under the MIT license; see LICENSE.txt.
// Source: src/util/mpz.cpp:1385-1410; src/util/mpn.cpp:371-399.
// Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func TestNumeralInsertionReceiptUnusedSmallBranchAndForeignTupleGuard(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	q, e := NewRationalService(RationalServiceConfig{Integers: IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	if e != nil {
		t.Fatal(e)
	}
	defer q.Close()
	v, e := q.NewValue()
	if e != nil {
		t.Fatal(e)
	}
	defer q.Del(&v)
	var receipt NumeralInsertionReceipt
	out := numeralReceiptDiscardStream{}
	if e = q.InsertIntoTracked(out, &v, &receipt); e != nil {
		t.Fatal(e)
	}
	if receipt.Active() {
		t.Fatal("small source branch created lexical acquisition")
	}
	if e = q.CompleteNumeralInsertion(&receipt, false); e != arena.ErrInvalidSlot {
		t.Fatal("empty tuple grants authority", e)
	}
	if e = q.SetInt64(&v, 1<<40); e != nil {
		t.Fatal(e)
	}
	if e = q.InsertIntoTracked(out, &v, &receipt); e != nil {
		t.Fatal(e)
	}
	if receipt.Active() {
		t.Fatal("successful published source insertion retained completed metadata")
	}
	if e = q.CompleteNumeralInsertion(&receipt, true); e != arena.ErrInvalidSlot {
		t.Fatal("stale successful receipt grants ownership", e)
	}
}

type numeralReceiptDiscardStream struct{}

func (numeralReceiptDiscardStream) AppendInt32(int32) error    { return nil }
func (numeralReceiptDiscardStream) AppendCString(string) error { return nil }
func (numeralReceiptDiscardStream) AppendNumeralCString(v NumeralCStringView) error {
	for i := uint64(0); ; i++ {
		c, e := v.Byte(i)
		if e != nil {
			return e
		}
		if c == 0 {
			return nil
		}
	}
}

// Exact guarded primitive receipts share the actual InsertInto child engine.
// Wrong copies are passed directly; the parent's acquired tuple is untouched.
func TestNumeralInsertionReceiptActualCharacterConstructorPrefixesAndTupleGuards(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		succeeded := false
		for ordinal := int64(0); ordinal < 8; ordinal++ {
			b := arena.NewBudget(1 << 24)
			rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: bits})
			serviceMust(t, e)
			cfg := serviceConfig(b)
			cfg.Integers.Memory = rt
			cfg.Integers.Profile = memory.SmallAllocatorProfile{PointerBits: bits, Debug: true}
			q, e := NewRationalService(cfg)
			serviceMust(t, e)
			foreignConfig := serviceConfig(b)
			foreign, e := NewRationalService(foreignConfig)
			serviceMust(t, e)
			n := q.manager.z.natural
			var receipt NumeralInsertionReceipt
			beforeUsed, beforeRuntime := b.Used(), rt.Stats().TotalBytes
			b.FailAfter(ordinal)
			body := n.beginIntegerCharactersTracked(2048, q, &receipt)
			if !receipt.active || receipt.service != q || receipt.character != &n.charFormatting || receipt.invocation == 0 || !receipt.characterOwned {
				t.Fatal("actual constructor entry not stamped", receipt, body)
			}
			original := receipt
			raw, pending := n.charFormatting.buffer.raw, n.charFormatting.buffer.pending
			rtBefore := rt.Stats()
			bad := receipt
			bad.invocation++
			if e = q.CompleteNumeralInsertion(&bad, false); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("wrong child invocation", e)
			}
			if e = foreign.CompleteNumeralInsertion(&receipt, false); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("foreign service", e)
			}
			bad = receipt
			bad.character = nil
			if e = q.CompleteNumeralInsertion(&bad, true); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("foreign frame", e)
			}
			if !reflect.DeepEqual(original, receipt) || raw != n.charFormatting.buffer.raw || pending != n.charFormatting.buffer.pending || rt.Stats() != rtBefore {
				t.Fatal("rejected tuple changed actual acquisitions")
			}
			if body != nil {
				var failure memory.Failure
				if !errors.As(body, &failure) || failure.Kind != memory.FailureMemout {
					t.Fatal("actual Runtime reservation", bits, ordinal, body)
				}
			}
			integerPhysicalAudit(t, q.manager.z)
			if b.Used() != beforeUsed+rt.Stats().TotalBytes-beforeRuntime {
				t.Fatal("immediate exact constructor cold graph", bits, ordinal, b.Used(), beforeUsed, rt.Stats())
			}
			// No injected fault is lifted until exact child and provider graph checks.
			if body == nil && raw == (memory.Block{}) {
				t.Fatal("completed dynamic char constructor has no actual block")
			}
			b.FailAfter(-1)
			serviceMust(t, q.CompleteNumeralInsertion(&receipt, false))
			if receipt.characterCompleted && n.charFormatting.completed {
				t.Fatal("false ACK failed to end completed char")
			}
			serviceMust(t, q.CompleteNumeralInsertion(&receipt, true))
			if receipt.active || !receipt.retired || receipt.service != original.service || receipt.character != original.character || receipt.invocation != original.invocation || receipt.bodyCause != original.bodyCause || receipt.sourceCause != original.sourceCause {
				t.Fatal("retirement erased tuple/history", receipt, original)
			}
			if e = q.CompleteNumeralInsertion(&receipt, true); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("double ACK", e)
			}
			serviceMust(t, foreign.Close())
			serviceMust(t, q.Close())
			serviceMust(t, rt.Close())
			if b.Used() != 0 {
				t.Fatal("named real owner cleanup", bits, ordinal, b.Used())
			}
			if body == nil {
				succeeded = true
				break
			}
		}
		if !succeeded {
			t.Fatal("actual constructor horizon", bits)
		}
	}
}
