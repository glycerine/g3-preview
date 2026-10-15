package numeral

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
)

type mpfEndlWriter struct {
	events  []string
	flushes int
	failAt  int
	refusal error
}

func (w *mpfEndlWriter) Write(p []byte) (int, error) { return w.WriteString(string(p)) }
func (w *mpfEndlWriter) WriteString(s string) (int, error) {
	w.events = append(w.events, "write:"+s)
	return len(s), nil
}
func (w *mpfEndlWriter) Flush() error {
	w.flushes++
	w.events = append(w.events, "flush")
	if w.flushes == w.failAt {
		return w.refusal
	}
	return nil
}

func TestArbitraryFloatFMAOverflowTraceEndlFlushes(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	w := &mpfEndlWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var one ArbitraryFloat
	serviceMust(t, m.MkOne(11, 53, false, &one))
	serviceMust(t, m.traceFMAOverflow(&one))
	if !tags.ShouldTrace("mpf_dbg") {
		if len(w.events) != 0 {
			t.Fatal("erased trace reached writer", w.events)
		}
		serviceMust(t, m.Del(&one))
		return
	}
	joined := strings.Join(w.events, "|")
	if w.flushes != 3 || !strings.Contains(joined, "write:Add/Sub overflew into 4.xxx!|write:\n|flush|write:S*= ") || !strings.Contains(joined, "write:\n|flush|write:------------------------------------------------\n|flush") {
		t.Fatal("source endl boundaries and final TRACE flush", w.flushes, w.events)
	}
	w.events = nil
	w.flushes = 0
	w.failAt = 1
	w.refusal = io.ErrClosedPipe
	serviceMust(t, m.traceFMAOverflow(&one))
	if w.flushes != 2 || strings.Contains(strings.Join(w.events, ""), "S*= ") || strings.Contains(strings.Join(w.events, ""), "------------------------------------------------") {
		t.Fatal("refused first endl allowed later physical insertion", w.flushes, w.events)
	}
	w.events = nil
	serviceMust(t, m.traceFMAOverflow(&one))
	if len(w.events) != 1 || w.events[0] != "flush" {
		t.Fatal("failed trace stream wrote a later record", w.events)
	}
	serviceMust(t, m.Del(&one))
}

func TestArbitraryFloatFMAReachedEndlFlushes(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	w := &mpfEndlWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var x, y, z, out ArbitraryFloat
	for _, v := range []*ArbitraryFloat{&x, &y, &z} {
		serviceMust(t, m.SetDouble(v, 11, 53, 1.75))
	}
	serviceMust(t, m.FMA(ArbitraryFloatNearestEven, &x, &y, &z, &out))
	if !tags.ShouldTrace("mpf_dbg") {
		if len(w.events) != 0 {
			t.Fatal("erased FMA trace reached writer", w.events)
		}
	} else {
		for _, tc := range []struct{ line, endlCount int }{
			{752, 1}, {753, 1}, {754, 1}, {813, 6}, {830, 2},
			{844, 2}, {867, 1}, {871, 2}, {873, 2}, {892, 1},
			{899, 2}, {913, 2}, {925, 1}, {940, 2},
		} {
			start := -1
			for i := 0; i+3 < len(w.events); i++ {
				if w.events[i] == "write:src/util/mpf.cpp" && w.events[i+1] == "write::" && w.events[i+2] == "write:"+strconv.Itoa(tc.line) && w.events[i+3] == "write: ---------\n" {
					start = i + 4
					break
				}
			}
			if start < 0 {
				t.Fatalf("missing reached FMA trace site %d", tc.line)
			}
			count := 0
			for i := start; i < len(w.events) && w.events[i] != "write:------------------------------------------------\n"; i++ {
				if w.events[i] == "flush" {
					count++
				}
			}
			if count != tc.endlCount {
				t.Fatalf("FMA source endl count at %d: got %d, want %d", tc.line, count, tc.endlCount)
			}
		}
	}
	for _, v := range []*ArbitraryFloat{&x, &y, &z, &out} {
		serviceMust(t, m.Del(v))
	}
}

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
	_, e := m.traceFloatBinaryString(&one, 1, 0)
	budget.FailAfter(-1)
	if !errors.Is(e, arena.ErrInjectedFailure) || m.binaryString.active {
		t.Fatalf("binary trace temporary retained after refusal: %v, active %v", e, m.binaryString.active)
	}
	text, e := m.traceFloatBinaryString(&one, 1, 0)
	serviceMust(t, e)
	if !strings.Contains(text, "#b") || m.binaryString.active {
		t.Fatalf("binary trace temporary retry: %q, active %v", text, m.binaryString.active)
	}
	serviceMust(t, m.Del(&one))
}

