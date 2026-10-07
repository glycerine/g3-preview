// Source: src/cmd_context/tactic_manager.cpp:22-66;
// src/cmd_context/tactic_cmds.cpp:35-50; src/tactic/probe.cpp:39-70,180-192.
// Source author: Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package smtcore

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/tactic"
	"math"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
)

// These observe the actual cold factory contract; they do not implement a
// tactic or return any satisfiability result.

type tacticManagerTestReference struct {
	references uint32
	closed     bool
}

func (r *tacticManagerTestReference) IncRef() error { r.references++; return nil }
func (r *tacticManagerTestReference) DecRef() error {
	r.references--
	if r.references == 0 {
		r.closed = true
	}
	return nil
}
func (r *tacticManagerTestReference) Close() error { r.closed = true; return nil }

type tacticManagerTestFactory struct {
	params         *util.ParamsContext
	events         *[]int
	id             int
	called, closed bool
	mustOutlive    *tactic.Probe
}

func (f *tacticManagerTestFactory) Make(m *ast.ASTManager, p *util.ParamsRef) (tacticReference, error) {
	empty, e := f.params.Empty(p)
	if e != nil {
		return nil, e
	}
	if !empty || m == nil {
		return nil, errors.New("factory did not receive source empty params and manager")
	}
	f.called = true
	return &tacticManagerTestReference{}, nil
}
func (f *tacticManagerTestFactory) Close() error {
	if f.mustOutlive != nil && f.mustOutlive.Closed {
		return errors.New("probe deleted before source tactic finalization")
	}
	if f.closed {
		return errors.New("factory deleted twice")
	}
	f.closed = true
	*f.events = append(*f.events, f.id)
	return nil
}

func tacticRegistryFixture(t *testing.T) (*tacticManager, *util.ParamsContext, *ast.ASTManager) {
	t.Helper()
	c, sy, q := paramsTestOwner(t)
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: c.Budget()})
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	r, e := newTacticManager(sy, c.Budget(), m.ExprIDs.OutOfMemory)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	return r, c, m
}

func TestTacticManagerSourceOrderLookupFactoryAndFinalization(t *testing.T) {
	r, c, m := tacticRegistryFixture(t)
	var events []int
	var commands []*tacticCmd
	for i := 0; i < 40; i++ {
		name := astTestSymbol(t, m, fmt.Sprintf("tactic-%02d", 39-i))
		descr := fmt.Sprintf("descriptor %d", i)
		f := &tacticManagerTestFactory{params: c, events: &events, id: i}
		cmd, e := newTacticCmd(c, name, &descr, f)
		astTestMust(t, e)
		astTestMust(t, r.InsertTactic(cmd))
		commands = append(commands, cmd)
	}
	if n, e := r.NumTactics(); e != nil || n != 40 {
		t.Fatal("source tactic count", n, e)
	}
	for i, want := range commands {
		got, e := r.Tactic(uint32(i))
		astTestMust(t, e)
		found, e := r.FindTactic(want.name)
		astTestMust(t, e)
		if got != want || found != want || !want.descriptionPresent || want.description != fmt.Sprintf("descriptor %d", i) {
			t.Fatal("insertion order/lookup/description", i)
		}
	}
	missing, e := r.FindTactic(astTestSymbol(t, m, "missing-tactic"))
	astTestMust(t, e)
	if missing != nil {
		t.Fatal("source missing lookup must return null")
	}
	tactic, e := commands[0].Make(m)
	astTestMust(t, e)
	astTestMust(t, tactic.Close())
	if !commands[0].factory.(*tacticManagerTestFactory).called {
		t.Fatal("actual factory was not invoked")
	}
	capacity := cap(r.tactics)
	astTestMust(t, r.FinalizeTactics())
	if cap(r.tactics) != capacity || len(r.tactics) != 0 {
		t.Fatal("source vector reset must retain backing")
	}
	for i, got := range events {
		if got != i {
			t.Fatal("factories destroyed in hash order instead of source vector order", events)
		}
	}
	if len(events) != 40 {
		t.Fatal("owned factories were lost", events)
	}
	for _, cmd := range commands {
		got, e := r.FindTactic(cmd.name)
		astTestMust(t, e)
		if !cmd.closed || got != nil {
			t.Fatal("finalization did not clear source registrations")
		}
	}
	astTestMust(t, r.FinalizeTactics())
	if len(events) != 40 {
		t.Fatal("repeated finalization deleted factories twice")
	}
	reused, e := newTacticCmd(c, commands[0].name, nil, &tacticManagerTestFactory{params: c, events: &events, id: 99})
	astTestMust(t, e)
	m.BudgetField.FailAfter(0)
	e = r.InsertTactic(reused)
	m.BudgetField.FailAfter(-1)
	astTestMust(t, e)
	got, e := r.FindTactic(reused.name)
	astTestMust(t, e)
	if got != reused || cap(r.tactics) != capacity {
		t.Fatal("reset backing/lookup could not be reused without allocation")
	}
}

