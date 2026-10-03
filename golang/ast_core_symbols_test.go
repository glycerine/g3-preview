// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"strings"
	"testing"
)

func TestASTModelValueNameInlineAndOverflowAccounting(t *testing.T) {
	m := astTestManager(t)
	name := astTestSymbol(t, m, "S")
	want, e := m.modelValueName(name, -2147483648)
	astTestMust(t, e)
	text, e := m.symbols.String(want)
	astTestMust(t, e)
	if text != "S!val!-2147483648" {
		t.Fatal(text)
	}
	before := m.budget.Used()
	m.budget.FailAfter(0)
	var got Symbol
	var failure error
	n := testing.AllocsPerRun(100, func() { got, failure = m.modelValueName(name, -2147483648) })
	m.budget.FailAfter(-1)
	if failure != nil || got != want || n != 0 || m.budget.Used() != before {
		t.Fatal("inline64 warmed model name", got, failure, n, m.budget.Used(), before)
	}
	long := astTestSymbol(t, m, strings.Repeat("q", 130))
	before = m.budget.Used()
	qBefore, e := m.numerals.Stats()
	astTestMust(t, e)
	// The first 64→128 overflow allocation succeeds; 128→256 fails.
	// Its lexical buffer must be destroyed before this operation returns.
	m.budget.FailAfter(1)
	_, e = m.modelValueName(long, 0)
	m.budget.FailAfter(-1)
	qAfter, statsError := m.numerals.Stats()
	if !errors.Is(e, arena.ErrInjectedFailure) || m.budget.Used() != before || statsError != nil || qAfter != qBefore {
		t.Fatal("overflow buffer failure cleanup", e, m.budget.Used(), before)
	}
}

func TestASTSymbolBytesInputOwnsSpellingAndWarmLongHit(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	s, e := NewSymbolManager(SymbolConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := s.Close(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Error("retained bytes", b.Used())
		}
	}()
	for _, input := range [][]byte{nil, []byte("foo\x00ignored"), {0x80, 0xff, 0, 'x'}} {
		value, e := s.InternBytes(input)
		if e != nil {
			t.Fatal(e)
		}
		plain := input
		for i, c := range input {
			if c == 0 {
				plain = input[:i]
				break
			}
		}
		same, e := s.Intern(string(plain))
		if e != nil || same != value {
			t.Fatal("source CString identity", input, e)
		}
		if len(plain) != 0 {
			want := string(plain)
			input[0] ^= 1
			got, e := s.String(value)
			if e != nil || got != want {
				t.Fatal("borrowed byte input escaped", got, want, e)
			}
		}
	}
	input := []byte("abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789")
	value, e := s.InternBytes(input)
	if e != nil {
		t.Fatal(e)
	}
	baseline := s.Stats()
	b.FailAfter(0)
	var got Symbol
	var failure error
	allocations := testing.AllocsPerRun(100, func() { got, failure = s.InternBytes(input) })
	if failure != nil || got != value || allocations != 0 {
		t.Fatal("warmed long byte query", got, value, failure, allocations)
	}
	if s.Stats() != baseline {
		t.Fatal("hit changed owner graph")
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = s.InternBytes(input); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed owner", e)
	}
}

func TestASTSymbolBytesGrowthBeforeDuplicateAndActualGraph(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	s, e := NewSymbolManager(SymbolConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	for i := 0; i < 7; i++ {
		if _, e = s.InternBytes([]byte{'a' + byte(i)}); e != nil {
			t.Fatal(e)
		}
	}
	before := s.Stats()
	b.FailAfter(0)
	if _, e = s.InternBytes([]byte("a")); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("must prepare before duplicate", e)
	}
	if s.Stats() != before {
		t.Fatal("failed table growth changed region")
	}
	b.FailAfter(-1)
	if _, e = s.InternBytes([]byte("a")); e != nil {
		t.Fatal(e)
	}
	state := s.Stats()
	d := state.Region.Descriptors
	physical := state.OwnerBytes + state.Table.OwnerBytes + state.Table.TableBytes + state.Table.DirectoryBytes + state.Region.OwnerBytes + state.Region.PayloadBytes + state.Region.PageHeaderBytes + state.Region.DirectoryBytes + state.Region.LogBytes + state.Region.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
	if physical != b.Used() {
		t.Fatal("actual complete graph accounting", physical, b.Used(), state)
	}
	if state.Symbols != 7 || state.Region.Descriptors.Live != before.Region.Descriptors.Live {
		t.Fatal("duplicate copied spelling", before, state)
	}
}
