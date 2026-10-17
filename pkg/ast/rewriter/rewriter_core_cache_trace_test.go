package rewriter

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

type refusingCacheTraceWriter struct {
	calls int
	err   error
}

func (w *refusingCacheTraceWriter) Write([]byte) (int, error) {
	w.calls++
	return 0, w.err
}

type rewriterCacheTraceObserver struct {
	bytes.Buffer
	core         *rewriterCore
	key          ast.ASTHandle
	beforeInsert bool
	inspectErr   error
}

func (w *rewriterCacheTraceObserver) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte("--->")) {
		var value ast.ASTHandle
		value, w.inspectErr = w.core.cache.Find(w.key)
		w.beforeInsert = w.inspectErr == nil && value.IsNull()
	}
	return w.Buffer.Write(p)
}

func (w *rewriterCacheTraceObserver) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// Source: rewriter.cpp:45-59. The source record observes both terms before
// the concrete cache insertion and is erased in the ordinary build.
func TestRewriterCacheResultSourceTraceBeforeInsert(t *testing.T) {
	m := astTestManager(t)
	bindBVTraceGlobals(t, m)
	p := goalTestAtom(t, m, "cache-p")
	q := goalTestAtom(t, m, "cache-q")
	c, err := newRewriterCore(m, false)
	astTestMust(t, err)
	defer func() { astTestMust(t, c.Close()) }()
	output := &rewriterCacheTraceObserver{core: c, key: p}
	tags := diagnostic.NewTraceTagsWithWriter(output)
	c.traceTags = tags
	astTestMust(t, c.cacheResult(p, q))
	if output.Len() != 0 {
		t.Fatal("disabled cache trace", output.String())
	}
	astTestMust(t, c.cache.Reset())
	tags.Enable("rewriter_cache_result")
	astTestMust(t, c.cacheResult(p, q))
	value, err := c.cache.Find(p)
	astTestMust(t, err)
	if value != q {
		t.Fatal("source cache result", value, q)
	}
	if !tags.ShouldTrace("rewriter_cache_result") {
		if output.Len() != 0 {
			t.Fatal("erased cache trace", output.String())
		}
		return
	}
	want := "-------- [rewriter_cache_result] cache_result src/ast/rewriter/rewriter.cpp:52 ---------\n" +
		"cache-p\n--->\ncache-q\n------------------------------------------------\n"
	if got := output.String(); got != want || !output.beforeInsert || output.inspectErr != nil {
		t.Fatalf("cache trace got %q, want %q; before=%v inspection=%v", got, want, output.beforeInsert, output.inspectErr)
	}
	boolSort, err := m.MkBoolSort()
	astTestMust(t, err)
	astTestPin(t, m, boolSort)
	bound, err := m.MkVar(0, boolSort)
	astTestMust(t, err)
	astTestPin(t, m, bound)
	negated, err := m.MkNot(bound)
	astTestMust(t, err)
	astTestPin(t, m, negated)
	name, err := m.Symbols.Intern("cache-x")
	astTestMust(t, err)
	quantified, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true,
		Sorts: []ast.ASTHandle{boolSort}, Names: []usymbol.Symbol{name}, Body: negated})
	astTestMust(t, err)
	astTestPin(t, m, quantified)
	astTestMust(t, c.cache.Reset())
	output.Reset()
	output.key, output.beforeInsert = quantified, false
	astTestMust(t, c.cacheResult(quantified, quantified))
	if tags.ShouldTrace("rewriter_cache_result") {
		want = "-------- [rewriter_cache_result] cache_result src/ast/rewriter/rewriter.cpp:52 ---------\n" +
			"(forall ((cache-x Bool)) (not cache-x))\n--->\n(forall ((cache-x Bool)) (not cache-x))\n------------------------------------------------\n"
		if got := output.String(); got != want || !output.beforeInsert || output.inspectErr != nil {
			t.Fatalf("quantified cache trace got %q, want %q; before=%v inspection=%v", got, want, output.beforeInsert, output.inspectErr)
		}
	}
}

// A failed source ostream keeps badbit; it does not cancel either cache
// insertion or the next cache-result call.
func TestRewriterCacheResultContinuesAfterTraceWriterRefusal(t *testing.T) {
	m := astTestManager(t)
	bindBVTraceGlobals(t, m)
	p := goalTestAtom(t, m, "refused-p")
	q := goalTestAtom(t, m, "refused-q")
	r := goalTestAtom(t, m, "refused-r")
	c, err := newRewriterCore(m, false)
	astTestMust(t, err)
	defer func() { astTestMust(t, c.Close()) }()
	w := &refusingCacheTraceWriter{err: errors.New("trace writer refused")}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("rewriter_cache_result")
	c.traceTags = tags
	astTestMust(t, c.cacheResult(p, q))
	astTestMust(t, c.cacheResult(q, r))
	for _, tc := range []struct{ key, want ast.ASTHandle }{{p, q}, {q, r}} {
		got, e := c.cache.Find(tc.key)
		astTestMust(t, e)
		if got != tc.want {
			t.Fatal("trace refusal stopped source cache insertion", got, tc.want)
		}
	}
	if tags.ShouldTrace("rewriter_cache_result") {
		if e := c.cacheResult(ast.ASTHandle{}, q); !errors.Is(e, ast.ErrASTHandle) {
			t.Fatal("operand computation error was mistaken for failed ostream state", e)
		}
	}
	if tags.ShouldTrace("rewriter_cache_result") && w.calls != 1 {
		t.Fatal("failed stream was retried", w.calls)
	}
	if !tags.ShouldTrace("rewriter_cache_result") && w.calls != 0 {
		t.Fatal("erased trace reached writer", w.calls)
	}
}
