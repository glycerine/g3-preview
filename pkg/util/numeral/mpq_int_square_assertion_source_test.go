package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalIntegerSquareSourceAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertion only")
	}
	d, log := bindMPNDebugActions(t, "TC")
	m, _ := rationalManager(t, true)
	a := Rational{Num: NewInt(9), Den: NewInt(2)}
	out := Rational{Num: NewInt(7), Den: NewInt(13)}
	if exact, e := m.IsIntPerfectSquare(&a, &out); exact || !isMPNVerificationThrow(e) || out != (Rational{Num: NewInt(7), Den: NewInt(13)}) {
		t.Fatal("Throw before denominator reset", exact, e, out)
	}
	if exact, e := m.IsIntPerfectSquare(&a, &out); !exact || e != nil || out != (Rational{Num: NewInt(3), Den: NewInt(1)}) {
		t.Fatal("Continue checks source numerator", exact, e, out)
	}
	contents, e := os.ReadFile(log.Name())
	qopsMust(t, e)
	if n := strings.Count(string(contents), "File: src/util/mpq.h\nLine: 839\nis_int(a)\n"); n != 2 {
		t.Fatal("source assertion location and action count", n, string(contents))
	}
	qopsMust(t, d.EnableAssertions(false))
	out = Rational{Num: NewInt(7), Den: NewInt(13)}
	if exact, e := m.IsIntPerfectSquare(&a, &out); !exact || e != nil || out != (Rational{Num: NewInt(3), Den: NewInt(1)}) {
		t.Fatal("disabled assertion follows source body", exact, e, out)
	}
	qopsMust(t, d.Close())
	a = NewRational()
	out = Rational{Num: NewInt(7), Den: NewInt(13)}
	if exact, e := m.IsIntPerfectSquare(&a, &out); exact || !errors.Is(e, arena.ErrClosed) || out != (Rational{Num: NewInt(7), Den: NewInt(13)}) {
		t.Fatal("valid call reads closed debug owner before reset", exact, e, out)
	}
}
