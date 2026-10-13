package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestHardwareContextIntegerBoundsAndOpenConflicts(t *testing.T) {
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewHardwareContext(m, &control.Reslimit{Limit: 1000}, 20, 1, 8192)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	}()
	for _, integer := range []bool{true, false, true} {
		if _, err := c.NewVariable(integer); err != nil {
			t.Fatal(err)
		}
	}
	root, err := c.CreateRootNode()
	if err != nil {
		t.Fatal(err)
	}
	var zero, one, two, fractional numeral.HardwareFloat
	for _, pair := range []struct {
		out   *numeral.HardwareFloat
		value float64
	}{{&zero, 0}, {&one, 1}, {&two, 2}, {&fractional, 2.5}} {
		if err := m.SetFloat64(pair.out, pair.value); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.PropagateBound(root, 2, &zero, true, true); err != nil {
		t.Fatal(err)
	}
	if root.lower[2].open || m.ToFloat64(&root.lower[2].value) != 0 {
		t.Fatal("source hardware is_int treats zero as non-integer before ceil")
	}
	if err := c.PropagateBound(root, 0, &fractional, true, true); err != nil {
		t.Fatal(err)
	}
	if root.lower[0].open || m.ToFloat64(&root.lower[0].value) != 3 || root.lower[0].timestamp != 1 {
		t.Fatal("fractional strict integer lower bound must ceil and close")
	}
	child, err := c.NewChildNode(root)
	if err != nil {
		t.Fatal(err)
	}
	if child.lower[0] != root.lower[0] {
		t.Fatal("child inherited parent integer bound")
	}
	if err := c.PropagateBound(child, 1, &one, true, true); err != nil {
		t.Fatal(err)
	}
	if err := c.PropagateBound(child, 1, &one, false, false); err != nil {
		t.Fatal(err)
	}
	if child.conflictVar != 1 || root.conflictVar != ^uint32(0) || root.lower[1] != nil || root.upper[1] != nil {
		t.Fatal("open equality conflicts only in the child")
	}
	if err := c.PropagateBound(child, 1, &two, false, false); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("bound after conflict", err)
	}
	if err := c.PropagateBound(root, 0, &two, false, true); err != nil {
		t.Fatal(err)
	}
	if root.conflictVar != 0 || root.upper[0].open || m.ToFloat64(&root.upper[0].value) != 1 || m.ToFloat64(&two) != 2 {
		t.Fatal("strict integer upper bound and conflict")
	}
	if len(c.queue) != 5 || c.timestamp != 5 || c.queue[4] != root.trail {
		t.Fatal("source bound queue and timestamps")
	}
	if err := child.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestHardwareContextRootUnitAssertionResumeAndConflict(t *testing.T) {
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	limit := &control.Reslimit{Limit: 1}
	c, err := NewHardwareContext(m, limit, 20, 1, 8192)
	if err != nil {
		t.Fatal(err)
	}
	x, err := c.NewVariable(true)
	if err != nil {
		t.Fatal(err)
	}
	atoms := make([]*HardwareInequality, 3)
	for i, entry := range []struct {
		value int32
		lower bool
		axiom bool
	}{{1, true, true}, {2, false, false}, {3, true, true}} {
		var value numeral.HardwareFloat
		if err := m.SetInt32(&value, entry.value); err != nil {
			t.Fatal(err)
		}
		atoms[i], err = c.NewInequality(x, &value, entry.lower, false)
		if err != nil {
			t.Fatal(err)
		}
		if err := c.AddUnitClause(atoms[i], entry.axiom); err != nil {
			t.Fatal(err)
		}
	}
	root, err := c.CreateRootNode()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.AssertUnitClauses(root); !errors.Is(err, ErrHardwareContextCanceled) {
		t.Fatal("checkpoint between units", err)
	}
	if c.unitHead != 1 || len(c.queue) != 1 || root.trail.reason != hardwareReasonAxiom || root.conflictVar != ^uint32(0) {
		t.Fatal("completed unit prefix after cancellation")
	}
	limit.Limit = 10
	if err := c.AssertUnitClauses(root); err != nil {
		t.Fatal(err)
	}
	if c.unitHead != 3 || len(c.queue) != 3 || root.conflictVar != x ||
		root.trail.reason != hardwareReasonAxiom || root.trail.prev.reason != hardwareReasonAssumption {
		t.Fatal("resumed unit suffix and conflicting bounds")
	}
	if err := c.AssertUnitClauses(root); err != nil || len(c.queue) != 3 {
		t.Fatal("conflicted unit pass must stop", err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	for _, atom := range atoms {
		if atom.refs != 0 {
			t.Fatal("unit reference retained after context close")
		}
	}
}
