package rewriter

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: bv_trailing.cpp:39,46,87. A cached second count skips the source
// count body, while construction and destruction each emit once.
func TestBVTrailingLifecycleAndCountSourceTrace(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	bindBVTraceGlobals(t, m)
	_, err = m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	sort, err := m.BitVectorSort(8)
	astTestMust(t, err)
	x, err := m.MkVar(0, sort)
	astTestMust(t, err)
	astTestPin(t, m, x)
	extract, err := NewMkExtractProc(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, extract.Close()) }()
	tags.Enable("bv-trailing")
	p, err := NewBVTrailing(extract)
	astTestMust(t, err)
	min, max, err := p.Count(x)
	astTestMust(t, err)
	if min != 0 || max != 8 {
		t.Fatal("source first count", min, max)
	}
	astTestMust(t, p.ResetCache(2))
	min, max, err = p.Count(x)
	astTestMust(t, err)
	if min != 0 || max != 8 {
		t.Fatal("source cached count", min, max)
	}
	astTestMust(t, p.Close())
	astTestMust(t, p.Close())
	if !tags.ShouldTrace("bv-trailing") {
		if output.Len() != 0 {
			t.Fatal("erased lifecycle trace", output.String())
		}
		return
	}
	want := "-------- [bv-trailing] imp src/ast/rewriter/bv_trailing.cpp:39 ---------\nctor\n------------------------------------------------\n" +
		"-------- [bv-trailing] count_trailing src/ast/rewriter/bv_trailing.cpp:87 ---------\n(:var 0)\n:0 - 8\n------------------------------------------------\n" +
		"-------- [bv-trailing] cache src/ast/rewriter/bv_trailing.cpp:346 ---------\ncaching@5: (:var 0)[8]\n: 0-8\n------------------------------------------------\n" +
		"-------- [bv-trailing] reset_cache src/ast/rewriter/bv_trailing.cpp:370 ---------\nmay reset cache 5 2\n------------------------------------------------\n" +
		"-------- [bv-trailing] is_cached src/ast/rewriter/bv_trailing.cpp:362 ---------\ncached@5: (:var 0)[8]\n: 0-8\n------------------------------------------------\n" +
		"-------- [bv-trailing] ~imp src/ast/rewriter/bv_trailing.cpp:46 ---------\ndtor\n------------------------------------------------\n" +
		"-------- [bv-trailing] reset_cache src/ast/rewriter/bv_trailing.cpp:370 ---------\nmay reset cache 5 0\n------------------------------------------------\n" +
		"-------- [bv-trailing] reset_cache src/ast/rewriter/bv_trailing.cpp:372 ---------\nreset cache 5\n------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("lifecycle and count trace got %q, want %q", got, want)
	}
}

// A source ostream refusal leaves badbit set, but construction, counting,
// equality reduction, removal and owning cleanup still run to completion.
func TestBVTrailingTraceRefusalDoesNotStopSourceOperations(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	w := &refusingCacheTraceWriter{err: errors.New("trace writer refused")}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	bindBVTraceGlobals(t, m)
	_, err = m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	sort, err := m.BitVectorSort(8)
	astTestMust(t, err)
	x, err := m.MkVar(0, sort)
	astTestMust(t, err)
	astTestPin(t, m, x)
	zero, err := m.BitVectorNumeralUnsigned(0, 8)
	astTestMust(t, err)
	astTestPin(t, m, zero)
	extract, err := NewMkExtractProc(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, extract.Close()) }()
	tags.Enable("bv-trailing")
	p, err := NewBVTrailing(extract)
	astTestMust(t, err)
	for i := 0; i < 2; i++ {
		min, max, e := p.Count(x)
		astTestMust(t, e)
		if min != 0 || max != 8 {
			t.Fatal("trace refusal changed count", min, max)
		}
	}
	out := astTestPin(t, m, x)
	status, err := p.EqRemoveTrailing(x, x, out)
	astTestMust(t, err)
	if status != RewriteFailed {
		t.Fatal("trace refusal changed equality status", status)
	}
	removed, err := p.Remove(zero, 8, out)
	astTestMust(t, err)
	result, err := m.RefGet(out)
	astTestMust(t, err)
	if removed != 8 || !result.IsNull() {
		t.Fatal("trace refusal changed removal", removed, result)
	}
	astTestMust(t, p.Close())
	if tags.ShouldTrace("bv-trailing") && w.calls != 1 {
		t.Fatal("failed stream was retried", w.calls)
	}
	if !tags.ShouldTrace("bv-trailing") && w.calls != 0 {
		t.Fatal("erased trace reached writer", w.calls)
	}
}

// Source: bv_trailing.cpp:51,95-96. The two CTRACE conditions select the
// published result after removal, including its null result.
func TestBVTrailingEqualityAndRemovalSourceTrace(t *testing.T) {
	symbols, numerals, budget := astTestOwners(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	bindBVTraceGlobals(t, m)
	_, err = m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	sort, err := m.BitVectorSort(8)
	astTestMust(t, err)
	x, err := m.MkVar(0, sort)
	astTestMust(t, err)
	astTestPin(t, m, x)
	zero, err := m.BitVectorNumeralUnsigned(0, 8)
	astTestMust(t, err)
	astTestPin(t, m, zero)
	extract, err := NewMkExtractProc(m)
	astTestMust(t, err)
	defer func() { astTestMust(t, extract.Close()) }()
	tags.Enable("bv-trailing")
	p, err := NewBVTrailing(extract)
	astTestMust(t, err)
	out := astTestPin(t, m, x)
	_, err = p.EqRemoveTrailing(x, x, out)
	astTestMust(t, err)
	astTestMust(t, m.SetRef(out, x))
	_, err = p.Remove(x, 0, out)
	astTestMust(t, err)
	_, err = p.Remove(zero, 8, out)
	astTestMust(t, err)
	astTestMust(t, p.Close())
	if !tags.ShouldTrace("bv-trailing") {
		if output.Len() != 0 {
			t.Fatal("erased equality/removal trace", output.String())
		}
		return
	}
	got := output.String()
	for _, want := range []string{
		"eq_remove_trailing src/ast/rewriter/bv_trailing.cpp:51 ---------\n(:var 0)\n=\n(:var 0)\n",
		"remove_trailing src/ast/rewriter/bv_trailing.cpp:95 ---------\n(:var 0)\n--->\n(:var 0)\n",
		"remove_trailing src/ast/rewriter/bv_trailing.cpp:96 ---------\n#x00\n---> [EMPTY]\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing source trace %q in %q", want, got)
		}
	}
}
