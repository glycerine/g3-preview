package numeral

import (
	"errors"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalFloatGetterSourceAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertion only")
	}
	d, log := bindMPNDebugActions(t, "TC")
	m, _ := rationalManager(t, true)
	a := Rational{Num: NewInt(6), Den: NewInt(-2)}
	if _, e := m.Float64(&a); !isMPNVerificationThrow(e) || a != (Rational{Num: NewInt(6), Den: NewInt(-2)}) {
		t.Fatal("Throw after both source component conversions", e, a)
	}
	if v, e := m.Float64(&a); e != nil || v != -3 {
		t.Fatal("Continue follows source quotient", v, e)
	}
	contents, e := os.ReadFile(log.Name())
	qopsMust(t, e)
	if n := strings.Count(string(contents), "File: src/util/mpq.cpp\nLine: 302\nd > 0.0\n"); n != 2 {
		t.Fatal("source assertion location and action count", n, string(contents))
	}
	qopsMust(t, d.EnableAssertions(false))
	a.Den = NewInt(0)
	if v, e := m.Float64(&a); e != nil || !math.IsInf(v, 1) {
		t.Fatal("disabled assertion retains IEEE division by zero", v, e)
	}
	qopsMust(t, d.Close())
	a.Den = NewInt(2)
	if _, e := m.Float64(&a); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("valid conversion reads closed debug owner", e)
	}
}
