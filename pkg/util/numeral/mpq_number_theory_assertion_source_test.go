package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalNumberTheorySourceAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	for _, tc := range []struct {
		name, line, expression string
		call                   func(*RationalManager, *Rational, *Rational, *Rational, *Rational, *Rational) error
	}{
		{"gcd", "602", "is_int(a) && is_int(b)", func(m *RationalManager, x, y, a, b, g *Rational) error {
			return m.GCD(x, y, g)
		}},
		{"extended-gcd", "610", "is_int(r1) && is_int(r2)", (*RationalManager).ExtendedGCD},
		{"lcm", "620", "is_int(a) && is_int(b)", func(m *RationalManager, x, y, a, b, g *Rational) error {
			return m.LCM(x, y, g)
		}},
		{"divides", "628", "is_int(a) && is_int(b)", func(m *RationalManager, x, y, a, b, g *Rational) error {
			_, e := m.Divides(x, y)
			return e
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, log := bindMPNDebugActions(t, "TC")
			m, _ := rationalManager(t, true)
			x := Rational{Num: NewInt(6), Den: NewInt(2)}
			y := Rational{Num: NewInt(10), Den: NewInt(2)}
			a, b, g := NewRational(), NewRational(), NewRational()
			qopsMust(t, m.SetInt64(&g, 71))
			if e := tc.call(m, &x, &y, &a, &b, &g); !isMPNVerificationThrow(e) || g.Num != NewInt(71) || g.Den != NewInt(1) {
				t.Fatal("Throw before number-theory action", e, g)
			}
			if e := tc.call(m, &x, &y, &a, &b, &g); e != nil {
				t.Fatal("Continue follows source number-theory action", e)
			}
			contents, e := os.ReadFile(log.Name())
			qopsMust(t, e)
			if n := strings.Count(string(contents), "File: src/util/mpq.h\nLine: "+tc.line+"\n"+tc.expression+"\n"); n != 2 {
				t.Fatal("source assertion location and action count", n, string(contents))
			}
			qopsMust(t, m.SetInt64(&x, 6))
			qopsMust(t, m.SetInt64(&y, 10))
			qopsMust(t, m.SetInt64(&g, 71))
			qopsMust(t, d.Close())
			if e := tc.call(m, &x, &y, &a, &b, &g); !errors.Is(e, arena.ErrClosed) || g.Num != NewInt(71) {
				t.Fatal("valid call reads closed assertion owner", e, g)
			}
		})
	}
}
