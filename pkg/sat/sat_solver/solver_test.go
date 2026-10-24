package sat_solver_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/parsers/smt2"
	incsat "github.com/glycerine/g3/pkg/sat/sat_solver"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func newSolverFixture(t *testing.T) (*encoderFixture, *incsat.Solver) {
	t.Helper()
	f := newEncoderFixture(t, false)
	s, err := incsat.NewIncrementalSolver(f.manager, &f.params, f.services)
	if s != nil {
		t.Cleanup(func() { incMust(t, s.Close()) })
	}
	incMust(t, err)
	return f, s
}
func solverForm(t *testing.T, f *encoderFixture, term string) ast.ASTHandle {
	t.Helper()
	g := f.goal(t, "(assert "+term+")")
	n, err := g.Size()
	incMust(t, err)
	if n != 1 {
		t.Fatal("expected one original formula", term, n)
	}
	h, err := g.Form(0)
	incMust(t, err)
	return h
}
func assertSolverModel(t *testing.T, f *encoderFixture, s *incsat.Solver, forms []ast.ASTHandle) {
	t.Helper()
	md, err := s.AcquireModel()
	incMust(t, err)
	if md == nil {
		t.Fatal("missing SAT model")
	}
	defer func() { incMust(t, md.DecRef()) }()
	ev, err := model.NewModelEvaluator(md, f.services)
	incMust(t, err)
	defer func() { incMust(t, ev.Close()) }()
	incMust(t, ev.SetModelCompletion(true))
	value, err := f.manager.NewRef(ast.ASTHandle{})
	incMust(t, err)
	defer func() { incMust(t, f.manager.DelRef(&value)) }()
	for _, h := range forms {
		incMust(t, ev.Evaluate(h, value))
		v, err := f.manager.RefGet(value)
		incMust(t, err)
		if !f.manager.IsTrue(v) {
			t.Fatal("original formula false in reconstructed model")
		}
	}
}

func TestIncrementalSolverChecksScopesAndModelsCPP(t *testing.T) {
	f, s := newSolverFixture(t)
	var script strings.Builder
	script.WriteString(`(set-option :ackermannization.sat_backend true)(set-option :ackermannization.inc_sat_backend true)(set-logic QF_UFBV)(declare-const x (_ BitVec 4))(declare-const y (_ BitVec 4))`)
	var answers []string
	check := func(want lbool.LBool) {
		t.Helper()
		got, err := s.CheckSat(nil)
		incMust(t, err)
		if got != want {
			t.Fatal("check", got, want)
		}
		answers = append(answers, map[lbool.LBool]string{lbool.True: "sat", lbool.False: "unsat"}[got])
		script.WriteString(`(check-sat-using qfufbv_ackr)`)
	}
	add := func(term string) ast.ASTHandle {
		t.Helper()
		h := solverForm(t, f, term)
		incMust(t, s.Assert(h))
		script.WriteString("(assert " + term + ")")
		return h
	}
	a := add(`(= x #x1)`)
	check(lbool.True)
	assertSolverModel(t, f, s, []ast.ASTHandle{a})
	incMust(t, s.Push())
	script.WriteString(`(push 1)`)
	add(`(= x #x2)`)
	check(lbool.False)
	incMust(t, s.Pop(1))
	script.WriteString(`(pop 1)`)
	check(lbool.True)
	assertSolverModel(t, f, s, []ast.ASTHandle{a})
	incMust(t, s.Push())
	script.WriteString(`(push 1)`)
	b := add(`(= y #x7)`)
	check(lbool.True)
	assertSolverModel(t, f, s, []ast.ASTHandle{a, b})
	incMust(t, s.Pop(100))
	script.WriteString(`(pop 1)`)
	b = add(`(= y #x9)`)
	check(lbool.True)
	assertSolverModel(t, f, s, []ast.ASTHandle{a, b})
	// A model returned to a caller survives complete backend destruction.
	md, err := s.AcquireModel()
	incMust(t, err)
	incMust(t, s.Close())
	ev, err := model.NewModelEvaluator(md, f.services)
	incMust(t, err)
	value, err := f.manager.NewRef(ast.ASTHandle{})
	incMust(t, err)
	for _, h := range []ast.ASTHandle{a, b} {
		incMust(t, ev.Evaluate(h, value))
		v, err := f.manager.RefGet(value)
		incMust(t, err)
		if !f.manager.IsTrue(v) {
			t.Fatal("model did not survive solver destruction")
		}
	}
	incMust(t, f.manager.DelRef(&value))
	incMust(t, ev.Close())
	incMust(t, md.DecRef())
	if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
		cmd := exec.Command(reference, "-in")
		cmd.Stdin = strings.NewReader(script.String())
		out, err := cmd.CombinedOutput()
		incMust(t, err)
		if strings.Join(strings.Fields(string(out)), " ") != strings.Join(answers, " ") {
			t.Fatal("C++ incremental checks", string(out), answers)
		}
	}
}

