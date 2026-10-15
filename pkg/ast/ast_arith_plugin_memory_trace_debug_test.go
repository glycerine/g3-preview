//go:build g3debug && g3trace

package ast

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"

	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestASTArithmeticPluginMemoryWrapperRegistrationAndClose(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	tags.Enable("memory")
	id, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	p := m.arithmeticPlugin
	if p == nil || !p.published || p.family != id {
		t.Fatal("arithmetic plugin was not published")
	}
	identity := fmt.Sprintf("%p", p)
	alloc := "alloc " + identity + " src/ast/reg_decl_plugins.cpp:32 arith_decl_plugin " + strconv.FormatUint(p.header, 10) + "\n"
	dealloc := "dealloc " + identity + " src/ast/ast.cpp:1412\n"
	if strings.Count(output.String(), alloc) != 1 || strings.Contains(output.String(), dealloc) {
		t.Fatalf("arithmetic allocation before close: %q", output.String())
	}
	again, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	if again != id || m.arithmeticPlugin != p || strings.Count(output.String(), alloc) != 1 {
		t.Fatal("duplicate registration allocated another plugin")
	}
	astTestMust(t, m.Close())
	astTestMust(t, m.Close())
	got := output.String()
	if strings.Count(got, alloc) != 1 || strings.Count(got, dealloc) != 1 ||
		strings.Index(got, alloc) > strings.Index(got, dealloc) {
		t.Fatalf("arithmetic plugin memory records: %q", got)
	}
}
