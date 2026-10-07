// Source: src/tactic/model_converter.cpp:127-151,164-174;
// src/util/buffer.h:25-48,68-82,94-96; src/util/vector.h:443-447.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package tactic

import (
	"fmt"

	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	"github.com/glycerine/g3/pkg/util"
)

func TestTacticStoredModelLabelsSnapshotInlineSpillAndAppend(t *testing.T) {
	for _, n := range []int{0, 16, 17, 40} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			m := astTestManager(t)
			md := modelTestCore(t, m)
			labels := make([]util.Symbol, n)
			for i := range labels {
				labels[i] = astTestSymbol(t, m, fmt.Sprintf("stored-label-%d", i))
			}
			snapshot := append([]util.Symbol(nil), labels...)
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
			want := append([]util.Symbol{prefix}, snapshot...)
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

func TestTacticStoredModelLabelAppendRefusalKeepsSourcePrefix(t *testing.T) {
	m := astTestManager(t)
	md := modelTestCore(t, m)
	labels := []util.Symbol{astTestSymbol(t, m, "first"), astTestSymbol(t, m, "second"), astTestSymbol(t, m, "third")}
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
	if e != arena.ErrInjectedFailure {
		t.Fatal("destination growth refusal lost", e)
	}
	if !reflect.DeepEqual(out.Values, []util.Symbol{prefix, labels[0]}) {
		t.Fatal("source elementwise append prefix lost", out.Values)
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
	c, e = newTacticStoredModelWithLabels(nil, []util.Symbol{{}})
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
