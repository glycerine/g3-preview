//go:build g3debug && g3trace

package model

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestCopiedFunctionMemoryWrapperFollowsASTProcessWithUnboundTrace(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	b := arena.NewBudget(1 << 25)
	p := umemory.NewProcessAccounting()
	r, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, Process: p})
	must(err)
	if first, err := r.BeginSourceInitialize(math.MaxUint32); err != nil || !first {
		t.Fatalf("source initialization: first=%v err=%v", first, err)
	}
	must(r.CompleteSourceInitialize())
	s, err := usymbol.NewManager(usymbol.Config{Budget: b})
	must(err)
	q, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{
		Memory: r, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}},
	}})
	must(err)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: b})
	must(err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	f, err := NewFunctionInterpretation(m, 1)
	must(err)
	f.memoryTrace = tags
	copy, err := f.copyWithMemoryTrace(true)
	must(err)
	if !strings.Contains(output.String(), "src/model/func_interp.cpp:91 func_interp") {
		t.Fatalf("missing copied function allocation: %q", output.String())
	}
	if active, err := p.BeginFinalize(); err != nil || !active {
		t.Fatalf("source finalization: active=%v err=%v", active, err)
	}
	must(copy.Close())
	must(f.Close())
	if strings.Contains(output.String(), "dealloc ") {
		t.Fatalf("copied function deallocated during finalization: %q", output.String())
	}
	must(m.Close())
	must(p.CompleteFinalize())
	must(q.Close())
	must(s.Close())
	must(r.Close())
	if used := b.Used(); used != 0 {
		t.Fatalf("owner graph retained %d bytes", used)
	}
}
