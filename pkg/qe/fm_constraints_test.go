package qe

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func (f *fmFixture) store() *fmStore {
	f.t.Helper()
	s, err := newFMStore(&f.r)
	fmMust(f.t, err)
	f.t.Cleanup(func() { fmMust(f.t, s.Close()) })
	return s
}

func fmNumbers(t *testing.T, s *fmStore, c *fmConstraint, constant string, coefficients ...string) {
	t.Helper()
	n := s.recognizer.manager.NumeralService()
	text, err := n.String(&c.constant)
	fmMust(t, err)
	if text != constant {
		t.Fatalf("constant=%s, want %s", text, constant)
	}
	if len(c.coefficients) != len(coefficients) {
		t.Fatalf("coefficient count=%d, want %d", len(c.coefficients), len(coefficients))
	}
	for i, want := range coefficients {
		text, err := n.String(&c.coefficients[i])
		fmMust(t, err)
		if text != want {
			t.Fatalf("coefficient[%d]=%s, want %s", i, text, want)
		}
	}
}

func fmConstraintCase(f *fmFixture, mode string) (form ast.ASTHandle, constant string, coefficients []string, strict bool) {
	m := f.r.manager
	sort := f.integer
	if strings.HasPrefix(mode, "real") {
		sort = f.real
	}
	x, y := f.variable(0, sort), f.variable(1, sort)
	if strings.HasPrefix(mode, "real") {
		half := f.own(m.ArithmeticRealNumeralText("1/2"))
		product := f.own(m.MkArithmeticMul(half, x))
		if mode == "real-fraction" {
			other := f.own(m.MkArithmeticMul(f.own(m.ArithmeticRealNumeralText("-2/3")), y))
			return f.own(m.MkArithmeticLE(f.own(m.MkArithmeticAdd(product, other)), f.own(m.ArithmeticRealNumeralText("5/6")))), "5", []string{"3", "-4"}, false
		}
		if mode == "real-large" {
			product = f.own(m.MkArithmeticMul(f.own(m.ArithmeticRealNumeralText("18446744073709551617/7")), x))
			return f.own(m.MkArithmeticLE(product, f.own(m.ArithmeticRealNumeralText("36893488147419103234/7")))), "2", []string{"1"}, false
		}
		bound := f.own(m.MkArithmeticGE(product, f.own(m.ArithmeticRealNumeralText("1/3"))))
		return f.own(m.MkNot(bound)), "2", []string{"3"}, true
	}
	if mode == "coerced-int-strict" {
		coerced := f.own(m.MkFamilyApp(f.r.arithmetic, 14, nil, []ast.ASTHandle{x}))
		bound := f.own(m.MkArithmeticGE(coerced, f.own(m.ArithmeticRealNumeralText("7/3"))))
		return f.own(m.MkNot(bound)), "2", []string{"1"}, false
	}
	if mode == "mixed" {
		coerced := f.own(m.MkFamilyApp(f.r.arithmetic, 14, nil, []ast.ASTHandle{x}))
		real := f.variable(1, f.real)
		return f.own(m.MkArithmeticLE(f.own(m.MkArithmeticAdd(coerced, real)), f.own(m.ArithmeticRealNumeralText("3/2")))), "3", []string{"2", "2"}, false
	}
	product := f.own(m.MkArithmeticMul(f.number("2"), x))
	switch mode {
	case "integer-ge":
		return f.own(m.MkArithmeticGE(product, f.number("6"))), "-3", []string{"-1"}, false
	case "integer-not-ge":
		return f.own(m.MkNot(f.own(m.MkArithmeticGE(product, f.number("6"))))), "5", []string{"2"}, false
	case "integer-not-le":
		return f.own(m.MkNot(f.own(m.MkArithmeticLE(product, f.number("6"))))), "-7", []string{"-2"}, false
	case "clause":
		ineq := f.own(m.MkArithmeticLE(product, f.number("6")))
		p, q := f.constant("fm-p", f.boolean), f.constant("fm-q", f.boolean)
		return f.own(m.MkOrArray([]ast.ASTHandle{p, f.own(m.MkNot(q)), ineq})), "3", []string{"1"}, false
	default:
		second := f.own(m.MkArithmeticMul(f.number("4"), y))
		return f.own(m.MkArithmeticLE(f.own(m.MkArithmeticAdd(product, second)), f.number("6"))), "3", []string{"1", "2"}, false
	}
}

