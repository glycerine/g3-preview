package tactic

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/cooperate"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func satModelFixture(t *testing.T) (*ast.ASTManager, model.ModelEvaluatorConfig) {
	t.Helper()
	m := astTestManager(t)
	satOK(t, m.RegisterDeclarationPlugins())
	c, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	satOK(t, err)
	t.Cleanup(func() { satOK(t, c.Close()) })
	catalog, err := uparams.NewGlobalParamsCatalog(m.Symbols, uparams.GlobalParamsCatalogConfig{Budget: m.Budget()})
	satOK(t, err)
	t.Cleanup(func() { satOK(t, catalog.Close()) })
	globals, err := uparams.NewGlobalParamsService(m.Symbols, c, uparams.GlobalParamsConfig{Budget: m.Budget(), Catalog: catalog})
	satOK(t, err)
	satOK(t, globals.InitializeSource())
	t.Cleanup(func() { satOK(t, globals.Close()) })
	coop, err := cooperate.NewCooperation(m.Budget())
	satOK(t, err)
	t.Cleanup(func() { satOK(t, coop.Close()) })
	return m, model.ModelEvaluatorConfig{Context: c, Globals: globals, Memory: m.NumeralService().RuntimeOwner(), Cooperation: coop}
}

func satModelSolver(t *testing.T, m *ast.ASTManager, services model.ModelEvaluatorConfig) *sat.Solver {
	t.Helper()
	c := services.Context
	p, err := c.NewValue()
	satOK(t, err)
	t.Cleanup(func() { satOK(t, c.Del(&p)) })
	name := "max_conflicts"
	satOK(t, c.SetUIntName(&p, &name, 0))
	runtime, err := sat.NewRuntime(sat.RuntimeConfig{Budget: m.Budget(), ParamsContext: c, Process: m.NumeralService().RuntimeOwner().ProcessOwner()})
	satOK(t, err)
	t.Cleanup(func() { satOK(t, runtime.Close()) })
	s, err := sat.NewSATSolver(sat.SolverConfig{ParamsContext: c, Params: p, RLimit: m.ResourceLimit(), Budget: m.Budget(), Runtime: runtime})
	satOK(t, err)
	t.Cleanup(func() { satOK(t, s.Close()) })
	return s
}

func TestSATGoalRecoveryModelConverterCPP(t *testing.T) {
	for _, op := range []string{"or", "iff", "ite"} {
		t.Run(op, func(t *testing.T) {
			m, services := satModelFixture(t)
			s := satModelSolver(t, m, services)
			tr, err := NewGoalTranslator(m, s)
			satOK(t, err)
			t.Cleanup(func() { satOK(t, tr.Close()) })
			p := goalTestAtom(t, m, "p")
			q := goalTestAtom(t, m, "q")
			x := goalTestAtom(t, m, "r")
			f := goalTestApp(t, m, ast.ASTOpOr, p, q)
			source := "(or p q)"
			if op == "iff" {
				f = goalTestApp(t, m, ast.ASTOpIff, f, x)
				source = "(= (or p q) r)"
			}
			if op == "ite" {
				f = goalTestApp(t, m, ast.ASTOpIte, p, f, x)
				source = "(ite p (or p q) r)"
			}
			g := goalTestNew(t, m, false, false)
			satOK(t, g.Assert(f))
			satOK(t, tr.CompileGoal(g))
			satOK(t, g.Reset())
			status, err := s.Check(nil)
			satOK(t, err)
			if status != lbool.Undef {
				t.Fatal("max_conflicts=0 source frontier", status)
			}
			satOK(t, s.PopToBaseLevel())
			mc, err := RecoverSATGoal(tr, g, false, ^uint64(0), services)
			satOK(t, err)
			if mc == nil {
				t.Fatal("missing recovery model converter")
			}
			t.Cleanup(func() { satOK(t, mc.Close()) })
			var actual bytes.Buffer
			satOK(t, mc.Display(&actual))
			satOK(t, tr.Close())
			satOK(t, s.Close()) // Copied elimination entries outlive the solver.
			if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
				cmd := exec.Command(reference, "-in")
				cmd.Stdin = strings.NewReader("(set-option :produce-models true)(declare-const p Bool)(declare-const q Bool)(declare-const r Bool)(assert " + source + ")(apply sat-preprocess :print false :print_model_converter true)")
				cpp, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatal(err, string(cpp))
				}
				fresh := regexp.MustCompile(`k![0-9]+`)
				if fresh.ReplaceAllString(actual.String(), "helper") != fresh.ReplaceAllString(string(cpp), "helper") {
					t.Fatalf("converter differs\nGo: %s\nC++: %s", actual.String(), cpp)
				}
			}
			// A model of the recovered clauses is lifted and loses every helper.
			md, err := model.NewModel(m)
			satOK(t, err)
			var out tactic.ModelRef
			satOK(t, out.Set(md.Core()))
			t.Cleanup(func() { satOK(t, out.Close()) })
			yes, err := m.MkTrue()
			satOK(t, err)
			converter := mc.(*SATModelConverter)
			eliminated, err := converter.conversion.AppendVariables(nil)
			satOK(t, err)
			removed := make(map[sat.BoolVar]bool)
			for _, v := range eliminated {
				removed[v] = true
			}
			for v, atom := range converter.atoms {
				if removed[sat.BoolVar(v)] {
					continue
				}
				d, err := m.GetDecl(atom)
				satOK(t, err)
				satOK(t, md.Core().RegisterConstant(d, yes))
			}
			satOK(t, mc.ApplyUnindexed(&out))
			got, err := out.Get()
			satOK(t, err)
			n, err := got.NumConstants()
			satOK(t, err)
			want := uint32(3)
			if op == "or" {
				want = 2
			}
			if n != want {
				t.Fatal("helper count", n, want)
			}
		})
	}
}