func TestIncrementalSolverAssumptionsAndTrackedCore(t *testing.T) {
	f, s := newSolverFixture(t)
	p := solverForm(t, f, `p`)
	q := solverForm(t, f, `q`)
	np := solverForm(t, f, `(not p)`)
	incMust(t, s.AssertTracked(p, q))
	got, err := s.CheckSat([]ast.ASTHandle{np})
	incMust(t, err)
	if got != lbool.False {
		t.Fatal(got)
	}
	core, err := s.UnsatCore()
	incMust(t, err)
	seen := map[ast.ASTHandle]bool{}
	for _, h := range core {
		seen[h] = true
	}
	if len(core) != 2 || !seen[np] || !seen[q] {
		t.Fatal("tracked assumption core", core)
	}
	got, err = s.CheckSat(nil)
	incMust(t, err)
	if got != lbool.True {
		t.Fatal(got)
	}
	assertSolverModel(t, f, s, []ast.ASTHandle{p, q})
	retained, err := s.UnsatCore()
	incMust(t, err)
	if len(retained) != len(core) {
		t.Fatal("source retains the previous core after SAT")
	}
	complex := solverForm(t, f, `(or (not p) (not q))`)
	before, err := s.NumAssertions()
	incMust(t, err)
	got, err = s.CheckSat([]ast.ASTHandle{complex})
	incMust(t, err)
	if got != lbool.False {
		t.Fatal(got)
	}
	core, err = s.UnsatCore()
	incMust(t, err)
	seen = map[ast.ASTHandle]bool{}
	for _, h := range core {
		seen[h] = true
	}
	if len(core) != 2 || !seen[complex] || !seen[q] {
		t.Fatal("complex assumption not restored in core", core)
	}
	after, err := s.NumAssertions()
	incMust(t, err)
	if after != before+1 {
		t.Fatal("source complex-assumption guard was not retained")
	}
	got, err = s.CheckSat(nil)
	incMust(t, err)
	if got != lbool.True {
		t.Fatal("assumption guard remained activated", got)
	}
}

func TestIncrementalSolverAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"constructor", "check", "assumption", "model", "remodel", "interpreted"} {
		t.Run(phase, func(t *testing.T) {
			success := false
			for stop := int64(0); stop < 8192 && !success; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					f := newEncoderFixture(t, false)
					var s *incsat.Solver
					var err error
					if phase != "constructor" {
						s, err = incsat.NewIncrementalSolver(f.manager, &f.params, f.services)
						incMust(t, err)
					}
					t.Cleanup(func() {
						f.manager.Budget().FailAfter(-1)
						if s != nil {
							incMust(t, s.Close())
						}
					})
					var assumption ast.ASTHandle
					if phase != "constructor" {
						incMust(t, s.Assert(solverForm(t, f, `(= (bvadd x y) #x3)`)))
						if phase == "assumption" {
							assumption = solverForm(t, f, `(= x #x1)`)
						}
						if phase == "model" || phase == "remodel" {
							status, e := s.CheckSat(nil)
							incMust(t, e)
							if status != lbool.True {
								t.Fatal(status)
							}
						}
						if phase == "remodel" {
							md, e := s.AcquireModel()
							incMust(t, e)
							incMust(t, md.DecRef())
							incMust(t, s.Push())
							incMust(t, s.Assert(solverForm(t, f, `(= x #x2)`)))
							status, e := s.CheckSat(nil)
							incMust(t, e)
							if status != lbool.True {
								t.Fatal(status)
							}
						}
						if phase == "interpreted" {
							// A positive-arity uninterpreted application survives the
							// pipeline and reaches the vector diagnostic formatter.
							incMust(t, smt2.ParseCommands(f.context, strings.NewReader(`(declare-fun fn ((_ BitVec 4)) (_ BitVec 4))`), io.Discard))
							incMust(t, s.Assert(solverForm(t, f, `(= (fn x) y)`)))
						}
					}
					f.manager.Budget().FailAfter(stop)
					switch phase {
					case "constructor":
						s, err = incsat.NewIncrementalSolver(f.manager, &f.params, f.services)
					case "model", "remodel":
						var md *model.ModelCore
						md, err = s.AcquireModel()
						f.manager.Budget().FailAfter(-1)
						if md != nil {
							incMust(t, md.DecRef())
						}
					case "assumption":
						_, err = s.CheckSat([]ast.ASTHandle{assumption})
					default:
						_, err = s.CheckSat(nil)
					}
					f.manager.Budget().FailAfter(-1)
					if err == nil {
						success = true
					} else if !errors.Is(err, arena.ErrInjectedFailure) {
						t.Fatal(err)
					}
				})
				if t.Failed() {
					t.Fatal("reservation frontier", phase, stop)
				}
			}
			if !success {
				t.Fatal("no successful reservation frontier", phase)
			}
		})
	}
}

