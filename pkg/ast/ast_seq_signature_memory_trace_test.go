package ast

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestASTSequenceSignatureMemoryWrappersSourceOrder(t *testing.T) {
	symbols, numerals, _ := astTestOwners(t)
	budget := arena.NewBudget(1 << 28)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	_, err = m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	_, err = m.RegisterArrayPlugin()
	astTestMust(t, err)
	tags.Enable("memory")
	_, err = m.RegisterSequencePlugin()
	astTestMust(t, err)
	p := m.sequencePlugin
	first, strIndex, last := p.signatures[0], p.signatures[39], p.signatures[43]
	if first == nil || strIndex == nil || last == nil {
		t.Fatal("source signature owners missing")
	}
	astTestMust(t, m.Close())
	if budget.Used() != 0 {
		t.Fatal("sequence signature owners retained", budget.Used())
	}
	got := output.String()
	if !udiag.DebugBuild || !tags.ShouldTrace("memory") {
		if got != "" {
			t.Fatal("source Z3DEBUG signature wrappers must erase", got)
		}
		return
	}
	if strings.Count(got, " psig ") != 42 || strings.Count(got, " src/ast/seq_decl_plugin.cpp:352\n") != 42 {
		t.Fatal("every source psig allocation and indexed finalization must have one record", got)
	}
	size := strconv.FormatUint(uint64(reflect.TypeOf(astSequenceSignature{}).Size()), 10)
	alloc := func(s *astSequenceSignature, line int) string {
		return "alloc " + fmt.Sprintf("%p", s) + " src/ast/seq_decl_plugin.cpp:" + strconv.Itoa(line) + " psig " + size + "\n"
	}
	dealloc := func(s *astSequenceSignature) string {
		return "dealloc " + fmt.Sprintf("%p", s) + " src/ast/seq_decl_plugin.cpp:352\n"
	}
	parts := []string{alloc(first, 530), alloc(strIndex, 557), alloc(last, 572), dealloc(first), dealloc(strIndex), dealloc(last)}
	previous := -1
	for _, part := range parts {
		at := strings.Index(got, part)
		if at <= previous || strings.Count(got, part) != 1 {
			t.Fatalf("source signature wrapper order for %q in %q", part, got)
		}
		previous = at
	}
}
