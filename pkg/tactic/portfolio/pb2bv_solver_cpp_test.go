package portfolio_test

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPB2BVSolverQFFDTruthTableReference(t *testing.T) {
	for _, tc := range []struct {
		name, mode string
		weights    [3]int64
		bound      int64
	}{
		{"at-most", "card-le", [3]int64{1, 1, 1}, 1},
		{"at-least", "card-ge", [3]int64{1, 1, 1}, 2},
		{"weighted-le", "le", [3]int64{2, 3, 4}, 5},
		{"weighted-ge", "ge", [3]int64{2, 3, 4}, 5},
		{"weighted-eq", "eq", [3]int64{1, 2, 3}, 4},
		{"zero-bound", "le", [3]int64{2, 3, 4}, 0},
		{"wide-coefficients", "le", [3]int64{4294967296, 4294967296, 1}, 4294967296},
		{"arithmetic-ite", "ite", [3]int64{2, 3, 4}, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, s := pbSolverFixture(t)
			m := f.manager
			vars := pbAtoms(t, m, 3)
			values := make([]unumeral.OwnedRational, 4)
			for i := range values {
				v, e := m.NumeralService().NewValue()
				enumMust(t, e)
				values[i] = v
				value := tc.bound
				if i < 3 {
					value = tc.weights[i]
				}
				enumMust(t, m.NumeralService().SetInt64(&values[i], value))
				t.Cleanup(func() { enumMust(t, m.NumeralService().Del(&values[i])) })
			}
			var h ast.ASTHandle
			var err error
			var body string
			switch tc.mode {
			case "card-le":
				h, err = m.MkPseudoBooleanAtMost(vars, uint32(tc.bound))
				body = fmt.Sprintf("((_ at-most %d) pb-b0 pb-b1 pb-b2)", tc.bound)
			case "card-ge":
				h, err = m.MkPseudoBooleanAtLeast(vars, uint32(tc.bound))
				body = fmt.Sprintf("((_ at-least %d) pb-b0 pb-b1 pb-b2)", tc.bound)
			case "le":
				h, err = m.MkPseudoBooleanLE(values[:3], vars, &values[3])
			case "ge":
				h, err = m.MkPseudoBooleanGE(values[:3], vars, &values[3])
			case "eq":
				h, err = m.MkPseudoBooleanEQ(values[:3], vars, &values[3])
			case "ite":
				zeroValue, e := m.NumeralService().NewValue()
				enumMust(t, e)
				defer func() { enumMust(t, m.NumeralService().Del(&zeroValue)) }()
				zero, e := m.ArithmeticNumeral(&zeroValue, true)
				enumMust(t, e)
				enumPin(t, m, zero)
				var terms []ast.ASTHandle
				for i := range vars {
					num, e := m.ArithmeticNumeral(&values[i], true)
					enumMust(t, e)
					term, e := m.MkIte(vars[i], num, zero)
					enumMust(t, e)
					terms = append(terms, term)
				}
				sum, e := m.MkArithmeticAdd(terms[0], terms[1])
				enumMust(t, e)
				sum, e = m.MkArithmeticAdd(sum, terms[2])
				enumMust(t, e)
				bound, e := m.ArithmeticNumeral(&values[3], true)
				enumMust(t, e)
				h, err = m.MkArithmeticLE(sum, bound)
				body = fmt.Sprintf("(<= (+ (ite pb-b0 %d 0) (ite pb-b1 %d 0) (ite pb-b2 %d 0)) %d)", tc.weights[0], tc.weights[1], tc.weights[2], tc.bound)
			}
			enumMust(t, err)
			enumPin(t, m, h)
			if body == "" {
				op := map[string]string{"le": "pble", "ge": "pbge", "eq": "pbeq"}[tc.mode]
				body = fmt.Sprintf("((_ %s %d %d %d %d) pb-b0 pb-b1 pb-b2)", op, tc.bound, tc.weights[0], tc.weights[1], tc.weights[2])
			}
			if tc.name == "wide-coefficients" {
				// The old C++ SMT-LIB parser restricts indexed PB fields to
				// unsigned32. Its ordinary integer syntax can express the same
				// weighted predicate without truncating the actual coefficients.
				body = fmt.Sprintf("(<= (+ (ite pb-b0 %d 0) (ite pb-b1 %d 0) (ite pb-b2 %d 0)) %d)", tc.weights[0], tc.weights[1], tc.weights[2], tc.bound)
			}
			enumMust(t, s.Assert(h))
			var script, actual strings.Builder
			script.WriteString(`(set-logic QF_FD)(declare-const pb-b0 Bool)(declare-const pb-b1 Bool)(declare-const pb-b2 Bool)(assert ` + body + `)`)
			for mask := uint(0); mask < 8; mask++ {
				enumMust(t, s.Push())
				script.WriteString("(push 1)")
				sum := int64(0)
				for i, v := range vars {
					name := fmt.Sprint("pb-b", i)
					if mask&(1<<i) == 0 {
						v, err = m.MkNot(v)
						enumMust(t, err)
						script.WriteString("(assert (not " + name + "))")
					} else {
						sum += tc.weights[i]
						script.WriteString("(assert " + name + ")")
					}
					enumMust(t, s.Assert(v))
				}
				status, e := s.CheckSat(nil)
				enumMust(t, e)
				want := sum <= tc.bound
				if tc.mode == "ge" || tc.mode == "card-ge" {
					want = sum >= tc.bound
				}
				if tc.mode == "eq" {
					want = sum == tc.bound
				}
				if status == lbool.Undef || (status == lbool.True) != want {
					t.Fatal("independent arithmetic truth table", mask, sum, tc.bound, status)
				}
				fmt.Fprintln(&actual, enumStatus(status))
				script.WriteString("(check-sat)(pop 1)")
				enumMust(t, s.Pop(1))
			}
			if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
				cmd := exec.Command(reference, "-in")
				cmd.Stdin = strings.NewReader(script.String())
				cpp, e := cmd.CombinedOutput()
				if e != nil {
					t.Fatal(e, string(cpp))
				}
				if enumCanonical(actual.String()) != enumCanonical(string(cpp)) {
					t.Fatalf("Go PB/enum wrappers + SMT: %s\nC++ QF_FD: %s", actual.String(), cpp)
				}
			}
		})
	}
}
