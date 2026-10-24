package bv_test

import (
	"bytes"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/tactic"
)

func bvSizeLiteral(value int64, width uint32) string {
	v := new(big.Int).Mod(big.NewInt(value), new(big.Int).Lsh(big.NewInt(1), uint(width)))
	return fmt.Sprintf("(_ bv%s %d)", v.String(), width)
}
func TestBVSizeReductionExactGoalsCPP(t *testing.T) {
	for _, width := range []uint32{1, 4, 8, 65, 257} {
		for _, bounds := range [][2]int64{{-1, -1}, {0, 0}, {1, 1}, {-1, 0}, {-2, -1}, {-2, 0}, {-2, 2}, {-4, -1}, {-4, 3}, {-8, -1}, {0, 1}, {0, 3}, {1, 3}, {1, 2}, {3, 2}} {
			limit := new(big.Int).Lsh(big.NewInt(1), uint(width-1))
			valid := true
			for _, v := range bounds {
				n := big.NewInt(v)
				if n.Cmp(new(big.Int).Neg(limit)) < 0 || n.Cmp(limit) >= 0 {
					valid = false
				}
			}
			if !valid {
				continue
			}
			t.Run(fmt.Sprint(width, "/", bounds), func(t *testing.T) {
				script := fmt.Sprintf(`(set-logic QF_BV)(declare-const x (_ BitVec %d))(assert (bvsle %s x))(assert (bvsle x %s))`, width, bvSizeLiteral(bounds[0], width), bvSizeLiteral(bounds[1], width))
				bvSizeExact(t, script)
			})
		}
	}
	for _, tc := range []struct{ name, forms string }{
		{"empty", ""},
		{"lower-only", `(assert (bvsle #xfe x))`},
		{"upper-only", `(assert (bvsle x #x02))`},
		{"negated", `(assert (not (bvsle x #xfd)))(assert (not (bvsle #x03 x)))`},
		{"improved", `(assert (bvsle #xf0 x))(assert (bvsle x #x20))(assert (bvsle #xfe x))(assert (bvsle x #x02))`},
		{"unsigned-disabled", `(assert (bvule #x00 x))(assert (bvule x #x03))`},
		{"strict-unsupported", `(assert (bvslt #xfd x))(assert (bvslt x #x03))`},
		{"greater-equal-unsupported", `(assert (bvsge x #xfe))(assert (bvsle x #x02))`},
		{"equality-unsupported", `(assert (= x #x02))`},
		{"overflow-upper-negation", `(assert (not (bvsle x #x7f)))(assert (bvsle #x00 x))`},
		{"overflow-lower-negation", `(assert (not (bvsle #x80 x)))(assert (bvsle x #xff))`},
		{"full-range", `(assert (bvsle #x80 x))(assert (bvsle x #x7f))`},
		{"positive-range", `(assert (bvsle #x00 x))(assert (bvsle x #x7f))`},
		{"negative-range", `(assert (bvsle #x80 x))(assert (bvsle x #xff))`},
		{"Boolean-container", `(assert (or (bvsle #xfe x) (bvsle x #x02)))`},
		{"compound-lhs", `(assert (bvsle #xfe (bvadd x x)))(assert (bvsle (bvadd x x) #x02))`},
		{"singleton-and-use", `(assert (bvsle #xff x))(assert (bvsle x #xff))(assert (= (bvadd x x) #xfe))`},
		{"multiple", `(assert (bvsle #xfe x))(assert (bvsle x #x02))(assert (bvsle #x00 y))(assert (bvsle y #x03))(assert (= (bvadd x y) #x02))`},
		{"already-inconsistent", `(assert false)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bvSizeExact(t, `(set-logic QF_BV)(declare-const x (_ BitVec 8))(declare-const y (_ BitVec 8))`+tc.forms)
		})
	}
}

func bvSizeExact(t *testing.T, script string) {
	t.Helper()
	c := bv1Context(t, script)
	m, err := c.Manager()
	bv1Must(t, err)
	g, err := tactic.NewGoal(m, false, true, false)
	bv1Must(t, err)
	t.Cleanup(func() { bv1Must(t, g.Close()) })
	bv1Must(t, c.AssertGoal(g))
	r, err := c.NewBuiltinTactic("reduce-bv-size")
	bv1Must(t, err)
	bv1Must(t, r.IncRef())
	t.Cleanup(func() { bv1Must(t, r.DecRef()) })
	var result tactic.TacticResult
	t.Cleanup(func() { bv1Must(t, result.Close()) })
	bv1Must(t, r.Apply(g, &result))
	if len(result.Goals) != 1 || result.Goals[0] != g {
		t.Fatal("missing borrowed goal")
	}
	var actual bytes.Buffer
	actual.WriteString("(goals\n")
	bv1Must(t, g.DisplayWithAttributes(&actual))
	actual.WriteString(")\n")
	if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
		cmd := exec.Command(reference, "-in")
		cmd.Stdin = strings.NewReader(script + `(apply reduce-bv-size)`)
		cpp, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(err, string(cpp))
		}
		if bv1Canonical(actual.String()) != bv1Canonical(string(cpp)) {
			t.Fatalf("Go: %s\nC++: %s", actual.String(), cpp)
		}
	}
}
