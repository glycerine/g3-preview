package memory

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

type processFatalPolicyProbe struct {
	budget *arena.Budget
	calls  []*Runtime
}

func (p *processFatalPolicyProbe) MemoryBudget() *arena.Budget { return p.budget }
func (p *processFatalPolicyProbe) HandleMemoryFailure(r *Runtime, f Failure) error {
	p.calls = append(p.calls, r)
	return f
}

func TestProcessFatalPolicyDispatchesAcrossRuntimeOwners(t *testing.T) {
	process := NewProcessAccounting()
	budget := arena.NewBudget(1 << 20)
	first, err := NewRuntime(RuntimeConfig{Budget: budget, Process: process})
	if err != nil {
		t.Fatal(err)
	}
	policy := &processFatalPolicyProbe{budget: budget}
	second, err := NewRuntime(RuntimeConfig{Budget: budget, Process: process, FatalPolicy: policy})
	if err != nil {
		t.Fatal(err)
	}
	third, err := NewRuntime(RuntimeConfig{Budget: budget, Process: process})
	if err != nil {
		t.Fatal(err)
	}
	if err := first.SetMaxSize(1); err != nil {
		t.Fatal(err)
	}
	for _, r := range []*Runtime{first, second, third} {
		if h, err := r.Allocate(1); h != (Block{}) || err == nil {
			t.Fatal("expected source memory failure", h, err)
		}
	}
	if len(policy.calls) != 3 || policy.calls[0] != first || policy.calls[1] != second || policy.calls[2] != third {
		t.Fatal("process policy was not shared by all runtime owners", policy.calls)
	}
	other := &processFatalPolicyProbe{budget: budget}
	before := budget.Used()
	if r, err := NewRuntime(RuntimeConfig{Budget: budget, Process: process, FatalPolicy: other}); r != nil || !errors.Is(err, arena.ErrConfig) || budget.Used() != before {
		t.Fatal("conflicting process policy changed storage", r, err, budget.Used(), before)
	}
	if r, err := NewRuntime(RuntimeConfig{Budget: budget, Process: process, FatalPolicy: policy}); err != nil {
		t.Fatal("same process policy rejected", err)
	} else if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	for _, r := range []*Runtime{third, second, first} {
		if err := r.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if budget.Used() != 0 {
		t.Fatal("runtime storage retained", budget.Used())
	}
}

func TestProcessFatalPolicyBindsAfterSuccessfulConstruction(t *testing.T) {
	process := NewProcessAccounting()
	budget := arena.NewBudget(1 << 20)
	policy := &processFatalPolicyProbe{budget: budget}
	budget.FailAfter(0)
	if r, err := NewRuntime(RuntimeConfig{Budget: budget, Process: process, FatalPolicy: policy}); r != nil || !errors.Is(err, arena.ErrInjectedFailure) || process.fatalPolicy != nil {
		t.Fatal("failed constructor bound the process policy", r, err)
	}
	budget.FailAfter(-1)
	r, err := NewRuntime(RuntimeConfig{Budget: budget, Process: process, FatalPolicy: policy})
	if err != nil || process.fatalPolicy != policy {
		t.Fatal("successful constructor did not bind the policy", r, err)
	}
	if err := r.Close(); err != nil || budget.Used() != 0 {
		t.Fatal("runtime teardown", err, budget.Used())
	}
}
