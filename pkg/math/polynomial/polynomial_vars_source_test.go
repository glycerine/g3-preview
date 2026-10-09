// Source: src/math/polynomial/polynomial.cpp:3186-3205,7359-7361;
// src/test/polynomial.cpp:837-879.
package polynomial

import (
	"fmt"
	"sort"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialVarsOriginalDormantAssertions(t *testing.T) {
	m := polynomialDiscriminantFixture(t)
	c := m.core
	parser := polynomialDiscriminantExpression{t: t, c: c}
	for i := range parser.variables {
		x, e := c.monomials.MkVar()
		serviceMust(t, e)
		parser.variables[i] = polynomialCaller(t, c, polynomialVarsVariable(t, c, x))
	}
	xs, e := NewPolynomialVariableDegrees(c.memory)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, xs.CloseChecked()) })
	// Literal original expressions with x0..x4 mapped to a,b,c,d,x solely for
	// the existing fixture parser. Original ENSURE size and sorted equality.
	rows := []struct {
		literal, expression string
		want                []uint32
	}{
		{"(x0 + 1)*((x0^2) + (x3^2))*(x2*x3)", "(a+1)*((a^2)+(d^2))*(c*d)", []uint32{0, 2, 3}},
		{"(x0 + x2)*((x0^2) + (x3^2))*(x2*x3)", "(a+c)*((a^2)+(d^2))*(c*d)", []uint32{0, 2, 3}},
		{"((x1 + x4 + 1)^5)", "((b+x+1)^5)", []uint32{1, 4}},
		{"((x1 + x4*x2 + 1)^4) + x0 + (x3^2)", "((b+x*c+1)^4)+a+(d^2)", []uint32{0, 1, 2, 3, 4}},
		{"(x3 + 1)^5", "(d+1)^5", []uint32{3}},
		{"x0*x1*x2", "a*b*c", []uint32{0, 1, 2}},
		{"x0*x1*x2 + 1", "a*b*c+1", []uint32{0, 1, 2}},
	}
	for _, row := range rows {
		p := parser.evaluate(row.expression)
		serviceMust(t, m.Vars(p, xs))
		n, e := xs.size()
		serviceMust(t, e)
		if n != uint32(len(row.want)) {
			t.Fatalf("%s size %d", row.literal, n)
		}
		got := make([]uint32, n)
		for i := range got {
			got[i], e = variableAt(xs, uint32(i))
			serviceMust(t, e)
		}
		sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
		for i := range got {
			if got[i] != row.want[i] {
				t.Fatalf("%s vars %v", row.literal, got)
			}
		}
	}
}
func polynomialVarsVariable(t *testing.T, c *MultivariatePolynomialCore, x uint32) Polynomial {
	t.Helper()
	v, e := c.MkVariable(x, 1)
	serviceMust(t, e)
	return v
}

func TestPolynomialVarsEveryActualReservationAndNamedGraph(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		stop := false
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, p, _ := polynomialLexFixture(t)
			c := m.core
			xs, e := NewPolynomialVariableDegrees(c.memory)
			serviceMust(t, e)
			c.integer.Budget().FailAfter(ordinal)
			returned := m.Vars(p, xs)
			polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, nil, xs.header, nil)
			n, e := xs.size()
			serviceMust(t, e)
			for i := uint32(0); i < n; i++ {
				x, e := variableAt(xs, i)
				serviceMust(t, e)
				seen, e := c.substitution.found.at(x)
				serviceMust(t, e)
				if seen != (returned != nil) {
					t.Fatal("literal final reset prefix", returned, x, seen)
				}
			}
			c.integer.Budget().FailAfter(-1)
			if returned == nil {
				if n != 2 {
					t.Fatal(n)
				}
				stop = true
				success = true
			} else if !numericPhysicalReservationRefusal(returned) {
				t.Fatal(returned)
			}
			serviceMust(t, xs.CloseChecked())
		})
		if stop {
			break
		}
	}
	if !success {
		t.Fatal("no first-success horizon")
	}
}

func TestPolynomialVarsNativePushAndNormalReset(t *testing.T) {
	m, p, _ := polynomialLexFixture(t)
	c := m.core
	xs, e := NewPolynomialVariableDegrees(c.memory)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, xs.CloseChecked()) })
	serviceMust(t, c.substitution.found.reserve(2))
	first, e := c.monomialCore(p, 0)
	serviceMust(t, e)
	x, e := c.monomials.varCore(first, 0)
	serviceMust(t, e)
	c.integer.Budget().FailAfter(0)
	returned := m.Vars(p, xs)
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, returned)
	n, e := xs.size()
	serviceMust(t, e)
	seen, e := c.substitution.found.at(x)
	serviceMust(t, e)
	if n != 2 || seen {
		t.Fatal("native push and normal reset", n, seen)
	}
	for i := uint32(0); i < n; i++ {
		collected, e := variableAt(xs, i)
		serviceMust(t, e)
		seen, e := c.substitution.found.at(collected)
		serviceMust(t, e)
		if seen {
			t.Fatal("normal suffix did not reset collected variable", collected)
		}
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, nil, xs.header, nil)
	serviceMust(t, m.Vars(p, xs))
	n, e = xs.size()
	serviceMust(t, e)
	if n != 2 {
		t.Fatal("repeat collection", n)
	}
	seen, e = c.substitution.found.at(x)
	serviceMust(t, e)
	if seen {
		t.Fatal("normal suffix did not reset collected variable")
	}
}