func TestTacticManagerSourceProbeReferencesAndStructuralValues(t *testing.T) {
	r, _, m := tacticRegistryFixture(t)
	g := goalTestNew(t, m, false, false)
	p := goalTestAtom(t, m, "probe-p")
	q := goalTestAtom(t, m, "probe-q")
	astTestMust(t, g.Assert(p))
	astTestMust(t, g.Assert(goalTestApp(t, m, ast.ASTOpOr, p, q)))
	astTestMust(t, g.SetDepth(7))
	for i, kind := range []tactic.ProbeKind{tactic.ProbeConstant, tactic.ProbeDepth, tactic.ProbeSize, tactic.ProbeNumExprs} {
		probe, e := tactic.NewProbe(m.BudgetField, kind, -3.25)
		astTestMust(t, e)
		name := astTestSymbol(t, m, fmt.Sprintf("probe-%d", i))
		info, e := newTacticProbeInfo(m.Symbols, m.BudgetField, name, nil, probe)
		astTestMust(t, e)
		if probe.References != 1 {
			t.Fatal("probe_info did not retain its probe")
		}
		astTestMust(t, r.InsertProbe(info))
		found, e := r.FindProbe(name)
		astTestMust(t, e)
		listed, e := r.Probe(uint32(i))
		astTestMust(t, e)
		if found != info || listed != info {
			t.Fatal("probe order/lookup changed")
		}
		value, e := found.probe.(*tactic.Probe).Eval(g)
		astTestMust(t, e)
		want := []float64{-3.25, 7, 2, 3}[i] // p, q, (or p q) are the three distinct expressions.
		if value != want {
			t.Fatal("literal structural probe", kind, value, want)
		}
	}
	kept := r.probes[0].probe.(*tactic.Probe)
	astTestMust(t, kept.IncRef())
	if n, e := r.NumProbes(); e != nil || n != 4 {
		t.Fatal(n, e)
	}
	astTestMust(t, r.FinalizeProbes())
	if kept.Closed || kept.References != 1 {
		t.Fatal("registry finalization consumed an external probe root")
	}
	astTestMust(t, kept.DecRef())
	if !kept.Closed {
		t.Fatal("probe's final reference did not delete it")
	}
	for _, tc := range []struct {
		value float64
		want  bool
	}{{0, false}, {math.Copysign(0, -1), false}, {-1, true}, {math.NaN(), true}, {math.Inf(1), true}} {
		if tactic.ProbeIsTrue(tc.value) != tc.want {
			t.Fatal("source probe::result bool conversion", tc.value, tc.want)
		}
	}
}

func TestTacticManagerSourceDuplicateAssertionAndNamespaceSeparation(t *testing.T) {
	r, c, m := tacticRegistryFixture(t)
	name := astTestSymbol(t, m, "same-name")
	var events []int
	one, e := newTacticCmd(c, name, nil, &tacticManagerTestFactory{params: c, events: &events, id: 1})
	astTestMust(t, e)
	astTestMust(t, r.InsertTactic(one))
	two, e := newTacticCmd(c, name, nil, &tacticManagerTestFactory{params: c, events: &events, id: 2})
	astTestMust(t, e)
	e = r.InsertTactic(two)
	if ast.ASTSourceDebug {
		if e == nil {
			t.Fatal("source duplicate SASSERT was not checked")
		}
		astTestMust(t, two.Close())
	} else {
		astTestMust(t, e)
		got, e := r.FindTactic(name)
		astTestMust(t, e)
		if got != two || len(r.tactics) != 2 {
			t.Fatal("release source overwrite/vector append was changed")
		}
	}
	info, e := newTacticProbeInfo(m.Symbols, m.BudgetField, name, nil, nil)
	astTestMust(t, e)
	astTestMust(t, r.InsertProbe(info))
	got, e := r.FindProbe(name)
	astTestMust(t, e)
	if got != info || got.probe != nil || got.descriptionPresent || one.descriptionPresent {
		t.Fatal("independent probe namespace/null ref changed")
	}
}

