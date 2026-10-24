package ackermannization_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ackermannization"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/smtcore"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

type lackrFixture struct {
	m        *ast.ASTManager
	services model.ModelEvaluatorConfig
	params   uparams.ParamsRef
	forms    []ast.ASTHandle
	backend  *smtcore.SMTSolver
	stats    ackermannization.Stats
	r        *ackermannization.Lackr
}

func newLackrFixture(t *testing.T, script string, eager bool, construct bool) *lackrFixture {
	t.Helper()
	c := ackContext(t, script)
	m, err := c.Manager()
	ackMust(t, err)
	services := ackServices(t, c)
	f := &lackrFixture{m: m, services: services}
	f.params, err = services.Context.NewValue()
	ackMust(t, err)
	t.Cleanup(func() { ackMust(t, services.Context.Del(&f.params)) })
	key := "eager"
	ackMust(t, services.Context.SetBoolName(&f.params, &key, eager))
	g, err := tactic.NewGoal(m, false, true, false)
	ackMust(t, err)
	t.Cleanup(func() { ackMust(t, g.Close()) })
	ackMust(t, c.AssertGoal(g))
	n, err := g.Size()
	ackMust(t, err)
	for i := uint32(0); i < n; i++ {
		h, err := g.Form(i)
		ackMust(t, err)
		f.forms = append(f.forms, h)
	}
	f.backend, err = smtcore.NewSMTSolver(m, nil, "QF_BV", smtcore.SMTSolverConfig{Params: services.Context, Globals: services.Globals, ModelEvaluator: services})
	ackMust(t, err)
	t.Cleanup(func() { ackMust(t, f.backend.Close()) })
	ackMust(t, f.backend.SetProduceModels(true))
	if construct {
		f.r, err = ackermannization.NewLackr(m, &f.params, &f.stats, f.forms, f.backend, services)
		if f.r != nil {
			t.Cleanup(func() { ackMust(t, f.r.Close()) })
		}
		ackMust(t, err)
	}
	return f
}