// Returning ErrorTransport is a supported Go callback domain, distinct from
// the C++ mathematical algorithm. Calls must refuse before owner mutation.
func TestPolynomialVarsReturningTransportOwnerReentry(t *testing.T) {
	for _, which := range []string{"full-source", "full-own", "core-source", "core-own", "output-source", "output-own", "same-call"} {
		t.Run(which, func(t *testing.T) {
			b := arena.NewBudget(1 << 28)
			var m *MultivariatePolynomialManager
			var xs *PolynomialVariableDegrees
			var p Polynomial
			armed, seen := false, false
			rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f util.Failure) error {
				if armed {
					armed = false
					seen = true
					state := m.core.memory.State()
					used := b.Used()
					source, closed := m.sourceStarted, m.closed
					values := append([]uint32(nil), xs.values...)
					var returned error
					switch which {
					case "full-source":
						returned = m.SourceFinalize()
					case "full-own":
						returned = m.CloseChecked()
					case "core-source":
						returned = m.core.SourceFinalizePrefix()
					case "core-own":
						returned = m.core.CloseChecked()
					case "output-source":
						returned = xs.SourceFinalize()
					case "output-own":
						returned = xs.CloseChecked()
					case "same-call":
						returned = m.Vars(p, xs)
					}
					sameValues := len(values) == len(xs.values)
					for i := range values {
						sameValues = sameValues && values[i] == xs.values[i]
					}
					if returned != ErrPolynomialPending || state != m.core.memory.State() || used != b.Used() || source != m.sourceStarted || closed != m.closed || !sameValues {
						t.Errorf("executing callback changed owner %v source=%v closed=%v", returned, m.sourceStarted, m.closed)
					}
				}
				return f
			}})
			serviceMust(t, e)
			cfg := serviceConfig(b).Integers
			cfg.Memory = rt
			q, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
			serviceMust(t, e)
			m, e = NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
			serviceMust(t, e)
			x, e := m.core.monomials.MkVar()
			serviceMust(t, e)
			p = polynomialVarsVariable(t, m.core, x)
			serviceMust(t, m.core.IncRef(p))
			xs, e = NewPolynomialVariableDegrees(rt)
			serviceMust(t, e)
			b.FailAfter(0)
			armed = true
			returned := m.Vars(p, xs)
			armed = false
			if !seen || !numericPhysicalReservationRefusal(returned) {
				t.Error("actual reserve callback absent", seen, returned)
			}
			b.FailAfter(-1)
			serviceMust(t, xs.CloseChecked())
			if !m.closed {
				serviceMust(t, m.core.decRefCore(p))
				serviceMust(t, m.CloseChecked())
			}
			serviceMust(t, q.CloseChecked())
			serviceMust(t, rt.Close())
			if b.Used() != 0 {
				t.Error("named owner charge", b.Used())
			}
		})
	}
}

func TestPolynomialVarsLiteralResetEmptyWarmAndBorrowGuards(t *testing.T) {
	m, p, _ := polynomialLexFixture(t)
	c := m.core
	xs, e := NewPolynomialVariableDegrees(c.memory)
	serviceMust(t, e)
	serviceMust(t, m.Vars(p, xs))
	n, e := xs.size()
	serviceMust(t, e)
	if n != 2 {
		t.Fatal(n)
	}
	// mkTerms places the maximal y^3 term first: this query preserves encounter
	// order and does not sort output or read coefficient values.
	first, e := variableAt(xs, 0)
	serviceMust(t, e)
	second, e := variableAt(xs, 1)
	serviceMust(t, e)
	if first != 1 || second != 0 {
		t.Fatal("first encounter order", first, second)
	}
	used := c.integer.Budget().Used()
	c.integer.Budget().FailAfter(0)
	serviceMust(t, m.Vars(p, xs))
	if used != c.integer.Budget().Used() {
		t.Fatal("warm variable storage changed")
	}
	serviceMust(t, m.Vars(c.zero, xs))
	n, e = xs.size()
	serviceMust(t, e)
	if n != 0 {
		t.Fatal("source empty output reset", n)
	}
	c.integer.Budget().FailAfter(-1)
	copied := *xs
	if e = m.Vars(p, &copied); e != ErrPolynomialOwner {
		t.Fatal("copied output", e)
	}
	ownerCopy := *m
	if e = ownerCopy.Vars(p, xs); e != ErrPolynomialOwner {
		t.Fatal("copied manager", e)
	}
	// Source reaches output reset/reserve before reading p.size. Invalid Go
	// input ownership is guarded at that reached borrow, not an early shortcut.
	serviceMust(t, variablePush(xs, 17))
	if e = m.Vars(Polynomial{}, xs); e == nil {
		t.Fatal("missing polynomial accepted")
	}
	n, e = xs.size()
	serviceMust(t, e)
	if n != 0 {
		t.Fatal("older output reset prefix erased", n)
	}
	serviceMust(t, xs.CloseChecked())
}
