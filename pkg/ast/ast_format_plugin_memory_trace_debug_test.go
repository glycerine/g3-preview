//go:build g3debug && g3trace

package ast

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

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