func TestLackrSolverEagerLazyCPPAndModelLifetime(t *testing.T) {
	const decls = `(set-option :produce-models true)(set-logic QF_UFBV)(declare-const x (_ BitVec 4))(declare-const y (_ BitVec 4))(declare-fun f ((_ BitVec 4)) (_ BitVec 4))(declare-fun g ((_ BitVec 4)) (_ BitVec 4))`
	for _, tc := range []struct {
		name, forms string
		want        lbool.LBool
		refine      bool
	}{
		{"empty", "", lbool.True, false},
		{"inconsistent-abstraction", `(assert false)`, lbool.False, false},
		{"one-application", `(assert (= (f x) #x3))`, lbool.True, false},
		{"satisfiable-congruence", `(assert (not (= (f x) (f y))))`, lbool.True, true},
		{"conflicting-congruence", `(assert (= x y))(assert (not (= (f x) (f y))))`, lbool.False, true},
		{"nested-conflict", `(assert (= x y))(assert (not (= (g (f x)) (g (f y)))))`, lbool.False, true},
		{"computed-arguments", `(assert (= (bvadd x #x1) y))(assert (not (= (f (bvadd x #x1)) (f y))))`, lbool.False, true},
		{"distinct-values", `(assert (not (= (f #x1) (f #x2))))`, lbool.True, false},
	} {
		for _, eager := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/eager-%v", tc.name, eager), func(t *testing.T) {
				f := newLackrFixture(t, decls+tc.forms, eager, true)
				var trace bytes.Buffer
				tags := diagnostic.NewTraceTagsWithWriter(&trace)
				tags.Enable("lackr")
				ackMust(t, f.r.BindTrace(tags))
				status, err := f.r.Solve()
				ackMust(t, err)
				if status != tc.want {
					t.Fatal("solver status", status, tc.want)
				}
				if tc.refine && tc.want == lbool.False && f.stats.Constraints == 0 {
					t.Fatal("contradiction bypassed congruence encoding")
				}
				if eager && f.stats.Iterations != 0 || !eager && f.stats.Iterations == 0 {
					t.Fatal("eager/lazy iteration statistics", f.stats)
				}
				if tags.ShouldTrace("lackr") {
					label := "run sat 0"
					if !eager {
						label = "lazy check: 1"
					}
					if !strings.Contains(trace.String(), label) {
						t.Fatal("solver path trace", trace.String())
					}
				}
				if os.Getenv("G3_CPP_REFERENCE") != "" {
					cpp := strings.TrimSpace(ackCPP(t, fmt.Sprintf("(set-option :ackermannization.eager %v)", eager)+decls+tc.forms+`(check-sat-using qfufbv_ackr)`))
					if cpp != map[lbool.LBool]string{lbool.True: "sat", lbool.False: "unsat"}[status] {
						t.Fatal("C++ Ackermannization", cpp, status)
					}
				}
				if status != lbool.True {
					return
				}
				info, err := f.r.Info()
				ackMust(t, err)
				t.Cleanup(func() { ackMust(t, info.DecRef()) })
				md, err := f.r.AcquireModel()
				ackMust(t, err)
				if md == nil {
					t.Fatal("satisfiable reduction omitted its abstract model")
				}
				mc, err := ackermannization.NewFixedModelConverter(f.m, info, md, f.services)
				ackMust(t, md.DecRef())
				ackMust(t, err)
				t.Cleanup(func() { ackMust(t, mc.Close()) })
				ackMust(t, f.r.Close())
				ackMust(t, f.backend.Close())
				var out tactic.ModelRef
				t.Cleanup(func() { ackMust(t, out.Close()) })
				ackMust(t, mc.ApplyModel(&out, 0))
				ackMust(t, mc.Close())
				concrete, err := out.Get()
				ackMust(t, err)
				for i := uint32(0); ; i++ {
					n, err := concrete.NumConstants()
					ackMust(t, err)
					if i >= n {
						break
					}
					d, err := concrete.Constant(i)
					ackMust(t, err)
					name, err := f.m.GetASTName(d)
					ackMust(t, err)
					s, err := f.m.Symbols.String(name)
					ackMust(t, err)
					if s != "x" && s != "y" {
						t.Fatal("abstraction constant escaped reconstruction", s)
					}
				}
				ev, err := model.NewModelEvaluator(concrete, f.services)
				ackMust(t, err)
				t.Cleanup(func() { ackMust(t, ev.Close()) })
				ackMust(t, ev.SetModelCompletion(true))
				value, err := f.m.NewRef(ast.ASTHandle{})
				ackMust(t, err)
				t.Cleanup(func() { ackMust(t, f.m.DelRef(&value)) })
				for _, original := range f.forms {
					ackMust(t, ev.Evaluate(original, value))
					v, err := f.m.RefGet(value)
					ackMust(t, err)
					if !f.m.IsTrue(v) {
						t.Fatal("reconstructed model violates an original formula")
					}
				}
			})
		}
	}
}

type lackrScriptBackend struct {
	solver.Solver
	checks int
	first  lbool.LBool
}

func (s *lackrScriptBackend) CheckSat(a []ast.ASTHandle) (lbool.LBool, error) {
	s.checks++
	if s.checks == 1 {
		return s.first, nil
	}
	return s.Solver.CheckSat(a)
}

func TestLackrUnknownFirstCheckAndConstructorSnapshot(t *testing.T) {
	for _, eager := range []bool{true, false} {
		t.Run(fmt.Sprint(eager), func(t *testing.T) {
			f := newLackrFixture(t, `(set-logic QF_UFBV)(declare-const p Bool)(assert p)`, eager, false)
			backend := &lackrScriptBackend{Solver: f.backend, first: lbool.Undef}
			r, err := ackermannization.NewLackr(f.m, &f.params, &f.stats, f.forms, backend, f.services)
			ackMust(t, err)
			t.Cleanup(func() { ackMust(t, r.Close()) })
			// Update changes eager selection but does not replace the saved
			// simplifier policy. An unrelated max_steps=0 must not take effect.
			selected := !eager
			mode := "eager"
			ackMust(t, f.services.Context.SetBoolName(&f.params, &mode, selected))
			key := "max_steps"
			ackMust(t, f.services.Context.SetUIntName(&f.params, &key, 0))
			ackMust(t, r.UpdateParams(&f.params))
			status, err := r.Solve()
			ackMust(t, err)
			if selected && (backend.checks != 2 || status != lbool.True) {
				t.Fatal("eager mode stopped after initial unknown", backend.checks, status)
			}
			if !selected && (backend.checks != 1 || status != lbool.Undef) {
				t.Fatal("lazy mode continued after unknown", backend.checks, status)
			}
		})
	}
}

