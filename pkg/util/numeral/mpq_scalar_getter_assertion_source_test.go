package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalScalarGetterSourceAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	for _, tc := range []struct {
		name, line, expression string
		get                    func(*RationalManager, *Rational) (int64, error)
	}{
		{"uint64", "776", "is_uint64(a)", func(m *RationalManager, a *Rational) (int64, error) {
			v, e := m.Uint64(a)
			return int64(v), e
		}},
		{"int64", "778", "is_int64(a)", (*RationalManager).Int64},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, log := bindMPNDebugActions(t, "TC")
			m, _ := rationalManager(t, true)
			a := Rational{Num: NewInt(5), Den: NewInt(2)}
			if _, e := tc.get(m, &a); !isMPNVerificationThrow(e) || a.Num != NewInt(5) || a.Den != NewInt(2) {
				t.Fatal("Throw before integer getter", e, a)
			}
			if v, e := tc.get(m, &a); e != nil || v != 5 {
				t.Fatal("Continue projects source numerator", v, e)
			}
			contents, e := os.ReadFile(log.Name())
			qopsMust(t, e)
			if n := strings.Count(string(contents), "File: src/util/mpq.h\nLine: "+tc.line+"\n"+tc.expression+"\n"); n != 2 {
				t.Fatal("source assertion site and action count", n, string(contents))
			}
			qopsMust(t, d.EnableAssertions(false))
			if v, e := tc.get(m, &a); e != nil || v != 5 {
				t.Fatal("disabled assertion projects source numerator", v, e)
			}
			a = Rational{Num: NewInt(5), Den: NewInt(1)}
			qopsMust(t, d.Close())
			if _, e := tc.get(m, &a); !errors.Is(e, arena.ErrClosed) || a.Num != NewInt(5) {
				t.Fatal("valid getter reads closed assertion owner", e, a)
			}
		})
	}
}
