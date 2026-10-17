package polynomial

import (
	"errors"
	"strconv"
	"testing"
)

func TestAlgebraicRootConstructionPreservesOlderFactorInvocation(t *testing.T) {
	found := false
	for allowance := uint64(0); allowance < 512 && !found; allowance++ {
		t.Run(strconv.FormatUint(allowance, 10), func(t *testing.T) {
			c, up, control := algebraicControlledFixture(t)
			pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{0, 0, 2}}, {-2, [3]uint32{}}})
			old, e := up.NewFactors()
			serviceMust(t, e)
			defer func() { serviceMust(t, c.config.Factors.CleanupRetained()); serviceMust(t, old.Close()) }()
			polynomial := algebraicPolynomial(t, up, -2, 0, 0, 0, 1)
			view, e := up.wholeView(polynomial)
			serviceMust(t, e)
			control.limit = control.visits + allowance
			_, body := c.config.Factors.Factor(view, old, c.factorParams)
			control.limit = 1000000
			service := c.config.Factors
			if body == nil || service.scope == nil {
				return
			}
			found = true
			receipt, scope := service.frontend.receipt, service.scope
			var out Algebraic
			e = c.MkPolynomialRoot(pm, p, 2, &out)
			if !errors.Is(e, ErrUPolynomialCleanupPending) || out != (Algebraic{}) {
				t.Fatal("new root did not refuse old factor", e)
			}
			serviceMust(t, c.CleanupRetained())
			if service.frontend.receipt != receipt || service.scope != scope {
				t.Fatal("root construction cleanup consumed old factor")
			}
		})
	}
	if !found {
		t.Fatal("did not reach actual pending factorization")
	}
}
func TestAlgebraicSexprRootExecutingOwnerAndConverterRefusals(t *testing.T) {
	c, up, sx, sy, q, _ := algebraicSexprRootFixture(t)
	x := sxPolySym(t, sx, sy, "x")
	two := sxPolyNum(t, sx, q, "2")
	expr := sxPolyApp(t, sx, sy, "-", sxPolyApp(t, sx, sy, "^", x, two), two)
	old := up.config.Checkpoint
	var parsing, isolating uint32
	up.config.Checkpoint = func() bool {
		s := &c.rootConstruction
		if s.running {
			if s.parser != nil && s.parser.converting {
				parsing++
				header := s.parser.header
				if e := s.parser.Close(); !errors.Is(e, ErrSexprPolynomialPending) {
					t.Fatal("executing converter close", e)
				}
				if e := s.parser.Cleanup(); !errors.Is(e, ErrSexprPolynomialPending) {
					t.Fatal("executing converter cleanup", e)
				}
				if s.parser.header != header {
					t.Fatal("converter guard changed header")
				}
			} else {
				isolating++
			}
			if e := c.CleanupRetained(); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatal("executing root cleanup", e)
			}
			if e := c.CleanupIsolation(); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatal("executing root-vector cleanup", e)
			}
			if e := c.Close(); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatal("executing algebraic owner close", e)
			}
			var another Algebraic
			if e := c.MkSexprRoot(sx, expr, 2, &another); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatal("reentrant root expression", e)
			}
		}
		return old()
	}
	var out Algebraic
	defer func() { serviceMust(t, c.Del(&out)) }()
	serviceMust(t, c.MkSexprRoot(sx, expr, 2, &out))
	if parsing == 0 || isolating == 0 {
		t.Fatal("did not reach both parsing and root isolation", parsing, isolating)
	}
}
