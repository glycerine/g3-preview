// Source: src/ast/pp_params.hpp:5-51.
package ast_test

import (
	"testing"

	_ "github.com/glycerine/g3/pkg/ackermannization"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	_ "github.com/glycerine/g3/pkg/ast/fpa"
	_ "github.com/glycerine/g3/pkg/ast/normal_forms"
	_ "github.com/glycerine/g3/pkg/ast/rewriter"
	_ "github.com/glycerine/g3/pkg/math/polynomial"
	_ "github.com/glycerine/g3/pkg/math/realclosure"
	_ "github.com/glycerine/g3/pkg/model"
	_ "github.com/glycerine/g3/pkg/muz/base"
	_ "github.com/glycerine/g3/pkg/nlsat"
	_ "github.com/glycerine/g3/pkg/opt"
	_ "github.com/glycerine/g3/pkg/parsers/util"
	_ "github.com/glycerine/g3/pkg/sat"
	_ "github.com/glycerine/g3/pkg/smtcore"
	_ "github.com/glycerine/g3/pkg/solver"
	_ "github.com/glycerine/g3/pkg/tactic/sls"
	_ "github.com/glycerine/g3/pkg/tactic/smtlogics"
	_ "github.com/glycerine/g3/pkg/util/lp"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func ppMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func ppFixture(t *testing.T) (*uparams.ParamsContext, *uparams.GlobalParamsService, *uparams.ParamsRef) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	sy, err := usymbol.NewManager(usymbol.Config{Budget: b, PageSize: 64, DescriptorPageSize: 2})
	ppMust(t, err)
	q, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	ppMust(t, err)
	c, err := uparams.NewParamsContext(sy, q, uparams.ParamsConfig{Budget: b, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	ppMust(t, err)
	catalog, err := uparams.NewGlobalParamsCatalog(sy, uparams.GlobalParamsCatalogConfig{Budget: b, Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	ppMust(t, err)
	g, err := uparams.NewGlobalParamsService(sy, c, uparams.GlobalParamsConfig{Budget: b, DescriptorPageSize: 2, DescriptorWordsPageSize: 16, Catalog: catalog})
	ppMust(t, err)
	ppMust(t, g.InitializeSource())
	local, err := c.NewValue()
	ppMust(t, err)
	t.Cleanup(func() {
		ppMust(t, c.Del(&local))
		ppMust(t, g.Close())
		ppMust(t, catalog.Close())
		ppMust(t, c.Close())
		ppMust(t, q.Close())
		ppMust(t, sy.Close())
		if b.Used() != 0 {
			t.Errorf("owner graph retained %d bytes", b.Used())
		}
	})
	return c, g, &local
}

func TestPPParamsAllGettersDefaultsSnapshotAndLocalPrecedence(t *testing.T) {
	c, g, local := ppFixture(t)
	p, err := ast.NewPPParams(c, g, nil)
	ppMust(t, err)
	u1, err := p.MaxIndent()
	ppMust(t, err)
	u2, err := p.MaxNumLines()
	ppMust(t, err)
	u3, err := p.MaxWidth()
	ppMust(t, err)
	u4, err := p.MaxRibbon()
	ppMust(t, err)
	u5, err := p.MaxDepth()
	ppMust(t, err)
	u6, err := p.MinAliasSize()
	ppMust(t, err)
	b1, err := p.Decimal()
	ppMust(t, err)
	u7, err := p.DecimalPrecision()
	ppMust(t, err)
	b2, err := p.BVLiterals()
	ppMust(t, err)
	b3, err := p.FPRealLiterals()
	ppMust(t, err)
	b4, err := p.BVNeg()
	ppMust(t, err)
	b5, err := p.FlatAssoc()
	ppMust(t, err)
	b6, err := p.FixedIndent()
	ppMust(t, err)
	b7, err := p.SingleLine()
	ppMust(t, err)
	b8, err := p.Bounded()
	ppMust(t, err)
	b9, err := p.PrettyProof()
	ppMust(t, err)
	b10, err := p.SimplifyImplies()
	ppMust(t, err)
	if u1 != ^uint32(0) || u2 != ^uint32(0) || u3 != 80 || u4 != 80 || u5 != 5 || u6 != 10 || b1 || u7 != 10 || !b2 || b3 || b4 || !b5 || b6 || b7 || b8 || b9 || !b10 {
		t.Fatal("generated pp defaults", u1, u2, u3, u4, u5, u6, b1, u7, b2, b3, b4, b5, b6, b7, b8, b9, b10)
	}
	ppMust(t, p.Close())

	values := map[string]string{"max_indent": "101", "max_num_lines": "102", "max_width": "103", "max_ribbon": "104", "max_depth": "105", "min_alias_size": "106", "decimal": "true", "decimal_precision": "107", "bv_literals": "false", "fp_real_literals": "true", "bv_neg": "true", "flat_assoc": "false", "fixed_indent": "true", "single_line": "true", "bounded": "true", "pretty_proof": "true", "simplify_implies": "false"}
	for key, value := range values {
		ppMust(t, g.SourceSetName("pp."+key, value))
	}
	p, err = ast.NewPPParams(c, g, local)
	ppMust(t, err)
	t.Cleanup(func() { ppMust(t, p.Close()) })
	ppMust(t, g.SourceSetName("pp.max_width", "999"))
	u1, err = p.MaxIndent()
	ppMust(t, err)
	u2, err = p.MaxNumLines()
	ppMust(t, err)
	u3, err = p.MaxWidth()
	ppMust(t, err)
	u4, err = p.MaxRibbon()
	ppMust(t, err)
	u5, err = p.MaxDepth()
	ppMust(t, err)
	u6, err = p.MinAliasSize()
	ppMust(t, err)
	u7, err = p.DecimalPrecision()
	ppMust(t, err)
	b1, err = p.Decimal()
	ppMust(t, err)
	b2, err = p.BVLiterals()
	ppMust(t, err)
	b3, err = p.FPRealLiterals()
	ppMust(t, err)
	b4, err = p.BVNeg()
	ppMust(t, err)
	b5, err = p.FlatAssoc()
	ppMust(t, err)
	b6, err = p.FixedIndent()
	ppMust(t, err)
	b7, err = p.SingleLine()
	ppMust(t, err)
	b8, err = p.Bounded()
	ppMust(t, err)
	b9, err = p.PrettyProof()
	ppMust(t, err)
	b10, err = p.SimplifyImplies()
	ppMust(t, err)
	if u1 != 101 || u2 != 102 || u3 != 103 || u4 != 104 || u5 != 105 || u6 != 106 || u7 != 107 || !b1 || b2 || !b3 || !b4 || b5 || !b6 || !b7 || !b8 || !b9 || b10 {
		t.Fatal("constructor snapshot", u1, u2, u3, u4, u5, u6, u7, b1, b2, b3, b4, b5, b6, b7, b8, b9, b10)
	}
	key := "max_width"
	ppMust(t, c.SetUIntName(local, &key, 211))
	key = "decimal"
	ppMust(t, c.SetBoolName(local, &key, false))
	u3, err = p.MaxWidth()
	ppMust(t, err)
	b1, err = p.Decimal()
	ppMust(t, err)
	if u3 != 211 || b1 {
		t.Fatal("local precedence", u3, b1)
	}
	copyOwner := *p
	if _, err = copyOwner.MaxDepth(); err == nil {
		t.Fatal("copied facade accepted")
	}
	ppMust(t, p.Close())
	if _, err = p.FlatAssoc(); err == nil {
		t.Fatal("closed facade accepted")
	}
}
