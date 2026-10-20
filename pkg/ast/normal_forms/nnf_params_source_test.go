// Source: src/ast/normal_forms/nnf_params.hpp:5-27.
package normal_forms_test

import (
	"testing"

	_ "github.com/glycerine/g3/pkg/ackermannization"
	"github.com/glycerine/g3/pkg/arena"
	_ "github.com/glycerine/g3/pkg/ast/fpa"
	normalforms "github.com/glycerine/g3/pkg/ast/normal_forms"
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

func nnfParamsMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}

func nnfParamsFixture(t *testing.T) (*uparams.ParamsContext, *uparams.GlobalParamsService, *uparams.ParamsRef, *usymbol.Manager) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	sy, e := usymbol.NewManager(usymbol.Config{Budget: b, PageSize: 64, DescriptorPageSize: 2})
	nnfParamsMust(t, e)
	q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{
		Cells: arena.Config{PageSize: 2, Budget: b},
		Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}},
	}})
	nnfParamsMust(t, e)
	c, e := uparams.NewParamsContext(sy, q, uparams.ParamsConfig{Budget: b, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	nnfParamsMust(t, e)
	catalog, e := uparams.NewGlobalParamsCatalog(sy, uparams.GlobalParamsCatalogConfig{Budget: b, Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	nnfParamsMust(t, e)
	g, e := uparams.NewGlobalParamsService(sy, c, uparams.GlobalParamsConfig{Budget: b, DescriptorPageSize: 2, DescriptorWordsPageSize: 16, Catalog: catalog})
	nnfParamsMust(t, e)
	nnfParamsMust(t, g.InitializeSource())
	local, e := c.NewValue()
	nnfParamsMust(t, e)
	t.Cleanup(func() {
		nnfParamsMust(t, c.Del(&local))
		nnfParamsMust(t, g.Close())
		nnfParamsMust(t, catalog.Close())
		nnfParamsMust(t, c.Close())
		nnfParamsMust(t, q.Close())
		nnfParamsMust(t, sy.Close())
		if b.Used() != 0 {
			t.Errorf("owner graph retained %d bytes", b.Used())
		}
	})
	return c, g, &local, sy
}

func TestNNFParamsDefaultsSnapshotAndLocalPrecedence(t *testing.T) {
	c, g, local, sy := nnfParamsFixture(t)
	p, e := normalforms.NewNNFParams(c, g, nil)
	nnfParamsMust(t, e)
	max, e := p.MaxMemory()
	nnfParamsMust(t, e)
	skHack, e := p.SkHack()
	nnfParamsMust(t, e)
	mode, e := p.Mode()
	nnfParamsMust(t, e)
	ignore, e := p.IgnoreLabels()
	nnfParamsMust(t, e)
	skolemize, e := p.Skolemize()
	nnfParamsMust(t, e)
	spelling, e := sy.String(mode)
	nnfParamsMust(t, e)
	if max != ^uint32(0) || skHack || spelling != "skolem" || ignore || !skolemize {
		t.Fatal("generated defaults", max, skHack, spelling, ignore, skolemize)
	}
	nnfParamsMust(t, p.Close())

	nnfParamsMust(t, g.SourceSetName("nnf.max_memory", "17"))
	nnfParamsMust(t, g.SourceSetName("nnf.sk_hack", "true"))
	nnfParamsMust(t, g.SourceSetName("nnf.mode", "full"))
	nnfParamsMust(t, g.SourceSetName("nnf.ignore_labels", "true"))
	nnfParamsMust(t, g.SourceSetName("nnf.skolemize", "false"))
	p, e = normalforms.NewNNFParams(c, g, local)
	nnfParamsMust(t, e)
	t.Cleanup(func() { nnfParamsMust(t, p.Close()) })
	// The constructor owns a snapshot; later process-global writes do not alter it.
	nnfParamsMust(t, g.SourceSetName("nnf.max_memory", "23"))
	max, e = p.MaxMemory()
	nnfParamsMust(t, e)
	mode, e = p.Mode()
	nnfParamsMust(t, e)
	spelling, e = sy.String(mode)
	nnfParamsMust(t, e)
	if max != 17 || spelling != "full" {
		t.Fatal("module snapshot", max, spelling)
	}
	key := "max_memory"
	nnfParamsMust(t, c.SetUIntName(local, &key, 31))
	key = "mode"
	quantifiers, e := sy.Intern("quantifiers")
	nnfParamsMust(t, e)
	nnfParamsMust(t, c.SetSymbolName(local, &key, quantifiers))
	max, e = p.MaxMemory()
	nnfParamsMust(t, e)
	mode, e = p.Mode()
	nnfParamsMust(t, e)
	spelling, e = sy.String(mode)
	nnfParamsMust(t, e)
	if max != 31 || spelling != "quantifiers" {
		t.Fatal("local precedence", max, spelling)
	}
	copyOwner := *p
	if _, e = copyOwner.MaxMemory(); e == nil {
		t.Fatal("copied facade accepted")
	}
	nnfParamsMust(t, p.Close())
	if _, e = p.Skolemize(); e == nil {
		t.Fatal("closed facade accepted")
	}
}
