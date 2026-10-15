//go:build g3debug && g3trace

package aig

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestAIGMemoryWrappersFollowASTProcessWithUnboundTrace(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	p := umemory.NewProcessAccounting()
	r, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, Process: p})
	aigMust(t, err)
	if first, err := r.BeginSourceInitialize(math.MaxUint32); err != nil || !first {
		t.Fatalf("source initialization: first=%v err=%v", first, err)
	}
	aigMust(t, r.CompleteSourceInitialize())
	s, err := symbol.NewManager(symbol.Config{Budget: b})
	aigMust(t, err)
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Memory: r, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}},
	}})
	aigMust(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: b, TraceTags: tags})
	aigMust(t, err)
	owner, err := newManagerWithGateEncoding(m, true, true)
	aigMust(t, err)
	for _, site := range []string{"src/tactic/aig/aig_tactic.cpp:34 aig_manager", "src/tactic/aig/aig.cpp:1684 imp"} {
		if !strings.Contains(output.String(), site) {
			t.Fatalf("missing AIG allocation %q: %q", site, output.String())
		}
	}
	before := output.Len()
	if active, err := p.BeginFinalize(); err != nil || !active {
		t.Fatalf("source finalization: active=%v err=%v", active, err)
	}
	aigMust(t, owner.Close())
	owner.traceTacticDeallocated()
	aigMust(t, m.Close())
	if tail := output.String()[before:]; strings.Contains(tail, "dealloc ") {
		t.Fatalf("AIG owner emitted deallocation during process finalization: %q", tail)
	}
	aigMust(t, p.CompleteFinalize())
	aigMust(t, q.Close())
	aigMust(t, s.Close())
	aigMust(t, r.Close())
	if used := b.Used(); used != 0 {
		t.Fatalf("owner graph retained %d bytes", used)
	}
}
