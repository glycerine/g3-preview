package arith

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

type boundShortWriter struct{}

func (boundShortWriter) Write(p []byte) (int, error) { return 0, nil }

func TestBoundEquationDisplaySourceOrderDeadConstraintAndWriterError(t *testing.T) {
	m, _ := equationManager(t)
	s, err := NewBoundEquationStore(m)
	equationMust(t, err)
	defer func() { equationMust(t, s.Close()) }()
	for _, x := range []uint32{0, 2, 3} {
		equationMust(t, s.MkVar(x, false))
	}
	value := numeral.NewRational()
	defer func() { equationMust(t, m.Del(&value)) }()
	equationMust(t, m.SetString(&value, "1/3"))
	_, err = s.AssertLower(0, &value, true)
	equationMust(t, err)
	equationMust(t, m.SetString(&value, "2/3"))
	_, err = s.AssertUpper(0, &value, false)
	equationMust(t, err)
	coeff := []numeral.Int{numeral.NewInt(1), numeral.NewInt(1)}
	ids := []uint32{2, 3}
	_, err = s.MkIntEquation(coeff, ids)
	equationMust(t, err)
	for i := range coeff {
		equationMust(t, m.Integers().Del(&coeff[i]))
	}
	equationMust(t, s.DelVar(2))
	var out bytes.Buffer
	equationMust(t, s.Display(&out))
	want := "1/3 | 0.333333 < x0 <= 2/3 | 0.666667\n" +
		"-oo < x3 < oo\n" +
		"1*x2 + 1*x3 = 0\n"
	if got := out.String(); got != want {
		t.Fatalf("display = %q, want %q", got, want)
	}
	if err := s.Display(boundShortWriter{}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short writer error = %v", err)
	}
}