func TestFMConstraintParsingNormalizationAndExpression(t *testing.T) {
	for _, mode := range []string{"integer-sum", "integer-ge", "integer-not-ge", "integer-not-le", "real-fraction", "real-strict", "real-large", "coerced-int-strict", "mixed", "clause"} {
		t.Run(mode, func(t *testing.T) {
			f := newFMFixture(t)
			form, constant, coefficients, strict := fmConstraintCase(f, mode)
			yes, err := f.r.clause(form)
			fmMust(t, err)
			if !yes {
				t.Fatal("fixture not recognized")
			}
			s := f.store()
			c, err := s.parse(form)
			fmMust(t, err)
			fmMust(t, s.normalize(c))
			fmNumbers(t, s, c, constant, coefficients...)
			if c.strict != strict {
				t.Fatalf("strict=%v, want %v", c.strict, strict)
			}
			out, err := s.expression(c)
			fmMust(t, err)
			t.Cleanup(func() { fmMust(t, f.r.manager.DelRef(&out)) })
			h, err := f.r.manager.RefGet(out)
			fmMust(t, err)
			// The returned formula must own its coefficients and variables after
			// all source constraint storage and store-owned refs are destroyed.
			fmMust(t, s.Close())
			n, err := f.r.manager.GetNumArgs(h)
			fmMust(t, err)
			if n == 0 {
				t.Fatal("missing reconstructed formula")
			}
		})
	}
}

func TestFMConstraintSetSwapEraseAndOwnerGuards(t *testing.T) {
	f := newFMFixture(t)
	s := f.store()
	form, _, _, _ := fmConstraintCase(f, "integer-sum")
	var cs []*fmConstraint
	for i := 0; i < 3; i++ {
		c, err := s.parse(form)
		fmMust(t, err)
		cs = append(cs, c)
		s.insertTodo(c)
		s.insertTodo(c)
	}
	if len(s.todo) != 3 {
		t.Fatal("duplicate inserted")
	}
	s.eraseTodo(cs[0])
	if s.popTodo() != cs[1] || s.popTodo() != cs[2] || s.popTodo() != nil {
		t.Fatal("source swap-erase/LIFO order")
	}
	copy := *s
	if err := copy.Close(); err == nil {
		t.Fatal("copied store accepted")
	}
	fmMust(t, s.Close())
	if _, err := s.parse(form); err == nil {
		t.Fatal("closed store accepted")
	}
}

func TestFMConstraintCopiesConstructorOperands(t *testing.T) {
	f := newFMFixture(t)
	s := f.store()
	n := f.r.manager.NumeralService()
	x, err := s.variable(f.variable(0, f.integer))
	fmMust(t, err)
	y, err := s.variable(f.variable(1, f.integer))
	fmMust(t, err)
	p, err := s.boolean(f.constant("fm-p", f.boolean))
	fmMust(t, err)
	scope, err := s.begin()
	fmMust(t, err)
	a, err := scope.number()
	fmMust(t, err)
	fmMust(t, n.SetInt64(a, 2))
	b, err := scope.number()
	fmMust(t, err)
	fmMust(t, n.SetInt64(b, -3))
	constant, err := scope.number()
	fmMust(t, err)
	fmMust(t, n.SetInt64(constant, 4))
	literals, variables := []int32{p}, []uint32{x, y}
	c, err := s.newConstraint(literals, variables, []*numeral.OwnedRational{a, b}, constant, false)
	fmMust(t, err)
	literals[0] = -p
	variables[0] = y
	fmMust(t, n.SetInt64(a, 99))
	fmMust(t, n.SetInt64(constant, 99))
	s.finish(scope, &err)
	fmMust(t, err)
	fmNumbers(t, s, c, "4", "2", "-3")
	if c.literals[0] != p || c.variables[0] != x {
		t.Fatal("constraint aliases caller literal/variable slices")
	}
}

func TestFMConstraintAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"constructor", "parse", "normalize", "expression", "register", "register-constant", "subsume", "delete", "close", "trace", "closed-trace"} {
		t.Run(phase, func(t *testing.T) {
			success := false
			for stop := int64(0); stop < 4096 && !success; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					f := newFMFixture(t)
					form, _, _, _ := fmConstraintCase(f, "real-large")
					var trace bytes.Buffer
					if phase == "trace" || phase == "closed-trace" || phase == "subsume" || phase == "register-constant" {
						var writer io.Writer = &trace
						if phase == "closed-trace" {
							writer = fmClosedTrace{}
						}
						tags := diagnostic.NewTraceTagsWithWriter(writer)
						tags.Enable("qe_lite")
						t.Cleanup(tags.Finalize)
						f.r.config.Trace = tags
					}
					var s *fmStore
					var c *fmConstraint
					var out ast.ASTRef
					var err error
					if phase != "constructor" {
						s = f.store()
					}
					if phase == "normalize" || phase == "expression" || phase == "register" || phase == "subsume" || phase == "delete" || phase == "close" {
						c, err = s.parse(form)
						fmMust(t, err)
					}
					if phase == "register-constant" {
						p, err := s.boolean(f.constant("fm-p", f.boolean))
						fmMust(t, err)
						c = f.rawConstraint(s, []int32{p}, nil, nil, "-1", false)
					}
					if phase == "subsume" {
						_, err = s.register(c)
						fmMust(t, err)
						other, err := s.parse(form)
						fmMust(t, err)
						_, err = s.register(other)
						fmMust(t, err)
						trace.Reset()
					}
					f.r.manager.Budget().FailAfter(stop)
					switch phase {
					case "constructor":
						s, err = newFMStore(&f.r)
					case "normalize":
						err = s.normalize(c)
					case "expression":
						out, err = s.expression(c)
					case "register", "register-constant":
						_, err = s.register(c)
					case "subsume":
						err = s.subsume()
					case "delete":
						err = s.deleteConstraint(c)
					case "close":
						err = s.Close()
					default:
						c, err = s.parse(form)
					}
					f.r.manager.Budget().FailAfter(-1)
					var failure memory.ConstructionFailure
					if errors.As(err, &failure) {
						fmMust(t, failure.Cleanup())
					}
					if out != (ast.ASTRef{}) {
						fmMust(t, f.r.manager.DelRef(&out))
					}
					if phase == "constructor" && s != nil {
						fmMust(t, s.Close())
					}
					if err == nil {
						success = true
						if phase == "trace" && f.r.config.Trace.ShouldTrace("qe_lite") {
							for _, text := range []string{"src/qe/qe_lite.cpp:1697", "src/qe/qe_lite.cpp:1655", "src/qe/qe_lite.cpp:1677", "src/qe/qe_lite.cpp:1255", "before mk_constraint: ", "add_constraint: "} {
								if !strings.Contains(trace.String(), text) {
									t.Fatal("missing trace", text, trace.String())
								}
							}
						}
					} else if !errors.Is(err, arena.ErrInjectedFailure) {
						var memout memory.Failure
						if !errors.As(err, &memout) || memout.Kind != memory.FailureMemout {
							t.Fatal(err)
						}
					}
				})
				if t.Failed() {
					t.Fatalf("%s allocation frontier %d", phase, stop)
				}
			}
			if !success {
				t.Fatal("no successful frontier")
			}
		})
	}
}
