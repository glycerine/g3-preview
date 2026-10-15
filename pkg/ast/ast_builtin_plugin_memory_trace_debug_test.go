//go:build g3debug && g3trace

package ast

import (
	"bytes"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unsafe"

	"github.com/glycerine/g3/pkg/arena"
	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTMemoryWrappersFollowOwnerProcessWithUnboundTrace(t *testing.T) {
	b := arena.NewBudget(1 << 26)
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
	tags := udiag.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	name := uparams.LiteralParamsStringView(filepath.Join(t.TempDir(), "ast-finalize.log"))
	m, err := NewSourceASTManager(ASTManagerConfig{Symbols: s, Numerals: q, Budget: b, TraceTags: tags}, &name)
	astTestMust(t, err)
	if m.TraceFile == nil {
		t.Fatal("source AST trace file was not constructed")
	}
	sort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "OwnerSort"), nil)
	astTestMust(t, err)
	ref, err := m.NewRef(sort)
	astTestMust(t, err)
	if !strings.Contains(output.String(), "alloc ") {
		t.Fatalf("missing AST allocation records: %q", output.String())
	}
	before := output.Len()
	if active, err := p.BeginFinalize(); err != nil || !active {
		t.Fatalf("source finalization: active=%v err=%v", active, err)
	}
	collector := newASTDeclCollector(m, true)
	collector.sorts = []ASTHandle{sort}
	astTestMust(t, collector.orderDeps())
	astTestMust(t, m.DelRef(&ref))
	astTestMust(t, m.Close())
	if tail := output.String()[before:]; !strings.Contains(tail, "sort_set ") || strings.Contains(tail, "dealloc ") {
		t.Fatalf("AST file/dependency/manager records during process finalization: %q", tail)
	}
	astTestMust(t, p.CompleteFinalize())
	astTestMust(t, q.Close())
	astTestMust(t, s.Close())
	astTestMust(t, r.Close())
	if used := b.Used(); used != 0 {
		t.Fatalf("owner graph retained %d bytes", used)
	}
}

func TestASTBuiltinPluginMemoryWrappersFollowBothManagerDestructors(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	child := m.Format
	if child == nil {
		t.Fatal("source format manager was not constructed")
	}
	size := strconv.FormatUint(uint64(unsafe.Sizeof(astPluginBinding{})), 10)
	for _, owner := range []*ASTManager{child, m} {
		for id, site := range astBuiltinPluginAllocation {
			alloc := "alloc " + owner.builtinPluginTraceIdentity(id) + " src/ast/ast.cpp:" + strconv.Itoa(site.line) + " " + site.name + " " + size + "\n"
			if strings.Count(output.String(), alloc) != 1 {
				t.Fatalf("builtin allocation before close: %q: %q", alloc, output.String())
			}
		}
	}
	astTestMust(t, m.Close())
	astTestMust(t, m.Close())
	got := output.String()
	for _, owner := range []*ASTManager{child, m} {
		previous := -1
		for id, site := range astBuiltinPluginAllocation {
			identity := owner.builtinPluginTraceIdentity(id)
			alloc := "alloc " + identity + " src/ast/ast.cpp:" + strconv.Itoa(site.line) + " " + site.name + " " + size + "\n"
			dealloc := "dealloc " + identity + " src/ast/ast.cpp:1412\n"
			ai, di := strings.Index(got, alloc), strings.Index(got, dealloc)
			if ai < 0 || di <= ai || strings.Count(got, alloc) != 1 || strings.Count(got, dealloc) != 1 || di <= previous {
				t.Fatalf("builtin lifetime/order for %q: %q", identity, got)
			}
			previous = di
		}
	}
}
