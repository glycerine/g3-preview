// Source: src/util/timeit.h:25-31; src/util/timeit.cpp:25-58.
package util

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestTimeitOriginalMemoryCaptureAndFormatting(t *testing.T) {
	r, b := memoryRuntimeFixture(t, false, 64)
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	memoryRuntimeRequire(t, e)
	defer func() { memoryRuntimeRequire(t, out.Close()) }()
	var now uint64 = 100
	reads := 0
	var p Timeit
	memoryRuntimeRequire(t, p.Construct(true, TimeitConfig{Runtime: r, Output: out, Message: LiteralParamsStringView("isolate\x00ignored"), Clock: func() (uint64, error) { reads++; return now, nil }}))
	if reads != 1 || r.Stats().LiveBlocks != 1 {
		t.Fatal("imp allocation/start", reads, r.Stats())
	}
	size, e := r.Size(p.imp.block)
	memoryRuntimeRequire(t, e)
	if size != 56 {
		t.Fatal("source imp extent", size)
	}
	initial, e := p.startMemory()
	memoryRuntimeRequire(t, e)
	if initial != 64.0/(1024*1024) {
		t.Fatal("own allocation included in baseline", initial)
	}
	extra, e := r.Allocate(1024*1024 - 8)
	memoryRuntimeRequire(t, e)
	now += 1_250_000_000
	memoryRuntimeRequire(t, out.SetWidth(4))
	memoryRuntimeRequire(t, out.SetFill('_'))
	memoryRuntimeRequire(t, p.Close())
	memoryRuntimeRequire(t, p.Close())
	text, e := out.String()
	memoryRuntimeRequire(t, e)
	if text != "___(isolate :time 1.25 :before-memory 0.00 :after-memory 1.00)\n" {
		t.Fatal(text)
	}
	if reads != 2 || r.Stats().LiveBlocks != 1 {
		t.Fatal("destructor does not restart watch", reads, r.Stats())
	}
	if state := out.Stats().State; state.FloatMode != NumericFixedFloat || state.Precision != 2 {
		t.Fatal("source flags retained", state)
	}
	memoryRuntimeRequire(t, out.AppendFloat64(2.5))
	text, e = out.String()
	memoryRuntimeRequire(t, e)
	if !strings.HasSuffix(text, "\n2.50") {
		t.Fatal(text)
	}
	memoryRuntimeRequire(t, r.Deallocate(&extra))
}

func TestTimeitDisabledErasureAndDefaultStreamBorrow(t *testing.T) {
	var disabled Timeit
	if e := disabled.Construct(false, TimeitConfig{Message: ParamsStringView{PresentField: true, Mode: 255}, Clock: func() (uint64, error) { t.Fatal("disabled clock"); return 0, nil }}); e != nil {
		t.Fatal(e)
	}
	memoryRuntimeRequire(t, disabled.Close())
	memoryRuntimeRequire(t, disabled.Cleanup())
	r, b := memoryRuntimeFixture(t, false, 64)
	f, e := os.CreateTemp(t.TempDir(), "timeit")
	memoryRuntimeRequire(t, e)
	t.Cleanup(func() { f.Close() })
	streams, e := NewMemoryStandardStreams(MemoryStandardStreamsConfig{Budget: b, Stderr: f})
	memoryRuntimeRequire(t, e)
	var p Timeit
	memoryRuntimeRequire(t, p.Construct(true, TimeitConfig{Runtime: r, Streams: streams, Message: LiteralParamsStringView("default"), Clock: func() (uint64, error) { return 100, nil }}))
	copy := p
	if copy.Close() != ErrTimeitOwner || copy.Cleanup() != ErrTimeitOwner {
		t.Fatal("copied owner")
	}
	memoryRuntimeRequire(t, p.Close())
	_, e = f.Seek(0, io.SeekStart)
	memoryRuntimeRequire(t, e)
	text, e := io.ReadAll(f)
	memoryRuntimeRequire(t, e)
	if string(text) != "(default :time 0.00 :before-memory 0.00 :after-memory 0.00)\n" {
		t.Fatal(string(text))
	}
	memoryRuntimeRequire(t, streams.Close())
	if _, e = f.Write([]byte("caller")); e != nil {
		t.Fatal("borrowed file closed", e)
	}
}

