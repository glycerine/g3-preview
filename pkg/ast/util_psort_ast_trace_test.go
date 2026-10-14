//go:build g3trace

package ast

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestPsortASTCardSourceEntryTrace(t *testing.T) {
	m := astTestManager(t)
	c, e := NewPsortASTContext(m)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, c.Close()) })
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: m.BudgetField})
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	s, e := NewPsortASTNetwork(c, r, m.BudgetField, nil)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	out, e := NewASTLiteralVector(m.BudgetField)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, out.Close()) })

	var buf bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&buf)
	astTestMust(t, s.BindTrace(tags))
	astTestMust(t, s.card(0, 0, psortASTPointer{}, out))
	if buf.Len() != 0 {
		t.Fatalf("disabled pb tag emitted %q", buf.String())
	}
	tags.Enable("pb")
	astTestMust(t, s.card(0, 0, psortASTPointer{}, out))
	want := "-------- [pb] card src/util/sorting_network.h:598 ---------\n" +
		"card k: 0 n: 0\n" +
		"------------------------------------------------\n" +
		"-------- [pb] sorting src/util/sorting_network.h:744 ---------\n" +
		"sorting: 0\n" +
		"------------------------------------------------\n"
	if got := buf.String(); got != want {
		t.Fatalf("source card entry record:\n got %q\nwant %q", got, want)
	}
}

func TestPsortASTScalarEntryTracesPrecedeSourceGuards(t *testing.T) {
	m := astTestManager(t)
	c, e := NewPsortASTContext(m)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, c.Close()) })
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: m.BudgetField})
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	s, e := NewPsortASTNetwork(c, r, m.BudgetField, nil)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	out, e := NewASTLiteralVector(m.BudgetField)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, out.Close()) })

	var buf bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&buf)
	tags.Enable("pb")
	astTestMust(t, s.BindTrace(tags))
	bad := psortASTPointer{}
	if e := s.merge(1, bad, 1, bad, out); !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("merge source operand guard: %v", e)
	}
	if e := s.mergeSmall(1, 1, bad, 1, bad, out); !errors.Is(e, arena.ErrBounds) {
		t.Fatalf("smerge source operand guard: %v", e)
	}
	if e := s.directMerge(0, 0, bad, 0, bad, out); !errors.Is(e, ErrPsortUninitializedComparison) {
		t.Fatalf("dsmerge source comparison guard: %v", e)
	}
	if e := s.directSort(0, 0, bad, out); !errors.Is(e, ErrPsortUninitializedComparison) {
		t.Fatalf("dsorting source comparison guard: %v", e)
	}
	want := "-------- [pb] merge src/util/sorting_network.h:640 ---------\nmerge a: 1 b: 1\n------------------------------------------------\n" +
		"-------- [pb] smerge src/util/sorting_network.h:805 ---------\nsmerge: c:1 a:1 b:1\n------------------------------------------------\n" +
		"-------- [pb] dsmerge src/util/sorting_network.h:923 ---------\ndsmerge: c:0 a:0 b:0\n------------------------------------------------\n" +
		"-------- [pb] dsorting src/util/sorting_network.h:982 ---------\ndsorting m: 0 n: 0\n------------------------------------------------\n"
	if got := buf.String(); got != want {
		t.Fatalf("source scalar entry sequence:\n got %q\nwant %q", got, want)
	}
}

func TestPsortASTInterleaveEntryTraceBeforeOutput(t *testing.T) {
	m := astTestManager(t)
	c, e := NewPsortASTContext(m)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, c.Close()) })
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: m.BudgetField})
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	s, e := NewPsortASTNetwork(c, r, m.BudgetField, nil)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	a, e := NewASTLiteralVector(m.BudgetField)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, a.Close()) })
	b, e := NewASTLiteralVector(m.BudgetField)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, b.Close()) })
	out, e := NewASTLiteralVector(m.BudgetField)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, out.Close()) })
	h, e := m.MkFalse()
	astTestMust(t, e)
	astTestMust(t, a.Push(h))

	var buf bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&buf)
	tags.Enable("pb")
	astTestMust(t, s.BindTrace(tags))
	astTestMust(t, s.interleave(a, b, out))
	sz, e := out.Size()
	astTestMust(t, e)
	got, e := out.At(0)
	astTestMust(t, e)
	if sz != 1 || got != h {
		t.Fatal("source interleave output", sz, got)
	}
	want := "-------- [pb] interleave src/util/sorting_network.h:712 ---------\n" +
		"interleave: 1 0\n" +
		"------------------------------------------------\n"
	if got := buf.String(); got != want {
		t.Fatalf("source interleave entry:\n got %q\nwant %q", got, want)
	}
}
