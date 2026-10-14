package numeral

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestArbitraryFloatFMAEntryDebugTraceBeforeShortcuts(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var zero, one, out ArbitraryFloat
	serviceMust(t, m.MkZero(11, 53, false, &zero))
	serviceMust(t, m.MkOne(11, 53, false, &one))
	for _, x := range []*ArbitraryFloat{&zero, &one} {
		output.Reset()
		serviceMust(t, m.FMA(ArbitraryFloatNearestEven, x, &one, &one, &out))
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("erased FMA trace: %q", got)
			}
			continue
		}
		last := -1
		for _, site := range []struct{ line, label string }{{"752", "X = "}, {"753", "Y = "}, {"754", "Z = "}} {
			entry := "-------- [mpf_dbg] fma src/util/mpf.cpp:" + site.line + " ---------\n" + site.label
			at := strings.Index(got, entry)
			if at <= last {
				t.Fatalf("FMA entry order at %s: %q", site.line, got)
			}
			last = at
		}
		if x == &zero && strings.Count(got, "-------- [mpf_dbg] fma src/util/mpf.cpp:") != 3 {
			t.Fatalf("shortcut reached FMA inner trace: %q", got)
		}
	}
	serviceMust(t, m.Del(&zero))
	serviceMust(t, m.Del(&one))
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatFMASwapAndArithmeticBranchTrace(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var one, z, out ArbitraryFloat
	serviceMust(t, m.MkOne(11, 53, false, &one))
	for _, tc := range []struct {
		name           string
		addend         float64
		swap, subtract bool
	}{
		{"large positive addend", 100, true, false},
		{"small negative addend", -0.5, false, true},
	} {
		serviceMust(t, m.SetDouble(&z, 11, 53, tc.addend))
		output.Reset()
		serviceMust(t, m.FMA(ArbitraryFloatNearestEven, &one, &one, &z, &out))
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("%s erased trace: %q", tc.name, got)
			}
			continue
		}
		prefix := "-------- [mpf_dbg] fma src/util/mpf.cpp:"
		swapAt := strings.Index(got, prefix+"852 ---------\nSwap!\n")
		if (swapAt >= 0) != tc.swap {
			t.Fatalf("%s swap trace: %q", tc.name, got)
		}
		branch, opposite := "892 ---------\nADDING\n", "880 ---------\nSUBTRACTING\n"
		if tc.subtract {
			branch, opposite = opposite, branch
		}
		at := strings.Index(got, prefix+branch)
		if at < 0 || strings.Contains(got, prefix+opposite) || tc.swap && at <= swapAt {
			t.Fatalf("%s selected FMA branch: %q", tc.name, got)
		}
	}
	serviceMust(t, m.Del(&one))
	serviceMust(t, m.Del(&z))
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatFMAUnpackedDecimalAndBinaryTrace(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var one, out ArbitraryFloat
	serviceMust(t, m.MkOne(11, 53, false, &one))
	serviceMust(t, m.FMA(ArbitraryFloatNearestEven, &one, &one, &one, &out))
	if compiled {
		got := output.String()
		const header = "-------- [mpf_dbg] fma src/util/mpf.cpp:813 ---------\n"
		start := strings.Index(got, header)
		if start < 0 || strings.Count(got, header) != 1 {
			t.Fatalf("missing unpacked source block: %q", got)
		}
		body := got[start+len(header):]
		end := strings.Index(body, "------------------------------------------------\n")
		if end < 0 {
			t.Fatalf("unterminated unpacked source block: %q", got)
		}
		lines := strings.Split(strings.TrimSuffix(body[:end], "\n"), "\n")
		if len(lines) != 6 {
			t.Fatalf("unpacked six source records: %q", lines)
		}
		for i, line := range lines {
			label := []string{"A = ", "B = ", "C = "}[i%3]
			if !strings.HasPrefix(line, label) || len(line) == len(label) {
				t.Fatalf("unpacked source order at %d: %q", i, lines)
			}
			if i >= 3 && !strings.Contains(line, "#b") {
				t.Fatalf("binary source text at %d: %q", i, line)
			}
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased unpacked source trace: %q", output.String())
	}
	serviceMust(t, m.Del(&one))
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatFMABinaryTraceTemporaryFailureCleanup(t *testing.T) {
	m, _, budget := mpfNumeralHelpersFixture(t)
	var one ArbitraryFloat
	serviceMust(t, m.MkOne(11, 53, false, &one))
	budget.FailAfter(0)
	_, e := m.traceFMABinaryString(&one, 1, 0)
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || m.binaryString.active {
		t.Fatalf("binary trace temporary retained after refusal: %v, active %v", e, m.binaryString.active)
	}
	text, e := m.traceFMABinaryString(&one, 1, 0)
	serviceMust(t, e)
	if !strings.Contains(text, "#b") || m.binaryString.active {
		t.Fatalf("binary trace temporary retry: %q, active %v", text, m.binaryString.active)
	}
	serviceMust(t, m.Del(&one))
}

func TestArbitraryFloatFMAUnpackedTraceFormatterRefusalPropagates(t *testing.T) {
	m, _, budget := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var one ArbitraryFloat
	serviceMust(t, m.MkOne(11, 53, false, &one))
	if !tags.ShouldTrace("mpf_dbg") {
		serviceMust(t, m.traceFMAUnpacked(&one, &one, &one))
		serviceMust(t, m.Del(&one))
		return
	}
	budget.FailAfter(0)
	e := m.traceFMAUnpacked(&one, &one, &one)
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || m.binaryString.active || !strings.Contains(output.String(), "src/util/mpf.cpp:813") {
		t.Fatalf("reached formatter refusal: %v, active %v, output %q", e, m.binaryString.active, output.String())
	}
	output.Reset()
	serviceMust(t, m.traceFMAUnpacked(&one, &one, &one))
	if !strings.Contains(output.String(), "C = #b") {
		t.Fatalf("unpacked trace retry: %q", output.String())
	}
	serviceMust(t, m.Del(&one))
}

func TestArbitraryFloatFMAProductAndShiftedAddendTrace(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var one, addend, out ArbitraryFloat
	serviceMust(t, m.MkOne(11, 53, false, &one))
	serviceMust(t, m.SetDouble(&addend, 11, 53, 100))
	serviceMust(t, m.FMA(ArbitraryFloatNearestEven, &one, &one, &addend, &out))
	if compiled {
		got := output.String()
		last := -1
		for _, site := range []struct{ line, label string }{{"830", "M = "}, {"844", "C_= "}} {
			header := "-------- [mpf_dbg] fma src/util/mpf.cpp:" + site.line + " ---------\n"
			at := strings.Index(got, header)
			if at <= last || strings.Count(got, header) != 1 {
				t.Fatalf("FMA product/addend source order at %s: %q", site.line, got)
			}
			last = at
			body := got[at+len(header):]
			end := strings.Index(body, "------------------------------------------------\n")
			if end < 0 {
				t.Fatalf("FMA trace footer at %s: %q", site.line, got)
			}
			lines := strings.Split(strings.TrimSuffix(body[:end], "\n"), "\n")
			if len(lines) != 2 || !strings.HasPrefix(lines[0], site.label) || !strings.HasPrefix(lines[1], site.label) || !strings.Contains(lines[1], "#b") {
				t.Fatalf("FMA decimal/binary block at %s: %q", site.line, lines)
			}
		}
		if swap := strings.Index(got, "src/util/mpf.cpp:852"); swap <= last {
			t.Fatalf("swap preceded shifted addend: %q", got)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased product/addend trace: %q", output.String())
	}
	serviceMust(t, m.Del(&one))
	serviceMust(t, m.Del(&addend))
	serviceMust(t, m.Del(&out))
}
