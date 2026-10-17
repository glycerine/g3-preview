package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatIEEESourceAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	for _, tc := range []struct {
		name, line, expression string
		ebits, sbits           uint32
		exponent               int64
		significand            uint64
		float                  bool
		boundsAfterContinue    bool
	}{
		{"double width", "1713", "x.ebits <= 11 && x.sbits <= 53", 8, 54, 0, 1, false, true},
		{"float width", "1740", "x.ebits <= 8 && x.sbits <= 24", 8, 25, 0, 1, true, true},
		{"float significand", "1745", "q < 4294967296ull", 8, 24, 0, 1 << 32, true, false},
		{"float exponent", "1755", "q < 4294967296ll", 8, 24, 1 << 32, 0, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, log := bindMPNDebugActions(t, "TC")
			b := arena.NewBudget(1 << 26)
			m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
			serviceMust(t, e)
			var x ArbitraryFloat
			var sig Int
			serviceMust(t, m.integer.SetUint64(&sig, tc.significand))
			serviceMust(t, m.SetComponents(&x, tc.ebits, tc.sbits, false, tc.exponent, &sig))
			serviceMust(t, m.integer.Del(&sig))
			get := func() (float64, error) {
				if tc.float {
					v, e := m.ToFloat(&x)
					return float64(v), e
				}
				return m.ToDouble(&x)
			}
			if _, e := get(); !isMPNVerificationThrow(e) {
				t.Fatal("Throw stops at source assertion", e)
			}
			checkContinue := func(label string) {
				t.Helper()
				v, e := get()
				if tc.boundsAfterContinue {
					if !errors.Is(e, arena.ErrBounds) {
						t.Fatal(label, "source undefined shift guard", e)
					}
				} else if e != nil || v != 1 {
					t.Fatal(label, "source narrowed IEEE bits", v, e)
				}
			}
			checkContinue("Continue")
			contents, e := os.ReadFile(log.Name())
			serviceMust(t, e)
			if n := strings.Count(string(contents), "File: src/util/mpf.cpp\nLine: "+tc.line+"\n"+tc.expression+"\n"); n != 2 {
				t.Fatal("source assertion site and action count", n, string(contents))
			}
			serviceMust(t, d.EnableAssertions(false))
			checkContinue("disabled")
			serviceMust(t, m.Del(&x))
			serviceMust(t, m.SourceFinalize())
			serviceMust(t, m.Close())
			if b.Used() != 0 {
				t.Fatal("unreleased numerical owner", b.Used())
			}
		})
	}
}
