// Source: src/util/memory_manager.cpp:124-149,274-305,326-377;
// src/util/memory_manager.h:49-71.
package memory

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func runtimeSizeTFixture(t *testing.T, tls bool) *Runtime {
	t.Helper()
	b := arena.NewBudget(1 << 22)
	r, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 32, ThreadLocal: tls})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		b.FailAfter(-1)
		if e := r.Close(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Error("physical graph leak", b.Used())
		}
	})
	return r
}
func TestMemoryRuntimeActualSizeT32AllocateReallocateEntry(t *testing.T) {
	for _, tls := range []bool{false, true} {
		t.Run(map[bool]string{false: "global", true: "tls"}[tls], func(t *testing.T) {
			r := runtimeSizeTFixture(t, tls)
			h, e := r.Allocate(1 << 32)
			if e != nil {
				t.Fatal("source zero request still allocates SIZE_T header", e)
			}
			if r.Stats().PhysicalBytes != 4 {
				t.Fatal(r.Stats())
			}
			if e = r.Reallocate(&h, (1<<32)+3); e != nil {
				t.Fatal("realloc actual source entry", e)
			}
			if e = r.Write(h, 2, 91); e != nil {
				t.Fatal(e)
			}
			v, e := r.Read(h, 2)
			if e != nil || v != 91 {
				t.Fatal(v, e)
			}
			if r.Stats().PhysicalBytes != 7 || r.State().GlobalCount+r.State().LocalCount != 2 {
				t.Fatal(r.State(), r.Stats())
			}
			if e = r.Deallocate(&h); e != nil {
				t.Fatal(e)
			}
		})
	}
}
func TestMemoryRuntimeActualSizeT32LimitSetters(t *testing.T) {
	r := runtimeSizeTFixture(t, false)
	if e := r.SetMaxSize((1 << 32) + 1); e != nil {
		t.Fatal("source setter narrowing", e)
	}
	h, e := r.Allocate(0)
	var f Failure
	if !errors.As(e, &f) || f.Kind != FailureMemout || h != (Block{}) || r.State().GlobalSize != 4 || !r.State().OutOfMemory || r.Stats().LiveBlocks != 0 {
		t.Fatal("actual preallocation limit prefix", h, e, r.State())
	}
	if e = r.SetMaxSize(1 << 32); e != nil {
		t.Fatal(e)
	}
	if e = r.SetMaxCount((1 << 32) + 2); e != nil {
		t.Fatal(e)
	}
	h, e = r.Allocate(0)
	if e != nil {
		t.Fatal(e)
	}
	if e = r.SetWatermark((1 << 32) + 8); e != nil {
		t.Fatal(e)
	}
	above, e := r.AboveWatermark()
	if e != nil || above {
		t.Fatal("strict equality threshold", above, e, r.State())
	}
	if e = r.SetWatermark((1 << 32) + 7); e != nil {
		t.Fatal(e)
	}
	above, e = r.AboveWatermark()
	if e != nil || !above {
		t.Fatal(above, e)
	}
	v, e := r.Allocate(0)
	if !errors.As(e, &f) || f.Kind != FailureAllocExceeded || v != (Block{}) || r.State().GlobalCount != 3 || r.State().GlobalSize != 12 {
		t.Fatal("source count limit prefix", v, e, r.State())
	}
	if e = r.SetWatermark(1 << 32); e != nil {
		t.Fatal(e)
	}
	above, e = r.AboveWatermark()
	if e != nil || above {
		t.Fatal("zero disables watermark", above, e)
	}
	if e = r.Deallocate(&h); e != nil {
		t.Fatal(e)
	}
}

