package ufbv

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestUFBVForwardRemovalTypedIdentityTrace(t *testing.T) {
	m := cacheFixture(t)
	b, e := m.MkBoolSort()
	cacheMust(t, e)
	x, e := m.MkVar(0, b)
	cacheMust(t, e)
	f, e := m.MkFuncDecl(symbol.Numerical(941), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	cacheMust(t, e)
	lhs, e := m.MkApp(f, []ast.ASTHandle{x})
	cacheMust(t, e)
	cacheMust(t, m.IncRef(lhs))
	t.Cleanup(func() { cacheMust(t, m.DecRef(lhs)) })
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{b},
		Names: []symbol.Symbol{symbol.Numerical(942)}, Body: x,
	})
	cacheMust(t, e)
	cacheMust(t, m.IncRef(q))
	t.Cleanup(func() { cacheMust(t, m.DecRef(q)) })
	r, e := newRewriteRuleIndex(m)
	cacheMust(t, e)
	t.Cleanup(func() { cacheMust(t, r.close()) })
	cacheMust(t, r.insert(q, lhs, x))
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("demodulator_fwd")
	r.trace = tags
	cacheMust(t, r.remove(f, q))
	if !tags.ShouldTrace("demodulator_fwd") {
		if output.Len() != 0 {
			t.Fatal("erased removal trace emitted", output.String())
		}
		return
	}
	identity := "@" + strconv.FormatUint(q.Origin(), 16) + ":" +
		strconv.FormatUint(uint64(q.Slot()), 16) + ":" +
		strconv.FormatUint(uint64(q.Generation()), 16)
	want := "-------- [demodulator_fwd] remove_fwd_idx src/tactic/ufbv/ufbv_rewriter.cpp:241 ---------\n" +
		"REMOVE: " + identity + "\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("typed Go removal identity trace:\ngot  %q\nwant %q", got, want)
	}
	cacheMust(t, r.insert(q, lhs, x))
	refused := &refusingUFBVTraceWriter{}
	refusedTags := diagnostic.NewTraceTagsWithWriter(refused)
	refusedTags.Enable("demodulator_fwd")
	r.trace = refusedTags
	cacheMust(t, r.remove(f, q))
	if refusedTags.ShouldTrace("demodulator_fwd") && refused.calls != 1 {
		t.Fatal("removal trace stream refusal stopped cleanup", refused.calls)
	}
}
