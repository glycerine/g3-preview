// Source: src/util/mpz.cpp:1385-1410; src/util/mpn.cpp:371-399.
package util

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
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
			rt, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: bits})
			serviceMust(t, e)
			cfg := serviceConfig(b)
			cfg.Integers.Memory = rt
			cfg.Integers.Profile = SmallAllocatorProfile{PointerBits: bits, Debug: true}
			q, e := NewRationalService(cfg)
			serviceMust(t, e)
			foreignConfig := serviceConfig(b)
			foreign, e := NewRationalService(foreignConfig)
			serviceMust(t, e)
			n := q.Manager.Z.Natural
			var receipt NumeralInsertionReceipt
			beforeUsed, beforeRuntime := b.Used(), rt.Stats().TotalBytes
			b.FailAfter(ordinal)
			body := n.beginIntegerCharactersTracked(2048, q, &receipt)
			if !receipt.active || receipt.service != q || receipt.Character != &n.CharFormatting || receipt.invocation == 0 || !receipt.characterOwned {
				t.Fatal("actual constructor entry not stamped", receipt, body)
			}
			original := receipt
			buffer := append([]byte(nil), n.CharFormatting.Buffer...)
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
			bad.Character = nil
			if e = q.CompleteNumeralInsertion(&bad, true); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("foreign frame", e)
			}
			if !reflect.DeepEqual(original, receipt) || !bytes.Equal(buffer, n.CharFormatting.Buffer) || rt.Stats() != rtBefore {
				t.Fatal("rejected tuple changed actual acquisitions")
			}
			if body != nil {
				var failure Failure
				if !errors.As(body, &failure) || failure.Kind != FailureMemout {
					t.Fatal("actual Runtime reservation", bits, ordinal, body)
				}
			}
			integerPhysicalAudit(t, q.Manager.Z)
			if b.Used() != beforeUsed+rt.Stats().TotalBytes-beforeRuntime {
				t.Fatal("immediate exact constructor cold graph", bits, ordinal, b.Used(), beforeUsed, rt.Stats())
			}
			// No injected fault is lifted until exact child and provider graph checks.

			b.FailAfter(-1)
			serviceMust(t, q.CompleteNumeralInsertion(&receipt, false))
			if receipt.characterCompleted && n.CharFormatting.Completed {
				t.Fatal("false ACK failed to end completed char")
			}
			serviceMust(t, q.CompleteNumeralInsertion(&receipt, true))
			if receipt.active || !receipt.retired || receipt.service != original.service || receipt.Character != original.Character || receipt.invocation != original.invocation || receipt.bodyCause != original.bodyCause || receipt.sourceCause != original.sourceCause {
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
