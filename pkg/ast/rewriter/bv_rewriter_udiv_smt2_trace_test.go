// Source: src/ast/rewriter/bv_rewriter.cpp:1038-1099; src/util/trace.h:40-46.
package rewriter

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestBVRewriterUnsignedDivisionSMT2SourceTrace(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	context, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	astTestMust(t, err)
	catalog, err := uparams.NewGlobalParamsCatalog(m.Symbols, uparams.GlobalParamsCatalogConfig{Budget: m.Budget()})
	astTestMust(t, err)
	globals, err := uparams.NewGlobalParamsService(m.Symbols, context, uparams.GlobalParamsConfig{Budget: m.Budget(), Catalog: catalog})
	astTestMust(t, err)
	astTestMust(t, globals.InitializeSource())
	previous, err := uparams.BindProcessGlobalParamsService(globals)
	astTestMust(t, err)
	t.Cleanup(func() {
		_, e := uparams.BindProcessGlobalParamsService(previous)
		astTestMust(t, e)
		astTestMust(t, globals.Close())
		astTestMust(t, catalog.Close())
		astTestMust(t, context.Close())
	})
	local, err := c.NewValue()
	astTestMust(t, err)
	defer func() { astTestMust(t, c.Del(&local)) }()
	p, err := NewBVRewriter(m, c, g, &local)
	astTestMust(t, err)
	defer func() { astTestMust(t, p.Close()) }()
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("udiv2mul")
	tags.Enable("bv_udiv")
	astTestMust(t, p.BindTrace(tags))
	xs := bvMergeVars(t, m, 2)
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	_, err = p.RewriteUnsignedDivision(xs[0], bvAddNum(t, m, 3, 8), out)
	astTestMust(t, err)
	key := "hi_div0"
	astTestMust(t, c.SetBoolName(&local, &key, false))
	astTestMust(t, p.Update(&local))
	status, err := p.RewriteUnsignedDivision(xs[0], xs[1], out)
	astTestMust(t, err)
	if status != Rewrite2 {
		t.Fatal("symbolic source branch", status)
	}
	key = "udiv2mul"
	astTestMust(t, c.SetBoolName(&local, &key, true))
	astTestMust(t, p.Update(&local))
	status, err = p.RewriteUnsignedDivision(xs[0], bvAddNum(t, m, 3, 8), out)
	astTestMust(t, err)
	if status != Rewrite1 {
		t.Fatal("configured inverse branch", status)
	}
	if !tags.ShouldTrace("bv_udiv") {
		if output.Len() != 0 {
			t.Fatal("erased trace emitted", output.String())
		}
		return
	}
	record := func(tag, line, body string) string {
		return "-------- [" + tag + "] mk_bv_udiv_core src/ast/rewriter/bv_rewriter.cpp:" + line + " ---------\n" + body + "------------------------------------------------\n"
	}
	want := record("bv_udiv", "1038", "hi_div0: 1\n") +
		record("udiv2mul", "1040", "#x03 udiv2mul: 0\n") +
		record("bv_udiv", "1038", "hi_div0: 0\n") +
		record("udiv2mul", "1040", "(:var 1) udiv2mul: 0\n") +
		record("bv_udiv", "1097", "(:var 0)\n(:var 1)\n---->\n(ite (= (:var 1) #x00) (bvudiv0 (:var 0)) (bvudiv_i (:var 0) (:var 1)))\n") +
		record("bv_udiv", "1038", "hi_div0: 0\n") +
		record("udiv2mul", "1040", "#x03 udiv2mul: 1\n") +
		record("udiv2mul", "1075", "using udiv2mul\n")
	if got := output.String(); got != want {
		t.Fatalf("source unsigned division records: got %q, want %q", got, want)
	}
	tags.Disable("udiv2mul")
	tags.Disable("bv_udiv")
	_, err = p.RewriteUnsignedDivision(xs[0], xs[1], out)
	astTestMust(t, err)
	if got := output.String(); got != want {
		t.Fatalf("disabled tags emitted: %q", got)
	}
}
