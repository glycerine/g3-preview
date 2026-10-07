// Source: src/tactic/model_converter.cpp:127-151,164-174;
// src/util/buffer.h:25-48,68-82,94-96; src/util/vector.h:443-447.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func TestTacticStoredModelRealSATBindingOwnershipAndReplacement(t *testing.T) {
	m := astTestManager(t)
	s, _, _ := satSolverCheckFixture(t)
	tr, e := newSATGoalTranslator(m, s)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, tr.Close()) })
	sort, e := m.MkBoolSort()
	satOK(t, e)
	p, e := m.MkConst(astTestSymbol(t, m, "stored-p"), sort)
	satOK(t, e)
	satOK(t, tr.Compile([]ASTHandle{p}))
	result, e := s.Check(nil)
	satOK(t, e)
	if result != lTrue {
		t.Fatal(result)
	}
	values, e := s.AppendModel(nil)
	satOK(t, e)
	md, e := buildSATBooleanModel(tr, values)
	satOK(t, e)
	d, e := m.GetDecl(p)
	satOK(t, e)
	c, e := newTacticStoredModel(md)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	if md.references != 1 {
		t.Fatal("wrapper did not retain model", md.references)
	}
	satOK(t, tr.Close())
	satOK(t, s.Close())
	if _, e = m.GetKind(p); e == nil {
		t.Fatal("test unexpectedly retained the atom")
	}
	old, e := newModelCore(m)
	satOK(t, e)
	var out tacticModelRef
	t.Cleanup(func() { satOK(t, out.Close()); satOK(t, old.Close()) })
	satOK(t, out.Set(old))
	satOK(t, c.ApplyModel(&out, ^uint32(0)))
	if !old.closed {
		t.Fatal("replacement leaked old model")
	}
	satOK(t, c.ApplyUnindexed(&out))
	got, e := out.Get()
	satOK(t, e)
	if got != md || md.references != 2 {
		t.Fatal("stored assignment/alias ownership", got, md.references)
	}
	truth, e := m.MkTrue()
	satOK(t, e)
	binding, e := got.ConstantInterpretation(d)
	satOK(t, e)
	if binding != truth {
		t.Fatal("retrieved model violates original assertion")
	}
	satOK(t, c.Cancel())
	satOK(t, out.Close())
	if md.closed || md.references != 1 {
		t.Fatal("stored model lost before wrapper")
	}
	satOK(t, c.IncRef())
	satOK(t, c.DecRef())
	if !md.closed {
		t.Fatal("wrapper final DEC did not retire stored model")
	}
	if _, e = m.GetKind(d); e == nil {
		t.Fatal("stored declaration leaked")
	}
}

func TestTacticStoredModelLabelsSnapshotInlineSpillAndAppend(t *testing.T) {
	for _, n := range []int{0, 16, 17, 40} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			m := astTestManager(t)
			md := modelTestCore(t, m)
			labels := make([]Symbol, n)
			for i := range labels {
				labels[i] = astTestSymbol(t, m, fmt.Sprintf("stored-label-%d", i))
			}
			snapshot := append([]Symbol(nil), labels...)
			prefix := astTestSymbol(t, m, "prefix")
			before := m.budget.Used()
			satOK(t, md.IncRef())
			c, e := newTacticStoredModelWithLabels(md, labels)
			satOK(t, e)
			t.Cleanup(func() { satOK(t, c.Close()) })
			clear(labels)
			out := tacticModelLabels{budget: m.budget}
			t.Cleanup(func() { out.Close() })
			satOK(t, out.Append(prefix))
			satOK(t, c.ApplyLabels(&out, 123))
			satOK(t, c.ApplyLabels(&out, 0))
			want := append([]Symbol{prefix}, snapshot...)
			want = append(want, snapshot...)
			if !reflect.DeepEqual(out.values, want) {
				t.Fatal("stored labels did not copy/append", out.values, want)
			}
			satOK(t, c.Close())
			out.Close()
			if md.closed || md.references != 1 {
				t.Fatal("wrapper released external model reference")
			}
			if m.budget.Used() != before {
				t.Fatal("stored label buffer/header leak", before, m.budget.Used())
			}
			satOK(t, md.DecRef())
		})
	}
}

func TestTacticStoredModelLabelAppendRefusalKeepsSourcePrefix(t *testing.T) {
	m := astTestManager(t)
	md := modelTestCore(t, m)
	labels := []Symbol{astTestSymbol(t, m, "first"), astTestSymbol(t, m, "second"), astTestSymbol(t, m, "third")}
	c, e := newTacticStoredModelWithLabels(md, labels)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	out := tacticModelLabels{budget: m.budget}
	t.Cleanup(func() { out.Close() })
	prefix := astTestSymbol(t, m, "prefix")
	satOK(t, out.Append(prefix))
	m.budget.FailAfter(0)
	t.Cleanup(func() { m.budget.FailAfter(-1) })
	e = c.ApplyLabels(&out, 0)
	if e != arena.ErrInjectedFailure {
		t.Fatal("destination growth refusal lost", e)
	}
	if !reflect.DeepEqual(out.values, []Symbol{prefix, labels[0]}) {
		t.Fatal("source elementwise append prefix lost", out.values)
	}
	if md.references != 1 || md.closed {
		t.Fatal("label refusal changed stored-model ownership")
	}
	m.budget.FailAfter(-1)
}

func TestTacticStoredModelNullShortcutsCompositionAndOwnerGuards(t *testing.T) {
	m := astTestManager(t)
	m.budget.FailAfter(0)
	c, e := newTacticStoredModel(nil)
	satOK(t, e)
	if c != nil {
		t.Fatal("null model allocated wrapper")
	}
	c, e = newTacticStoredModelWithLabels(nil, []Symbol{{}})
	satOK(t, e)
	if c != nil {
		t.Fatal("null model/labels allocated wrapper")
	}
	m.budget.FailAfter(-1)
	a := modelTestCore(t, m)
	b := modelTestCore(t, m)
	ca, e := newTacticStoredModel(a)
	satOK(t, e)
	cb, e := newTacticStoredModel(b)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, ca.Close()); satOK(t, cb.Close()) })
	combined, e := concatTacticModels(m.budget, ca, cb)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, combined.Close()) })
	var out tacticModelRef
	t.Cleanup(func() { satOK(t, out.Close()) })
	satOK(t, combined.ApplyModel(&out, 9))
	got, e := out.Get()
	satOK(t, e)
	if got != a {
		t.Fatal("concat stored-model precedence")
	}
	concrete := ca.(*tacticStoredModelConverter)
	copyConverter := *concrete
	if e = copyConverter.ApplyModel(&out, 0); e != errTacticModelConverterOwner {
		t.Fatal("copied wrapper accepted", e)
	}
	satOK(t, combined.Close())
	if !b.closed || a.closed {
		t.Fatal("composed stored-model lifetimes", a.closed, b.closed)
	}
	satOK(t, out.Close())
	if !a.closed {
		t.Fatal("retrieved model reference leaked")
	}
	if e = ca.ApplyUnindexed(&out); e != errTacticModelConverterOwner {
		t.Fatal("retired wrapper accepted", e)
	}
}
