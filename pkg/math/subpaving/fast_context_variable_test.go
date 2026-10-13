package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestFastContextTacticVariableGrowthBeforeRoot(t *testing.T) {
	m, c, value, root := fastMonomialFixture(t, nil)
	x, err := c.NewVariable(true)
	if err != nil || x != 0 {
		t.Fatal("first integer variable", x, err)
	}
	y, err := c.NewVariable(false)
	if err != nil || y != 1 {
		t.Fatal("second real variable", y, err)
	}
	if len(c.isInt) != 2 || !c.isInt[x] || c.isInt[y] || len(c.watchers) != 2 ||
		len(c.watchOrder) != 2 || len(root.lower) != 2 || len(root.upper) != 2 {
		t.Fatal("source variable and root slot growth")
	}
	if err := m.SetInt32(value, 1); err != nil {
		t.Fatal(err)
	}
	sum, err := c.DefinePolynomial(value, []FastPolynomialTerm{{Variable: x, Coefficient: value}, {Variable: y, Coefficient: value}})
	if err != nil || sum != 2 || c.isInt[sum] {
		t.Fatal("real polynomial output", sum, err)
	}
	product, err := c.DefineMonomial([]FastPower{{Variable: x, Degree: 2}})
	if err != nil || product != 3 || !c.isInt[product] {
		t.Fatal("integer monomial output", product, err)
	}
	if len(root.lower) != 4 || len(root.upper) != 4 {
		t.Fatal("definition slots", len(root.lower), len(root.upper))
	}
	if err := c.InitializeRoot(); err != nil {
		t.Fatal("root with tactic-created variables", err)
	}
	if _, err := c.NewVariable(true); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("late variable must not invalidate initialized root", err)
	}
}
