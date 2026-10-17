package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalBitwiseSourceAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	for _, tc := range []struct {
		name, line, expression string
		want                   int32
		call                   func(*RationalManager, *Rational, *Rational, *Rational) error
	}{
		{"or", "635", "is_int(a) && is_int(b)", 7, (*RationalManager).BitwiseOr},
		{"and", "643", "is_int(a) && is_int(b)", 1, (*RationalManager).BitwiseAnd},
		{"xor", "651", "is_int(a) && is_int(b)", 6, (*RationalManager).BitwiseXor},
		{"not", "659", "is_int(a)", 122, func(m *RationalManager, a, b, out *Rational) error {
			return m.BitwiseNot(7, a, out)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, log := bindMPNDebugActions(t, "TC")
			m, _ := rationalManager(t, true)
			a := Rational{Num: NewInt(5), Den: NewInt(2)}
			b, out := NewRational(), NewRational()
			qopsMust(t, m.SetInt64(&b, 3))
			qopsMust(t, m.SetInt64(&out, 71))
			if e := tc.call(m, &a, &b, &out); !isMPNVerificationThrow(e) || out.Num != NewInt(71) {
				t.Fatal("Throw before numerator bitwise operation", e, out)
			}
			if e := tc.call(m, &a, &b, &out); e != nil || out.Num != NewInt(tc.want) || out.Den != NewInt(1) {
				t.Fatal("Continue follows source bitwise action", e, out)
			}
			contents, e := os.ReadFile(log.Name())
			qopsMust(t, e)
			if n := strings.Count(string(contents), "File: src/util/mpq.h\nLine: "+tc.line+"\n"+tc.expression+"\n"); n != 2 {
				t.Fatal("source assertion site and action count", n, string(contents))
			}
			qopsMust(t, m.SetInt64(&a, 5))
			qopsMust(t, m.SetInt64(&out, 71))
			qopsMust(t, d.Close())
			if e := tc.call(m, &a, &b, &out); !errors.Is(e, arena.ErrClosed) || out.Num != NewInt(71) {
				t.Fatal("valid bitwise call reads closed assertion owner", e, out)
			}
		})
	}
}
