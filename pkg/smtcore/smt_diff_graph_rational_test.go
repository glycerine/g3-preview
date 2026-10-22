package smtcore

import (
	"errors"
	"math/big"
	"reflect"
	"testing"
)

func diffRat(value int64) *big.Rat { return new(big.Rat).SetInt64(value) }

func TestSMTRationalDiffGraphFeasibleActivationAndAssignment(t *testing.T) {
	g := NewSMTRationalDiffGraph()
	for variable := int32(0); variable < 4; variable++ {
		if err := g.InitVariable(variable); err != nil {
			t.Fatal(err)
		}
	}
	edges := [][3]int64{{0, 1, 2}, {1, 2, -5}, {2, 3, 1}}
	for index, item := range edges {
		id, err := g.AddEdge(int32(item[0]), int32(item[1]), diffRat(item[2]), NewSMTLiteral(int32(index+1), false))
		if err != nil {
			t.Fatal(err)
		}
		feasible, err := g.EnableEdge(id)
		if err != nil || !feasible {
			t.Fatal("feasible activation", index, feasible, err)
		}
	}
	if !g.IsFeasible() || g.NumNodes() != 4 || g.NumEdges() != 3 {
		t.Fatal("graph dimensions or feasibility")
	}
	want := []*big.Rat{diffRat(0), diffRat(0), diffRat(-5), diffRat(-4)}
	for variable := int32(0); variable < 4; variable++ {
		got, err := g.Assignment(variable)
		if err != nil || got.Cmp(want[variable]) != 0 {
			t.Fatal("assignment", variable, got, err)
		}
	}
}

func TestSMTRationalDiffGraphNegativeCycleExplanationAndRollback(t *testing.T) {
	g := NewSMTRationalDiffGraph()
	for variable := int32(0); variable < 3; variable++ {
		if err := g.InitVariable(variable); err != nil {
			t.Fatal(err)
		}
	}
	literals := []SMTLiteral{NewSMTLiteral(7, false), NewSMTLiteral(8, true), NewSMTLiteral(9, false)}
	input := [][3]int64{{0, 1, 1}, {1, 2, 1}, {2, 0, -3}}
	for index, item := range input {
		id, err := g.AddEdge(int32(item[0]), int32(item[1]), diffRat(item[2]), literals[index])
		if err != nil {
			t.Fatal(err)
		}
		feasible, err := g.EnableEdge(id)
		if err != nil {
			t.Fatal(err)
		}
		if feasible != (index != 2) {
			t.Fatal("negative cycle frontier", index, feasible)
		}
	}
	if g.IsFeasible() {
		t.Fatal("negative cycle reported feasible")
	}
	got, err := g.NegativeCycleExplanations(false)
	if err != nil {
		t.Fatal(err)
	}
	want := []SMTLiteral{literals[2], literals[1], literals[0]}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("cycle order", got, want)
	}
	for variable := int32(0); variable < 3; variable++ {
		value, err := g.Assignment(variable)
		if err != nil || value.Sign() != 0 {
			t.Fatal("failed activation did not restore assignment", variable, value, err)
		}
	}
}

func TestSMTRationalDiffGraphScopesAndDomainGuards(t *testing.T) {
	g := NewSMTRationalDiffGraph()
	if err := g.InitVariable(1); err != nil {
		t.Fatal(err)
	}
	base, err := g.AddEdge(0, 1, diffRat(4), SMTTrueLiteral)
	if err != nil {
		t.Fatal(err)
	}
	if feasible, err := g.EnableEdge(base); err != nil || !feasible {
		t.Fatal(feasible, err)
	}
	if err = g.Push(); err != nil {
		t.Fatal(err)
	}
	temporary, err := g.AddEdge(1, 0, diffRat(-4), SMTFalseLiteral)
	if err != nil {
		t.Fatal(err)
	}
	if feasible, err := g.EnableEdge(temporary); err != nil || !feasible {
		t.Fatal(feasible, err)
	}
	if err = g.Pop(1); err != nil || g.NumEdges() != 1 || !g.IsFeasible() {
		t.Fatal("scope rollback", g.NumEdges(), err)
	}
	if _, err = g.EnableEdge(temporary); !errors.Is(err, errSMTDiffGraphDomain) {
		t.Fatal("retired edge guard", err)
	}
	if err = g.Pop(0); !errors.Is(err, errSMTDiffGraphDomain) {
		t.Fatal("zero scope guard", err)
	}
	if err = g.InitVariable(0); !errors.Is(err, errSMTDiffGraphDomain) {
		t.Fatal("incident variable reset guard", err)
	}
}
