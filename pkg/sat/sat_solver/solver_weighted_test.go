package sat_solver_test

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/parsers/smt2"
	incsat "github.com/glycerine/g3/pkg/sat/sat_solver"
	"github.com/glycerine/g3/pkg/util/lbool"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestIncrementalSolverWeightedDisplayCPP(t *testing.T) {
	f, s := newSolverFixture(t)
	incMust(t, s.Assert(solverForm(t, f, `(or p q)`)))
	soft := []ast.ASTHandle{solverForm(t, f, `p`), solverForm(t, f, `q`)}
	var out strings.Builder
	incMust(t, s.DisplayWeighted(&out, soft, []uint32{2, 7}))
	want := "p wcnf 4 5 8\nc soft 2\n8 1 2 0\n8 1 -3 0\n8 2 -4 0\n2 2 0\n7 3 0\n"
	if out.String() != want {
		t.Fatalf("actual SAT weighted output\n%s\nwant\n%s", out.String(), want)
	}
	if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
		cmd := exec.Command(reference, "-in")
		cmd.Stdin = strings.NewReader(`(set-option :opt.dump_benchmarks true)(set-logic QF_BV)(declare-const p Bool)(declare-const q Bool)(assert (or p q))(assert-soft p :weight 2)(assert-soft q :weight 7)(check-sat)`)
		cpp, err := cmd.CombinedOutput()
		incMust(t, err)
		if string(cpp) != want+"sat\n" {
			t.Fatal("C++ incremental weighted output mismatch", string(cpp), out.String())
		}
	}
	n, err := s.NumAssertions()
	incMust(t, err)
	if n != 3 {
		t.Fatal("soft guards not retained", n)
	}
	status, err := s.CheckSat(nil)
	incMust(t, err)
	if status != lbool.True {
		t.Fatal("weighted output activated soft guards permanently", status)
	}
}

func TestIncrementalSolverWeightedRationalValidationAndOverflow(t *testing.T) {
	f, s := newSolverFixture(t)
	soft := []ast.ASTHandle{solverForm(t, f, `p`)}
	service := f.manager.NumeralService()
	weight, err := service.NewValue()
	incMust(t, err)
	t.Cleanup(func() { incMust(t, service.CleanupValue(&weight)) })
	for _, text := range []string{"1/2", "-1", "4294967296"} {
		incMust(t, service.SetString(&weight, text))
		var out strings.Builder
		err = incsat.DisplayWeighted(&out, s, soft, []unumeral.OwnedRational{weight})
		var refusal z3exception.DefaultException
		if !errors.As(err, &refusal) || err.Error() != "Cannot display weights that are not integers" || out.Len() != 0 {
			t.Fatal("source weight refusal", text, err, out.String())
		}
		n, e := s.NumAssertions()
		incMust(t, e)
		if n != 0 {
			t.Fatal("invalid rational mutated backend", n)
		}
	}
	incMust(t, service.SetUnsigned(&weight, math.MaxUint32))
	var out strings.Builder
	incMust(t, incsat.DisplayWeighted(&out, s, soft, []unumeral.OwnedRational{weight}))
	if !strings.HasPrefix(out.String(), "p wcnf 2 2 0\n") || !strings.Contains(out.String(), "4294967295 0 0\n") {
		t.Fatal("source unsigned hard-weight wrap", out.String())
	}
}

func TestIncrementalSolverWeightedAllocationFrontiers(t *testing.T) {
	for _, repeat := range []bool{false, true} {
		t.Run(fmt.Sprint(repeat), func(t *testing.T) {
			success := false
			for stop := int64(0); stop < 8192 && !success; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					f, s := newSolverFixture(t)
					incMust(t, s.Assert(solverForm(t, f, `(or p q)`)))
					soft := []ast.ASTHandle{solverForm(t, f, `p`), solverForm(t, f, `q`)}
					if repeat {
						incMust(t, s.DisplayWeighted(io.Discard, soft, []uint32{2, 7}))
					}
					f.manager.Budget().FailAfter(stop)
					err := s.DisplayWeighted(io.Discard, soft, []uint32{2, 7})
					f.manager.Budget().FailAfter(-1)
					if err == nil {
						success = true
					} else if !errors.Is(err, arena.ErrInjectedFailure) {
						t.Fatal(err)
					}
				})
				if t.Failed() {
					t.Fatal("weighted reservation frontier", repeat, stop)
				}
			}
			if !success {
				t.Fatal("no successful weighted reservation frontier")
			}
		})
	}
}

func TestIncrementalSolverWeightedBoundsAndWriterRecovery(t *testing.T) {
	f, s := newSolverFixture(t)
	p := solverForm(t, f, `p`)
	if err := s.DisplayWeighted(io.Discard, []ast.ASTHandle{p}, nil); !errors.Is(err, arena.ErrBounds) {
		t.Fatal("missing source weights must not default to one", err)
	}
	err := s.DisplayWeighted(closedIncrementalWriter{}, []ast.ASTHandle{p}, []uint32{2})
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal("writer error", err)
	}
	status, err := s.CheckSat(nil)
	incMust(t, err)
	if status != lbool.True {
		t.Fatal("writer failure left activated guards", status)
	}
	var out strings.Builder
	incMust(t, s.DisplayWeighted(&out, []ast.ASTHandle{p}, []uint32{7}))
	if !strings.Contains(out.String(), "c soft 1\n") || !strings.Contains(out.String(), "7 ") {
		t.Fatal("writer failure retained old weights", out.String())
	}
}

func TestIncrementalSolverWeightedBitVectorsCPP(t *testing.T) {
	for _, width := range []int{1, 4, 65, 257} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			f, s := newSolverFixture(t)
			declaration := fmt.Sprintf(`(declare-const wide (_ BitVec %d))`, width)
			incMust(t, smt2.ParseCommands(f.context, strings.NewReader(declaration), io.Discard))
			second := 2
			if width == 1 {
				second = 0
			}
			terms := []string{fmt.Sprintf(`(= wide (_ bv1 %d))`, width), fmt.Sprintf(`(= wide (_ bv%d %d))`, second, width)}
			var soft []ast.ASTHandle
			var expressions []ast.ASTHandle
			for _, term := range terms {
				expressions = append(expressions, solverForm(t, f, term))
			}
			// opt::context::mk_atomic purifies theory-containing soft terms
			// before the retained inc_sat_display call. Reproduce that actual
			// caller input, including its hard proxy equalities.
			boolean, err := f.manager.MkBoolSort()
			incMust(t, err)
			for _, expression := range expressions {
				proxy, err := f.manager.MkFreshConst("weighted", boolean)
				incMust(t, err)
				definition, err := f.manager.MkEq(proxy, expression)
				incMust(t, err)
				incMust(t, s.Assert(definition))
				soft = append(soft, proxy)
			}
			var out strings.Builder
			incMust(t, s.DisplayWeighted(&out, soft, []uint32{2, 7}))
			if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
				script := `(set-option :opt.dump_benchmarks true)(set-logic QF_BV)` + declaration + "(assert-soft " + terms[0] + " :weight 2)(assert-soft " + terms[1] + " :weight 7)(check-sat)"
				cmd := exec.Command(reference, "-in")
				cmd.Stdin = strings.NewReader(script)
				cpp, err := cmd.CombinedOutput()
				incMust(t, err)
				if string(cpp) != out.String()+"sat\n" {
					t.Fatal("C++ BV weighted output mismatch", string(cpp), out.String())
				}
			}
		})
	}
}
