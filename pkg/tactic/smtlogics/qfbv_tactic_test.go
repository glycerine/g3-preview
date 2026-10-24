package smtlogics_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	"github.com/glycerine/g3/pkg/parsers/smt2"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/memory"
)

func TestQFBVTacticCPP(t *testing.T) {
	for _, tc := range []struct{ name, options, decls, forms, command, query string }{
		{"empty", "", "", "", "qfbv", ""},
		{"equality-extract", "", `(declare-const x (_ BitVec 4))(declare-const y (_ BitVec 4))`, `(assert (not (= ((_ extract 1 0) x) ((_ extract 1 0) y))))`, "qfbv", `(get-value ((not (= ((_ extract 1 0) x) ((_ extract 1 0) y)))))`},
		{"signed-model", "", `(declare-const x (_ BitVec 8))`, `(assert (bvsle #xf8 x))(assert (bvsle x #xff))(assert (not (= x #xff)))`, "qfbv", `(get-value ((bvsle #xf8 x) (bvsle x #xff) (not (= x #xff))))`},
		{"multiply-unsat", "", `(declare-const x (_ BitVec 4))`, `(assert (= (bvmul x x) #x2))`, "qfbv", ""},
		{"multiply-model", "", `(declare-const x (_ BitVec 4))(declare-const y (_ BitVec 4))`, `(assert (= (bvmul x y) #x6))(assert (bvult x y))`, "qfbv", `(get-value ((= (bvmul x y) #x6) (bvult x y)))`},
		{"uf-reconstruction", "", `(declare-const x (_ BitVec 4))(declare-const y (_ BitVec 4))(declare-fun f ((_ BitVec 4)) (_ BitVec 4))`, `(assert (distinct x y))(assert (bvult (f x) (f y)))`, "qfbv", `(get-value ((distinct x y) (bvult (f x) (f y))))`},
		{"uf-fallback", "", `(declare-const x (_ BitVec 4))(declare-const y (_ BitVec 4))(declare-fun f ((_ BitVec 4)) (_ BitVec 4))`, `(assert (= x y))(assert (not (= (f x) (f y))))`, "(using-params qfbv :div0_ackermann_limit 0)", ""},
		{"div0-fallback", "", `(declare-const x (_ BitVec 4))(declare-const y (_ BitVec 4))`, `(assert (bvult (bvudiv x #x0) (bvudiv y #x0)))`, "(using-params qfbv :hi_div0 false :div0_ackermann_limit 0)", `(get-value ((bvult (bvudiv x #x0) (bvudiv y #x0))))`},
		{"proof-route", `(set-option :produce-proofs true)`, `(declare-const x (_ BitVec 4))`, `(assert (= (bvmul x x) #x2))`, "qfbv", ""},
		{"core-route", `(set-option :produce-unsat-cores true)`, `(declare-const x (_ BitVec 4))`, `(assert (! (= (bvmul x x) #x2) :named impossible))`, "qfbv", `(get-unsat-core)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			script := `(set-logic ALL)(set-option :produce-models true)` + tc.options + tc.decls + tc.forms + `(check-sat-using ` + tc.command + `)` + tc.query
			c, err := cmd_context.NewStandaloneContext()
			qfaufliaMust(t, err)
			t.Cleanup(func() { qfaufliaMust(t, c.Close()) })
			var output bytes.Buffer
			qfaufliaMust(t, smt2.ParseCommands(c, strings.NewReader(script), &output))
			if strings.Contains(output.String(), "unknown") || strings.Contains(output.String(), "error") {
				t.Fatal(output.String())
			}
			if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
				command := exec.Command(reference, "-in")
				command.Stdin = strings.NewReader(script)
				cpp, err := command.CombinedOutput()
				qfaufliaMust(t, err)
				if strings.Join(strings.Fields(output.String()), " ") != strings.Join(strings.Fields(string(cpp)), " ") {
					t.Fatalf("Go: %s\nC++: %s", output.String(), cpp)
				}
			}
		})
	}
}

func TestQFBVTacticConstructorAllocationFrontiers(t *testing.T) {
	success := false
	for stop := int64(0); stop < 8192 && !success; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			c := qfliaContext(t, `(set-logic ALL)`)
			m, err := c.Manager()
			qfaufliaMust(t, err)
			m.Budget().FailAfter(stop)
			r, err := c.NewBuiltinTactic("qfbv")
			m.Budget().FailAfter(-1)
			if r != nil {
				qfaufliaMust(t, r.IncRef())
				qfaufliaMust(t, r.DecRef())
			}
			var failure memory.ConstructionFailure
			if errors.As(err, &failure) {
				qfaufliaMust(t, failure.Cleanup())
			}
			var memout memory.Failure
			if err == nil {
				success = true
			} else if !errors.Is(err, arena.ErrInjectedFailure) && !(errors.As(err, &memout) && memout.Kind == memory.FailureMemout) {
				t.Fatal(err)
			}
		})
	}
	if !success {
		t.Fatal("no successful factory reservation frontier")
	}
}

func TestQFBVTacticModelSurvivesStrategyClose(t *testing.T) {
	c := qfliaContext(t, `(set-logic QF_BV)(set-option :produce-models true)
(declare-const x (_ BitVec 4))(declare-const y (_ BitVec 4))
(assert (= (bvmul x y) #x6))(assert (bvult x y))`)
	m, err := c.Manager()
	qfaufliaMust(t, err)
	var terms []ast.ASTHandle
	for _, name := range []string{"x", "y"} {
		h, err := c.MakeApp(name, nil, nil, ast.ASTHandle{})
		qfaufliaMust(t, err)
		qfaufliaPin(t, m, h)
		terms = append(terms, h)
	}
	r, err := c.NewBuiltinTactic("qfbv")
	qfaufliaMust(t, err)
	qfaufliaMust(t, r.IncRef())
	closed := false
	t.Cleanup(func() {
		if !closed {
			qfaufliaMust(t, r.DecRef())
		}
	})
	g, err := tactic.NewGoal(m, false, true, false)
	qfaufliaMust(t, err)
	t.Cleanup(func() { qfaufliaMust(t, g.Close()) })
	qfaufliaMust(t, c.AssertGoal(g))
	var result tactic.TacticResult
	t.Cleanup(func() { qfaufliaMust(t, result.Close()) })
	qfaufliaMust(t, r.Apply(g, &result))
	if !result.DecidedSat() || result.ModelConverter == nil {
		t.Fatal("missing reconstructable SAT result")
	}
	qfaufliaMust(t, r.Cleanup())
	qfaufliaMust(t, r.DecRef())
	closed = true
	var reference tactic.ModelRef
	t.Cleanup(func() { qfaufliaMust(t, reference.Close()) })
	qfaufliaMust(t, tactic.ModelConverterToModel(m, result.ModelConverter, &reference))
	md, err := reference.Get()
	qfaufliaMust(t, err)
	values := make([]int, 2)
	for i, term := range terms {
		d, err := m.GetDecl(term)
		qfaufliaMust(t, err)
		value, err := md.ConstantInterpretation(d)
		qfaufliaMust(t, err)
		values[i] = -1
		for candidate := uint32(0); candidate < 16; candidate++ {
			numeral, err := m.BitVectorNumeralUnsigned(candidate, 4)
			qfaufliaMust(t, err)
			if numeral == value {
				values[i] = int(candidate)
				break
			}
		}
	}
	if values[0] < 0 || values[1] < 0 || values[0] >= values[1] || (values[0]*values[1])%16 != 6 {
		t.Fatal("reconstructed values violate the original formulas", values)
	}
	n, err := md.NumConstants()
	qfaufliaMust(t, err)
	if n != 2 {
		t.Fatal("helper declarations escaped model reconstruction", n)
	}
}
