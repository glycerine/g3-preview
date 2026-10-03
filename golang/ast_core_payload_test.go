// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
	"testing"
)

func TestASTSymbolPayloadCodecAndPhysicalCharges(t *testing.T) {
	m := astTestManager(t)
	text := astTestSymbol(t, m, "codec-spelling")
	marked, e := text.mark()
	astTestMust(t, e)
	numerical := NumericalSymbol(math.MaxUint32)
	markedNumber, e := numerical.mark()
	astTestMust(t, e)
	values := []Symbol{{}, NumericalSymbol(0), numerical, text, marked, markedNumber, DummySymbol()}
	before := m.allocator.Stats()
	p, e := m.allocator.Allocate(uint32(len(values)) * 24)
	astTestMust(t, e)
	for i, s := range values {
		astTestMust(t, m.writeSymbol(p, uint32(i), s))
		got, e := m.readSymbol(p, uint32(i))
		astTestMust(t, e)
		if got != s {
			t.Fatal("scalar token roundtrip", i, got, s)
		}
	}
	after := m.allocator.Stats()
	if after.RequestedBytes-before.RequestedBytes != uint64(len(values))*24 {
		t.Fatal("actual encoded payload length", before, after)
	}
	if reflect.TypeOf(text.generation).Size() != 4 || reflect.TypeOf(text.tag).Size() != 4 {
		t.Fatal("codec field widths changed")
	}
	if after.OwnerBytes+after.PayloadBytes+after.MetadataBytes+after.DirectoryBytes <= after.RequestedBytes {
		t.Fatal("allocator overhead was not charged", after)
	}
	// 2^29*24 wraps a uint32 byte offset to zero. This must fail before writing
	// element zero, using bounded live storage rather than a huge allocation.
	if e = m.writeSymbol(p, 1<<29, text); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("overflowed codec offset", e)
	}
	got, e := m.readSymbol(p, 0)
	astTestMust(t, e)
	if got != (Symbol{}) {
		t.Fatal("overflow mutated element zero")
	}
	if _, e = m.readSymbol(p, 1<<29); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("overflowed reader offset", e)
	}
	other, e := NewSymbolManager(SymbolConfig{})
	astTestMust(t, e)
	foreign, e := other.Intern("foreign")
	astTestMust(t, e)
	if e = m.writeSymbol(p, 0, foreign); !errors.Is(e, ErrSymbolOwner) {
		t.Fatal("foreign spelling accepted", e)
	}
	astTestMust(t, other.Close())
	astTestMust(t, m.allocator.Deallocate(uint32(len(values))*24, p))
	if _, e = m.readSymbol(p, 0); e == nil {
		t.Fatal("dead payload accepted")
	}
}

func TestASTPayloadArithmeticFiniteBoundaries(t *testing.T) {
	limit := uint64(math.MaxUint32)
	if host := uint64(^uint(0) >> 1); host < limit {
		limit = host
	}
	count := uint32(limit / 24)
	if n, e := astPayloadBytes(count, 24); e != nil || n != count*24 {
		t.Fatal(n, e)
	}
	if _, e := astPayloadBytes(count+1, 24); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if _, e := astPayloadOffset(1<<29, 24); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
}
