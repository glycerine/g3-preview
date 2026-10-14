// Source: src/util/mpf.cpp:541,670,752; numeric formatting inside TRACE.
package numeral

import (
	"bytes"
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func closeMpfAfterTraceRefusal(t *testing.T, m *ArbitraryFloatManager) {
	t.Helper()
	if e := m.Close(); e != nil {
		z := m.integer
		if !errors.Is(e, arena.ErrInvalidSlot) || !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
			t.Fatalf("unexpected manager source stop: %v", e)
		}
		index := int(z.SourceFinalizeStage-2) / 2
		ref := z.Tmp[index]
		if z.SourceFinalizeStage%2 == 1 {
			ref = z.Arg[index]
		}
		if ref != 0 {
			t.Fatalf("unproved source scratch hole: %v", ref)
		}
		serviceMust(t, m.Close())
	}
}

func TestArbitraryFloatArithmeticDebugTraceFormatterRefusals(t *testing.T) {
	probe := diagnostic.NewTraceTagsWithWriter(&bytes.Buffer{})
	probe.Enable("mpf_dbg")
	if !probe.ShouldTrace("mpf_dbg") {
		return
	}
	for _, tc := range []struct {
		name, site, next string
		invoke           func(*ArbitraryFloatManager, *ArbitraryFloat, *ArbitraryFloat, *ArbitraryFloat) error
		cleanup          func(*ArbitraryFloatManager) error
	}{
		{"add_sub", "541", "542", func(m *ArbitraryFloatManager, x, y, out *ArbitraryFloat) error {
			return m.Add(ArbitraryFloatNearestEven, x, y, out)
		}, (*ArbitraryFloatManager).CleanupAddSub},
		{"div", "670", "671", func(m *ArbitraryFloatManager, x, y, out *ArbitraryFloat) error {
			return m.Div(ArbitraryFloatNearestEven, x, y, out)
		}, (*ArbitraryFloatManager).CleanupDiv},
		{"fma", "752", "753", func(m *ArbitraryFloatManager, x, y, out *ArbitraryFloat) error {
			return m.FMA(ArbitraryFloatNearestEven, x, y, y, out)
		}, (*ArbitraryFloatManager).CleanupFMA},
	} {
		t.Run(tc.name, func(t *testing.T) {
			found := false
			for ordinal := int64(0); ordinal < 128 && !found; ordinal++ {
				t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
					m, _, budget := mpfNumeralHelpersFixture(t)
					var output bytes.Buffer
					tags := diagnostic.NewTraceTagsWithWriter(&output)
					tags.Enable("mpf_dbg")
					serviceMust(t, m.BindTrace(tags))
					var x, y, out ArbitraryFloat
					serviceMust(t, m.MkOne(11, 53, false, &x))
					serviceMust(t, m.MkOne(11, 53, false, &y))
					beforeX, beforeY, beforeOut := x, y, out
					budget.FailAfter(ordinal)
					body := tc.invoke(m, &x, &y, &out)
					budget.FailAfter(-1)
					if body != nil && !numericPhysicalReservationRefusal(body) {
						t.Fatalf("unexpected arithmetic refusal: %v", body)
					}
					header := []byte(tc.name + " src/util/mpf.cpp:" + tc.site + " ---------")
					next := []byte(tc.name + " src/util/mpf.cpp:" + tc.next + " ---------")
					if body != nil && bytes.Contains(output.Bytes(), header) && !bytes.Contains(output.Bytes(), next) {
						found = true
						if x != beforeX || y != beforeY || tc.name != "add_sub" && out != beforeOut {
							t.Fatal("first debug formatter refusal changed operands or output")
						}
						if tc.name != "add_sub" && (m.div.active || m.fma.active) {
							t.Fatal("entry formatter refusal reached arithmetic scope")
						}
					}
					serviceMust(t, tc.cleanup(m))
					serviceMust(t, m.Del(&out))
					serviceMust(t, m.Del(&x))
					serviceMust(t, m.Del(&y))
					closeMpfAfterTraceRefusal(t, m)
				})
			}
			if !found {
				t.Fatalf("formatter refusal did not reach source line %s", tc.site)
			}
		})
	}
}
