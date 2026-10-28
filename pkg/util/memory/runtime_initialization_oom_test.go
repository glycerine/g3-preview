package memory

// Source: src/util/memory_manager.cpp:90-113, especially the reached assignment at102.

import (
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Independent scalar source-prefix fixtures, not a generated mem_initialize driver.
func TestMemoryRuntimeInitializationClearAfterActualAllocationFailure(t *testing.T) {
	for _, tls := range []bool{false, true} {
		for _, bits := range []uint8{32, 64} {
			for _, physical := range []bool{false, true} {
				name := strconv.FormatBool(tls) + "/" + strconv.Itoa(int(bits)) + "/physical=" + strconv.FormatBool(physical)
				t.Run(name, func(t *testing.T) {
					m, b := memoryRuntimeFixture(t, tls, bits)
					old, e := m.Allocate(8)
					memoryRuntimeRequire(t, e)
					memoryRuntimeRequire(t, m.Write(old, 0, 71))
					memoryRuntimeRequire(t, m.SetMaxCount(100))
					memoryRuntimeRequire(t, m.SetWatermark(3))
					if physical {
						b.FailAfter(0)
					} else {
						memoryRuntimeRequire(t, m.SetMaxSize(1))
					}
					n := uint64(16)
					if tls && !physical {
						n = 100000
					}
					failed, e := m.Allocate(n)
					memoryRuntimeCode(t, e, uint32(101))
					if !m.State().OutOfMemory {
						t.Fatal("real failure did not reach OOM flag")
					}
					// Physical/cold roots are named before restoring the failure control.
					before, stats, used := m.State(), m.Stats(), b.Used()
					limits := [3]int64{m.accounting.maxSize, m.accounting.maxCount, m.accounting.watermark}
					if e = m.ClearOutOfMemoryForInitialization(); e != nil {
						t.Fatal(e)
					}
					expected := before
					expected.OutOfMemory = false
					if m.State() != expected || m.Stats() != stats || b.Used() != used || limits != ([3]int64{m.accounting.maxSize, m.accounting.maxCount, m.accounting.watermark}) {
						t.Fatal("initialization scalar changed other accounting/backing", m.State(), expected)
					}
					value, e := m.Read(old, 0)
					memoryRuntimeRequire(t, e)
					if value != 71 {
						t.Fatal("live bytes changed", value)
					}
					if failed != (Block{}) {
						size, sizeError := m.Size(failed)
						if sizeError != nil || size != n {
							t.Fatal("late failure root extent changed", size, sizeError)
						}
						if _, e = m.Read(failed, 0); !errors.Is(e, ErrRuntimeUninitialized) {
							t.Fatal("late allocation initialization state changed", e)
						}
					}
					memoryRuntimeRequire(t, m.ClearOutOfMemoryForInitialization())
					if m.State() != expected || m.Stats() != stats || b.Used() != used {
						t.Fatal("scalar reassignment changed graph")
					}
					// This is only cpp102: no first-initialize guard, child callbacks, max-size
					// update or fully-initialized publication is claimed by the leaf fixture.
					b.FailAfter(-1)
					if failed != (Block{}) {
						memoryRuntimeRequire(t, m.Deallocate(&failed))
					}
					memoryRuntimeRequire(t, m.Deallocate(&old))
				})
			}
		}
	}
}
func TestMemoryRuntimeInitializationClearReachedOwnerGuard(t *testing.T) {
	m, b := memoryRuntimeFixture(t, false, 64)
	b.FailAfter(0)
	_, e := m.Allocate(8)
	memoryRuntimeCode(t, e, uint32(101))
	before, stats, used := m.State(), m.Stats(), b.Used()
	shallow := new(Runtime)
	reflect.ValueOf(shallow).Elem().Set(reflect.ValueOf(m).Elem())
	if e = shallow.ClearOutOfMemoryForInitialization(); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatal("copied owner", e)
	}
	var missing *Runtime
	if e = missing.ClearOutOfMemoryForInitialization(); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatal("nil owner", e)
	}
	if m.State() != before || m.Stats() != stats || b.Used() != used {
		t.Fatal("foreign guard changed actual owner")
	}
	b.FailAfter(-1)
	memoryRuntimeRequire(t, m.Close())
	if e = m.ClearOutOfMemoryForInitialization(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("ended owner", e)
	}
}