func TestSATModelConverterReconstructionAndAuxiliaryFiltering(t *testing.T) {
	for _, qValue := range []bool{false, true} {
		t.Run(fmt.Sprint(qValue), func(t *testing.T) {
			m, services := satModelFixture(t)
			var trace bytes.Buffer
			services.Trace = diagnostic.NewTraceTagsWithWriter(&trace)
			services.Trace.Enable("sat_mc")
			r, err := newSATModelConverter(m, services)
			satOK(t, err)
			t.Cleanup(func() { satOK(t, r.Close()) })
			p := goalTestAtom(t, m, "eliminated")
			q := goalTestAtom(t, m, "retained")
			aux := goalTestAtom(t, m, "helper")
			missing := goalTestAtom(t, m, "missing")
			neg := goalTestApp(t, m, ast.ASTOpNot, q)
			for i, atom := range []ast.ASTHandle{p, q, aux, neg, missing} {
				satOK(t, r.Insert(atom, i == 2))
			}
			// Source elim-var log for (p or q) and (not p or not q).
			entry, err := r.conversion.Mk(0, 0)
			satOK(t, err)
			satOK(t, r.conversion.InsertBinary(entry, sat.MakeLiteral(0, false), sat.MakeLiteral(1, false)))
			satOK(t, r.conversion.InsertBinary(entry, sat.MakeLiteral(0, true), sat.MakeLiteral(1, true)))
			// Source blocked-literal reconstruction may flip an existing helper.
			entry, err = r.conversion.Mk(1, 2)
			satOK(t, err)
			satOK(t, r.conversion.InsertBinary(entry, sat.MakeLiteral(2, true), sat.MakeLiteral(1, false)))
			md, err := model.NewModel(m)
			satOK(t, err)
			var out tactic.ModelRef
			satOK(t, out.Set(md.Core()))
			t.Cleanup(func() { satOK(t, out.Close()) })
			qd, err := m.GetDecl(q)
			satOK(t, err)
			ad, err := m.GetDecl(aux)
			satOK(t, err)
			yes, err := m.MkTrue()
			satOK(t, err)
			no, err := m.MkFalse()
			satOK(t, err)
			qAST := no
			if qValue {
				qAST = yes
			}
			satOK(t, md.Core().RegisterConstant(qd, qAST))
			satOK(t, md.Core().RegisterConstant(ad, yes))
			satOK(t, r.ApplyUnindexed(&out))
			if services.Trace.ShouldTrace("sat_mc") && (!strings.Contains(trace.String(), "goal2sat.cpp:598") || !strings.Contains(trace.String(), "goal2sat.cpp:644") || !strings.Contains(trace.String(), "after sat_mc")) {
				t.Fatal(trace.String())
			}
			got, err := out.Get()
			satOK(t, err)
			pd, err := m.GetDecl(p)
			satOK(t, err)
			pv, err := got.ConstantInterpretation(pd)
			satOK(t, err)
			if m.IsTrue(pv) == qValue || (!m.IsTrue(pv) && !m.IsFalse(pv)) {
				t.Fatal("eliminated variable was not reconstructed", pv)
			}
			qv, err := got.ConstantInterpretation(qd)
			satOK(t, err)
			if qv != qAST {
				t.Fatal("retained interpretation changed")
			}
			av, err := got.ConstantInterpretation(ad)
			satOK(t, err)
			if !av.IsNull() {
				t.Fatal("auxiliary variable escaped filter")
			}
			d, err := m.GetDecl(missing)
			satOK(t, err)
			mv, err := got.ConstantInterpretation(d)
			satOK(t, err)
			if !mv.IsNull() {
				t.Fatal("model completion must remain disabled")
			}
			if n, err := got.NumConstants(); err != nil || n != 2 {
				t.Fatal("interpreted atom or helper was registered", n, err)
			}
		})
	}
}

