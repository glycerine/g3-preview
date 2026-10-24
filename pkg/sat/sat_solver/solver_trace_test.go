package sat_solver_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/parsers/smt2"
	incsat "github.com/glycerine/g3/pkg/sat/sat_solver"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
	utext "github.com/glycerine/g3/pkg/util/text"
)

type closedIncrementalWriter struct{}

func (closedIncrementalWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestIncrementalSolverTraceSitesAndClosedStream(t *testing.T) {
	for _, closed := range []bool{false, true} {
		t.Run(map[bool]string{false: "trace", true: "closed"}[closed], func(t *testing.T) {
			f := newEncoderFixture(t, false)
			var output bytes.Buffer
			var writer io.Writer = &output
			if closed {
				writer = closedIncrementalWriter{}
			}
			tags := diagnostic.NewTraceTagsWithWriter(writer)
			tags.Enable("sat")
			f.services.Trace = tags
			s, err := incsat.NewIncrementalSolver(f.manager, &f.params, f.services)
			incMust(t, err)
			t.Cleanup(func() { incMust(t, s.Close()) })
			incMust(t, s.Assert(solverForm(t, f, `(=> p (= x #x3))`)))
			p := solverForm(t, f, `p`)
			x, err := f.context.MakeApp("x", nil, nil, ast.ASTHandle{})
			incMust(t, err)
			q := solverForm(t, f, `q`)
			status, _, err := s.GetConsequences([]ast.ASTHandle{p}, []ast.ASTHandle{x, q})
			incMust(t, err)
			if status != lbool.True {
				t.Fatal(status)
			}
			status, err = s.CheckSat([]ast.ASTHandle{p})
			incMust(t, err)
			if status != lbool.True {
				t.Fatal(status)
			}
			md, err := s.AcquireModel()
			incMust(t, err)
			incMust(t, md.DecRef())
			status, err = s.CheckSat([]ast.ASTHandle{p, solverForm(t, f, `(not (= x #x3))`)})
			incMust(t, err)
			if status != lbool.False {
				t.Fatal(status)
			}
			if !closed && tags.ShouldTrace("sat") {
				for _, text := range []string{"inc_sat_solver.cpp:231", "inc_sat_solver.cpp:399", "inc_sat_solver.cpp:416", "inc_sat_solver.cpp:266", "inc_sat_solver.cpp:291", "inc_sat_solver.cpp:484", "inc_sat_solver.cpp:489", "inc_sat_solver.cpp:606", "inc_sat_solver.cpp:644", "retrieve model present", "unhandled variable q", "asm2fml:", "core:"} {
					if !strings.Contains(output.String(), text) {
						t.Fatal("missing source trace", text, output.String())
					}
				}
			}
		})
	}
}

func TestIncrementalSolverVerboseUnknownAndClosedStream(t *testing.T) {
	f, s := newSolverFixture(t)
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: f.manager.Budget()})
	incMust(t, err)
	t.Cleanup(func() { incMust(t, out.Close()) })
	previous, err := diagnostic.BorrowProcessVerboseStream()
	incMust(t, err)
	level, err := diagnostic.ProcessVerbosityLevel()
	incMust(t, err)
	incMust(t, diagnostic.SetProcessVerboseStream(out))
	incMust(t, diagnostic.SetProcessVerbosityLevel(1))
	t.Cleanup(func() {
		incMust(t, diagnostic.SetProcessVerboseStream(previous))
		incMust(t, diagnostic.SetProcessVerbosityLevel(level))
	})
	incMust(t, smt2.ParseCommands(f.context, strings.NewReader(`(declare-fun fn (Bool) Bool)`), io.Discard))
	incMust(t, s.Assert(solverForm(t, f, `(fn p)`)))
	status, err := s.CheckSat(nil)
	incMust(t, err)
	if status != lbool.Undef {
		t.Fatal(status)
	}
	text, err := out.String()
	incMust(t, err)
	if text != "interpreted atoms sent to SAT solver \n(fn p)\n" {
		t.Fatalf("source verbose unsupported-atom message: %q", text)
	}
	incMust(t, out.Close())
	incMust(t, s.Assert(solverForm(t, f, `(fn q)`)))
	status, err = s.CheckSat(nil)
	incMust(t, err)
	if status != lbool.Undef {
		t.Fatal("closed verbose stream changed refusal", status)
	}
}

