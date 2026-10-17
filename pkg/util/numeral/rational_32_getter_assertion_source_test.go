package numeral

import (
	"errors"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRational32GetterSourceAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	for _, tc := range []struct {
		name, line, expression string
		input                  uint64
		get                    func(*RationalManager, *Rational) (uint32, error)
	}{
		{"unsigned", "108", "is_unsigned()", uint64(math.MaxUint32) + 1, func(m *RationalManager, a *Rational) (uint32, error) {
			return m.Unsigned(a)
		}},
		{"int32", "121", "is_int32()", uint64(math.MaxInt32) + 1, func(m *RationalManager, a *Rational) (uint32, error) {
			v, e := m.Int32(a)
			return uint32(v), e
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, log := bindMPNDebugActions(t, "TC")
			m, _ := rationalManager(t, true)
			a := NewRational()
			qopsMust(t, m.Z.SetUint64(&a.Num, tc.input))
			if _, e := tc.get(m, &a); !isMPNVerificationThrow(e) {
				t.Fatal("Throw before the delegated 64-bit getter", e)
			}
			if v, e := tc.get(m, &a); e != nil || v != uint32(tc.input) {
				t.Fatal("Continue delegates and narrows", v, e)
			}
			contents, e := os.ReadFile(log.Name())
			qopsMust(t, e)
			if n := strings.Count(string(contents), "File: src/util/rational.h\nLine: "+tc.line+"\n"+tc.expression+"\n"); n != 2 {
				t.Fatal("source assertion site and action count", n, string(contents))
			}
			qopsMust(t, d.EnableAssertions(false))
			if v, e := tc.get(m, &a); e != nil || v != uint32(tc.input) {
				t.Fatal("disabled assertion delegates and narrows", v, e)
			}
			qopsMust(t, m.Z.SetUint64(&a.Num, 7))
			qopsMust(t, d.Close())
			if _, e := tc.get(m, &a); !errors.Is(e, arena.ErrClosed) {
				t.Fatal("valid getter reads closed debug owner", e)
			}
			qopsMust(t, m.Del(&a))
		})
	}
}
