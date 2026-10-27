package nlsat

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/sat"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func intervalSMTValue(t *testing.T, m *IntervalSetManager, v *polynomial.Algebraic) string {
	t.Helper()
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: m.budget})
	assignmentMust(t, err)
	defer func() { assignmentMust(t, out.Close()) }()
	assignmentMust(t, m.am.DisplayRootSMT2Into(out, v))
	text, err := out.String()
	assignmentMust(t, err)
	return text
}
func intervalSMTSet(t *testing.T, m *IntervalSetManager, s *IntervalSet) string {
	t.Helper()
	if s == nil {
		return "false"
	}
	parts := make([]string, 0, len(s.intervals))
	for j := range s.intervals {
		i := &s.intervals[j]
		var bounds []string
		if !i.lowerInf {
			op := ">="
			if i.lowerOpen {
				op = ">"
			}
			bounds = append(bounds, "("+op+" x "+intervalSMTValue(t, m, &i.lower)+")")
		}
		if !i.upperInf {
			op := "<="
			if i.upperOpen {
				op = "<"
			}
			bounds = append(bounds, "("+op+" x "+intervalSMTValue(t, m, &i.upper)+")")
		}
		if len(bounds) == 0 {
			parts = append(parts, "true")
		} else {
			parts = append(parts, "(and "+strings.Join(bounds, " ")+")")
		}
	}
	return "(or " + strings.Join(parts, " ") + ")"
}
func intervalSMTInteger(n int32) string {
	if n < 0 {
		return fmt.Sprintf("(- %d.0)", -n)
	}
	return fmt.Sprintf("%d.0", n)
}
func intervalSMTSpec(s intervalSpec) string {
	var bounds []string
	if !s.lowerInf {
		op := ">="
		if s.lowerOpen {
			op = ">"
		}
		bounds = append(bounds, "("+op+" x "+intervalSMTInteger(s.lower)+")")
	}
	if !s.upperInf {
		op := "<="
		if s.upperOpen {
			op = "<"
		}
		bounds = append(bounds, "("+op+" x "+intervalSMTInteger(s.upper)+")")
	}
	if len(bounds) == 0 {
		return "true"
	}
	return "(and " + strings.Join(bounds, " ") + ")"
}
func TestIntervalUnionsAndWitnessPrebuiltCPPOracle(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	m := intervalFixture(t)
	pool := intervalPool(t, m)
	var script strings.Builder
	script.WriteString("(declare-const x Real)\n")
	count := 0
	for trial := 0; trial < 64; trial++ {
		a := intervalSpec{lower: int32(trial%5) - 2, upper: 2, lowerOpen: trial&1 != 0, upperOpen: trial&2 != 0, why: sat.MakeLiteral(1, false)}
		b := intervalSpec{lower: -2, upper: int32(trial/2%5) - 2, lowerOpen: trial&4 != 0, upperOpen: trial&8 != 0, why: sat.MakeLiteral(2, true)}
		if a.lower == a.upper {
			a.lowerOpen = false
			a.upperOpen = false
		}
		if b.lower == b.upper {
			b.lowerOpen = false
			b.upperOpen = false
		}
		if trial%7 == 0 {
			a.upperInf = true
			a.upperOpen = true
		}
		if trial%11 == 0 {
			b.lowerInf = true
			b.lowerOpen = true
		}
		s1, s2 := intervalMk(t, m, pool, a), intervalMk(t, m, pool, b)
		u, err := m.MkUnion(s1, s2)
		assignmentMust(t, err)
		assignmentMust(t, m.IncRef(u))
		fmt.Fprintf(&script, "(push)(assert (not (= %s (or %s %s))))(check-sat)(pop)\n", intervalSMTSet(t, m, u), intervalSMTSpec(a), intervalSMTSpec(b))
		count++
		assignmentMust(t, m.DecRef(u))
		assignmentMust(t, m.DecRef(s2))
		assignmentMust(t, m.DecRef(s1))
	}
	root := assignmentInteger(t, m.am, 2)
	assignmentMust(t, m.am.Root(root, 2, root))
	left, err := m.Mk(true, true, nil, true, false, root, sat.MakeLiteral(3, false))
	assignmentMust(t, err)
	assignmentMust(t, m.IncRef(left))
	defer func() { assignmentMust(t, m.DecRef(left)) }()
	right, err := m.Mk(true, false, root, true, true, nil, sat.MakeLiteral(4, false))
	assignmentMust(t, err)
	assignmentMust(t, m.IncRef(right))
	defer func() { assignmentMust(t, m.DecRef(right)) }()
	missing, err := m.MkUnion(left, right)
	assignmentMust(t, err)
	assignmentMust(t, m.IncRef(missing))
	defer func() { assignmentMust(t, m.DecRef(missing)) }()
	var witness polynomial.Algebraic
	assignmentMust(t, m.PeekInComplement(missing, false, &witness, false))
	defer func() { assignmentMust(t, m.am.Del(&witness)) }()
	fmt.Fprintf(&script, "(push)(assert (= x %s))(assert %s)(check-sat)(pop)\n", intervalSMTValue(t, m, &witness), intervalSMTSet(t, m, missing))
	count++
	cmd := exec.Command(reference, "-in")
	cmd.Stdin = strings.NewReader(script.String())
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err, string(raw))
	}
	statuses := strings.Fields(string(raw))
	if len(statuses) != count {
		t.Fatal("C++ errors or surplus output", string(raw))
	}
	for i, status := range statuses {
		if status != "unsat" {
			t.Fatal("interval union or witness differs", i, string(raw))
		}
	}
}
