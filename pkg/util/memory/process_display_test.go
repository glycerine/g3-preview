package memory

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

type peakReportWriter struct {
	bytes.Buffer
	call, failAt, partial int
	cause                 error
}

func (w *peakReportWriter) Write(p []byte) (int, error) {
	w.call++
	if w.call == w.failAt {
		if w.partial != 0 {
			if len(p) == 1 {
				return 0, nil
			}
			return w.Buffer.Write(p[:1])
		}
		return 0, w.cause
	}
	return w.Buffer.Write(p)
}

func TestProcessDisplayMaxUsageSharedPeakAndInsertionPrefixes(t *testing.T) {
	b := arena.NewBudget(2 << 20)
	p := NewProcessAccounting()
	first, e := NewRuntime(RuntimeConfig{Budget: b, Process: p})
	memoryRuntimeRequire(t, e)
	second, e := NewRuntime(RuntimeConfig{Budget: b, Process: p})
	memoryRuntimeRequire(t, e)
	a, e := first.Allocate((1 << 18) - 8)
	memoryRuntimeRequire(t, e)
	c, e := second.Allocate((3 << 18) - 8)
	memoryRuntimeRequire(t, e)
	if peak := p.MaximumUsed(); peak != 1<<20 {
		t.Fatal("shared peak bytes", peak)
	}
	memoryRuntimeRequire(t, first.Deallocate(&a))
	memoryRuntimeRequire(t, second.Deallocate(&c))
	if p.AllocationSize() != 0 || p.MaximumUsed() != 1<<20 {
		t.Fatal("peak lost after release", p.AllocationSize(), p.MaximumUsed())
	}
	parts := []string{"max. heap size:     ", "1", " Mbytes\n"}
	var full bytes.Buffer
	memoryRuntimeRequire(t, p.DisplayMaxUsage(&full))
	if got, want := full.String(), "max. heap size:     1 Mbytes\n"; got != want {
		t.Fatal(got, want)
	}
	failure := errors.New("peak output refused")
	for partial := 0; partial <= 1; partial++ {
		for at := 1; at <= len(parts); at++ {
			out := &peakReportWriter{failAt: at, partial: partial, cause: failure}
			e := p.DisplayMaxUsage(out)
			want := failure
			if partial != 0 {
				want = io.ErrShortWrite
			}
			prefix := ""
			for _, part := range parts[:at-1] {
				prefix += part
			}
			if partial != 0 && len(parts[at-1]) > 1 {
				prefix += parts[at-1][:1]
			}
			if e != want || out.String() != prefix || out.call != at {
				t.Fatal("source insertion prefix", partial, at, e, out.String(), prefix, out.call)
			}
		}
	}
	if e := (*ProcessAccounting)(nil).DisplayMaxUsage(&full); e != ErrRuntimeOwner {
		t.Fatal("nil process owner", e)
	}
	if e := p.DisplayMaxUsage(nil); !errors.Is(e, arena.ErrConfig) {
		t.Fatal("nil output", e)
	}
	memoryRuntimeRequire(t, second.Close())
	memoryRuntimeRequire(t, first.Close())
	if b.Used() != 0 {
		t.Fatal("retained raw memory budget", b.Used())
	}
}