func TestTacticManagerSourcePublishedDictionaryBeforeVectorRefusal(t *testing.T) {
	r, c, m := tacticRegistryFixture(t)
	var events []int
	name := astTestSymbol(t, m, "published-before-vector")
	cmd, e := newTacticCmd(c, name, nil, &tacticManagerTestFactory{params: c, events: &events, id: 0})
	astTestMust(t, e)
	m.BudgetField.FailAfter(0)
	e = r.InsertTactic(cmd)
	m.BudgetField.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("expected source vector reservation refusal", e)
	}
	got, e := r.FindTactic(name)
	astTestMust(t, e)
	if got != cmd || len(r.tactics) != 0 {
		t.Fatal("dictionary publication was rolled back or vector publication invented")
	}
	astTestMust(t, r.FinalizeTactics())
	if cmd.closed || len(events) != 0 {
		t.Fatal("source finalization deleted a command never owned by its vector")
	}
	astTestMust(t, cmd.Close()) // retained caller owns the unappended command.
}

func TestTacticManagerCopiedForeignAndClosedGuards(t *testing.T) {
	r, c, m := tacticRegistryFixture(t)
	var copied tacticManager
	reflect.ValueOf(&copied).Elem().Set(reflect.ValueOf(r).Elem())
	before := m.BudgetField.Used()
	if e := copied.Close(); e == nil || m.BudgetField.Used() != before {
		t.Fatal("copied registry acquired teardown authority")
	}
	foreign := astTestManager(t)
	if _, e := r.FindTactic(astTestSymbol(t, foreign, "foreign")); !errors.Is(e, util.ErrSymbolOwner) {
		t.Fatal("foreign symbol query", e)
	}
	var events []int
	cmd, e := newTacticCmd(c, astTestSymbol(t, m, "owned"), nil, &tacticManagerTestFactory{params: c, events: &events})
	astTestMust(t, e)
	astTestMust(t, r.InsertTactic(cmd))
	astTestMust(t, r.Close())
	if _, e := r.FindTactic(cmd.name); e == nil {
		t.Fatal("closed registry remained queryable")
	}
	if _, e := cmd.Make(m); e == nil {
		t.Fatal("deleted factory remained callable")
	}
}

func TestTacticManagerDestructorFinalizesTacticsBeforeProbes(t *testing.T) {
	r, c, m := tacticRegistryFixture(t)
	probe, e := tactic.NewProbe(m.BudgetField, tactic.ProbeConstant, 1)
	astTestMust(t, e)
	info, e := newTacticProbeInfo(m.Symbols, m.BudgetField, astTestSymbol(t, m, "ordered-probe"), nil, probe)
	astTestMust(t, e)
	astTestMust(t, r.InsertProbe(info))
	var events []int
	cmd, e := newTacticCmd(c, astTestSymbol(t, m, "ordered-tactic"), nil, &tacticManagerTestFactory{params: c, events: &events, mustOutlive: probe})
	astTestMust(t, e)
	astTestMust(t, r.InsertTactic(cmd))
	astTestMust(t, r.Close())
	if !probe.Closed || !info.closed || !cmd.closed {
		t.Fatal("source destructor did not finalize both owned namespaces")
	}
}

func TestTacticManagerMetadataCapturesSourceDescriptionValue(t *testing.T) {
	r, c, m := tacticRegistryFixture(t)
	var events []int
	description := "original description"
	cmd, e := newTacticCmd(c, astTestSymbol(t, m, "description-tactic"), &description, &tacticManagerTestFactory{params: c, events: &events})
	astTestMust(t, e)
	astTestMust(t, r.InsertTactic(cmd))
	info, e := newTacticProbeInfo(m.Symbols, m.BudgetField, astTestSymbol(t, m, "description-probe"), &description, nil)
	astTestMust(t, e)
	astTestMust(t, r.InsertProbe(info))
	description = "caller variable now points elsewhere"
	if !cmd.descriptionPresent || !info.descriptionPresent || cmd.description != "original description" || info.description != "original description" {
		t.Fatal("source m_descr(d) must capture pointer value, rather than follow the caller's pointer variable")
	}
	empty := ""
	emptyInfo, e := newTacticProbeInfo(m.Symbols, m.BudgetField, astTestSymbol(t, m, "empty-description"), &empty, nil)
	astTestMust(t, e)
	astTestMust(t, r.InsertProbe(emptyInfo))
	if !emptyInfo.descriptionPresent || emptyInfo.description != "" {
		t.Fatal("nonnull empty source description became null")
	}
}
