// Source: src/util/debug.cpp:37-42; src/util/debug.h:58-65.
package util

import (
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func debugNotificationFixture(t *testing.T) (*DebugContext, *MemoryStandardStreams, *arena.Budget, *os.File) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	paramsMust(t, e)
	d, e := NewDebugContext(r, b)
	paramsMust(t, e)
	f, e := os.CreateTemp(".", "debug-notification-*")
	paramsMust(t, e)
	streams, e := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: b, Stdout: f, Stderr: f})
	paramsMust(t, e)
	paramsMust(t, d.BindAssertionStreams(streams))
	t.Cleanup(func() {
		b.FailAfter(-1)
		paramsMust(t, d.Close())
		paramsMust(t, streams.Close())
		paramsMust(t, r.Close())
		f.Close()
		os.Remove(f.Name())
		if b.Used() != 0 {
			t.Fatal("notification graph", b.Used())
		}
	})
	return d, streams, b, f
}
func debugNotificationBytes(t *testing.T, f *os.File) string {
	t.Helper()
	_, e := f.Seek(0, io.SeekStart)
	paramsMust(t, e)
	s, e := io.ReadAll(f)
	paramsMust(t, e)
	return string(s)
}
func TestDebugNotificationExactStatementsSignedFlagsAndNoReservation(t *testing.T) {
	d, streams, b, f := debugNotificationFixture(t)
	out, e := streams.Stderr()
	paramsMust(t, e)
	paramsMust(t, out.SetWidth(25))
	paramsMust(t, out.SetFill('_'))
	paramsMust(t, out.SetIntegerBase(16))
	paramsMust(t, out.SetShowBase(true))
	paramsMust(t, out.SetUppercase(true))
	paramsMust(t, d.EnableAssertions(false)) // notify function itself never reads the switch.
	before := b.Used()
	debugNotificationPhysical(t, d, streams, b)
	b.FailAfter(0)
	paramsMust(t, d.NotifyAssertionViolation("src/file.cpp\x00unused", -1, "x != 0\x00unused"))
	debugNotificationPhysical(t, d, streams, b)
	if b.Used() != before {
		t.Fatal("file notification acquired storage")
	}
	b.FailAfter(-1)
	got := debugNotificationBytes(t, f)
	if got != "_____ASSERTION VIOLATION\nFile: src/file.cpp\nLine: 0XFFFFFFFF\nx != 0\n" {
		t.Fatal(got)
	}
	if out.State().Width != 0 || out.State().IntegerBase != 16 {
		t.Fatal("caller flags", out.State())
	}
}
func TestDebugNotificationBorrowedCStringActualLifetimePrefixes(t *testing.T) {
	for _, badFile := range []bool{false, true} {
		t.Run(map[bool]string{false: "condition", true: "filename"}[badFile], func(t *testing.T) {
			d, streams, _, f := debugNotificationFixture(t)
			filename := debugNotificationCString(t, d.region, "old.cpp")
			condition := debugNotificationCString(t, d.region, "a == b")
			paramsMust(t, d.region.Write(filename.Ref, 0, 'n'))
			if badFile {
				filename.Present = false
			} else {
				condition.Present = false
			}
			e := d.NotifyBorrowedAssertionViolation(d.region, filename, 17, d.region, condition)
			if !errors.Is(e, ErrParamsStreamFailure) {
				t.Fatal("null pointer reached guard", e)
			}
			want := "ASSERTION VIOLATION\nFile: "
			if !badFile {
				want += "nld.cpp\nLine: 17\n"
			}
			if got := debugNotificationBytes(t, f); got != want {
				t.Fatal("reached insertion prefix", got, want)
			}
			out, e := streams.Stderr()
			paramsMust(t, e)
			bad, e := out.Bad()
			paramsMust(t, e)
			if bad {
				t.Fatal("guarded null pointer does not emulate default stream failure")
			}
		})
	}
}
func TestDebugNotificationClosedFileStillReadsLaterBorrowedCondition(t *testing.T) {
	d, streams, b, f := debugNotificationFixture(t)
	other, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, other.Close()) })
	filename := debugNotificationCString(t, d.region, "f.cpp")
	condition := debugNotificationCString(t, other, "x")
	paramsMust(t, other.Close())
	paramsMust(t, f.Close())
	before := b.Used()
	b.FailAfter(0)
	e = d.NotifyBorrowedAssertionViolation(d.region, filename, 3, other, condition)
	if !errors.Is(e, arena.ErrClosed) {
		t.Fatal("badbit erases output, not later C-string length", e)
	}
	debugNotificationPhysical(t, d, streams, b)
	if b.Used() != before {
		t.Fatal("stopped notification allocated")
	}
	out, e := streams.Stderr()
	paramsMust(t, e)
	bad, e := out.Bad()
	paramsMust(t, e)
	if !bad {
		t.Fatal("actual file failure")
	}
}
func TestDebugNotificationBindingBorrowsClosedStreamAndRejectsForeignBudget(t *testing.T) {
	d, streams, _, _ := debugNotificationFixture(t)
	paramsMust(t, streams.Close())
	paramsMust(t, d.BindAssertionStreams(streams))
	e := d.NotifyAssertionViolation("f", 1, "x")
	if !errors.Is(e, arena.ErrClosed) {
		t.Fatal("pointed lifetime at notification", e)
	}
	b := arena.NewBudget(1 << 20)
	foreign, e := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: b})
	paramsMust(t, e)
	t.Cleanup(func() {
		paramsMust(t, foreign.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	})
	if e = d.BindAssertionStreams(foreign); !errors.Is(e, arena.ErrConfig) {
		t.Fatal("common cohort", e)
	}
}

