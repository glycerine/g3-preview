package numeral

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestArbitraryFloatPartialRemainderScalarTraceOrder(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg_rem")
	compiled := tags.ShouldTrace("mpf_dbg_rem")
	serviceMust(t, m.BindTrace(tags))
	const hidden = 1 << 23
	x := ArbitraryFloat{format: arbitraryFloatFormat(11, 24, false), exponent: 40, significand: NewInt(hidden)}
	y := ArbitraryFloat{format: arbitraryFloatFormat(11, 24, true), exponent: 0, significand: NewInt(hidden + hidden/2)}
	serviceMust(t, m.PartialRemainder(&x, &y, 0, false))
	got := output.String()
	if !compiled {
		if got != "" {
			t.Fatalf("erased partial remainder trace: %q", got)
		}
	} else {
		last := -1
		for _, site := range []struct{ line, text string }{
			{"1282", "x="},
			{"1322", "tie= "},
			{"1365", "exp_delta="},
			{"1394", "aligned subtrahend="},
			{"1398", "X_YQ_sig'="},
			{"1421", "final sticky="},
		} {
			header := "-------- [mpf_dbg_rem] partial_remainder src/util/mpf.cpp:" + site.line + " ---------\n" + site.text
			at := strings.Index(got, header)
			if at <= last || strings.Count(got, header) != 1 {
				t.Fatalf("partial remainder trace order at %s: %q", site.line, got)
			}
			last = at
		}
		if !strings.Contains(got, "d=0\npartial=0\n") || strings.Count(got, "------------------------------------------------\n") != 6 {
			t.Fatalf("partial remainder source fields: %q", got)
		}
	}
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.Del(&y))
}
