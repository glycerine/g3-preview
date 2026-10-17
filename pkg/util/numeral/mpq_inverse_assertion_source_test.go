package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalInverseSourceAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertion only")
	}
	d, log := bindMPNDebugActions(t, "TC")
	m, _ := rationalManager(t, true)
	a := NewRational()
	if e := m.InvInPlace(&a); !isMPNVerificationThrow(e) || a != NewRational() {
		t.Fatal("Throw before sign or swap", e, a)
	}
	if e := m.InvInPlace(&a); e != nil || a.Num != NewInt(1) || a.Den != NewInt(0) {
		t.Fatal("Continue follows source zero swap", e, a)
	}
	contents, e := os.ReadFile(log.Name())
	qopsMust(t, e)
	if n := strings.Count(string(contents), "File: src/util/mpq.h\nLine: 444\n!is_zero(a)\n"); n != 2 {
		t.Fatal("source assertion site and action count", n, string(contents))
	}
	qopsMust(t, m.Reset(&a))
	qopsMust(t, d.EnableAssertions(false))
	if e := m.InvInPlace(&a); e != nil || a.Num != NewInt(1) || a.Den != NewInt(0) {
		t.Fatal("disabled source assertion follows swap", e, a)
	}
	qopsMust(t, m.Reset(&a))
	qopsMust(t, d.Close())
	if e := m.InvInPlace(&a); !errors.Is(e, arena.ErrClosed) || a != NewRational() {
		t.Fatal("valid entry reads closed assertion owner", e, a)
	}
}