func TestSATModelConverterTranslationDropsEliminationLog(t *testing.T) {
	m, services := satModelFixture(t)
	r, err := newSATModelConverter(m, services)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, r.Close()) })
	p := goalTestAtom(t, m, "translated")
	a := goalTestAtom(t, m, "aux")
	satOK(t, r.Insert(p, false))
	satOK(t, r.Insert(a, true))
	entry, err := r.conversion.Mk(0, 0)
	satOK(t, err)
	satOK(t, r.conversion.InsertLiterals(entry, []sat.Literal{sat.MakeLiteral(0, false)}))
	var before bytes.Buffer
	satOK(t, r.Display(&before))
	if err := r.Display(nil); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
	copied := *r
	if err := copied.Close(); !errors.Is(err, tactic.ErrModelConverterOwner) {
		t.Fatal(err)
	}
	if !strings.Contains(before.String(), "(elim 0") || !strings.Contains(before.String(), "(0\n  translated)") {
		t.Fatal(before.String())
	}
	other, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: m.Symbols, Numerals: m.NumeralService(), Budget: m.Budget()})
	satOK(t, err)
	t.Cleanup(func() { satOK(t, other.Close()) })
	satOK(t, other.RegisterDeclarationPlugins())
	tr, err := ast.NewASTTranslation(m, other, true)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, tr.Close()) })
	copy, err := r.Translate(tr)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, copy.Close()) })
	c := copy.(*SATModelConverter)
	if c.conversion.Size() != 0 || len(c.atoms) != 2 {
		t.Fatal("source translation must omit SAT log and retain all atoms")
	}
	satOK(t, r.Close())
	var display bytes.Buffer
	satOK(t, c.Display(&display))
	if strings.Contains(display.String(), "(elim") || !strings.Contains(display.String(), "aux") {
		t.Fatal(display.String())
	}
	md, err := model.NewModel(other)
	satOK(t, err)
	var out tactic.ModelRef
	satOK(t, out.Set(md.Core()))
	t.Cleanup(func() { satOK(t, out.Close()) })
	yes, err := other.MkTrue()
	satOK(t, err)
	d, err := other.GetDecl(c.atoms[1])
	satOK(t, err)
	satOK(t, md.Core().RegisterConstant(d, yes))
	satOK(t, c.ApplyUnindexed(&out))
	got, err := out.Get()
	satOK(t, err)
	if n, err := got.NumConstants(); err != nil || n != 0 {
		t.Fatal("translated log/filter behavior", n, err)
	}
}

func TestSATModelConverterAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"constructor", "apply", "recover", "translate", "close"} {
		success := false
		for stop := int64(0); stop < 512; stop++ {
			t.Run(fmt.Sprint(phase, "/", stop), func(t *testing.T) {
				m, services := satModelFixture(t)
				s := satModelSolver(t, m, services)
				r, err := NewSATModelConverter(m, s, services)
				satOK(t, err)
				t.Cleanup(func() { satOK(t, r.Close()) })
				p := goalTestAtom(t, m, "frontier-p")
				a := goalTestAtom(t, m, "frontier-aux")
				satOK(t, r.Insert(p, false))
				satOK(t, r.Insert(a, true))
				var fresh *SATModelConverter
				var converted tactic.ModelConverter
				run := func() error { return r.Close() }
				if phase == "constructor" {
					run = func() error { fresh, err = NewSATModelConverter(m, s, services); return err }
				}
				if phase == "apply" {
					md, err := model.NewModel(m)
					satOK(t, err)
					var out tactic.ModelRef
					satOK(t, out.Set(md.Core()))
					t.Cleanup(func() { satOK(t, out.Close()) })
					yes, err := m.MkTrue()
					satOK(t, err)
					for _, atom := range []ast.ASTHandle{p, a} {
						d, err := m.GetDecl(atom)
						satOK(t, err)
						satOK(t, md.Core().RegisterConstant(d, yes))
					}
					run = func() error { return r.ApplyUnindexed(&out) }
				}
				if phase == "recover" {
					tr, err := NewGoalTranslator(m, s)
					satOK(t, err)
					t.Cleanup(func() { satOK(t, tr.Close()) })
					f := goalTestApp(t, m, ast.ASTOpIff, p, a)
					satOK(t, tr.Compile([]ast.ASTHandle{f}))
					g := goalTestNew(t, m, false, false)
					run = func() error { converted, err = RecoverSATGoal(tr, g, false, ^uint64(0), services); return err }
				}
				if phase == "translate" {
					destination, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: m.Symbols, Numerals: m.NumeralService(), Budget: m.Budget()})
					satOK(t, err)
					t.Cleanup(func() { satOK(t, destination.Close()) })
					tr, err := ast.NewASTTranslation(m, destination, true)
					satOK(t, err)
					t.Cleanup(func() { satOK(t, tr.Close()) })
					run = func() error { converted, err = r.Translate(tr); return err }
				}
				m.Budget().FailAfter(stop)
				err = run()
				m.Budget().FailAfter(-1)
				if fresh != nil {
					satOK(t, fresh.Close())
				}
				if converted != nil {
					satOK(t, converted.Close())
				}
				if err == nil {
					success = true
				} else if !errors.Is(err, arena.ErrInjectedFailure) {
					t.Fatal(err)
				}
			})
			if success {
				break
			}
		}
		if !success {
			t.Fatal("no successful reservation frontier", phase)
		}
	}
}
