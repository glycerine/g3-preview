//go:build g3debug && g3trace

package ast

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestASTFormatPluginMemoryWrapperLazyLifetime(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	child := m.Format
	if child == nil || child.formatPlugin != nil {
		t.Fatal("format plugin must be lazy")
	}
	tags.Enable("memory")
	id, err := m.GetFormatFamilyID()
	astTestMust(t, err)
	p := child.formatPlugin
	if p == nil || !p.published || p.family != id || !p.memoryTraceLive {
		t.Fatal("format plugin registration was not published")
	}
	alloc := "alloc " + p.traceIdentity() + " src/ast/format.cpp:105 format_decl_plugin " + strconv.FormatUint(p.header, 10) + "\n"
	if strings.Count(output.String(), alloc) != 1 {
		t.Fatalf("format allocation: %q", output.String())
	}
	again, err := m.GetFormatFamilyID()
	astTestMust(t, err)
	if again != id || strings.Count(output.String(), alloc) != 1 {
		t.Fatal("duplicate format allocation", again, id, output.String())
	}
	astTestMust(t, m.Close())
	astTestMust(t, m.Close())
	dealloc := "dealloc " + p.traceIdentity() + " src/ast/ast.cpp:1412\n"
	got := output.String()
	if ai, di := strings.Index(got, alloc), strings.Index(got, dealloc); ai < 0 || di <= ai || strings.Count(got, dealloc) != 1 || p.memoryTraceLive {
		t.Fatalf("format plugin lifetime: %q", got)
	}
}

func TestASTFormatPluginMemoryWrapperLaterFormatFailure(t *testing.T) {
	symbols, numerals, _ := astTestOwners(t)
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		budget := arena.NewBudget(1 << 30)
		var output bytes.Buffer
		tags := udiag.NewTraceTagsWithWriter(&output)
		m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
		astTestMust(t, err)
		child := m.Format
		tags.Enable("memory")
		budget.FailAfter(ordinal)
		_, err = m.FormatString("source prefix text")
		p := child.formatPlugin
		if p == nil || err == nil {
			budget.FailAfter(-1)
			astTestMust(t, m.Close())
			continue
		}
		if !errors.Is(err, arena.ErrInjectedFailure) || p.header == 0 || !p.published || !p.completed {
			t.Fatalf("later format failure at %d: error=%v owner=%+v", ordinal, err, p)
		}
		alloc := "alloc " + p.traceIdentity() + " src/ast/format.cpp:105 format_decl_plugin " + strconv.FormatUint(p.header, 10) + "\n"
		if strings.Count(output.String(), alloc) != 1 {
			t.Fatalf("published allocation record: %q", output.String())
		}
		budget.FailAfter(-1)
		astTestMust(t, m.Close())
		dealloc := "dealloc " + p.traceIdentity() + " src/ast/ast.cpp:1412\n"
		if strings.Count(output.String(), dealloc) != 1 || strings.Index(output.String(), dealloc) <= strings.Index(output.String(), alloc) || budget.Used() != 0 {
			t.Fatalf("owner lifetime after format failure: %q, bytes=%d", output.String(), budget.Used())
		}
		return
	}
	t.Fatal("no post-publication format refusal reached")
}