// Oversized caller arguments qualify a source size_t32 conversion profile.
// A wrapped zero physical extent is a source undefined-header domain, not a
// valid out-of-memory fixture. The typed adapter retains the actual block.
func TestMemoryRuntimeSizeT32WrappedHeaderlessNamedRoot(t *testing.T) {
	for _, tls := range []bool{false, true} {
		t.Run(map[bool]string{false: "global", true: "tls"}[tls], func(t *testing.T) {
			r := runtimeSizeTFixture(t, tls)
			h, e := r.Allocate((1 << 32) + uint64(^uint32(0)) - 3)
			if !errors.Is(e, arena.ErrBounds) || h == (Block{}) {
				t.Fatal("actual wrapped header access", h, e)
			}
			s := r.Stats()
			if s.LiveBlocks != 1 || s.PhysicalBytes != 0 || s.InitializationBytes != 0 || r.config.Budget.Used() != s.TotalBytes {
				t.Fatal("named headerless block graph", s)
			}
			st := r.State()
			wantCount := int64(1)
			if tls {
				wantCount = 0
			}
			if st.GlobalCount+st.LocalCount != wantCount || st.GlobalSize != 0 || st.LocalSize != 0 || st.OutOfMemory {
				t.Fatal("source header before TLS counter vs after global counter", st)
			}
			if _, e = r.Read(h, 0); !errors.Is(e, arena.ErrBounds) {
				t.Fatal("reached header remains guarded", e)
			}
			if e = r.CleanupPartial(&h); e != nil {
				t.Fatal(e)
			}
			if h != (Block{}) || r.State() != st || r.Stats().LiveBlocks != 0 || r.config.Budget.Used() != r.Stats().TotalBytes {
				t.Fatal("separate owning abort changed source counters", r.State(), st)
			}
		})
	}
}
func TestMemoryRuntimeSizeT32EveryActualPhysicalRefusal(t *testing.T) {
	for _, tls := range []bool{false, true} {
		t.Run(map[bool]string{false: "global", true: "tls"}[tls], func(t *testing.T) {
			failures, success := 0, 0
			for ordinal := int64(0); ordinal < 8; ordinal++ {
				t.Run(string(rune('A'+ordinal)), func(t *testing.T) {
					r := runtimeSizeTFixture(t, tls)
					r.config.Budget.FailAfter(ordinal)
					h, e := r.Allocate((1 << 32) + 3)
					s := r.Stats()
					if s.TotalBytes != r.config.Budget.Used() {
						t.Fatal("pre-faultlift physical graph", s, r.config.Budget.Used())
					}
					if e != nil {
						failures++
						var f Failure
						if !errors.As(e, &f) || f.Kind != FailureMemout || !r.State().OutOfMemory || h != (Block{}) || s.LiveBlocks != 0 {
							t.Fatal("real physical refusal", ordinal, h, e, r.State(), s)
						}
						want := int64(1)
						if tls {
							want = 0
						}
						if r.State().GlobalCount+r.State().LocalCount != want {
							t.Fatal("source allocation publication prefix", r.State())
						}
					} else {
						success++
						if h == (Block{}) || s.LiveBlocks != 1 || s.PhysicalBytes != 7 || s.InitializationBytes != 8 {
							t.Fatal("actual successful extent/root", h, s)
						}
					}
					r.config.Budget.FailAfter(-1)
					if h != (Block{}) {
						if e = r.Deallocate(&h); e != nil {
							t.Fatal(e)
						}
					}
				})
				if success != 0 {
					break
				}
			}
			if failures != 2 || success != 1 {
				t.Fatal("complete real reservation horizon", failures, success)
			}
		})
	}
}
func TestMemoryRuntimeSizeTEntryRetainsOwnerAndLP64Domains(t *testing.T) {
	r := runtimeSizeTFixture(t, false)
	copied := *r
	if e := copied.SetMaxSize(1 << 32); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatal(e)
	}
	if _, e := copied.Allocate(1 << 32); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatal(e)
	}
	if e := r.Close(); e != nil {
		t.Fatal(e)
	}
	if e := r.SetWatermark(1 << 32); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if _, e := r.Allocate(1 << 32); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	b := arena.NewBudget(1 << 22)
	wide, e := NewRuntime(RuntimeConfig{Budget: b, SizeTBits: 64})
	if e != nil {
		t.Fatal(e)
	}
	if e = wide.SetMaxSize(1 << 32); e != nil {
		t.Fatal(e)
	}
	if e = wide.SetMaxCount(1 << 32); e != nil {
		t.Fatal(e)
	}
	if e = wide.SetWatermark(1 << 32); e != nil {
		t.Fatal(e)
	}
	h, e := wide.Allocate(0)
	if e != nil {
		t.Fatal(e)
	}
	if above, e := wide.AboveWatermark(); e != nil || above {
		t.Fatal("LP64 unchanged", above, e)
	}
	if e = wide.Deallocate(&h); e != nil {
		t.Fatal(e)
	}
	if e = wide.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestMemoryRuntimeSizeT32EveryReallocatePhysicalRefusal(t *testing.T) {
	for _, tls := range []bool{false, true} {
		t.Run(map[bool]string{false: "global", true: "tls"}[tls], func(t *testing.T) {
			failures, success := 0, 0
			for ordinal := int64(0); ordinal < 4; ordinal++ {
				t.Run(string(rune('A'+ordinal)), func(t *testing.T) {
					r := runtimeSizeTFixture(t, tls)
					h, e := r.Allocate(1)
					if e != nil {
						t.Fatal(e)
					}
					if e = r.Write(h, 0, 137); e != nil {
						t.Fatal(e)
					}
					captured := h
					r.config.Budget.FailAfter(ordinal)
					e = r.Reallocate(&h, (1<<32)+9)
					if r.config.Budget.Used() != r.Stats().TotalBytes || r.Stats().LiveBlocks != 1 {
						t.Fatal("immediate named allocation graph", r.Stats())
					}
					st := r.State()
					wantGlobal, wantLocal := int64(13), int64(0)
					if tls {
						wantGlobal, wantLocal = 0, 13
					}
					if st.GlobalSize != wantGlobal || st.LocalSize != wantLocal || st.GlobalCount+st.LocalCount != 2 {
						t.Fatal("accounting before physical realloc prefix", st)
					}
					got, readError := r.Read(h, 0)
					if readError != nil || got != 137 {
						t.Fatal("actual copied byte", got, readError)
					}
					if e != nil {
						failures++
						var f Failure
						if !errors.As(e, &f) || f.Kind != FailureMemout || !st.OutOfMemory || h != captured || r.Stats().PhysicalBytes != 5 {
							t.Fatal("failed physical replacement retains exact old root", h, e, st)
						}
					} else {
						success++
						if h == captured || r.Stats().PhysicalBytes != 13 {
							t.Fatal("actual generation/extent replacement", h)
						}
						if _, e = r.Read(captured, 0); !errors.Is(e, ErrRuntimeOwner) {
							t.Fatal("captured old token must be stale", e)
						}
						if _, e = r.Read(h, 8); !errors.Is(e, ErrRuntimeUninitialized) {
							t.Fatal("new bytes were not source-initialized", e)
						}
					}
					r.config.Budget.FailAfter(-1)
					if e = r.Deallocate(&h); e != nil {
						t.Fatal(e)
					}
				})
				if success != 0 {
					break
				}
			}
			if failures != 1 || success != 1 {
				t.Fatal("actual replacement reservation horizon", failures, success)
			}
		})
	}
}

