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
			{"1301", "X/Y_exp="},
			{"1322", "tie= "},
			{"1331", "Q_exp="},
			{"1352", "YQ_sgn="},
			{"1365", "exp_delta="},
			{"1394", "aligned subtrahend="},
			{"1398", "X_YQ_sig'="},
			{"1410", "minuend="},
			{"1421", "final sticky="},
			{"1435", "partial remainder = go-words:"},
		} {
			header := "-------- [mpf_dbg_rem] partial_remainder src/util/mpf.cpp:" + site.line + " ---------\n" + site.text
			at := strings.Index(got, header)
			if at <= last || strings.Count(got, header) != 1 {
				t.Fatalf("partial remainder trace order at %s: %q", site.line, got)
			}
			last = at
		}
		for _, field := range []string{"d=0\npartial=0\n", "X/Y_sig_lrg=", "X/Y_rem=", "X/Y~=", "Q_sig=", "Q=", "YQ_exp=", "YQ_sig=", "YQ=", "subtrahend=", "X_YQ_sgn=", "X_YQ_exp=", "X_YQ_sig=", "X-YQ="} {
			if !strings.Contains(got, field) {
				t.Fatalf("missing partial remainder trace field %q: %q", field, got)
			}
		}
		if strings.Count(got, "------------------------------------------------\n") != 11 {
			t.Fatalf("partial remainder source fields: %q", got)
		}
	}
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.Del(&y))
}

func TestArbitraryFloatPartialRemainderZeroQuotientOmitsFinalTrace(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg_rem")
	compiled := tags.ShouldTrace("mpf_dbg_rem")
	serviceMust(t, m.BindTrace(tags))
	const hidden = 1 << 23
	x := ArbitraryFloat{format: arbitraryFloatFormat(11, 24, false), exponent: -1, significand: NewInt(hidden)}
	y := ArbitraryFloat{format: arbitraryFloatFormat(11, 24, false), exponent: 0, significand: NewInt(hidden + hidden/2)}
	before := x
	serviceMust(t, m.PartialRemainder(&x, &y, -1, false))
	got := output.String()
	if x != before {
		t.Fatalf("zero quotient changed input: before=%v after=%v", before, x)
	}
	if compiled {
		if !strings.Contains(got, "src/util/mpf.cpp:1331") || strings.Contains(got, "src/util/mpf.cpp:1435") || strings.Contains(got, "src/util/mpf.cpp:1352") {
			t.Fatalf("zero quotient trace continued after source return: %q", got)
		}
	} else if got != "" {
		t.Fatalf("erased zero quotient trace: %q", got)
	}
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.Del(&y))
}

func TestArbitraryFloatPartialRemainderZeroResidualEmitsFinalTrace(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg_rem")
	compiled := tags.ShouldTrace("mpf_dbg_rem")
	serviceMust(t, m.BindTrace(tags))
	const hidden = 1 << 23
	x := ArbitraryFloat{format: arbitraryFloatFormat(11, 24, true), exponent: 0, significand: NewInt(hidden)}
	y := x
	serviceMust(t, m.PartialRemainder(&x, &y, 0, false))
	zero, err := m.IsZero(&x)
	serviceMust(t, err)
	if !zero || !x.Sign() {
		t.Fatalf("equal negative operands did not produce negative zero: %+v", x)
	}
	got := output.String()
	if compiled {
		if !strings.Contains(got, "src/util/mpf.cpp:1398") || !strings.Contains(got, "src/util/mpf.cpp:1435 ---------\npartial remainder = go-words:") || strings.Contains(got, "src/util/mpf.cpp:1410") || strings.Contains(got, "src/util/mpf.cpp:1421") {
			t.Fatalf("zero residual trace did not skip nonzero rounding: %q", got)
		}
	} else if got != "" {
		t.Fatalf("erased zero residual trace: %q", got)
	}
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.Del(&y))
}
