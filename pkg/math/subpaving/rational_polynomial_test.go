package subpaving

import (
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRationalPolynomialRootForwardAndReversePropagation(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		name := "forward"
		if reverse {
			name = "reverse"
		}
		t.Run(name, func(t *testing.T) {
			c, m, value, x, y := rationalClauseFixture(t)
			two, three, constant := numeral.NewRational(), numeral.NewRational(), numeral.NewRational()
			t.Cleanup(func() {
				for _, e := range []error{m.Del(&two), m.Del(&three), m.Del(&constant)} {
					if e != nil {
						t.Error(e)
					}
				}
			})
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			must(m.SetInt32(&two, 2))
			must(m.SetInt32(&three, 3))
			must(m.SetInt32(&constant, 7))
			z, err := c.DefinePolynomial(&constant, []numeral.Rational{two, three}, []uint32{x, y})
			must(err)
			if z != 2 || c.variables[z] || c.polynomials[z] == nil {
				t.Fatal("polynomial definition and real variable registration")
			}
			unit := func(variable uint32, number int32) {
				must(m.SetInt32(value, number))
				_, err := c.AddInequality(variable, value, true, false, true)
				must(err)
				_, err = c.AddInequality(variable, value, false, false, true)
				must(err)
			}
			if reverse {
				unit(z, 10)
				unit(y, 2)
			} else {
				unit(x, 1)
				unit(y, 2)
			}
			must(c.InitializeRoot())
			target, expected := z, int32(8)
			if reverse {
				target, expected = x, 2
			}
			root := c.root
			if root == nil || root.conflictVar != math.MaxUint32 || root.lower[target] == nil || root.upper[target] == nil || root.lower[target].reasonDefinition != z || root.upper[target].reasonDefinition != z || len(c.queue) != 0 || c.initStage != 3 {
				t.Fatal("root polynomial propagation did not publish both justified bounds")
			}
			must(m.SetInt32(value, expected))
			for _, bound := range []*RationalBound{root.lower[target], root.upper[target]} {
				cmp, err := m.Compare(&bound.value, value)
				if err != nil || cmp != 0 {
					t.Fatal("wrong derived value; source calculation excludes stored constant", cmp, err)
				}
			}
			bounds, visits := c.numBounds, c.numVisited
			must(c.InitializeRoot())
			if c.numBounds != bounds || c.numVisited != visits {
				t.Fatal("completed polynomial root initialization replayed")
			}
		})
	}
}

func TestRationalPolynomialDuplicateCoefficientBufferAndWatchOrder(t *testing.T) {
	c, m, value, x, y := rationalClauseFixture(t)
	two, three, constant := numeral.NewRational(), numeral.NewRational(), numeral.NewRational()
	t.Cleanup(func() {
		for _, e := range []error{m.Del(&two), m.Del(&three), m.Del(&constant)} {
			if e != nil {
				t.Error(e)
			}
		}
	})
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(m.SetInt32(value, 5))
	a, err := c.NewInequality(x, value, true, false)
	must(err)
	b, err := c.NewInequality(y, value, true, false)
	must(err)
	clause, err := c.AddClause([]*RationalInequality{a, b})
	must(err)
	must(m.SetInt32(&two, 2))
	must(m.SetInt32(&three, 3))
	must(m.SetInt32(&constant, 7))
	z, err := c.DefinePolynomial(&constant, []numeral.Rational{two, three}, []uint32{x, x})
	must(err)
	p := c.polynomials[z]
	if p == nil || len(p.terms) != 2 || p.terms[0].variable != x || p.terms[1].variable != x || len(c.watchOrder[x]) != 3 || c.watchOrder[x][0].clause != clause || c.watchOrder[x][1].definition != z || c.watchOrder[x][2].definition != z {
		t.Fatal("source coefficient duplicates or insertion-order watches")
	}
	must(m.SetInt32(value, 3))
	cmp, err := m.Compare(&p.terms[0].coefficient, value)
	if err != nil || cmp != 0 {
		t.Fatal("last coefficient did not occupy first sorted term", cmp, err)
	}
	zero, err := m.IsZero(&p.terms[1].coefficient)
	if err != nil || !zero {
		t.Fatal("second duplicate term did not receive zero", zero, err)
	}
	zero, err = m.IsZero(&c.numBuffer[x])
	if err != nil || !zero {
		t.Fatal("scratch buffer did not relinquish coefficient", zero, err)
	}
}