func TestIncrementalSolverDIMACSRepeatedAndConsequencesCPP(t *testing.T) {
	f := newEncoderFixture(t, false)
	incMust(t, f.services.Globals.SourceSetName("sat.dimacs.display", "true"))
	var out strings.Builder
	s, err := incsat.NewIncrementalSolverWithConfig(f.manager, &f.params, incsat.IncrementalSolverConfig{Services: f.services, Output: &out})
	incMust(t, err)
	t.Cleanup(func() { incMust(t, s.Close()) })
	incMust(t, s.Assert(solverForm(t, f, `(or p q)`)))
	p := solverForm(t, f, `p`)
	q := solverForm(t, f, `q`)
	status, _, err := s.GetConsequences(nil, []ast.ASTHandle{p, q})
	incMust(t, err)
	if status != lbool.Undef {
		t.Fatal("DIMACS mode solved instead of printing", status)
	}
	want := "p cnf 2 1\n1 2 0\n"
	if out.String() != want {
		t.Fatal("consequence DIMACS", out.String(), want)
	}
	if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
		cmd := exec.Command(reference, "-in")
		cmd.Stdin = strings.NewReader(`(set-option :sat.dimacs.display true)(set-logic QF_BV)(declare-const p Bool)(declare-const q Bool)(assert (or p q))(get-consequences () (p q))`)
		cpp, err := cmd.CombinedOutput()
		incMust(t, err)
		if string(cpp) != want+"unknown\n\n" {
			t.Fatal("C++ DIMACS mismatch", string(cpp), out.String())
		}
	}
	for i := 0; i < 2; i++ {
		out.Reset()
		status, err = s.CheckSat(nil)
		incMust(t, err)
		if status != lbool.Undef || !strings.HasPrefix(out.String(), want) || !strings.Contains(out.String(), "c 0 p\n") || !strings.Contains(out.String(), "c 1 q\n") {
			t.Fatal("repeated DIMACS mapping", status, out.String())
		}
	}
}

func TestIncrementalSolverDiagnosticAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"trace-check", "trace-model", "dimacs"} {
		t.Run(phase, func(t *testing.T) {
			success := false
			for stop := int64(0); stop < 8192 && !success; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					f := newEncoderFixture(t, false)
					tags := diagnostic.NewTraceTagsWithWriter(io.Discard)
					tags.Enable("sat")
					f.services.Trace = tags
					if phase == "dimacs" {
						incMust(t, f.services.Globals.SourceSetName("sat.dimacs.display", "true"))
					}
					s, err := incsat.NewIncrementalSolverWithConfig(f.manager, &f.params, incsat.IncrementalSolverConfig{Services: f.services, Output: io.Discard})
					incMust(t, err)
					t.Cleanup(func() { incMust(t, s.Close()) })
					incMust(t, s.Assert(solverForm(t, f, `(= (bvadd x y) #x3)`)))
					if phase == "trace-model" {
						status, e := s.CheckSat(nil)
						incMust(t, e)
						if status != lbool.True {
							t.Fatal(status)
						}
					}
					f.manager.Budget().FailAfter(stop)
					if phase == "trace-model" {
						md, e := s.AcquireModel()
						err = e
						f.manager.Budget().FailAfter(-1)
						if md != nil {
							incMust(t, md.DecRef())
						}
					} else {
						var status lbool.LBool
						status, err = s.CheckSat(nil)
						if err == nil && status != map[bool]lbool.LBool{false: lbool.True, true: lbool.Undef}[phase == "dimacs"] {
							t.Fatal("diagnostic altered status", status)
						}
					}
					f.manager.Budget().FailAfter(-1)
					if err == nil {
						success = true
					} else if !errors.Is(err, arena.ErrInjectedFailure) {
						t.Fatal(err)
					}
				})
				if t.Failed() {
					t.Fatal("diagnostic reservation frontier", phase, stop)
				}
			}
			if !success {
				t.Fatal("no successful diagnostic frontier")
			}
		})
	}
}

func TestIncrementalSolverReasonCStringBoundary(t *testing.T) {
	_, s := newSolverFixture(t)
	incMust(t, s.SetReasonUnknown("reason\x00ignored"))
	reason, err := s.ReasonUnknown()
	incMust(t, err)
	if reason != "reason" {
		t.Fatal("source char pointer boundary", reason)
	}
}
