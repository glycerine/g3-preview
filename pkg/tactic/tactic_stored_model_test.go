package tactic

// Source: src/tactic/model_converter.cpp:127-151,164-174;
// src/util/buffer.h:25-48,68-82,94-96; src/util/vector.h:443-447.

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestTacticStoredModelDisplayAndTranslate(t *testing.T) {
	from := astTestManager(t)
	goalDisplayBindParams(t, from)
	to, err := ast.NewASTManager(ast.ASTManagerConfig{
		Symbols: from.Symbols, Numerals: from.NumeralService(), Budget: from.Budget(),
	})
	satOK(t, err)
	t.Cleanup(func() { satOK(t, to.Close()) })
	md, err := model.NewModel(from)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, md.Close()) })
	a := goalTestAtom(t, from, "stored-translated")
	d, err := from.GetDecl(a)
	satOK(t, err)
	yes, err := from.MkTrue()
	satOK(t, err)
	satOK(t, md.Core().RegisterConstant(d, yes))
	label := astTestSymbol(t, from, "stored-label")
	base, err := newTacticStoredModelWithLabels(md.Core(), []usymbol.Symbol{label})
	satOK(t, err)
	c := base.(*tacticStoredModelConverter)
	t.Cleanup(func() { satOK(t, c.Close()) })
	var display bytes.Buffer
	satOK(t, c.Display(&display))
	if got, want := display.String(), "(model->model-converter-wrapper\nstored-translated -> true\n)\n"; got != want {
		t.Fatalf("display = %q; want %q", got, want)
	}
	translator, err := ast.NewASTTranslation(from, to, true)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, translator.Close()) })
	translated, err := c.Translate(translator)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, translated.Close()) })
	var out ModelRef
	t.Cleanup(func() { satOK(t, out.Close()) })
	satOK(t, translated.ApplyModel(&out, 0))
	got, err := out.Get()
	satOK(t, err)
	if got.Manager() != to {
		t.Fatal("translated model manager")
	}
	wantDeclaration, err := translator.Translate(d)
	satOK(t, err)
	wantValue, err := translator.Translate(yes)
	satOK(t, err)
	value, err := got.ConstantInterpretation(wantDeclaration)
	satOK(t, err)
	if value != wantValue {
		t.Fatal("translated model binding", value, wantValue)
	}
	var labels ModelLabels
	satOK(t, translated.ApplyLabels(&labels, 0))
	if !reflect.DeepEqual(labels.Values, []usymbol.Symbol{label}) {
		t.Fatal("translated labels", labels.Values)
	}
}

func TestTacticStoredModelLabelsSnapshotAndAppend(t *testing.T) {
	for _, n := range []int{0, 16, 17, 40} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			m := astTestManager(t)
			md := modelTestCore(t, m)
			labels := make([]usymbol.Symbol, n)
			for i := range labels {
				labels[i] = astTestSymbol(t, m, fmt.Sprintf("stored-label-%d", i))
			}
			snapshot := append([]usymbol.Symbol(nil), labels...)
			prefix := astTestSymbol(t, m, "prefix")
			before := m.BudgetField.Used()
			satOK(t, md.IncRef())
			c, e := newTacticStoredModelWithLabels(md, labels)
			satOK(t, e)
			t.Cleanup(func() { satOK(t, c.Close()) })
			clear(labels)
			out := ModelLabels{Budget: m.BudgetField}
			t.Cleanup(func() { out.Close() })
			satOK(t, out.Append(prefix))
			satOK(t, c.ApplyLabels(&out, 123))
			satOK(t, c.ApplyLabels(&out, 0))
			want := append([]usymbol.Symbol{prefix}, snapshot...)
			want = append(want, snapshot...)
			if !reflect.DeepEqual(out.Values, want) {
				t.Fatal("stored labels did not copy/append", out.Values, want)
			}
			satOK(t, c.Close())
			out.Close()
			if md.Closed || md.References != 1 {
				t.Fatal("wrapper released external model reference")
			}
			if m.BudgetField.Used() != before {
				t.Fatal("stored label buffer/header leak", before, m.BudgetField.Used())
			}
			satOK(t, md.DecRef())
		})
	}
}

func TestTacticStoredModelLabelAppendUsesNativeSliceStorage(t *testing.T) {
	m := astTestManager(t)
	md := modelTestCore(t, m)
	labels := []usymbol.Symbol{astTestSymbol(t, m, "first"), astTestSymbol(t, m, "second"), astTestSymbol(t, m, "third")}
	c, e := newTacticStoredModelWithLabels(md, labels)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	out := ModelLabels{Budget: m.BudgetField}
	t.Cleanup(func() { out.Close() })
	prefix := astTestSymbol(t, m, "prefix")
	satOK(t, out.Append(prefix))
	m.BudgetField.FailAfter(0)
	t.Cleanup(func() { m.BudgetField.FailAfter(-1) })
	e = c.ApplyLabels(&out, 0)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(out.Values, []usymbol.Symbol{prefix, labels[0], labels[1], labels[2]}) {
		t.Fatal("label order", out.Values)
	}
	if md.References != 1 || md.Closed {
		t.Fatal("label refusal changed stored-model ownership")
	}
	m.BudgetField.FailAfter(-1)
}

func TestTacticStoredModelNullShortcutsCompositionAndOwnerGuards(t *testing.T) {
	m := astTestManager(t)
	m.BudgetField.FailAfter(0)
	c, e := NewStoredModel(nil)
	satOK(t, e)
	if c != nil {
		t.Fatal("null model allocated wrapper")
	}
	c, e = newTacticStoredModelWithLabels(nil, []usymbol.Symbol{{}})
	satOK(t, e)
	if c != nil {
		t.Fatal("null model/labels allocated wrapper")
	}
	m.BudgetField.FailAfter(-1)
	a := modelTestCore(t, m)
	b := modelTestCore(t, m)
	ca, e := NewStoredModel(a)
	satOK(t, e)
	cb, e := NewStoredModel(b)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, ca.Close()); satOK(t, cb.Close()) })
	combined, e := ConcatModels(m.BudgetField, ca, cb)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, combined.Close()) })
	var out ModelRef
	t.Cleanup(func() { satOK(t, out.Close()) })
	satOK(t, combined.ApplyModel(&out, 9))
	got, e := out.Get()
	satOK(t, e)
	if got != a {
		t.Fatal("concat stored-model precedence")
	}
	concrete := ca.(*tacticStoredModelConverter)
	copyConverter := *concrete
	if e = copyConverter.ApplyModel(&out, 0); e != ErrModelConverterOwner {
		t.Fatal("copied wrapper accepted", e)
	}
	satOK(t, combined.Close())
	if !b.Closed || a.Closed {
		t.Fatal("composed stored-model lifetimes", a.Closed, b.Closed)
	}
	satOK(t, out.Close())
	if !a.Closed {
		t.Fatal("retrieved model reference leaked")
	}
	if e = ca.ApplyUnindexed(&out); e != ErrModelConverterOwner {
		t.Fatal("retired wrapper accepted", e)
	}
}
