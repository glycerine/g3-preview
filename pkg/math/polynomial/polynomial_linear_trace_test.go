package polynomial

import (
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strings"
	"testing"
)

func checkLinearIterationTrace(t *testing.T, text string, compiled bool, rhs string) {
	t.Helper()
	if !compiled {
		if text != "" {
			t.Fatal("erased TRACE emitted", text)
		}
		return
	}
	expected := []string{"iteration 0\n2 0 4\n0 3 " + rhs + "\n", "iteration 1\n1 0 2\n0 3 " + rhs + "\n", "iteration 1\n1 0 2\n0 1 3\n", "iteration 0\n1 0 2\n0 1 3\n"}
	rest := text
	for _, body := range expected {
		i := strings.Index(rest, body)
		if i < 0 {
			t.Fatal("missing ordered source matrix trace", body, text)
		}
		rest = rest[i+len(body):]
	}
	if strings.Count(text, "iteration ") != 4 {
		t.Fatal("wrong iteration count", text)
	}
}
func TestLinearRationalIterationTraceSourceStates(t *testing.T) {
	q, _ := rationalManager(t, false)
	s, e := NewRationalLinearSolver(q)
	serviceMust(t, e)
	defer func() { serviceMust(t, s.Close()); serviceMust(t, q.BindTrace(nil)) }()
	serviceMust(t, s.Resize(2))
	for i := uint32(0); i < 2; i++ {
		row := [2]unumeral.Rational{unumeral.NewRational(), unumeral.NewRational()}
		rhs := unumeral.NewRational()
		n, b := int64(2), int64(4)
		if i == 1 {
			n, b = 3, 9
		}
		serviceMust(t, q.SetInt64(&row[i], n))
		serviceMust(t, q.SetInt64(&rhs, b))
		serviceMust(t, s.Add(i, row[:], &rhs))
	}
	var output strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("linear_eq_solver")
	serviceMust(t, q.BindTrace(tags))
	result := [2]unumeral.Rational{unumeral.NewRational(), unumeral.NewRational()}
	ok, e := s.Solve(result[:])
	serviceMust(t, e)
	if !ok {
		t.Fatal("diagonal system singular")
	}
	checkLinearIterationTrace(t, output.String(), tags.ShouldTrace("linear_eq_solver"), "9")
	for i := range result {
		text, e := q.String(&result[i])
		serviceMust(t, e)
		if text != []string{"2", "3"}[i] {
			t.Fatal(text)
		}
		serviceMust(t, q.Del(&result[i]))
	}
}
func TestLinearModularIterationTraceSourceStates(t *testing.T) {
	q, _ := rationalManager(t, false)
	m, e := unumeral.NewIntegerModularManager(q.Integers())
	serviceMust(t, e)
	defer func() { serviceMust(t, m.BindTrace(nil)); serviceMust(t, m.Close()) }()
	serviceMust(t, m.SetZPUint64(11))
	s, e := NewModularLinearSolver(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, s.Close()) }()
	serviceMust(t, s.Resize(2))
	for i := uint32(0); i < 2; i++ {
		row := [2]unumeral.Int{}
		b := unumeral.NewInt(4)
		row[i] = unumeral.NewInt(2)
		if i == 1 {
			row[i] = unumeral.NewInt(3)
			b = unumeral.NewInt(9)
		}
		serviceMust(t, s.Add(i, row[:], &b))
	}
	var output strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("linear_eq_solver")
	serviceMust(t, m.BindTrace(tags))
	var result [2]unumeral.Int
	ok, e := s.Solve(result[:])
	serviceMust(t, e)
	if !ok || result[0].Val != 2 || result[1].Val != 3 {
		t.Fatal("diagonal field solution", ok, result)
	}
	checkLinearIterationTrace(t, output.String(), tags.ShouldTrace("linear_eq_solver"), "-2")
	for i := range result {
		serviceMust(t, q.Integers().Del(&result[i]))
	}
}
