// Source: src/util/mpf.cpp:541-542,670-742,752-754; numeric formatting inside TRACE.
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
	// The integer-only records call Int.String without a budgeted reservation;
	// this campaign can reach formatter refusals only in the float records.
	for _, tc := range []struct {
		name    string
		sites   []string
		invoke  func(*ArbitraryFloatManager, *ArbitraryFloat, *ArbitraryFloat, *ArbitraryFloat) error
		cleanup func(*ArbitraryFloatManager) error
	}{
		{"add_sub", []string{"541", "542"}, func(m *ArbitraryFloatManager, x, y, out *ArbitraryFloat) error {
			return m.Add(ArbitraryFloatNearestEven, x, y, out)
		}, (*ArbitraryFloatManager).CleanupAddSub},
		{"div", []string{"670", "671", "722", "723", "734", "742"}, func(m *ArbitraryFloatManager, x, y, out *ArbitraryFloat) error {
			return m.Div(ArbitraryFloatNearestEven, x, y, out)
		}, (*ArbitraryFloatManager).CleanupDiv},
		{"fma", []string{"752", "753", "754"}, func(m *ArbitraryFloatManager, x, y, out *ArbitraryFloat) error {
			return m.FMA(ArbitraryFloatNearestEven, x, y, y, out)
		}, (*ArbitraryFloatManager).CleanupFMA},
	} {
		t.Run(tc.name, func(t *testing.T) {
			found := make(map[string]bool)
			for ordinal := int64(0); ordinal < 512 && len(found) != len(tc.sites); ordinal++ {
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
					for _, site := range tc.sites {
						header := []byte("-------- [mpf_dbg] " + tc.name + " src/util/mpf.cpp:" + site + " ---------\n")
						at := bytes.Index(output.Bytes(), header)
						if body == nil || at < 0 {
							continue
						}
						record := output.Bytes()[at+len(header):]
						end := bytes.Index(record, []byte("------------------------------------------------\n"))
						if end >= 0 {
							continue
						}
						found[site] = true
						if x != beforeX || y != beforeY || tc.name != "add_sub" && site == tc.sites[0] && out != beforeOut {
							t.Fatal("debug formatter refusal changed borrowed operands or entry output")
						}
						if tc.name != "add_sub" && site == tc.sites[0] && (m.div.active || m.fma.active) {
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
			for _, site := range tc.sites {
				if !found[site] {
					t.Errorf("formatter refusal did not reach source line %s", site)
				}
			}
		})
	}
}
