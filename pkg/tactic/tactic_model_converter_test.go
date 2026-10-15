// Source: src/tactic/model_converter.cpp:23-126,183-187;
// src/tactic/converter.h:26-66,96-124; src/util/ref.h:26-90.
package tactic

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	"reflect"
	"testing"
)

// A virtual-boundary fixture records dispatch and can mutate the actual
// constant model. It is not a production converter or a solver implementation.

type modelConverterTestLeaf struct {
	name   string
	calls  *[]string
	refs   uint32
	closed bool
	label  usymbol.Symbol
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
	_, e := c.DecRefConsumed()
	return e
}
func (c *modelConverterTestLeaf) DecRefConsumed() (bool, error) {
	if c.closed || c.refs == 0 {
		return false, ErrModelConverterOwner
	}
	c.refs--
	if c.refs == 0 {
		return true, c.Close()
	}
	return true, nil
}
func (c *modelConverterTestLeaf) Close() error {
	if !c.closed {
		c.closed = true
		*c.calls = append(*c.calls, "destroy:"+c.name)
	}
	return nil
}

type modelConverterStoppedRelease struct {
	modelConverterTestLeaf
	stop       error
	cleanupDue bool
	decrements uint32
}

func (c *modelConverterStoppedRelease) DecRef() error {
	_, e := c.DecRefConsumed()
	return e
}
func (c *modelConverterStoppedRelease) DecRefConsumed() (bool, error) {
	if c.closed || c.refs == 0 {
		return false, ErrModelConverterOwner
	}
	c.refs--
	c.decrements++
	if c.refs == 0 {
		c.cleanupDue = true
		return true, c.stop
	}
	return true, nil
}
func (c *modelConverterStoppedRelease) Close() error {
	if !c.cleanupDue {
		return ErrModelConverterOwner
	}
	c.cleanupDue = false
	return c.modelConverterTestLeaf.Close()
}

func TestTacticModelConcatConsumedChildCleanupResumesWithoutSecondDecrement(t *testing.T) {
	m := astTestManager(t)
	calls := []string{}
	stop := errors.New("child cleanup stopped")
	first := &modelConverterTestLeaf{name: "first", calls: &calls}
	second := &modelConverterStoppedRelease{modelConverterTestLeaf: modelConverterTestLeaf{name: "second", calls: &calls}, stop: stop}
	ordinary, e := ConcatModels(m.BudgetField, first, second)
	satOK(t, e)
	if e = ordinary.Close(); !errors.Is(e, stop) || second.refs != 0 || second.decrements != 1 || first.refs != 1 || len(calls) != 0 {
		t.Fatal("ordinary consumed prefix", e, second.refs, second.decrements, first.refs, calls)
	}
	satOK(t, ordinary.Close())
	if second.decrements != 1 || !reflect.DeepEqual(calls, []string{"destroy:second", "destroy:first"}) {
		t.Fatal("ordinary cleanup replayed decrement", second.decrements, calls)
	}

	calls = nil
	root := &modelConverterTestLeaf{name: "root", calls: &calls}
	stopped := &modelConverterStoppedRelease{modelConverterTestLeaf: modelConverterTestLeaf{name: "stopped", calls: &calls}, stop: stop}
	later := &modelConverterTestLeaf{name: "later", calls: &calls}
	star, e := concatTacticModelSubgoals(m.BudgetField, root, []ModelConverter{stopped, later}, []uint32{1, 1})
	satOK(t, e)
	if e = star.Close(); !errors.Is(e, stop) || stopped.refs != 0 || stopped.decrements != 1 || later.refs != 1 || root.refs != 1 || len(calls) != 0 {
		t.Fatal("star consumed prefix", e, stopped.refs, stopped.decrements, later.refs, root.refs, calls)
	}
	satOK(t, star.Close())
	if stopped.decrements != 1 || !reflect.DeepEqual(calls, []string{"destroy:stopped", "destroy:later", "destroy:root"}) {
		t.Fatal("star cleanup replayed decrement", stopped.decrements, calls)
	}
}

func TestTacticModelConcatRealStoredChildFinalReleaseNativeIDRecycle(t *testing.T) {
	m := astTestManager(t)
	md, e := model.NewModel(m)
	satOK(t, e)
	sort, e := m.MkUninterpretedSort(astTestSymbol(t, m, "concat-refusal-sort"), nil)
	satOK(t, e)
	value, e := m.GetSomeValue(sort)
	satOK(t, e)
	satOK(t, md.RegisterUniverse(sort, []ast.ASTHandle{value}))
	stored, e := NewStoredModel(&md.CoreField)
	satOK(t, e)
	calls := []string{}
	first := &modelConverterTestLeaf{name: "first", calls: &calls}
	combined, e := ConcatModels(m.BudgetField, first, stored)
	satOK(t, e)
	m.ExprIDs.Cleanup(m.ExprIDs.Range())
	m.BudgetField.FailAfter(0)
	e = combined.Close()
	m.BudgetField.FailAfter(-1)
	satOK(t, e)
	if first.refs != 0 || !combined.(*tacticConcatModelConverter).closed {
		t.Fatal("real child final release", first.refs)
	}
	if !reflect.DeepEqual(calls, []string{"destroy:first"}) || !stored.(*tacticStoredModelConverter).closed || !md.Closed {
		t.Fatal("real child cleanup order", calls)
	}
	satOK(t, md.Close())
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

func TestTacticModelReferenceCopyAssignmentOwnsSourceModel(t *testing.T) {
	m := astTestManager(t)
	a, e := model.NewModelCore(m)
	satOK(t, e)
	b, e := model.NewModelCore(m)
	satOK(t, e)
	var source, target ModelRef
	t.Cleanup(func() { satOK(t, target.Close()); satOK(t, source.Close()); satOK(t, a.Close()); satOK(t, b.Close()) })
	satOK(t, source.Set(a))
	satOK(t, target.Set(b))
	satOK(t, target.SetRef(&source))
	if !b.Closed || a.Closed || a.References != 2 {
		t.Fatal("copy assignment did not retain source before retiring target", a.References, a.Closed, b.Closed)
	}
	satOK(t, source.Set(nil))
	if a.Closed || a.References != 1 {
		t.Fatal("target did not own its independent model reference", a.References, a.Closed)
	}
	satOK(t, target.SetRef(&target))
	if a.Closed || a.References != 1 {
		t.Fatal("self-assignment changed reference ownership", a.References, a.Closed)
	}
	if e = target.SetRef(nil); e != ErrModelConverterOwner {
		t.Fatal("nil source accepted", e)
	}
	satOK(t, target.Close())
	if !a.Closed {
		t.Fatal("last reference did not retire copied model")
	}
	if e = source.SetRef(&target); e != ErrModelConverterOwner {
		t.Fatal("retired source accepted", e)
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
	if !reflect.DeepEqual(labels.Values, []usymbol.Symbol{b.label, root.label}) {
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
