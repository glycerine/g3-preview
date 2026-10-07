// Source: src/tactic/model_converter.cpp:23-126,183-187;
// src/tactic/converter.h:26-66,96-124; src/util/ref.h:26-90.
package tactic

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/model"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// A virtual-boundary fixture records dispatch and can mutate the actual
// constant model. It is not a production converter or a solver implementation.

type modelConverterTestLeaf struct {
	name   string
	calls  *[]string
	refs   uint32
	closed bool
	label  util.Symbol
	run    func(*ModelRef, uint32) error
	fail   error
}

func (c *modelConverterTestLeaf) IncRef() error {
	if c.closed {
		return ErrModelConverterOwner
	}
	c.refs++
	return nil
}
func (c *modelConverterTestLeaf) DecRef() error {
	if c.closed || c.refs == 0 {
		return ErrModelConverterOwner
	}
	c.refs--
	if c.refs == 0 {
		return c.Close()
	}
	return nil
}
func (c *modelConverterTestLeaf) Close() error {
	if !c.closed {
		c.closed = true
		*c.calls = append(*c.calls, "destroy:"+c.name)
	}
	return nil
}
func (c *modelConverterTestLeaf) ApplyModel(m *ModelRef, i uint32) error {
	*c.calls = append(*c.calls, fmt.Sprintf("model:%s:%d", c.name, i))
	if c.fail != nil {
		return c.fail
	}
	if c.run != nil {
		return c.run(m, i)
	}
	return nil
}
func (c *modelConverterTestLeaf) ApplyUnindexed(m *ModelRef) error {
	*c.calls = append(*c.calls, "unindexed:"+c.name)
	if c.fail != nil {
		return c.fail
	}
	if c.run != nil {
		return c.run(m, 0)
	}
	return nil
}
func (c *modelConverterTestLeaf) ApplyLabels(l *ModelLabels, i uint32) error {
	*c.calls = append(*c.calls, fmt.Sprintf("labels:%s:%d", c.name, i))
	if c.fail != nil {
		return c.fail
	}
	return l.Append(c.label)
}
func (c *modelConverterTestLeaf) Cancel() error {
	*c.calls = append(*c.calls, "cancel:"+c.name)
	return nil
}

func TestTacticModelReferenceOwningAliasAndReplacement(t *testing.T) {
	m := astTestManager(t)
	a, e := model.NewModelCore(m)
	satOK(t, e)
	b, e := model.NewModelCore(m)
	satOK(t, e)
	var r ModelRef
	t.Cleanup(func() { satOK(t, r.Close()); satOK(t, a.Close()); satOK(t, b.Close()) })
	satOK(t, r.Set(a))
	satOK(t, r.Set(a))
	if a.References != 1 || a.Closed {
		t.Fatal("alias assignment retired the model", a.References, a.Closed)
	}
	satOK(t, r.Set(b))
	if !a.Closed || b.References != 1 {
		t.Fatal("replacement ownership")
	}
	satOK(t, b.IncRef())
	satOK(t, r.Close())
	if b.Closed || b.References != 1 {
		t.Fatal("external reference lost")
	}
	satOK(t, b.DecRef())
	if !b.Closed {
		t.Fatal("final reference did not destroy model")
	}
	copyRef := r
	if e = copyRef.Set(nil); e != ErrModelConverterOwner {
		t.Fatal("copied owning model reference accepted", e)
	}
}

func TestTacticModelConcatStarSubgoalDispatchAndDestruction(t *testing.T) {
	m := astTestManager(t)
	calls := []string{}
	root := &modelConverterTestLeaf{name: "root", calls: &calls}
	a := &modelConverterTestLeaf{name: "a", calls: &calls}
	b := &modelConverterTestLeaf{name: "b", calls: &calls}
	c, e := concatTacticModelSubgoals(m.BudgetField, root, []ModelConverter{a, nil, b}, []uint32{2, 0, 3})
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	var model ModelRef
	for i := uint32(0); i < 5; i++ {
		satOK(t, c.ApplyModel(&model, i))
	}
	want := []string{"model:a:0", "model:root:0", "model:a:1", "model:root:0", "model:b:0", "model:root:2", "model:b:1", "model:root:2", "model:b:2", "model:root:2"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatal("flattened subgoal mapping", calls, want)
	}
	if e = c.ApplyModel(&model, 5); e != errTacticModelConverterGoal {
		t.Fatal("invalid goal accepted", e)
	}
	if e = c.ApplyUnindexed(&model); e != errTacticModelConverterGoal {
		t.Fatal("source UNREACHABLE accepted", e)
	}
	labels := ModelLabels{Budget: m.BudgetField}
	t.Cleanup(func() { labels.Close() })
	a.label = astTestSymbol(t, m, "a-label")
	b.label = astTestSymbol(t, m, "b-label")
	root.label = astTestSymbol(t, m, "root-label")
	satOK(t, c.ApplyLabels(&labels, 4))
	if !reflect.DeepEqual(labels.Values, []util.Symbol{b.label, root.label}) {
		t.Fatal("star labels", labels.Values)
	}
	calls = nil
	satOK(t, c.Cancel())
	satOK(t, c.Close())
	want = []string{"cancel:root", "cancel:a", "cancel:b", "destroy:a", "destroy:b", "destroy:root"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatal("source star cancellation/destruction order", calls, want)
	}
}

