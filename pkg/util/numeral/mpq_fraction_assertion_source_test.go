package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalFractionSourceAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	for _, tc := range []struct {
		name string
		line string
		set  func(*RationalManager, *Rational, bool) error
	}{
		{"signed32", "679", func(m *RationalManager, a *Rational, valid bool) error {
			if valid {
				return m.SetFractionSignedSource(a, 1, 2)
			}
			return m.SetFractionSignedSource(a, 1, 0)
		}},
		{"int64-uint64", "690", func(m *RationalManager, a *Rational, valid bool) error {
			if valid {
				return m.SetFraction(a, 1, 2)
			}
			return m.SetFraction(a, 1, 0)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, log := bindMPNDebugActions(t, "TC")
			m, _ := rationalManager(t, true)
			a := Rational{Num: NewInt(7), Den: NewInt(9)}
			if e := tc.set(m, &a, false); !isMPNVerificationThrow(e) || a != (Rational{Num: NewInt(7), Den: NewInt(9)}) {
				t.Fatal("Throw before numerator set", e, a)
			}
			if e := tc.set(m, &a, false); e != nil || a.Num != NewInt(1) || a.Den != NewInt(0) {
				t.Fatal("Continue follows source setter", e, a)
			}
			contents, e := os.ReadFile(log.Name())
			qopsMust(t, e)
			if n := strings.Count(string(contents), "File: src/util/mpq.h\nLine: "+tc.line+"\nd != 0\n"); n != 2 {
				t.Fatal("source assertion location and action count", n, string(contents))
			}
			qopsMust(t, m.Reset(&a))
			qopsMust(t, d.EnableAssertions(false))
			if e := tc.set(m, &a, false); e != nil || a.Num != NewInt(1) || a.Den != NewInt(0) {
				t.Fatal("disabled assertion follows source setter", e, a)
			}
			qopsMust(t, m.Reset(&a))
			qopsMust(t, d.Close())
			if e := tc.set(m, &a, true); !errors.Is(e, arena.ErrClosed) || a != NewRational() {
				t.Fatal("valid setter reads closed assertion owner", e, a)
			}
		})
	}
}