type lackrClosedWriter struct{}

func (lackrClosedWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestLackrQuantifierRefusalCancellationAndClosedTrace(t *testing.T) {
	for _, eager := range []bool{true, false} {
		t.Run(fmt.Sprint("quantifier/", eager), func(t *testing.T) {
			f := newLackrFixture(t, `(set-logic ALL)(declare-fun f (Bool) Bool)(assert (forall ((x Bool)) (f x)))`, eager, true)
			status, err := f.r.Solve()
			ackMust(t, err)
			if status != lbool.Undef || f.stats != (ackermannization.Stats{}) {
				t.Fatal("quantified input was solved or encoded", status, f.stats)
			}
		})
		t.Run(fmt.Sprint("cancellation/", eager), func(t *testing.T) {
			f := newLackrFixture(t, `(set-logic QF_UFBV)`, eager, false)
			backend := &lackrScriptBackend{Solver: f.backend, first: lbool.False}
			r, err := ackermannization.NewLackr(f.m, &f.params, &f.stats, nil, backend, f.services)
			ackMust(t, err)
			t.Cleanup(func() { ackMust(t, r.Close()) })
			f.m.ResourceLimit().IncCancel()
			status, err := r.Solve()
			f.m.ResourceLimit().DecCancel()
			if eager {
				ackMust(t, err)
				if status != lbool.False || backend.checks != 1 {
					t.Fatal("eager added an unreached checkpoint", status, backend.checks)
				}
			} else {
				var exception tactic.TacticException
				if !errors.As(err, &exception) || !strings.Contains(err.Error(), "canceled") || backend.checks != 0 || f.stats.Iterations != 1 {
					t.Fatal("lazy cancellation ordering", err, backend.checks, f.stats)
				}
			}
		})
		t.Run(fmt.Sprint("closed-trace/", eager), func(t *testing.T) {
			f := newLackrFixture(t, `(set-logic QF_UFBV)(declare-const x Bool)(declare-fun f (Bool) Bool)(assert (f x))`, eager, true)
			tags := diagnostic.NewTraceTagsWithWriter(lackrClosedWriter{})
			tags.Enable("lackr")
			tags.Enable("model_constructor")
			ackMust(t, f.r.BindTrace(tags))
			status, err := f.r.Solve()
			ackMust(t, err)
			if status != lbool.True {
				t.Fatal("trace stream changed the answer", status)
			}
		})
	}
}

func TestLackrAllocationFrontiers(t *testing.T) {
	for _, eager := range []bool{true, false} {
		for _, phase := range []string{"constructor", "solve"} {
			t.Run(fmt.Sprintf("%v/%s", eager, phase), func(t *testing.T) {
				success := false
				for stop := int64(0); stop < 4096 && !success; stop++ {
					t.Run(fmt.Sprint(stop), func(t *testing.T) {
						f := newLackrFixture(t, `(set-logic QF_UFBV)(declare-const x Bool)(declare-const y Bool)(declare-fun f (Bool) Bool)(assert (= x y))(assert (not (= (f x) (f y))))`, eager, phase == "solve")
						f.m.Budget().FailAfter(stop)
						var err error
						if phase == "constructor" {
							f.r, err = ackermannization.NewLackr(f.m, &f.params, &f.stats, f.forms, f.backend, f.services)
							f.m.Budget().FailAfter(-1)
							if f.r != nil {
								ackMust(t, f.r.Close())
							}
						} else {
							var status lbool.LBool
							status, err = f.r.Solve()
							f.m.Budget().FailAfter(-1)
							if err == nil && status != lbool.False {
								t.Fatal("congruence contradiction lost", status)
							}
						}
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
					t.Fatal("no successful reservation frontier")
				}
			})
		}
	}
}
