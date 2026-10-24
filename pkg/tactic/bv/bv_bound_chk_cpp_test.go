package bv_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/tactic"
)

func TestBVBoundCheckExactGoalsCPP(t *testing.T) {
	for _, limit := range []int{0, 2, 100} {
		for _, tc := range []struct{ name, form string }{
			{"empty", ""},
			{"inconsistent-goal", `(assert false)`},
			{"unsigned-contradiction", `(assert (or p (and (bvule x #x02) (bvule #x03 x))))`},
			{"signed-contradiction", `(assert (or p (and (bvsle x #xfe) (bvsle #x01 x))))`},
			{"unsigned-singleton", `(assert (or p (and (bvule x #x02) (bvule #x02 x))))`},
			{"signed-singleton", `(assert (or p (and (bvsle x #xfe) (bvsle #xfe x))))`},
			{"redundant", `(assert (or p (and (bvule x #x20) (bvule x #x10) (bvule #x02 x))))`},
			{"tautology", `(assert (or (bvule x #x02) (bvule #x03 x)))`},
			{"shared-conjunction", `(assert (or p (and (bvule x #x10) (and (bvule #x02 x) (bvule x #x02)))))(assert (or (not p) (and (bvule #x02 x) (bvule x #x02))))`},
			{"negated-bounds", `(assert (or p (and (not (bvule x #x02)) (not (bvule #x03 x)))))`},
			{"independent-variables", `(assert (or p (and (bvule x #x02) (bvule #x03 y))))`},
			{"mixed-signed-unsigned", `(assert (or p (and (bvsle x #xff) (bvule x #x7f))))`},
			{"nested-associative", `(assert (or p (and (bvule x #x10) (and (bvule #x02 x) (bvule x #x02)))))`},
			{"non-bv", `(assert (or p (and (> i 2) (< i 1))))`},
			{"ite-container", `(assert (= (ite (and (bvule x #x02) (bvule #x03 x)) x y) y))`},
		} {
			t.Run(fmt.Sprintf("%s/limit-%d", tc.name, limit), func(t *testing.T) {
				logic, integer := "QF_BV", ""
				if tc.name == "non-bv" {
					logic, integer = "ALL", `(declare-const i Int)`
				}
				script := fmt.Sprintf(`(set-option :rewriter.bv_ineq_consistency_test_max %d)(set-logic %s)(declare-const x (_ BitVec 8))(declare-const y (_ BitVec 8))(declare-const p Bool)`, limit, logic) + integer + tc.form
				bvBoundExact(t, script)
			})
		}
	}
	for _, width := range []uint32{1, 4, 65, 257} {
		t.Run(fmt.Sprint("wide/", width), func(t *testing.T) {
			script := fmt.Sprintf(`(set-option :rewriter.bv_ineq_consistency_test_max 100)(set-logic QF_BV)(declare-const x (_ BitVec %d))(declare-const p Bool)(assert (or p (and (bvule x %s) (bvule %s x))))`, width, bvSizeLiteral(0, width), bvSizeLiteral(1, width))
			bvBoundExact(t, script)
		})
	}
}

func bvBoundExact(t *testing.T, script string) {
	t.Helper()
	c := bv1Context(t, script)
	m, err := c.Manager()
	bv1Must(t, err)
	g, err := tactic.NewGoal(m, false, true, false)
	bv1Must(t, err)
	t.Cleanup(func() { bv1Must(t, g.Close()) })
	bv1Must(t, c.AssertGoal(g))
	r, err := c.NewBuiltinTactic("bv_bound_chk")
	bv1Must(t, err)
	bv1Must(t, r.IncRef())
	t.Cleanup(func() { bv1Must(t, r.DecRef()) })
	var result tactic.TacticResult
	t.Cleanup(func() { bv1Must(t, result.Close()) })
	bv1Must(t, r.Apply(g, &result))
	if len(result.Goals) != 1 || result.Goals[0] != g || result.ModelConverter != nil || result.ProofConverter != nil {
		t.Fatal("bound checker result metadata")
	}
	var actual bytes.Buffer
	actual.WriteString("(goals\n")
	bv1Must(t, g.DisplayWithAttributes(&actual))
	actual.WriteString(")\n")
	if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
		cmd := exec.Command(reference, "-in")
		cmd.Stdin = strings.NewReader(script + `(apply bv_bound_chk)`)
		cpp, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(err, string(cpp))
		}
		if bv1Canonical(actual.String()) != bv1Canonical(string(cpp)) {
			t.Fatalf("Go: %s\nC++: %s", actual.String(), cpp)
		}
	}
}
