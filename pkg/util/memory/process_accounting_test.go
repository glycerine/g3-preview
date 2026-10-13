package memory

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func processRuntimeFixture(t *testing.T, p *ProcessAccounting, tls bool) *Runtime {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	r, err := NewRuntime(RuntimeConfig{Budget: b, Process: p, ThreadLocal: tls, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
		if b.Used() != 0 {
			t.Error("retained budget", b.Used())
		}
	})
	return r
}

func TestProcessAccountingSharesSourceGlobalCounterAndLimit(t *testing.T) {
	p := NewProcessAccounting()
	a := processRuntimeFixture(t, p, false)
	b := processRuntimeFixture(t, p, false)
	x, err := a.Allocate(24)
	memoryRuntimeRequire(t, err)
	y, err := b.Allocate(8)
	memoryRuntimeRequire(t, err)
	if p.AllocationSize() != 48 || a.State().GlobalSize != 48 || b.State().GlobalSize != 48 {
		t.Fatal("shared size", p.AllocationSize(), a.State(), b.State())
	}
	memoryRuntimeRequire(t, b.SetMaxSize(48))
	failed, err := a.Allocate(1)
	memoryRuntimeCode(t, err, 101)
	if failed != (Block{}) || p.AllocationSize() != 57 || !p.OutOfMemory() || !b.State().OutOfMemory {
		t.Fatal("cross-runtime limit", failed, p.AllocationSize(), b.State())
	}
	memoryRuntimeRequire(t, a.Deallocate(&x))
	memoryRuntimeRequire(t, b.Deallocate(&y))
	if p.AllocationSize() != 9 {
		t.Fatal("source retains failed non-TLS allocation charge", p.AllocationSize())
	}
}

func TestProcessAccountingPreservesSeparateTLSLanes(t *testing.T) {
	p := NewProcessAccounting()
	a := processRuntimeFixture(t, p, true)
	b := processRuntimeFixture(t, p, true)
	x, err := a.Allocate(99992)
	memoryRuntimeRequire(t, err)
	y, err := b.Allocate(8)
	memoryRuntimeRequire(t, err)
	if p.AllocationSize() != 0 || a.State().LocalSize != 100000 || b.State().LocalSize != 16 {
		t.Fatal("unsynchronized lanes", p.AllocationSize(), a.State(), b.State())
	}
	z, err := a.Allocate(0)
	memoryRuntimeRequire(t, err)
	if p.AllocationSize() != 100008 || b.State().GlobalSize != 100008 || b.State().LocalSize != 16 {
		t.Fatal("shared synchronized size", p.AllocationSize(), b.State())
	}
	memoryRuntimeRequire(t, a.Deallocate(&x))
	memoryRuntimeRequire(t, a.Deallocate(&z))
	memoryRuntimeRequire(t, b.Deallocate(&y))
	if p.AllocationSize() != 0 {
		t.Fatal("released global size", p.AllocationSize())
	}
}

func TestProcessAccountingRejectsCopiedOwner(t *testing.T) {
	p := NewProcessAccounting()
	copy := *p
	_, err := NewRuntime(RuntimeConfig{Budget: arena.NewBudget(1024), Process: &copy})
	if err != arena.ErrConfig {
		t.Fatal(err)
	}
}
