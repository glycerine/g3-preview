package rewriter

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

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
}
