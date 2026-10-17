package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalIntegerForwardSourceAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	for _, tc := range []struct {
		name, line string
		want       int32
		call       func(*RationalManager, *Rational, *Rational, *Rational) error
	}{
		{"machine-div", "513", 1, (*RationalManager).MachineIDiv},
		{"idiv", "524", 1, (*RationalManager).IDiv},
		{"rem", "537", 2, (*RationalManager).Rem},
		{"mod", "550", 2, (*RationalManager).Mod},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, log := bindMPNDebugActions(t, "TC")
			m, _ := rationalManager(t, true)
			a, b, out := NewRational(), NewRational(), NewRational()
			qopsMust(t, m.SetFraction(&a, 5, 2))
			qopsMust(t, m.SetFraction(&b, 3, 2))
			qopsMust(t, m.SetFraction(&out, 7, 9))
			if e := tc.call(m, &a, &b, &out); !isMPNVerificationThrow(e) || out.Num != NewInt(7) || out.Den != NewInt(9) {
				t.Fatal("Throw before numerator operation", e, out)
			}
			if e := tc.call(m, &a, &b, &out); e != nil || out.Num != NewInt(tc.want) || out.Den != NewInt(1) {
				t.Fatal("Continue follows source numerator operation", e, out)
			}
			contents, e := os.ReadFile(log.Name())
			qopsMust(t, e)
			if n := strings.Count(string(contents), "File: src/util/mpq.h\nLine: "+tc.line+"\nis_int(a) && is_int(b)\n"); n != 2 {
				t.Fatal("source assertion site and action count", n, string(contents))
			}
			qopsMust(t, m.SetInt64(&a, 5))
			qopsMust(t, m.SetInt64(&b, 3))
			qopsMust(t, m.SetInt64(&out, 71))
			qopsMust(t, d.Close())
			if e := tc.call(m, &a, &b, &out); !errors.Is(e, arena.ErrClosed) || out.Num != NewInt(71) || out.Den != NewInt(1) {
				t.Fatal("valid operation reads closed source assertion switch", e, out)
			}
		})
	}
}