func TestTimeitEveryColdConstructorAndOutputReservation(t *testing.T) {
	for _, output := range []bool{false, true} {
		completed := false
		for ordinal := int64(0); ordinal < 64; ordinal++ {
			r, b := memoryRuntimeFixture(t, false, 64)
			out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
			memoryRuntimeRequire(t, e)
			calls := 0
			var p Timeit
			config := TimeitConfig{Runtime: r, Output: out, Message: LiteralParamsStringView(strings.Repeat("message", 20)), Clock: func() (uint64, error) { calls++; return uint64(calls) * 100, nil }}
			if !output {
				b.FailAfter(ordinal)
			}
			cause := p.Construct(true, config)
			if output {
				memoryRuntimeRequire(t, cause)
				b.FailAfter(ordinal)
				cause = p.Close()
			}
			if cause != nil {
				var f Failure
				if !errors.Is(cause, arena.ErrInjectedFailure) && (!errors.As(cause, &f) || f.Kind != FailureMemout) {
					t.Fatal(cause)
				}
				if p.Close() != cause {
					t.Fatal("sticky failure replay")
				}
			}
			stats := out.Stats()
			if b.Used() != r.Stats().TotalBytes+stats.OwnerBytes+stats.ScratchBytes+stats.OutputOwnerBytes+stats.OutputOverflowBytes {
				t.Fatal("immediate cold graph", ordinal, b.Used(), r.Stats(), stats)
			}
			text, e := out.String()
			memoryRuntimeRequire(t, e)
			want := "(" + strings.Repeat("message", 20) + " :time 0.00 :before-memory 0.00 :after-memory 0.00)\n"
			if !strings.HasPrefix(want, text) {
				t.Fatal("output prefix", text)
			}
			if !output && cause != nil && calls != 0 {
				t.Fatal("constructor clock before allocation")
			}
			b.FailAfter(-1)
			memoryRuntimeRequire(t, p.Cleanup())
			memoryRuntimeRequire(t, p.Cleanup())
			memoryRuntimeRequire(t, out.Close())
			memoryRuntimeRequire(t, r.Close())
			if b.Used() != 0 {
				t.Fatal("cold refund", ordinal, b.Used())
			}
			if cause == nil {
				if output && text != want {
					t.Fatal(text)
				}
				completed = true
				break
			}
		}
		if !completed {
			t.Fatal("completion horizon")
		}
	}
}

func TestTimeitClockAndBorrowFailurePrefixes(t *testing.T) {
	for _, failedRead := range []int{1, 2} {
		r, b := memoryRuntimeFixture(t, false, 64)
		out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
		memoryRuntimeRequire(t, e)
		failure := errors.New("timeit clock refusal")
		calls := 0
		var p Timeit
		e = p.Construct(true, TimeitConfig{Runtime: r, Output: out, Message: LiteralParamsStringView("clock"), Clock: func() (uint64, error) {
			calls++
			if calls == failedRead {
				return 0, failure
			}
			return 100, nil
		}})
		if failedRead == 2 {
			memoryRuntimeRequire(t, e)
			e = p.Close()
		}
		if e != failure || p.Close() != failure || calls != failedRead {
			t.Fatal(e, calls)
		}
		text, e := out.String()
		memoryRuntimeRequire(t, e)
		if text != "" || r.Stats().LiveBlocks != 1 {
			t.Fatal("failure before output retains imp", text, r.Stats())
		}
		memoryRuntimeRequire(t, p.Cleanup())
		memoryRuntimeRequire(t, out.Close())
	}
	r, b := memoryRuntimeFixture(t, false, 64)
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	memoryRuntimeRequire(t, e)
	var p Timeit
	memoryRuntimeRequire(t, p.Construct(true, TimeitConfig{Runtime: r, Output: out, Message: ParamsStringView{Mode: 255, PresentField: true}, Clock: func() (uint64, error) { return 100, nil }}))
	if e = p.Close(); e != ErrParamsOwner || p.Close() != e {
		t.Fatal("borrow read at destructor insertion", e)
	}
	text, e := out.String()
	memoryRuntimeRequire(t, e)
	if text != "(" {
		t.Fatal(text)
	}
	memoryRuntimeRequire(t, p.Cleanup())
	memoryRuntimeRequire(t, out.Close())
}

