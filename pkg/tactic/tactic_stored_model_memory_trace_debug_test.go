//go:build g3debug && g3trace

package tactic

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"strconv"
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

func TestStoredModelMemoryWrapperSourceAllocationAndRelease(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	satOK(t, err)
	t.Cleanup(func() { satOK(t, m.Close()) })
	md, err := model.NewModel(m)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, md.Close()) })
	satOK(t, md.Core().IncRef())
	tags.Enable("memory")
	if null, e := NewStoredModel(nil); e != nil || null != nil || output.Len() != 0 {
		t.Fatal("null model allocated a wrapper", null, e, output.String())
	}
	for _, tc := range []struct {
		line   int
		labels []usymbol.Symbol
	}{{line: 167}, {line: 173, labels: []usymbol.Symbol{usymbol.Numerical(9)}}} {
		output.Reset()
		var c ModelConverter
		if tc.labels == nil {
			c, err = NewStoredModel(md.Core())
		} else {
			c, err = newTacticStoredModelWithLabels(md.Core(), tc.labels)
		}
		satOK(t, err)
		stored := c.(*tacticStoredModelConverter)
		id := fmt.Sprintf("%p", stored)
		size := strconv.FormatUint(uint64(reflect.TypeOf(*stored).Size()), 10)
		allocation := "alloc " + id + " src/tactic/model_converter.cpp:" + strconv.Itoa(tc.line) + " model2mc " + size + "\n"
		if got := output.String(); got != "-------- [memory] allocate src/util/memory_manager.cpp:209 ---------\n"+allocation+"------------------------------------------------\n" {
			t.Fatalf("wrapper allocation %d: %q", tc.line, got)
		}
		satOK(t, c.Close())
		deallocation := "dealloc " + id + " src/tactic/converter.h:36\n"
		want := "-------- [memory] allocate src/util/memory_manager.cpp:209 ---------\n" + allocation + "------------------------------------------------\n" +
			"-------- [memory] deallocate src/util/memory_manager.cpp:204 ---------\n" + deallocation + "------------------------------------------------\n"
		if got := output.String(); got != want {
			t.Fatalf("wrapper release %d: got %q, want %q", tc.line, got, want)
		}
	}
	satOK(t, md.Core().DecRef())
}

func TestStoredModelMemoryWrapperUsesASTProcessDuringFinalize(t *testing.T) {
	b := arena.NewBudget(1 << 25)
	p := umemory.NewProcessAccounting()
	r, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, Process: p})
	satOK(t, err)
	if first, e := r.BeginSourceInitialize(math.MaxUint32); e != nil || !first {
		t.Fatal("source initialize", first, e)
	}
	satOK(t, r.CompleteSourceInitialize())
	s, err := usymbol.NewManager(usymbol.Config{Budget: b})
	satOK(t, err)
	q, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{
		Memory: r, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}},
	}})
	satOK(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: b, TraceTags: tags})
	satOK(t, err)
	md, err := model.NewModel(m)
	satOK(t, err)
	c, err := NewStoredModel(md.Core())
	satOK(t, err)
	if !strings.Contains(output.String(), "src/tactic/model_converter.cpp:167 model2mc") {
		t.Fatal("missing source allocation", output.String())
	}
	if active, e := p.BeginFinalize(); e != nil || !active {
		t.Fatal("source finalize", active, e)
	}
	satOK(t, c.Close())
	if strings.Contains(output.String(), "dealloc ") {
		t.Fatal("deallocation during process finalization", output.String())
	}
	satOK(t, m.Close())
	satOK(t, p.CompleteFinalize())
	satOK(t, q.Close())
	satOK(t, s.Close())
	satOK(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("owner graph leak", b.Used())
	}
}
