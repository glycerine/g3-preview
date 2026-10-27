package nlsat

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
)

func TestSignTablePrebuiltCPPCellSignOracle(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	ism := intervalFixture(t)
	table, err := newSignTable(ism.am)
	assignmentMust(t, err)
	defer func() { assignmentMust(t, table.close()) }()
	var expressions []string
	for family, values := range [][]int{{-3, -1, 4}, {-2, 0, 3}, {-8, -6, -4, -2, 0, 2, 4, 6, 8}} {
		roots := signRoots(t, ism.am, values)
		expression := "7.0"
		leading := 7
		if family%2 == 0 {
			expression = "(- 7.0)"
			leading = -7
		}
		powers := make([]int, len(values))
		for i, r := range values {
			powers[i] = 1
			if i%3 == 0 {
				powers[i] = 2
			}
			for j := 0; j < powers[i]; j++ {
				expression = "(* " + expression + " (- x " + intervalSMTInteger(int32(r)) + "))"
			}
		}
		signs := make([]int, len(values)+1)
		for i := range signs {
			signs[i] = leading
			for j := i; j < len(powers); j++ {
				if powers[j]%2 == 1 {
					signs[i] = -signs[i]
				}
			}
		}
		assignmentMust(t, table.add(roots, signs))
		assignmentMust(t, roots.Close())
		expressions = append(expressions, expression)
	}
	// Add the two exact irrational roots of x^2-2. The eventual evaluator
	// obtains these vectors from algebraic isolation; this table owns copies.
	root := assignmentInteger(t, ism.am, 2)
	assignmentMust(t, ism.am.Root(root, 2, root))
	negative := polynomial.Algebraic{}
	assignmentMust(t, ism.am.Copy(&negative, root))
	assignmentMust(t, ism.am.Neg(&negative))
	roots, err := ism.am.NewVector()
	assignmentMust(t, err)
	assignmentMust(t, roots.PushCopy(&negative))
	assignmentMust(t, roots.PushCopy(root))
	assignmentMust(t, table.add(roots, []int{73, -17, 113}))
	assignmentMust(t, roots.Close())
	assignmentMust(t, ism.am.Del(&negative))
	expressions = append(expressions, "(- (* x x) 2.0)")
	var script strings.Builder
	script.WriteString("(declare-const x Real)\n")
	count := 0
	for c := uint32(0); c < table.numCells(); c++ {
		cell := "true"
		if c%2 == 1 {
			id, err := table.rootID(c)
			assignmentMust(t, err)
			r, err := table.root(id)
			assignmentMust(t, err)
			cell = "(= x " + intervalSMTValue(t, ism, r) + ")"
		} else {
			var bounds []string
			if c > 0 {
				id, err := table.rootID(c - 1)
				assignmentMust(t, err)
				r, err := table.root(id)
				assignmentMust(t, err)
				bounds = append(bounds, "(> x "+intervalSMTValue(t, ism, r)+")")
			}
			if c+1 < table.numCells() {
				id, err := table.rootID(c + 1)
				assignmentMust(t, err)
				r, err := table.root(id)
				assignmentMust(t, err)
				bounds = append(bounds, "(< x "+intervalSMTValue(t, ism, r)+")")
			}
			cell = "(and " + strings.Join(bounds, " ") + ")"
		}
		for i, expression := range expressions {
			s, err := table.signAt(uint32(i), c)
			assignmentMust(t, err)
			op := "="
			if s < 0 {
				op = "<"
			} else if s > 0 {
				op = ">"
			}
			fmt.Fprintf(&script, "(push)(assert %s)(assert (not (%s %s 0.0)))(check-sat)(pop)\n", cell, op, expression)
			count++
		}
	}
	cmd := exec.Command(reference, "-in", "-T:20")
	cmd.Stdin = strings.NewReader(script.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err, string(out))
	}
	statuses := strings.Fields(string(out))
	if len(statuses) != count {
		t.Fatal("C++ errors or surplus output", len(statuses), count, string(out))
	}
	for i, status := range statuses {
		if status != "unsat" {
			t.Fatal("incorrect sign on cell", i, status)
		}
	}
	t.Logf("prebuilt C++ rejects all %d cell-sign disagreements", count)
}