func TestTimeitNullStaleAndClosedOutputBorrow(t *testing.T) {
	for _, mode := range []string{"null", "stale", "closed"} {
		r, b := memoryRuntimeFixture(t, false, 64)
		out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
		memoryRuntimeRequire(t, e)
		message := ParamsStringView{}
		var invalidate func()
		if mode == "stale" {
			c, _, _ := paramsTestOwner(t)
			v, e := c.NewValue()
			paramsMust(t, e)
			key, text := "timeit-message", "original"
			paramsMust(t, c.SetStringName(&v, &key, &text))
			message, e = c.GetStringViewFallbackName(&v, &key, nil, nil)
			paramsMust(t, e)
			invalidate = func() { paramsMust(t, c.Reset(&v)); paramsMust(t, c.Del(&v)) }
		}
		var p Timeit
		memoryRuntimeRequire(t, p.Construct(true, TimeitConfig{Runtime: r, Output: out, Message: message, Clock: func() (uint64, error) { return 100, nil }}))
		if invalidate != nil {
			invalidate()
		}
		if mode == "closed" {
			memoryRuntimeRequire(t, out.Close())
		}
		cause := p.Close()
		if mode == "null" {
			memoryRuntimeRequire(t, cause)
			text, e := out.String()
			memoryRuntimeRequire(t, e)
			if text != "(" || !out.FileBad || out.Stats().State.Precision != 2 || r.Stats().LiveBlocks != 0 {
				t.Fatal("nonthrowing null ostream insertion", text, out.Stats())
			}
		} else {
			if mode == "stale" && !errors.Is(cause, arena.ErrInvalidSlot) {
				t.Fatal(cause)
			}
			if mode == "closed" && cause != arena.ErrClosed {
				t.Fatal(cause)
			}
			if p.Close() != cause || r.Stats().LiveBlocks != 1 {
				t.Fatal("stopped borrowed insertion retains imp")
			}
		}
		memoryRuntimeRequire(t, p.Cleanup())
		memoryRuntimeRequire(t, out.Close())
	}
}

func TestTimeitTLSAccountingLimitAndReentry(t *testing.T) {
	for _, limit := range []bool{false, true} {
		r, b := memoryRuntimeFixture(t, true, 64)
		out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
		memoryRuntimeRequire(t, e)
		var preload Block
		if limit {
			preload, e = r.Allocate(99960)
			memoryRuntimeRequire(t, e)
			memoryRuntimeRequire(t, r.SetMaxSize(1))
		}
		var p Timeit
		calls := 0
		e = p.Construct(true, TimeitConfig{Runtime: r, Output: out, Message: LiteralParamsStringView("tls"), Clock: func() (uint64, error) {
			calls++
			if p.Close() != ErrTimerPending || p.Cleanup() != ErrTimerPending {
				t.Fatal("constructor/destructor callback reentry")
			}
			if calls == 1 {
				initial, e := p.startMemory()
				memoryRuntimeRequire(t, e)
				if initial != 0 {
					t.Fatal("source TLS unsynchronized memory baseline", initial)
				}
			} else {
				w := p.watchAdapter()
				state, e := w.readWatch()
				memoryRuntimeRequire(t, e)
				if !state.Running() {
					t.Fatal("stop clock prefix")
				}
			}
			return uint64(calls) * 100, nil
		}})
		if limit {
			memoryRuntimeCode(t, e, 101)
			if calls != 0 || p.imp.block == (Block{}) {
				t.Fatal("owned TLS failure before imp initialization")
			}
			if _, e = r.Read(p.imp.block, 0); e != ErrRuntimeUninitialized {
				t.Fatal(e)
			}
		} else {
			memoryRuntimeRequire(t, e)
			memoryRuntimeRequire(t, p.Close())
			if calls != 2 {
				t.Fatal("clock count", calls)
			}
		}
		memoryRuntimeRequire(t, p.Cleanup())
		if preload != (Block{}) {
			memoryRuntimeRequire(t, r.Deallocate(&preload))
		}
		memoryRuntimeRequire(t, out.Close())
	}
}