func TestTacticModelConcatShortcutsNullBranchesAndSharedOwners(t *testing.T) {
	m := astTestManager(t)
	calls := []string{}
	leaf := &modelConverterTestLeaf{name: "shared", calls: &calls}
	m.BudgetField.FailAfter(0)
	a, e := ConcatModels(m.BudgetField, nil, leaf)
	satOK(t, e)
	if a != leaf || leaf.refs != 0 {
		t.Fatal("null shortcut retained/allocated")
	}
	a, e = concatTacticModelSubgoals(m.BudgetField, leaf, []ModelConverter{nil, nil}, nil)
	satOK(t, e)
	if a != leaf {
		t.Fatal("all-null shortcut")
	}
	a, e = concatTacticModelSubgoals(m.BudgetField, nil, []ModelConverter{leaf}, nil)
	satOK(t, e)
	if a != leaf {
		t.Fatal("single-subgoal shortcut")
	}
	a, e = ConcatModels(m.BudgetField, nil, nil)
	satOK(t, e)
	if a != nil {
		t.Fatal("null concat")
	}
	a, e = concatTacticModelSubgoals(m.BudgetField, leaf, nil, nil)
	if tacticSourceDebug {
		if e != errTacticModelConverterGoal {
			t.Fatal("source zero-count SASSERT missing", e)
		}
	} else if e != nil || a != leaf || leaf.refs != 0 {
		t.Fatal("native zero-count all-null return", a, e, leaf.refs)
	}
	m.BudgetField.FailAfter(-1)
	c, e := ConcatModels(m.BudgetField, leaf, leaf)
	satOK(t, e)
	if leaf.refs != 2 {
		t.Fatal("two source refs not retained", leaf.refs)
	}
	satOK(t, c.Close())
	if !reflect.DeepEqual(calls, []string{"destroy:shared"}) {
		t.Fatal("shared child destroyed more than once", calls)
	}
	calls = nil
	b := &modelConverterTestLeaf{name: "branch", calls: &calls}
	c, e = concatTacticModelSubgoals(m.BudgetField, nil, []ModelConverter{nil, b}, []uint32{2, 1})
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	var model ModelRef
	satOK(t, c.ApplyModel(&model, 1))
	if len(calls) != 0 {
		t.Fatal("null branch dispatched")
	}
	satOK(t, c.ApplyModel(&model, 2))
	if !reflect.DeepEqual(calls, []string{"model:branch:0"}) {
		t.Fatal(calls)
	}
}

func TestTacticModelConcatStopsAtChildErrorAndRejectsCopiedOwner(t *testing.T) {
	m := astTestManager(t)
	calls := []string{}
	sentinel := errors.New("child converter error")
	a := &modelConverterTestLeaf{name: "a", calls: &calls}
	b := &modelConverterTestLeaf{name: "b", calls: &calls, fail: sentinel}
	c, e := ConcatModels(m.BudgetField, a, b)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, c.Close()) })
	var model ModelRef
	if e = c.ApplyModel(&model, 3); e != sentinel {
		t.Fatal("child error lost", e)
	}
	if !reflect.DeepEqual(calls, []string{"model:b:3"}) {
		t.Fatal("first ran after second failed", calls)
	}
	original := c.(*tacticConcatModelConverter)
	copyConverter := *original
	if e = copyConverter.IncRef(); e != ErrModelConverterOwner {
		t.Fatal("copied converter accepted", e)
	}
	satOK(t, c.Close())
	if e = c.ApplyModel(&model, 0); e != ErrModelConverterOwner {
		t.Fatal("retired converter accepted", e)
	}
	if _, e = concatTacticModelSubgoals(m.BudgetField, nil, []ModelConverter{a, b}, nil); e != errTacticModelConverterGoal {
		t.Fatal("required star size table missing", e)
	}
}

func TestTacticModelConcatStarCopiesVectorsAndRetainsExternalChild(t *testing.T) {
	m := astTestManager(t)
	calls := []string{}
	root := &modelConverterTestLeaf{name: "root", calls: &calls}
	child := &modelConverterTestLeaf{name: "child", calls: &calls}
	satOK(t, child.IncRef())
	seconds := make([]ModelConverter, 40)
	sizes := make([]uint32, 40)
	for i := range seconds {
		seconds[i], sizes[i] = child, 2
	}
	before := m.BudgetField.Used()
	c, e := concatTacticModelSubgoals(m.BudgetField, root, seconds, sizes)
	satOK(t, e)
	t.Cleanup(func() {
		satOK(t, c.Close())
		if !child.closed {
			satOK(t, child.DecRef())
		}
	})
	if child.refs != 41 {
		t.Fatal("source constructor did not retain every slot", child.refs)
	}
	// Source vectors copy pointer/size values; later caller assignments cannot
	// change the converter's dispatch table or its reference ownership.
	clear(seconds)
	clear(sizes)
	var model ModelRef
	satOK(t, applyTacticModelConverter(c, &model, 79))
	if !reflect.DeepEqual(calls, []string{"model:child:1", "model:root:39"}) {
		t.Fatal("caller mutation changed source vectors", calls)
	}
	satOK(t, applyTacticModelConverter(nil, nil, 99))
	satOK(t, c.Close())
	if child.closed || child.refs != 1 || !root.closed {
		t.Fatal("star destroyed external child or leaked first", child.refs, child.closed, root.closed)
	}
	if m.BudgetField.Used() != before {
		t.Fatal("converter header/vector backing leaked", before, m.BudgetField.Used())
	}
	satOK(t, child.DecRef())
	if !reflect.DeepEqual(calls, []string{"model:child:1", "model:root:39", "destroy:root", "destroy:child"}) {
		t.Fatal("external child lifetime", calls)
	}
}