func TestMemoryRuntimeSizeT32TLSLimitAfterPhysicalPublication(t *testing.T) {
	r := runtimeSizeTFixture(t, true)
	if e := r.SetMaxSize((1 << 32) + 100004); e != nil {
		t.Fatal(e)
	}
	if e := r.SetMaxCount(1 << 32); e != nil {
		t.Fatal(e)
	}
	h, e := r.Allocate((1 << 32) + 100001)
	var f Failure
	if !errors.As(e, &f) || f.Kind != FailureMemout || h == (Block{}) {
		t.Fatal("actual TLS postphysical limit", h, e)
	}
	s := r.Stats()
	if s.LiveBlocks != 1 || s.PhysicalBytes != 100005 || r.config.Budget.Used() != s.TotalBytes {
		t.Fatal("named returned block immediate graph", s)
	}
	st := r.State()
	if st.GlobalSize != 100005 || st.LocalSize != 0 || st.GlobalCount != 1 || st.LocalCount != 1 || !st.OutOfMemory {
		t.Fatal("actual synchronize prefix", st)
	}
	if e = r.Write(h, 100000, 83); e != nil {
		t.Fatal("returned published block remains real", e)
	}
	if e = r.Deallocate(&h); e != nil {
		t.Fatal(e)
	}
	st = r.State()
	if st.GlobalSize != 0 || st.LocalSize != 0 || st.GlobalCount != 2 || st.LocalCount != 1 || r.Stats().LiveBlocks != 0 {
		t.Fatal("source cumulative count quirk after true free", st)
	}
}
