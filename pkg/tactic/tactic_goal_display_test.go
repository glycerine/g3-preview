package tactic

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func goalDisplayBindParams(t *testing.T, m *ast.ASTManager) {
	t.Helper()
	b := m.BudgetField
	ctx, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: b})
	astTestMust(t, err)
	catalog, err := uparams.NewGlobalParamsCatalog(m.Symbols, uparams.GlobalParamsCatalogConfig{Budget: b})
	astTestMust(t, err)
	global, err := uparams.NewGlobalParamsService(m.Symbols, ctx, uparams.GlobalParamsConfig{Budget: b, Catalog: catalog})
	astTestMust(t, err)
	astTestMust(t, global.InitializeSource())
	previous, err := uparams.BindProcessGlobalParamsService(global)
	astTestMust(t, err)
	t.Cleanup(func() {
		_, e := uparams.BindProcessGlobalParamsService(previous)
		astTestMust(t, e)
		astTestMust(t, global.Close())
		astTestMust(t, catalog.Close())
		astTestMust(t, ctx.Close())
	})
}

func TestGoalSourceDefaultDisplayAndProcessPrinter(t *testing.T) {
	m := astTestManager(t)
	g := goalTestNew(t, m, false, false)
	var out bytes.Buffer
	astTestMust(t, g.Display(&out))
	if out.String() != "(goal)\n" {
		t.Fatalf("empty source goal = %q", out.String())
	}
	p := goalTestAtom(t, m, "display-p")
	q := goalTestAtom(t, m, "display-q")
	notQ := goalTestApp(t, m, ast.ASTOpNot, q)
	astTestMust(t, g.Assert(p))
	astTestMust(t, g.Assert(notQ))
	out.Reset()
	if err := g.Display(&out); !errors.Is(err, uparams.ErrGlobalParamsOwner) || out.String() != "(goal\n  " {
		t.Fatalf("missing source process printer: output=%q err=%v", out.String(), err)
	}
	goalDisplayBindParams(t, m)
	out.Reset()
	astTestMust(t, g.Display(&out))
	if out.String() != "(goal\n  display-p\n  (not display-q))\n" {
		t.Fatalf("source goal display = %q", out.String())
	}
	if err := g.Display(shortGoalWriter{}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short borrowed writer: %v", err)
	}
}

type shortGoalWriter struct{}

func (shortGoalWriter) Write(p []byte) (int, error) { return 0, nil }
