package api

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/interp"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

type interpolationFixture struct {
	t *testing.T
	c *InterpolationContext
	m *ast.ASTManager
	w *interp.Manager
}

func newInterpolationFixture(t *testing.T) *interpolationFixture {
	t.Helper()
	c, err := NewInterpolationContext()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
		if used := c.budget.Used(); used != 0 {
			t.Errorf("context retained %d bytes", used)
		}
	})
	return &interpolationFixture{t, c, c.manager, c.terms}
}

func (f *interpolationFixture) pin(h ast.ASTHandle, err error) ast.ASTHandle {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
	ref, err := f.m.NewRef(h)
	if err != nil {
		f.t.Fatal(err)
	}
	f.t.Cleanup(func() {
		if err := f.m.DelRef(&ref); err != nil {
			f.t.Error(err)
		}
	})
	return h
}

func (f *interpolationFixture) variable(name string, sort ast.ASTHandle) ast.ASTHandle {
	return f.pin(f.w.MakeVariable(name, sort))
}

func (f *interpolationFixture) make(op interp.Operator, args ...ast.ASTHandle) ast.ASTHandle {
	return f.pin(f.w.Make(op, args...))
}

func (f *interpolationFixture) verify(pattern ast.ASTHandle, results []ast.ASTHandle) {
	f.t.Helper()
	constraints, err := interp.CollectConjuncts(f.w, pattern)
	if err != nil {
		f.t.Fatal(err)
	}
	s, err := f.c.factory.New(f.m, nil, true, false, false, "")
	if err != nil {
		f.t.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			f.t.Error(err)
		}
	}()
	var diagnostics bytes.Buffer
	valid, err := interp.CheckTreeInterpolants(f.w, s, &diagnostics, constraints, pattern, results)
	if err != nil || !valid {
		f.t.Fatalf("invalid interpolant: %v: %s", err, diagnostics.String())
	}
}

func TestInterpolationContextBinary(t *testing.T) {
	for _, name := range []string{"boolean", "integer", "local-integer", "euf", "array", "quantifier", "local-false"} {
		t.Run(name, func(t *testing.T) {
			f := newInterpolationFixture(t)
			boolean := f.pin(f.w.BoolType())
			integer := f.pin(f.w.IntType())
			p := f.variable("p", boolean)
			x := f.variable("x", integer)
			y := f.variable("y", integer)
			zero := f.pin(f.w.MakeIntText("0"))
			two := f.pin(f.w.MakeIntText("2"))
			a, b := p, f.make(interp.OperatorNot, p)
			switch name {
			case "integer":
				a = f.make(interp.OperatorLT, x, zero)
				b = f.make(interp.OperatorGT, x, two)
			case "local-integer":
				a = f.make(interp.OperatorAnd, f.make(interp.OperatorEqual, y, f.make(interp.OperatorPlus, x, two)), f.make(interp.OperatorLE, y, zero))
				b = f.make(interp.OperatorGE, x, zero)
			case "euf":
				u := f.pin(f.w.Function("f", []ast.ASTHandle{integer}, integer))
				fx := f.pin(f.m.MkApp(u, []ast.ASTHandle{x}))
				fy := f.pin(f.m.MkApp(u, []ast.ASTHandle{y}))
				a = f.make(interp.OperatorEqual, x, y)
				b = f.make(interp.OperatorNot, f.make(interp.OperatorEqual, fx, fy))
			case "array":
				array := f.pin(f.w.ArrayType(integer, integer))
				ar := f.variable("ar", array)
				stored := f.make(interp.OperatorStore, ar, x, y)
				selected := f.make(interp.OperatorSelect, stored, x)
				a = f.make(interp.OperatorEqual, y, zero)
				b = f.make(interp.OperatorNot, f.make(interp.OperatorEqual, selected, zero))
			case "quantifier":
				pred := f.pin(f.w.Function("P", []ast.ASTHandle{integer}, boolean))
				px := f.pin(f.m.MkApp(pred, []ast.ASTHandle{x}))
				py := f.pin(f.m.MkApp(pred, []ast.ASTHandle{y}))
				a = f.pin(f.w.MakeQuantifier(interp.OperatorForall, []ast.ASTHandle{x}, px))
				b = f.make(interp.OperatorNot, py)
			case "local-false":
				a = f.make(interp.OperatorAnd, p, f.make(interp.OperatorNot, p))
				b = f.make(interp.OperatorGT, x, zero)
			}
			marked := f.pin(f.c.MkInterpolant(a))
			pattern := f.make(interp.OperatorAnd, marked, b)
			for attempt := 0; attempt < 2; attempt++ {
				results, err := f.c.ComputeInterpolant(pattern)
				if err != nil {
					t.Fatal(err)
				}
				if len(results) != 1 {
					t.Fatalf("got %d interpolants", len(results))
				}
				f.verify(pattern, results)
			}
		})
	}
}