func TestIncrementalSolverUnknownDiagnosticAndRecovery(t *testing.T) {
	f, s := newSolverFixture(t)
	md, err := s.AcquireModel()
	incMust(t, err)
	if md != nil {
		incMust(t, md.DecRef())
		t.Fatal("unchecked solver has model")
	}
	incMust(t, smt2.ParseCommands(f.context, strings.NewReader(`(declare-fun fn (Bool) Bool)`), io.Discard))
	incMust(t, s.Push())
	incMust(t, s.Assert(solverForm(t, f, `(fn p)`)))
	got, err := s.CheckSat(nil)
	incMust(t, err)
	if got != lbool.Undef {
		t.Fatal("UF atom accepted by SAT", got)
	}
	reason, err := s.ReasonUnknown()
	incMust(t, err)
	if reason != "interpreted atoms sent to SAT solver \n(fn p)" {
		t.Fatalf("source vector diagnostic: %q", reason)
	}
	incMust(t, s.Pop(1))
	h := solverForm(t, f, `(= x #x3)`)
	incMust(t, s.Assert(h))
	got, err = s.CheckSat(nil)
	incMust(t, err)
	if got != lbool.True {
		t.Fatal(got)
	}
	assertSolverModel(t, f, s, []ast.ASTHandle{h})
}

func TestIncrementalSolverWideModelsCPP(t *testing.T) {
	for _, width := range []int{1, 4, 65, 257} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			f, s := newSolverFixture(t)
			declarations := fmt.Sprintf(`(declare-const wide (_ BitVec %d))(declare-const rhs (_ BitVec %d))`, width, width)
			incMust(t, smt2.ParseCommands(f.context, strings.NewReader(declarations), io.Discard))
			terms := []string{fmt.Sprintf(`(= wide (_ bv1 %d))`, width), fmt.Sprintf(`(= rhs (bvadd wide (_ bv1 %d)))`, width)}
			var forms []ast.ASTHandle
			for _, term := range terms {
				h := solverForm(t, f, term)
				forms = append(forms, h)
				incMust(t, s.Assert(h))
			}
			got, err := s.CheckSat(nil)
			incMust(t, err)
			if got != lbool.True {
				t.Fatal(got)
			}
			assertSolverModel(t, f, s, forms)
			negated, err := f.manager.MkNot(forms[0])
			incMust(t, err)
			incMust(t, s.Assert(negated))
			got, err = s.CheckSat(nil)
			incMust(t, err)
			if got != lbool.False {
				t.Fatal("fixed-value contradiction", got)
			}
			if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
				script := `(set-option :ackermannization.sat_backend true)(set-option :ackermannization.inc_sat_backend true)(set-logic QF_UFBV)` + declarations
				for _, term := range terms {
					script += "(assert " + term + ")"
				}
				script += `(check-sat-using qfufbv_ackr)`
				script += "(assert (not " + terms[0] + "))"
				script += `(check-sat-using qfufbv_ackr)`
				cmd := exec.Command(reference, "-in")
				cmd.Stdin = strings.NewReader(script)
				out, err := cmd.CombinedOutput()
				incMust(t, err)
				if strings.Join(strings.Fields(string(out)), " ") != "sat unsat" {
					t.Fatal("C++ fixed-value contradiction", string(out))
				}
			}
		})
	}
}

func TestIncrementalSolverBaseAndShadowParameters(t *testing.T) {
	f, s := newSolverFixture(t)
	c := f.services.Context
	key := "max_steps"
	incMust(t, c.SetUIntName(&f.params, &key, 0))
	incMust(t, s.UpdateParams(&f.params))
	p, err := s.Params()
	incMust(t, err)
	value, err := c.GetUIntName(p, &key, 99)
	incMust(t, err)
	if value != 0 {
		t.Fatal("base parameter snapshot", value)
	}
	incMust(t, c.SetUIntName(&f.params, &key, 100))
	value, err = c.GetUIntName(p, &key, 99)
	incMust(t, err)
	if value != 0 {
		t.Fatal("base params aliased caller")
	}
	h := solverForm(t, f, `(= (bvadd x y) #x3)`)
	incMust(t, s.Assert(h))
	got, err := s.CheckSat(nil)
	incMust(t, err)
	if got != lbool.True {
		t.Fatal("user base max_steps leaked into shadow internal params", got)
	}
	assertSolverModel(t, f, s, []ast.ASTHandle{h})
}

func TestIncrementalSolverProofRefusal(t *testing.T) {
	f, s := newSolverFixture(t)
	incMust(t, s.Assert(solverForm(t, f, `p`)))
	incMust(t, f.manager.ToggleProofMode(ast.ASTProofEnabled))
	got, err := s.CheckSat(nil)
	var refusal z3exception.DefaultException
	if got != lbool.Undef || !errors.As(err, &refusal) || err.Error() != "generation of proof objects is not supported in this mode" {
		t.Fatal("source proof refusal", got, err)
	}
}