func TestArbitraryFloatFMABinaryTracePreservesEarlierStoppedString(t *testing.T) {
	m, _, budget := mpfNumeralHelpersFixture(t)
	var one ArbitraryFloat
	serviceMust(t, m.MkOne(11, 53, false, &one))
	var earlier utext.OwnedString
	budget.FailAfter(0)
	refused := m.ToStringBinary(&one, 1, 0, &earlier)
	budget.FailAfter(-1)
	if !numericPhysicalReservationRefusal(refused) || !m.binaryString.active {
		t.Fatal("expected earlier stopped string", refused)
	}
	if _, err := m.traceFloatBinaryString(&one, 1, 0); !errors.Is(err, ErrNumericPending) || !m.binaryString.active || m.binaryString.out != &earlier {
		t.Fatal("trace consumed an earlier string invocation", err)
	}
	serviceMust(t, m.CleanupBinaryString())
	earlier.OwnAbort()
	if got, err := m.traceFloatBinaryString(&one, 1, 0); err != nil || !strings.Contains(got, "#b") {
		t.Fatal("trace retry after owning cleanup", got, err)
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

func TestArbitraryFloatFMAAlignmentAndShiftedSignificandTrace(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var one, addend, out ArbitraryFloat
	serviceMust(t, m.MkOne(11, 53, false, &one))
	for _, tc := range []struct {
		name  string
		value float64
	}{
		{"unswapped", -0.5},
		{"swapped", 100},
	} {
		serviceMust(t, m.SetDouble(&addend, 11, 53, tc.value))
		output.Reset()
		serviceMust(t, m.FMA(ArbitraryFloatNearestEven, &one, &one, &addend, &out))
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("%s erased trace: %q", tc.name, got)
			}
			continue
		}
		last := -1
		for _, site := range []struct{ line, label string }{
			{"844", "C_= "}, {"867", "alignment shift by delta="}, {"871", "M'= "}, {"873", "C'= "},
		} {
			header := "-------- [mpf_dbg] fma src/util/mpf.cpp:" + site.line + " ---------\n"
			at := strings.Index(got, header)
			if at <= last || strings.Count(got, header) != 1 {
				t.Fatalf("%s alignment source order at %s: %q", tc.name, site.line, got)
			}
			last = at
			body := got[at+len(header):]
			end := strings.Index(body, "------------------------------------------------\n")
			if end < 0 || !strings.HasPrefix(body, site.label) {
				t.Fatalf("%s alignment body at %s: %q", tc.name, site.line, got)
			}
			if site.line == "867" {
				if !strings.Contains(body[:end], " -> sig = ") || !strings.Contains(body[:end], " sticky_rem = ") {
					t.Fatalf("%s alignment fields: %q", tc.name, body[:end])
				}
			} else if site.line == "871" || site.line == "873" {
				lines := strings.Split(strings.TrimSuffix(body[:end], "\n"), "\n")
				if len(lines) != 2 || !strings.HasPrefix(lines[1], site.label) || !strings.Contains(lines[1], "#b") {
					t.Fatalf("%s shifted significand lines: %q", tc.name, lines)
				}
			}
		}
	}
	serviceMust(t, m.Del(&one))
	serviceMust(t, m.Del(&addend))
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatFMAResultAndRenormalizationTrace(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var one, addend, out ArbitraryFloat
	serviceMust(t, m.MkOne(11, 53, false, &one))
	for _, tc := range []struct {
		name  string
		value float64
	}{
		{"addition", 1},
		{"subtraction", -0.5},
	} {
		serviceMust(t, m.SetDouble(&addend, 11, 53, tc.value))
		output.Reset()
		serviceMust(t, m.FMA(ArbitraryFloatNearestEven, &one, &one, &addend, &out))
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("%s erased trace: %q", tc.name, got)
			}
			continue
		}
		last := -1
		for _, site := range []struct {
			line   string
			first  string
			second string
		}{
			{"899", "S'= ", "S'= #b"},
			{"925", "R*= #b", " (renormalized, delta="},
			{"940", "sum[-1:sbits+2] = ", "R = #b"},
		} {
			header := "-------- [mpf_dbg] fma src/util/mpf.cpp:" + site.line + " ---------\n"
			at := strings.Index(got, header)
			if at <= last || strings.Count(got, header) != 1 {
				t.Fatalf("%s result source order at %s: %q", tc.name, site.line, got)
			}
			last = at
			body := got[at+len(header):]
			end := strings.Index(body, "------------------------------------------------\n")
			if end < 0 || !strings.HasPrefix(body[:end], site.first) || !strings.Contains(body[:end], site.second) {
				t.Fatalf("%s result body at %s: %q", tc.name, site.line, body)
			}
		}
		overflowHeader := "-------- [mpf_dbg] fma src/util/mpf.cpp:913 ---------\n"
		at := strings.Index(got, overflowHeader)
		if at >= 0 {
			t.Fatalf("%s unexpectedly reached overflow branch: %q", tc.name, got)
		}
	}
	output.Reset()
	serviceMust(t, m.traceFMAOverflow(&one))
	if compiled {
		got := output.String()
		if !strings.HasPrefix(got, "-------- [mpf_dbg] fma src/util/mpf.cpp:913 ---------\nAdd/Sub overflew into 4.xxx!\nS*= #b") || strings.Count(got, "------------------------------------------------\n") != 1 {
			t.Fatalf("overflow record format: %q", got)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased overflow record: %q", output.String())
	}
	serviceMust(t, m.Del(&one))
	serviceMust(t, m.Del(&addend))
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatFMATraceSourceInsertions(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	w := &mpffArithmeticTraceWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpf_dbg")
	if !tags.ShouldTrace("mpf_dbg") {
		return
	}
	serviceMust(t, m.BindTrace(tags))
	var one, addend, out ArbitraryFloat
	serviceMust(t, m.MkOne(11, 53, false, &one))
	serviceMust(t, m.SetDouble(&addend, 11, 53, -0.5))
	serviceMust(t, m.FMA(ArbitraryFloatNearestEven, &one, &one, &addend, &out))
	for _, label := range []string{"M = ", "C_= ", "M'= ", "C'= ", "S'= "} {
		found := false
		for i := 0; i+5 < len(w.parts); i++ {
			if w.parts[i] == label && w.parts[i+1] != "" && w.parts[i+2] == "\n" && w.parts[i+3] == label && strings.HasPrefix(w.parts[i+4], "#b") && w.parts[i+5] == "\n" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing FMA paired insertion %q in %q", label, w.parts)
		}
	}
	foundAlignment := false
	for i := 0; i+6 < len(w.parts); i++ {
		if w.parts[i] == "alignment shift by delta=" && w.parts[i+1] != "" && w.parts[i+2] == " -> sig = " && w.parts[i+3] != "" && w.parts[i+4] == " sticky_rem = " && w.parts[i+5] != "" && w.parts[i+6] == "\n" {
			foundAlignment = true
			break
		}
	}
	if !foundAlignment {
		t.Fatalf("missing FMA alignment insertions in %q", w.parts)
	}
	if !hasFloatTraceValueInsertion(w.parts, "sum[-1:sbits+2] = ") {
		t.Fatalf("missing FMA result significand insertions in %q", w.parts)
	}
	if !hasFloatTraceValueInsertion(w.parts, "R = ") {
		t.Fatalf("missing FMA result binary insertions in %q", w.parts)
	}
	foundRenorm := false
	for i := 0; i+5 < len(w.parts); i++ {
		if w.parts[i] == "R*= " && strings.HasPrefix(w.parts[i+1], "#b") && w.parts[i+2] == " (renormalized, delta=" && w.parts[i+3] != "" && w.parts[i+4] == ")" && w.parts[i+5] == "\n" {
			foundRenorm = true
			break
		}
	}
	if !foundRenorm {
		t.Fatalf("missing FMA renormalization insertions in %q", w.parts)
	}
	w.parts = nil
	serviceMust(t, m.traceFMAOverflow(&one))
	if !hasMPFFTraceParts(w.parts, []string{"Add/Sub overflew into 4.xxx!", "\n", "S*= "}) || !hasFloatTraceValueInsertion(w.parts, "S*= ") {
		t.Fatalf("missing FMA overflow insertion boundaries in %q", w.parts)
	}
	serviceMust(t, m.Del(&one))
	serviceMust(t, m.Del(&addend))
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatFMAOverflowTraceSourceBranch(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	w := &mpffArithmeticTraceWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var x, y, z, out ArbitraryFloat
	for _, value := range []*ArbitraryFloat{&x, &y, &z} {
		serviceMust(t, m.SetDouble(value, 11, 53, 1.75))
	}
	serviceMust(t, m.FMA(ArbitraryFloatNearestEven, &x, &y, &z, &out))
	got, e := m.ToDouble(&out)
	serviceMust(t, e)
	if got != 4.8125 {
		t.Fatalf("overflow-branch FMA result: %v", got)
	}
	if compiled {
		all := strings.Join(w.parts, "")
		first := strings.Index(all, "fma src/util/mpf.cpp:899 ---------")
		overflow := strings.Index(all, "fma src/util/mpf.cpp:913 ---------")
		next := strings.Index(all, "fma src/util/mpf.cpp:925 ---------")
		if first < 0 || overflow <= first || next <= overflow || !hasMPFFTraceParts(w.parts, []string{"Add/Sub overflew into 4.xxx!", "\n", "S*= "}) || !hasFloatTraceValueInsertion(w.parts, "S*= ") {
			t.Fatalf("real FMA overflow trace insertion order: %q", w.parts)
		}
	} else if len(w.parts) != 0 {
		t.Fatalf("erased FMA overflow trace: %q", w.parts)
	}
	for _, value := range []*ArbitraryFloat{&x, &y, &z, &out} {
		serviceMust(t, m.Del(value))
	}
}
