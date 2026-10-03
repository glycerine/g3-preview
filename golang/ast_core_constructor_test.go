// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestASTConstructorRejectsClosedBorrowedNumeralOwnerBeforeGrowth(t *testing.T) {
	s, q, b := astTestOwners(t)
	astTestMust(t, q.Close())
	before := b.Used()
	m, e := NewASTManager(ASTManagerConfig{Symbols: s, Numerals: q, Budget: b})
	if m != nil {
		m.Close()
	}
	if !errors.Is(e, arena.ErrClosed) || m != nil || b.Used() != before {
		t.Fatal("closed borrowed owner", m != nil, e, b.Used(), before)
	}
}

func TestASTSymbolCodecRejectsClosedSpellingOwner(t *testing.T) {
	m := astTestManager(t)
	s := astTestSymbol(t, m, "closed codec owner")
	p, e := m.allocator.Allocate(astSymbolBytes)
	astTestMust(t, e)
	astTestMust(t, m.writeSymbol(p, 0, s))
	astTestMust(t, m.symbols.Close())
	if _, e = m.readSymbol(p, 0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed read", e)
	}
	if e = m.writeSymbol(p, 0, s); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed write", e)
	}
	astTestMust(t, m.allocator.Deallocate(astSymbolBytes, p))
}
