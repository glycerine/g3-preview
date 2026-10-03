package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"strconv"
	"strings"
	"testing"
)

func pfstr(s string) *string { return &s }
func pfnew(t *testing.T, c PrintfConfig) *PrintfFormatter {
	t.Helper()
	f, e := NewPrintfFormatter(c)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { f.Close() })
	return f
}
func pfformat(t *testing.T, f *PrintfFormatter, format, want string, args ...PrintfArg) {
	t.Helper()
	s, e := f.Format(format, args)
	if e != nil || s != want {
		t.Fatalf("format %q = %q, want %q: %v", format, s, want, e)
	}
}
func TestPrintfSourceExceptionConsumerAndCallsiteForms(t *testing.T) {
	f := pfnew(t, PrintfConfig{})
	pfformat(t, f, "Format %d %s", "Format 12 twelve", PrintfInt32(12), PrintfCString(pfstr("twelve")))
	pfformat(t, f, "%s at line %u '%s' found '%s'\n", "identifier at line 7 'name' found 'bad'\n", PrintfCString(pfstr("identifier")), PrintfUInt32(7), PrintfCString(pfstr("name")), PrintfCString(pfstr("bad")))
	x, e := NewDefaultExceptionFormatted(f, "Format %d %s", []PrintfArg{PrintfInt32(12), PrintfCString(pfstr("twelve"))})
	if e != nil || x.Message() != "Format 12 twelve" || x.HasErrorCode() {
		t.Fatal(x, e)
	}
	pfformat(t, f, "%ld", "-9223372036854775808", PrintfInt64(-9223372036854775808))
	pfformat(t, f, "literal %%", "literal %", PrintfInt32(999))
}
func TestPrintfIntegerFlagsWidthsPrecisionsAndDomains(t *testing.T) {
	f := pfnew(t, PrintfConfig{})
	for _, x := range []struct {
		format, want string
		arg          PrintfArg
	}{{"%+06d", "+00042", PrintfInt32(42)}, {"% d", " 42", PrintfInt32(42)}, {"%#08x", "0x00002a", PrintfUInt32(42)}, {"%-#8x", "0x2a    ", PrintfUInt32(42)}, {"%#o", "052", PrintfUInt32(42)}, {"%#X", "0X2A", PrintfUInt32(42)}, {"%.0d", "", PrintfInt32(0)}, {"%#.0o", "0", PrintfUInt32(0)}, {"%08.3d", "     012", PrintfInt32(12)}, {"%hhd", "-1", PrintfInt32(255)}, {"%hhu", "255", PrintfInt32(-1)}, {"%hd", "-32768", PrintfInt32(32768)}, {"%hu", "65535", PrintfUInt32(65535)}, {"%lld", "-9223372036854775808", PrintfInt64(-9223372036854775808)}, {"%llu", "18446744073709551615", PrintfUInt64(^uint64(0))}, {"%zu", "18446744073709551615", PrintfUInt64(^uint64(0))}} {
		pfformat(t, f, x.format, x.want, x.arg)
	}
	pfformat(t, f, "%*.*s", "ab    ", PrintfInt32(-6), PrintfInt32(2), PrintfCString(pfstr("abc")))
	pfformat(t, f, "%.*d", "12", PrintfInt32(-1), PrintfInt32(12))
	short := pfnew(t, PrintfConfig{Profile: PrintfNonWindows32})
	pfformat(t, short, "%ld %zu", "-2147483648 4294967295", PrintfInt32(-2147483648), PrintfUInt32(^uint32(0)))
	if _, e := short.Format("%ld", []PrintfArg{PrintfInt64(4)}); e != ErrPrintfArgument {
		t.Fatal(e)
	}
}
func TestPrintfCBytesNullPrecisionAndCharacterCount(t *testing.T) {
	f := pfnew(t, PrintfConfig{})
	pfformat(t, f, "%s|%.2s|%c", "A|\xff\xfe|\xff", PrintfCString(pfstr("A\x00B")), PrintfCString(pfstr("\xff\xfeC")), PrintfChar(255))
	pfformat(t, f, "%s", "(null)", PrintfCString(nil))
	for p := int32(0); p < 6; p++ {
		pfformat(t, f, "[%.*s]", "[]", PrintfInt32(p), PrintfCString(nil))
	}
	pfformat(t, f, "[%.6s]", "[(null)]", PrintfCString(nil))
	pfformat(t, f, "x%cTAIL", "x", PrintfChar(0))
	pfformat(t, f, "abc\x00%d", "abc")
}
func TestPrintfSourceBufferCapacitiesTruncationAndGrowLoop(t *testing.T) {
	for _, profile := range []PrintfProfile{PrintfNonWindowsAMD64, PrintfNonWindows32} {
		b := arena.NewBudget(^uint64(0))
		events := []PrintfEvent{}
		f := pfnew(t, PrintfConfig{Budget: b, Profile: profile, Observe: func(e PrintfEvent) { events = append(events, e) }})
		base := b.Used()
		text := strings.Repeat("x", 1100)
		out, e := f.Format("%s", []PrintfArg{PrintfCString(&text)})
		if e != nil {
			t.Fatal(e)
		}
		if profile == PrintfNonWindowsAMD64 {
			if len(out) != 1023 {
				t.Fatal(len(out))
			}
		} else if out != text {
			t.Fatal(len(out))
		}
		if b.Used() != base || f.Stats().WorkspaceBytes != 0 {
			t.Fatal(b.Used(), base, f.Stats())
		}
		calls := []PrintfEvent{}
		caps := []uint32{}
		for _, e := range events {
			if e.Kind == PrintfGrow {
				caps = append(caps, e.Capacity)
			}
			if e.Kind == PrintfCall {
				calls = append(calls, e)
			}
		}
		if profile == PrintfNonWindowsAMD64 {
			want := []uint32{2, 3, 5, 8, 12, 18, 27, 41, 62, 93, 140, 210, 315, 473, 710, 1065}
			if len(caps) != len(want) {
				t.Fatal(caps)
			}
			for i, c := range want {
				if caps[i] != c {
					t.Fatal(caps)
				}
			}
			if len(calls) != 1 || calls[0].Size != 1024 || calls[0].Capacity != 1065 || calls[0].Count != 1100 {
				t.Fatal(calls)
			}
		} else {
			want := []uint32{128, 257, 515, 1031, 2063}
			if len(calls) != len(want) {
				t.Fatal(calls)
			}
			for i, n := range want {
				if calls[i].Size != n || calls[i].Count != 1100 {
					t.Fatal(calls)
				}
			}
			if calls[0].Capacity != 140 || calls[1].Capacity != 315 || calls[2].Capacity != 710 || calls[3].Capacity != 1065 || calls[4].Capacity != 2397 {
				t.Fatal(calls)
			}
		}
		if events[len(events)-1].Kind != PrintfEnd {
			t.Fatal(events)
		}
		f.Close()
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
}
func TestPrintfLinuxI64WidthAndExplicitMismatchBoundary(t *testing.T) {
	f := pfnew(t, PrintfConfig{})
	pfformat(t, f, "%I64u", strings.Repeat(" ", 63)+"1", PrintfUInt32(1))
	pfformat(t, f, "%I64u", strings.Repeat(" ", 54)+"4294967295", PrintfUInt32(^uint32(0)))
	// The fork passes uint64 to this GNU width64/%u form. A permanent explicit
	// low32 diagnostic fixture records observed LinuxABI bits without pretending
	// that original mismatched C-varargs argument is defined behavior.
	original := uint64(4294967297)
	pfformat(t, f, "%I64u", strings.Repeat(" ", 63)+"1", PrintfUInt32(uint32(original)))
	if _, e := f.Format("%I64u", []PrintfArg{PrintfUInt64(original)}); e != ErrPrintfArgument {
		t.Fatal(e)
	}
}
func TestPrintfMalformedUnsupportedAndSignedCountOverflow(t *testing.T) {
	f := pfnew(t, PrintfConfig{})
	pfformat(t, f, "bad %", "bad %")
	pfformat(t, f, "%2147483648d", "%2147483648d", PrintfInt32(7))
	pfformat(t, f, "x%2147483647d", "x%2147483647d", PrintfInt32(7))
	pfformat(t, f, "%*d", "%*d", PrintfInt32(-2147483648), PrintfInt32(7))
	for _, s := range []string{"%f", "%p", "%n", "%ls", "%lc", "%1$d", "%'d", "%m"} {
		if _, e := f.Format(s, []PrintfArg{PrintfInt32(7)}); e != ErrPrintfUnsupported {
			t.Fatal(s, e)
		}
	}
	if _, e := f.Format("%s", nil); e != ErrPrintfArgument {
		t.Fatal(e)
	}
	if _, e := f.Format("%u", []PrintfArg{PrintfInt64(1)}); e != ErrPrintfArgument {
		t.Fatal(e)
	}
	short := pfnew(t, PrintfConfig{Profile: PrintfNonWindows32})
	if _, e := short.Format("bad %", nil); e != ErrPrintfMalformed {
		t.Fatal(e)
	}
	if _, e := short.Format("%2147483648d", []PrintfArg{PrintfInt32(1)}); e != ErrPrintfCountOverflow {
		t.Fatal(e)
	}
}
func TestPrintfBudgetFailureEventsAndOwnerGuards(t *testing.T) {
	fail, ok := 0, 0
	for prefix := int64(0); prefix < 20; prefix++ {
		b := arena.NewBudget(^uint64(0))
		f := pfnew(t, PrintfConfig{Budget: b})
		base := b.Used()
		b.FailAfter(prefix)
		_, e := f.Format("%d", []PrintfArg{PrintfInt32(1)})
		b.FailAfter(-1)
		if e != nil {
			fail++
		} else {
			ok++
		}
		if b.Used() != base || f.Stats().WorkspaceBytes != 0 {
			t.Fatal(prefix, e, b.Used(), base)
		}
		f.Close()
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
	if fail == 0 || ok == 0 {
		t.Fatal(fail, ok)
	}
	b := arena.NewBudget(^uint64(0))
	events := []PrintfEvent{}
	f := pfnew(t, PrintfConfig{Budget: b, Profile: PrintfNonWindows32, Observe: func(e PrintfEvent) { events = append(events, e) }})
	base := b.Used()
	b.FailAfter(11)
	text := strings.Repeat("x", 200)
	_, e := f.Format("%s", []PrintfArg{PrintfCString(&text)})
	b.FailAfter(-1)
	if e != arena.ErrInjectedFailure || b.Used() != base {
		t.Fatal(e, b.Used())
	}
	begins, ends, calls := 0, 0, 0
	for _, e := range events {
		switch e.Kind {
		case PrintfBegin:
			begins++
		case PrintfEnd:
			ends++
		case PrintfCall:
			calls++
		}
	}
	if begins != 1 || ends != 0 || calls != 1 {
		t.Fatal(begins, ends, calls)
	}
	copied := PrintfFormatter{self: f}
	if _, e := copied.Format("x", nil); e != ErrPrintfOwner {
		t.Fatal(e)
	}
	f.Close()
	if _, e := f.Format("x", nil); e != arena.ErrClosed {
		t.Fatal(e)
	}
	b.FailAfter(0)
	if _, e := NewPrintfFormatter(PrintfConfig{Budget: b}); e != arena.ErrInjectedFailure || b.Used() != 0 {
		t.Fatal(e, b.Used())
	}
}
func TestPrintfIndependentFiniteIntegerAndPaddingOracle(t *testing.T) {
	f := pfnew(t, PrintfConfig{})
	seed := uint64(17)
	for i := 0; i < 1000; i++ {
		seed = seed*6364136223846793005 + 1
		v := int64(seed)
		pfformat(t, f, "%lld", strconv.FormatInt(v, 10), PrintfInt64(v))
		pfformat(t, f, "%llx", strconv.FormatUint(seed, 16), PrintfUInt64(seed))
		pfformat(t, f, "%llo", strconv.FormatUint(seed, 8), PrintfUInt64(seed))
	}
	for width := 0; width < 12; width++ {
		for precision := 0; precision < 8; precision++ {
			text := "\xffabc\x00ignored"
			n := precision
			if n > 4 {
				n = 4
			}
			want := text[:n]
			if width > n {
				want = strings.Repeat(" ", width-n) + want
			}
			pfformat(t, f, "%*.*s", want, PrintfInt32(int32(width)), PrintfInt32(int32(precision)), PrintfCString(&text))
		}
	}
}
func TestPrintfSignedCountOverflowOnEscapedPercent(t *testing.T) {
	f := pfnew(t, PrintfConfig{})
	pfformat(t, f, "%2147483647d%%", "%2147483647d%%", PrintfInt32(1))
}
func TestPrintfBudgetPeakExactAndObserverPanicCleanup(t *testing.T) {
	probe := pfnew(t, PrintfConfig{})
	owner := probe.Stats().OwnerBytes
	for _, extra := range []uint64{1774, 1775} {
		b := arena.NewBudget(owner + extra)
		f := pfnew(t, PrintfConfig{Budget: b})
		_, e := f.Format("x", nil)
		if extra == 1774 && e != arena.ErrBudget {
			t.Fatal(extra, e)
		}
		if extra == 1775 && e != nil {
			t.Fatal(extra, e)
		}
		if b.Used() != owner {
			t.Fatal("released scratch", b.Used(), owner)
		}
		f.Close()
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
	b := arena.NewBudget(^uint64(0))
	f := pfnew(t, PrintfConfig{Budget: b, Observe: func(e PrintfEvent) {
		if e.Kind == PrintfBegin {
			panic("observer")
		}
	}})
	base := b.Used()
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("missing observer panic")
			}
		}()
		f.Format("x", nil)
	}()
	if b.Used() != base || f.Stats().WorkspaceBytes != 0 {
		t.Fatal(b.Used(), base)
	}
	f.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestPrintfByteBoundaryAndBackendEventClassification(t *testing.T) {
	events := []PrintfEvent{}
	f := pfnew(t, PrintfConfig{Observe: func(e PrintfEvent) { events = append(events, e) }})
	text := strings.Repeat("é", 600)
	out, e := f.Format("%s", []PrintfArg{PrintfCString(&text)})
	if e != nil || out != text[:1023] || out[len(out)-1] != 0xc3 {
		t.Fatal(len(out), e)
	}
	events = nil
	pfformat(t, f, "bad %", "bad %")
	call := PrintfEvent{}
	for _, e := range events {
		if e.Kind == PrintfCall {
			call = e
		}
	}
	if call.Count != -1 || call.Failure != PrintfMalformedFailure || events[len(events)-1].Kind != PrintfEnd {
		t.Fatal(events)
	}
	events = nil
	if _, e := f.Format("prefix %f", nil); e != ErrPrintfUnsupported {
		t.Fatal(e)
	}
	for _, e := range events {
		if e.Kind == PrintfCall {
			call = e
		}
	}
	if call.Failure != PrintfUnsupportedFailure || call.Count != 7 || events[len(events)-1].Kind == PrintfEnd {
		t.Fatal(events)
	}
	events = nil
	pfformat(t, f, "x%2147483647d", "x%2147483647d", PrintfInt32(1))
	for _, e := range events {
		if e.Kind == PrintfCall {
			call = e
		}
	}
	if call.Count != -1 || call.Failure != PrintfCountFailure {
		t.Fatal(events)
	}
}