func TestRationalPolynomialScopedCoefficientGrowthAndReuse(t *testing.T) {
	c, m, _, _, _ := rationalClauseFixture(t)
	var last uint32
	for i := 0; i < 19; i++ {
		var err error
		last, err = c.NewVariable(false)
		if err != nil {
			t.Fatal(err)
		}
	}
	constant, first, second := numeral.NewRational(), numeral.NewRational(), numeral.NewRational()
	t.Cleanup(func() {
		for _, e := range []error{m.Del(&constant), m.Del(&first), m.Del(&second)} {
			if e != nil {
				t.Error(e)
			}
		}
	})
	if err := m.SetInt64(&first, 1<<40); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt64(&second, 1<<41); err != nil {
		t.Fatal(err)
	}
	for _, coefficient := range []*numeral.Rational{&first, &second} {
		z, err := c.DefinePolynomial(&constant, []numeral.Rational{*coefficient}, []uint32{last})
		if err != nil {
			t.Fatal(err)
		}
		p := c.polynomials[z]
		if p == nil || len(p.terms) != 1 || p.terms[0].variable != last {
			t.Fatal("source mk_sum lost high-index coefficient")
		}
		cmp, err := m.Compare(&p.terms[0].coefficient, coefficient)
		if err != nil || cmp != 0 {
			t.Fatal("source indexed copy/swap changed coefficient", cmp, err)
		}
		zero, err := m.IsZero(&c.numBuffer[last])
		if err != nil || !zero {
			t.Fatal("source swap did not clear reusable slot", zero, err)
		}
	}
	if len(c.numBuffer) != len(c.variables)-1 || len(c.numBuffer) <= 16 {
		t.Fatal("source reserve did not grow logical indexed buffer", len(c.numBuffer), len(c.variables))
	}
}

func TestRationalPolynomialQueuedClauseBoundTriggersDefinition(t *testing.T) {
	c, m, value, x, y := rationalClauseFixture(t)
	two, three, constant := numeral.NewRational(), numeral.NewRational(), numeral.NewRational()
	t.Cleanup(func() {
		for _, e := range []error{m.Del(&two), m.Del(&three), m.Del(&constant)} {
			if e != nil {
				t.Error(e)
			}
		}
	})
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(m.SetInt32(&two, 2))
	must(m.SetInt32(&three, 3))
	z, err := c.DefinePolynomial(&constant, []numeral.Rational{two, three}, []uint32{x, y})
	must(err)
	must(m.SetInt32(value, 1))
	xLower, err := c.NewInequality(x, value, true, false)
	must(err)
	must(m.SetInt32(value, 3))
	yLower, err := c.NewInequality(y, value, true, false)
	must(err)
	clause, err := c.AddClause([]*RationalInequality{yLower, xLower})
	must(err)
	must(m.SetInt32(value, 2))
	if _, err := c.AddInequality(y, value, true, false, true); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AddInequality(y, value, false, false, true); err != nil {
		t.Fatal(err)
	}
	must(c.InitializeRoot())
	root := c.root
	if root == nil || root.conflictVar != math.MaxUint32 || root.lower[x] == nil || root.lower[x].reasonClause != clause || root.lower[z] == nil || root.lower[z].reasonDefinition != z || root.upper[z] != nil || len(c.queue) != 0 {
		t.Fatal("clause bound did not trigger the watched polynomial forward lower bound")
	}
	must(m.SetInt32(value, 8))
	cmp, err := m.Compare(&root.lower[z].value, value)
	if err != nil || cmp != 0 {
		t.Fatal("watched polynomial lower bound", cmp, err)
	}
}

func TestRationalPolynomialReverseIntegerCandidateNormalization(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	m, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewRationalContext(m, &control.Reslimit{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	two, zero, value := numeral.NewRational(), numeral.NewRational(), numeral.NewRational()
	t.Cleanup(func() {
		for _, e := range []error{c.Close(), m.Del(&two), m.Del(&zero), m.Del(&value), m.CloseChecked()} {
			if e != nil {
				t.Error(e)
			}
		}
		if budget.Used() != 0 {
			t.Error("integer polynomial retained numeral budget", budget.Used())
		}
	})
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	x, err := c.NewVariable(true)
	must(err)
	must(m.SetInt32(&two, 2))
	z, err := c.DefinePolynomial(&zero, []numeral.Rational{two}, []uint32{x})
	must(err)
	if !c.variables[z] {
		t.Fatal("integer definition did not retain source is_int predicate")
	}
	must(m.SetInt32(&value, 5))
	_, err = c.AddInequality(z, &value, true, false, true)
	must(err)
	must(c.InitializeRoot())
	bound := c.root.lower[x]
	if bound == nil || bound.open || bound.reasonDefinition != z {
		t.Fatal("reverse integer candidate was not normalized and justified")
	}
	must(m.SetInt32(&value, 3))
	cmp, err := m.Compare(&bound.value, &value)
	if err != nil || cmp != 0 {
		t.Fatal("ceil(5/2) integer lower bound", cmp, err)
	}
}