func debugNotificationPhysical(t *testing.T, d *DebugContext, streams *MemoryStandardStreams, b *arena.Budget) {
	t.Helper()
	x := d.region.Stats()
	a := x.Descriptors
	want := x.OwnerBytes + x.PayloadBytes + x.PageHeaderBytes + x.DirectoryBytes + x.LogBytes + x.ScopeBytes + a.OwnerBytes + a.RecordBytes + a.MetadataBytes + a.DirectoryBytes
	if !d.closed {
		want += d.ownerBytes
	}
	if !streams.closed {
		want += streams.header
	}
	for _, out := range []*NumericTextStream{streams.stdout, streams.StderrField} {
		h := out.Stats()
		want += h.OwnerBytes + h.ScratchBytes + h.OutputOwnerBytes + h.OutputOverflowBytes
	}
	if want != b.Used() {
		t.Fatal("actual caller-named notification cold graph", want, b.Used())
	}
}
func TestDebugNotificationValidBorrowedCStringFinitePrefixesAndMutation(t *testing.T) {
	d, streams, b, f := debugNotificationFixture(t)
	cases := []string{"", "abc", "raw\x00suffix", "\xc3\xa9", "space text"}
	for i, file := range cases {
		filename := debugNotificationCString(t, d.region, file)
		condition := debugNotificationCString(t, d.region, cases[len(cases)-1-i])
		if file != "" {
			paramsMust(t, d.region.Write(filename.Ref, 0, 'Z'))
			file = "Z" + file[1:]
		}
		before := b.Used()
		b.FailAfter(0)
		paramsMust(t, d.NotifyBorrowedAssertionViolation(d.region, filename, int32(i), d.region, condition))
		debugNotificationPhysical(t, d, streams, b)
		if b.Used() != before {
			t.Fatal("borrowed notification allocated")
		}
		b.FailAfter(-1)
		got := debugNotificationBytes(t, f)
		filePrefix, _, _ := strings.Cut(file, "\x00")
		conditionPrefix, _, _ := strings.Cut(cases[len(cases)-1-i], "\x00")
		want := "ASSERTION VIOLATION\nFile: " + filePrefix + "\nLine: " + strconv.Itoa(i) + "\n" + conditionPrefix + "\n"
		if got != want {
			t.Fatal(i, got, want)
		}
		paramsMust(t, f.Truncate(0))
		_, e := f.Seek(0, io.SeekStart)
		paramsMust(t, e)
	}
}
func TestDebugNotificationUnboundAndOwnerGuards(t *testing.T) {
	d, streams, b, _ := debugNotificationFixture(t)
	paramsMust(t, d.BindAssertionStreams(nil))
	debugNotificationPhysical(t, d, streams, b)
	if e := d.NotifyAssertionViolation("f", 1, "x"); !errors.Is(e, ErrDebugNotificationBinding) {
		t.Fatal("explicit unbound service", e)
	}
	paramsMust(t, d.BindAssertionStreams(streams))
	copy := *d
	if e := copy.NotifyAssertionViolation("f", 1, "x"); !errors.Is(e, ErrDebugContextOwner) {
		t.Fatal("copied owner", e)
	}
	paramsMust(t, d.Close())
	if e := d.NotifyAssertionViolation("f", 1, "x"); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed context", e)
	}
	debugNotificationPhysical(t, d, streams, b)
}

func debugNotificationCString(t *testing.T, r *arena.BytesRegion, text string) GlobalParamCString {
	t.Helper()
	ref, e := r.Allocate(uint32(len(text) + 1))
	paramsMust(t, e)
	for i := 0; i < len(text); i++ {
		paramsMust(t, r.Write(ref, uint32(i), text[i]))
	}
	paramsMust(t, r.Write(ref, uint32(len(text)), 0))
	return GlobalParamCString{Origin: r.OwnerID(), Ref: ref, Present: true}
}

func TestDebugNotificationForeignCStringProviderRemainsBorrowed(t *testing.T) {
	d, streams, b, f := debugNotificationFixture(t)
	foreign := arena.NewBudget(1 << 20)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: foreign}})
	paramsMust(t, e)
	t.Cleanup(func() {
		foreign.FailAfter(-1)
		paramsMust(t, region.Close())
		if foreign.Used() != 0 {
			t.Fatal("foreign caller graph", foreign.Used())
		}
	})
	filename := debugNotificationCString(t, region, "other.cpp")
	condition := debugNotificationCString(t, region, "kept")
	before := foreign.Used()
	b.FailAfter(0)
	foreign.FailAfter(0)
	paramsMust(t, d.NotifyBorrowedAssertionViolation(region, filename, 2, region, condition))
	debugNotificationPhysical(t, d, streams, b)
	if foreign.Used() != before {
		t.Fatal("borrowed provider acquired/refunded ownership")
	}
	if got := debugNotificationBytes(t, f); got != "ASSERTION VIOLATION\nFile: other.cpp\nLine: 2\nkept\n" {
		t.Fatal(got)
	}
	b.FailAfter(-1)
	foreign.FailAfter(-1)
	paramsMust(t, d.Close())
	c, e := region.Read(condition.Ref, 0)
	paramsMust(t, e)
	if c != 'k' {
		t.Fatal("context close destroyed caller text")
	}
}
