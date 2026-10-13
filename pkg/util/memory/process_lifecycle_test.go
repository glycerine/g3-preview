package memory

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestProcessSourceLifecycleGuards(t *testing.T) {
	p := NewProcessAccounting()
	r := processRuntimeFixture(t, p, false)
	if err := r.SetMaxSize(1); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Allocate(8); err == nil || !p.OutOfMemory() {
		t.Fatal("source allocation did not set sticky OOM", err)
	}
	first, err := r.BeginSourceInitialize(math.MaxUint32)
	if err != nil || !first || p.OutOfMemory() || p.accounting.maxSize != 1 || !p.initialized || p.fullyInitialized {
		t.Fatal("first initialization prefix", first, err, p)
	}
	if _, err := r.BeginSourceInitialize(40); !errors.Is(err, ErrProcessLifecyclePending) || p.accounting.maxSize != 1 {
		t.Fatal("pending initialization changed limit", err)
	}
	if _, err := p.BeginFinalize(); !errors.Is(err, ErrProcessLifecyclePending) {
		t.Fatal("finalization during generated initialization", err)
	}
	if err := r.CompleteSourceInitialize(); err != nil || !p.fullyInitialized {
		t.Fatal("generated initialization publication", err)
	}
	if err := r.CompleteSourceInitialize(); !errors.Is(err, ErrProcessLifecycleOrder) {
		t.Fatal("duplicate completion", err)
	}
	first, err = r.BeginSourceInitialize(40)
	if err != nil || first || p.accounting.maxSize != 40 {
		t.Fatal("repeated initialization updates limit", first, err, p.accounting.maxSize)
	}
	active, err := p.BeginFinalize()
	if err != nil || !active || !p.finalizing {
		t.Fatal("finalization prefix", active, err)
	}
	if _, err := r.BeginSourceInitialize(80); !errors.Is(err, ErrProcessLifecyclePending) || p.accounting.maxSize != 40 {
		t.Fatal("initialization during finalization", err)
	}
	if _, err := p.BeginFinalize(); !errors.Is(err, ErrProcessLifecyclePending) {
		t.Fatal("duplicate finalization", err)
	}
	if err := p.CompleteFinalize(); err != nil || p.initialized || p.finalizing || !p.fullyInitialized {
		t.Fatal("finalization completion", err, p)
	}
	active, err = p.BeginFinalize()
	if err != nil || active {
		t.Fatal("null finalization", active, err)
	}
	first, err = r.BeginSourceInitialize(80)
	if err != nil || !first || p.accounting.maxSize != 80 || !p.fullyInitialized {
		t.Fatal("reinitialize after finalization", first, err, p)
	}
	if err := r.CompleteSourceInitialize(); err != nil {
		t.Fatal(err)
	}
}

func TestProcessSourceLifecycleRejectsInvalidOwners(t *testing.T) {
	r, err := NewRuntime(RuntimeConfig{Budget: arena.NewBudget(1024)})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if _, err := r.BeginSourceInitialize(10); !errors.Is(err, ErrRuntimeOwner) {
		t.Fatal("standalone runtime", err)
	}
	if err := r.CompleteSourceInitialize(); !errors.Is(err, ErrRuntimeOwner) {
		t.Fatal("standalone runtime completion", err)
	}
	p := NewProcessAccounting()
	copy := *p
	if _, err := copy.BeginFinalize(); !errors.Is(err, ErrRuntimeOwner) {
		t.Fatal("copied process", err)
	}
	if err := copy.CompleteFinalize(); !errors.Is(err, ErrRuntimeOwner) {
		t.Fatal("copied process completion", err)
	}
	if err := p.CompleteFinalize(); !errors.Is(err, ErrProcessLifecycleOrder) {
		t.Fatal("completion without begin", err)
	}
}

func TestProcessSourceInitializeConvertsSizeTBeforeSentinel(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		p := NewProcessAccounting()
		r, err := NewRuntime(RuntimeConfig{Budget: arena.NewBudget(1024), Process: p, SizeTBits: bits})
		if err != nil {
			t.Fatal(bits, err)
		}
		if err = r.SetMaxSize(123); err != nil {
			t.Fatal(bits, err)
		}
		wideSentinel := uint64(1)<<32 | math.MaxUint32
		first, err := r.BeginSourceInitialize(wideSentinel)
		want := int64(wideSentinel)
		if bits == 32 {
			want = 123
		}
		if err != nil || !first || p.accounting.maxSize != want {
			t.Fatal("first size_t conversion", bits, first, err, p.accounting.maxSize, want)
		}
		if err = r.CompleteSourceInitialize(); err != nil {
			t.Fatal(bits, err)
		}
		first, err = r.BeginSourceInitialize(wideSentinel)
		if err != nil || first || p.accounting.maxSize != want {
			t.Fatal("repeated size_t conversion", bits, first, err, p.accounting.maxSize, want)
		}
		if bits == 32 {
			first, err = r.BeginSourceInitialize(uint64(1) << 32)
			if err != nil || first || p.accounting.maxSize != 0 {
				t.Fatal("wrapped non-sentinel", first, err, p.accounting.maxSize)
			}
		}
		if err = r.Close(); err != nil {
			t.Fatal(bits, err)
		}
	}
}
