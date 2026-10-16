//go:build g3debug

// Source: src/model/func_interp.cpp:200-216; src/util/trace.h:40-46.
package model_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestFunctionInterpretationDuplicateNewEntrySourceTrace(t *testing.T) {
	c, symbols, numerals := paramsTestOwner(t)
	budget := arena.NewBudget(1 << 25)
	catalog, err := uparams.NewGlobalParamsCatalog(symbols, uparams.GlobalParamsCatalogConfig{Budget: budget})
	astTestMust(t, err)
	globals, err := uparams.NewGlobalParamsService(symbols, c, uparams.GlobalParamsConfig{Budget: budget, Catalog: catalog})
	astTestMust(t, err)
	astTestMust(t, globals.InitializeSource())
	previous, err := uparams.BindProcessGlobalParamsService(globals)
	astTestMust(t, err)
	t.Cleanup(func() {
		_, e := uparams.BindProcessGlobalParamsService(previous)
		astTestMust(t, e)
		astTestMust(t, globals.Close())
		astTestMust(t, catalog.Close())
		if budget.Used() != 0 {
			t.Fatalf("trace owners retained %d bytes", budget.Used())
		}
	})
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	f := modelTestFunction(t, m, 1)
	yes, err := m.MkTrue()
	astTestMust(t, err)
	no, err := m.MkFalse()
	astTestMust(t, err)
	astTestPin(t, m, yes)
	astTestPin(t, m, no)
	astTestMust(t, f.InsertNewEntry([]ast.ASTHandle{yes}, no))
	before := f.NumEntries()
	if err = f.InsertNewEntry([]ast.ASTHandle{yes}, yes); !errors.Is(err, ast.ErrASTDomain) || output.Len() != 0 {
		t.Fatalf("disabled duplicate trace: entries=%d err=%v output=%q", f.NumEntries(), err, output.String())
	}
	tags.Enable("func_interp_bug")
	err = f.InsertNewEntry([]ast.ASTHandle{yes}, yes)
	if !errors.Is(err, ast.ErrASTDomain) || f.NumEntries() != before {
		t.Fatalf("duplicate assertion: entries=%d err=%v", f.NumEntries(), err)
	}
	if !tags.ShouldTrace("func_interp_bug") {
		if output.Len() != 0 {
			t.Fatal("erased source trace", output.String())
		}
		return
	}
	want := "-------- [func_interp_bug] insert_new_entry src/model/func_interp.cpp:202 ---------\n" +
		"Old: false\nArgs:true\nNew: true\nArgs:true\nOld: false\n" +
		"------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source duplicate record: got %q, want %q", got, want)
	}
}
