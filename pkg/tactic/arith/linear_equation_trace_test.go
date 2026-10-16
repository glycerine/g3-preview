package arith

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLinearEquationSourceConstructionTrace(t *testing.T) {
	m, l := equationManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("linear_equation_mk")
	tags.Enable("linear_equation_bug")
	equationMust(t, l.BindTrace(tags))

	ras := []numeral.Rational{numeral.NewRational(), numeral.NewRational()}
	equationMust(t, m.SetString(&ras[0], "1/2"))
	equationMust(t, m.SetString(&ras[1], "-2/3"))
	eq, err := l.MkRational(ras, []uint32{0, 1}, true)
	equationMust(t, err)
	equationMust(t, eq.Close())
	for i := range ras {
		equationMust(t, m.Del(&ras[i]))
	}

	as := []numeral.Int{numeral.NewInt(6), numeral.NewInt(9)}
	eq, err = l.MkInt(as, []uint32{1, 3}, true)
	equationMust(t, err)
	equationMust(t, eq.Close())
	for i := range as {
		equationMust(t, m.Integers().Del(&as[i]))
	}
	if !tags.ShouldTrace("linear_equation_mk") {
		if output.Len() != 0 {
			t.Fatalf("erased TRACE wrote %q", output.String())
		}
		return
	}
	record := func(tag, function string, line int, body string) string {
		return "-------- [" + tag + "] " + function + " src/tactic/arith/linear_equation.cpp:" + strconv.Itoa(line) + " ---------\n" + body + "------------------------------------------------\n"
	}
	var want strings.Builder
	want.WriteString(record("linear_equation_mk", "mk", 71, "lcm: 6\n"))
	want.WriteString(record("linear_equation_mk", "mk", 76, "before as[0]: 1/2\n"))
	want.WriteString(record("linear_equation_mk", "mk", 78, "after as[0]: 3\n"))
	want.WriteString(record("linear_equation_mk", "mk", 76, "before as[1]: -2/3\n"))
	want.WriteString(record("linear_equation_mk", "mk", 78, "after as[1]: -4\n"))
	want.WriteString(record("linear_equation_bug", "mk_core", 99, "3*x0 -4*x1 \n"))
	want.WriteString(record("linear_equation_bug", "mk_core", 121, "g: 1\n3*x0 -4*x1 \n"))
	want.WriteString(record("linear_equation_bug", "mk_core", 99, "6*x1 9*x3 \n"))
	want.WriteString(record("linear_equation_bug", "mk_core", 121, "g: 3\n2*x1 3*x3 \n"))
	if got := output.String(); got != want.String() {
		t.Fatalf("source construction TRACE:\n got %q\nwant %q", got, want.String())
	}
}