func TestInterpolationContextTree(t *testing.T) {
	f := newInterpolationFixture(t)
	boolean := f.pin(f.w.BoolType())
	p, q := f.variable("p", boolean), f.variable("q", boolean)
	first := f.pin(f.c.MkInterpolant(p))
	second := f.pin(f.c.MkInterpolant(q))
	root := f.make(interp.OperatorNot, f.make(interp.OperatorAnd, p, q))
	branching := f.make(interp.OperatorAnd, first, second, root)
	nested := f.make(interp.OperatorAnd, f.pin(f.c.MkInterpolant(f.make(interp.OperatorAnd, first, q))), root)
	for _, pattern := range []ast.ASTHandle{branching, nested} {
		results, err := f.c.ComputeInterpolant(pattern)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 2 {
			t.Fatalf("got %d tree interpolants", len(results))
		}
		f.verify(pattern, results)
	}
}

type unknownInterpolationSolver struct{ solver.Solver }

func (s unknownInterpolationSolver) CheckSat([]ast.ASTHandle) (lbool.LBool, error) {
	return lbool.Undef, nil
}

func TestInterpolationContextErrorsAndIsolation(t *testing.T) {
	f := newInterpolationFixture(t)
	g := newInterpolationFixture(t)
	integer := f.pin(f.w.IntType())
	x := f.variable("x", integer)
	if _, err := f.c.MkInterpolant(x); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("integer marker", err)
	}
	if _, err := f.c.ComputeInterpolant(x); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("integer pattern", err)
	}
	boolean := f.pin(f.w.BoolType())
	p := f.variable("p", boolean)
	if _, err := g.c.MkInterpolant(p); err == nil {
		t.Fatal("accepted foreign AST")
	}
	pattern := f.pin(f.c.MkInterpolant(p))
	if _, err := f.c.ComputeInterpolant(pattern); !errors.Is(err, ErrInterpolationSAT) {
		t.Fatal("SAT", err)
	}
	factory := f.c.factory
	f.c.factory = solver.FactoryFunc(func(m *ast.ASTManager, p *uparams.ParamsRef, proofs, models, cores bool, logic string) (solver.Solver, error) {
		s, err := factory.New(m, p, proofs, models, cores, logic)
		return unknownInterpolationSolver{s}, err
	})
	_, err := f.c.ComputeInterpolant(pattern)
	f.c.factory = factory
	if !errors.Is(err, ErrInterpolationUnknown) {
		t.Fatal("UNKNOWN", err)
	}
	copyContext := *f.c
	if _, err := copyContext.MkInterpolant(p); err == nil {
		t.Fatal("accepted copied context")
	}
	c, err := NewInterpolationContext()
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Close(); err != nil {
		t.Fatal(err)
	}
	if err = c.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = c.ComputeInterpolant(p); err == nil {
		t.Fatal("accepted closed context")
	}
}
