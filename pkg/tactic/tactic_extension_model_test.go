// Source: src/tactic/extension_model_converter.cpp:49-85;
// src/tactic/model_converter.h:30-41.
package tactic

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util"
	"testing"
)

func extensionFixture(t *testing.T) (*ast.ASTManager, *model.Model, *ModelRef, model.ModelEvaluatorConfig) {
	ctx, sy, q := paramsTestOwner(t)
	budget := arena.NewBudget(^uint64(0))
	catalog, e := util.NewGlobalParamsCatalog(sy, util.GlobalParamsCatalogConfig{Budget: budget, Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: budget}}, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: budget}}})
	satOK(t, e)
	globals, e := util.NewGlobalParamsService(sy, ctx, util.GlobalParamsConfig{Budget: budget, Catalog: catalog, DescriptorPageSize: 2, DescriptorWordsPageSize: 16})
	satOK(t, e)
	satOK(t, globals.InitializeSource())
	memory, e := util.NewRuntime(util.RuntimeConfig{Budget: budget})
	satOK(t, e)
	cooperate, e := util.NewCooperation(budget)
	satOK(t, e)
	t.Cleanup(func() {
		satOK(t, cooperate.Close())
		satOK(t, memory.Close())
		satOK(t, globals.Close())
		satOK(t, catalog.Close())
		if budget.Used() != 0 {
			t.Fatal("extension fixture leak", budget.Used())
		}
	})
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: budget})
	satOK(t, e)
	t.Cleanup(func() { satOK(t, m.Close()) })
	md, e := model.NewModel(m)
	satOK(t, e)
	out := &ModelRef{}
	satOK(t, out.Set(md.Core()))
	t.Cleanup(func() { satOK(t, out.Close()) })
	return m, md, out, model.ModelEvaluatorConfig{Context: ctx, Globals: globals, Memory: memory, Cooperation: cooperate}
}
func TestTacticExtensionModelOriginalReverseDefinitionOrder(t *testing.T) {
	m, md, out, config := extensionFixture(t)
	c, e := NewExtensionModel(m, config)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	a := goalTestAtom(t, m, "extension-a")
	b := goalTestAtom(t, m, "extension-b")
	da, e := m.GetDecl(a)
	satOK(t, e)
	db, e := m.GetDecl(b)
	satOK(t, e)
	yes, e := m.MkTrue()
	satOK(t, e)
	satOK(t, c.Insert(da, b))
	satOK(t, c.Insert(db, yes))
	satOK(t, c.ApplyModel(out, 0))
	for _, d := range []ast.ASTHandle{da, db} {
		value, e := md.Core().ConstantInterpretation(d)
		satOK(t, e)
		if value != yes {
			t.Fatal("definitions must run in reverse order", d, value)
		}
	}
	got, e := out.Get()
	satOK(t, e)
	if got != md.Core() {
		t.Fatal("extension unexpectedly replaced the model")
	}
}
func TestTacticExtensionModelFunctionFallbackCompletionAndDefaultCallbacks(t *testing.T) {
	m, md, out, config := extensionFixture(t)
	c, e := NewExtensionModel(m, config)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	b, e := m.MkBoolSort()
	satOK(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "extension-function"), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	satOK(t, e)
	v, e := m.MkVar(0, b)
	satOK(t, e)
	astTestPin(t, m, v)
	atom := goalTestAtom(t, m, "extension-unknown")
	body, e := m.MkOr(v, atom)
	satOK(t, e)
	astTestPin(t, m, body)
	satOK(t, c.Insert(d, body))
	satOK(t, c.ApplyUnindexed(out)) // source inherited unindexed operation is empty
	if md.Core().NumFunctionDeclarations() != 0 {
		t.Fatal("unindexed base callback was not empty")
	}
	satOK(t, c.ApplyLabels(nil, 99))
	satOK(t, c.Cancel())
	satOK(t, c.ApplyModel(out, 0))
	fi, e := md.Core().FunctionInterpretation(d)
	satOK(t, e)
	if fi == nil || fi.NumEntries() != 0 {
		t.Fatal("extension function not registered")
	}
	fallback, e := fi.Else()
	satOK(t, e)
	if fallback != v {
		t.Fatal("completed and reduced function fallback", fallback)
	}
	decl, e := m.GetDecl(atom)
	satOK(t, e)
	value, e := md.Core().ConstantInterpretation(decl)
	satOK(t, e)
	no, e := m.MkFalse()
	satOK(t, e)
	if value != no {
		t.Fatal("model completion was disabled")
	}
}

func TestTacticExtensionInvocationRefusalsCleanupAndReuse(t *testing.T) {
	m, _, out, config := extensionFixture(t)
	c, e := NewExtensionModel(m, config)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	b, e := m.MkBoolSort()
	satOK(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "extension-refused-function"), []ast.ASTHandle{b}, b, ast.ASTFuncDeclInfo{})
	satOK(t, e)
	v, e := m.MkVar(0, b)
	satOK(t, e)
	astTestPin(t, m, v)
	satOK(t, c.Insert(d, v))
	satOK(t, c.ApplyModel(out, 0))
	satOK(t, out.Set(nil))
	baseline := m.Budget().Used()
	for fault := int64(0); fault < 256; fault++ {
		md, e := model.NewModel(m)
		satOK(t, e)
		var r ModelRef
		satOK(t, r.Set(md.Core()))
		m.Budget().FailAfter(fault)
		body := c.ApplyModel(&r, 0)
		m.Budget().FailAfter(-1)
		// A stopped Go lexical destructor stays named in the converter until drain.
		satOK(t, c.drainInvocation())
		satOK(t, r.Close())
		if m.Budget().Used() != baseline {
			t.Fatal("invocation owner leak", fault, baseline, m.Budget().Used())
		}
		if body == nil {
			return
		}
	}
	t.Fatal("invocation never completed")
}
func TestTacticExtensionOwningTrailsCopiedClosedAndGoalGuards(t *testing.T) {
	m, _, out, config := extensionFixture(t)
	c, e := NewExtensionModel(m, config)
	satOK(t, e)
	d := modelTestDecl(t, m, "extension-owned-declaration")
	yes, e := m.MkTrue()
	satOK(t, e)
	beforeD, e := m.GetRefCount(d)
	satOK(t, e)
	beforeV, e := m.GetRefCount(yes)
	satOK(t, e)
	for i := 0; i < 20; i++ {
		satOK(t, c.Insert(d, yes))
	}
	afterD, e := m.GetRefCount(d)
	satOK(t, e)
	afterV, e := m.GetRefCount(yes)
	satOK(t, e)
	if afterD != beforeD+20 || afterV != beforeV+20 {
		t.Fatal("owning duplicate trails", beforeD, afterD, beforeV, afterV)
	}
	copied := *c
	if e = copied.ApplyModel(out, 0); e != ErrModelConverterOwner {
		t.Fatal("copied converter", e)
	}
	e = c.ApplyModel(out, 1)
	if tacticSourceDebug {
		if e != errTacticModelConverterGoal {
			t.Fatal("source goal assertion", e)
		}
	} else {
		satOK(t, e)
	}
	satOK(t, c.Close())
	afterV, e = m.GetRefCount(yes)
	satOK(t, e)
	// Native invocation may register one additional model-owned constant value.
	expected := beforeV
	if !tacticSourceDebug {
		expected++
	}
	if afterV != expected {
		t.Fatal("expression trail cleanup", afterV, expected)
	}
	if e = c.ApplyModel(out, 0); e != ErrModelConverterOwner {
		t.Fatal("closed converter", e)
	}
}
