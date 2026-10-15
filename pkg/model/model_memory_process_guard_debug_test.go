//go:build g3debug && g3trace

package model_test

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestModelMemoryWrappersFollowASTProcessWithUnboundTrace(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	p := umemory.NewProcessAccounting()
	r, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, Process: p})
	astTestMust(t, err)
	if first, err := r.BeginSourceInitialize(math.MaxUint32); err != nil || !first {
		t.Fatalf("source initialization: first=%v err=%v", first, err)
	}
	astTestMust(t, r.CompleteSourceInitialize())
	s, err := usymbol.NewManager(usymbol.Config{Budget: b})
	astTestMust(t, err)
	q, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{
		Memory: r, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}},
	}})
	astTestMust(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: b, TraceTags: tags})
	astTestMust(t, err)
	md, err := model.NewModel(m)
	astTestMust(t, err)
	sort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "model-process-sort"), nil)
	astTestMust(t, err)
	value, err := m.GetSomeValue(sort)
	astTestMust(t, err)
	astTestMust(t, md.RegisterUniverse(sort, []ast.ASTHandle{value}))
	copy, err := md.Copy()
	astTestMust(t, err)
	before := output.Len()
	for _, site := range []string{"src/model/model.cpp:69 model", "src/model/model.cpp:139 ptr_vector<expr>"} {
		if !strings.Contains(output.String(), site) {
			t.Fatalf("missing source allocation %q: %q", site, output.String())
		}
	}
	if active, err := p.BeginFinalize(); err != nil || !active {
		t.Fatalf("source finalization: active=%v err=%v", active, err)
	}
	astTestMust(t, copy.Close())
	astTestMust(t, md.Close())
	astTestMust(t, m.Close())
	if tail := output.String()[before:]; strings.Contains(tail, "dealloc ") {
		t.Fatalf("model owner emitted deallocation during process finalization: %q", tail)
	}
	astTestMust(t, p.CompleteFinalize())
	astTestMust(t, q.Close())
	astTestMust(t, s.Close())
	astTestMust(t, r.Close())
	if used := b.Used(); used != 0 {
		t.Fatalf("owner graph retained %d bytes", used)
	}
}
