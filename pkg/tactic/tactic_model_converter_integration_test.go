package

// Source: src/tactic/model_converter.cpp:23-126,183-187;
// src/tactic/converter.h:26-66,96-124; src/util/ref.h:26-90.
// Licensed under the MIT license. Source author: Leonardo de Moura.
tactic_test

import (
	"errors"
	sattactic "github.com/glycerine/g3/pkg/sat/tactic"
	"github.com/glycerine/g3/pkg/util/lbool"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"

	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
)

func TestTacticModelConcatActualModelReverseOrderLabelsAndLifetime(t *testing.T) {
	m := astTestManager(t)
	s, _, _ := satSolverCheckFixture(t)
	tr, e := sattactic.NewGoalTranslator(m, s)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, tr.Close()) })
	p := goalTestAtom(t, m, "converter-p")
	satOK(t, tr.Compile([]ast.ASTHandle{p}))
	result, e := s.Check(nil)
	satOK(t, e)
	if result != lbool.True {
		t.Fatal("real solver failed", result)
	}
	values, e := s.AppendModel(nil)
	satOK(t, e)
	md, e := sattactic.BuildBooleanModel(tr, values)
	satOK(t, e)
	var r tactic.ModelRef
	satOK(t, r.Set(md))
	t.Cleanup(func() { satOK(t, r.Close()) })
	d, e := m.GetDecl(p)
	satOK(t, e)
	truth, e := m.MkTrue()
	satOK(t, e)
	falsity, e := m.MkFalse()
	satOK(t, e)
	calls := []string{}
	first := &modelConverterTestLeaf{name: "first", calls: &calls, label: astTestSymbol(t, m, "first-label")}
	second := &modelConverterTestLeaf{name: "second", calls: &calls, label: astTestSymbol(t, m, "second-label")}
	second.run = func(r *tactic.ModelRef, _ uint32) error {
		model, e := r.Get()
		if e != nil {
			return e
		}
		return model.RegisterConstant(d, falsity)
	}
	first.run = func(r *tactic.ModelRef, _ uint32) error {
		model, e := r.Get()
		if e != nil {
			return e
		}
		v, e := model.ConstantInterpretation(d)
		if e != nil {
			return e
		}
		if v != falsity {
			return errors.New("first ran before second")
		}
		return model.RegisterConstant(d, truth)
	}
	c, e := tactic.ConcatModels(m.BudgetField, first, second)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	satOK(t, c.IncRef())
	satOK(t, c.ApplyModel(&r, 7))
	satOK(t, c.ApplyUnindexed(&r))
	var labels tactic.ModelLabels
	labels.Budget = m.BudgetField
	t.Cleanup(func() { labels.Close() })
	prefix := astTestSymbol(t, m, "prefix")
	satOK(t, labels.Append(prefix))
	satOK(t, c.ApplyLabels(&labels, 9))
	satOK(t, c.Cancel())
	if !reflect.DeepEqual(labels.Values, []usymbol.Symbol{prefix, second.label, first.label}) {
		t.Fatal("label append order", labels.Values)
	}
	satOK(t, c.DecRef())
	want := []string{"model:second:7", "model:first:0", "unindexed:second", "unindexed:first", "labels:second:9", "labels:first:0", "cancel:second", "cancel:first", "destroy:second", "destroy:first"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatal(calls, want)
	}
	got, e := md.ConstantInterpretation(d)
	satOK(t, e)
	if got != truth {
		t.Fatal("converter sequence corrupted model")
	}
}
