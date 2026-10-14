//go:build g3trace

package ast

import (
	"bytes"
	"testing"

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
		"------------------------------------------------\n"
	if got := buf.String(); got != want {
		t.Fatalf("source card entry record:\n got %q\nwant %q", got, want)
	}
}
