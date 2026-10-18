package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

type rationalBorrowTraceProbe struct {
	t                      *testing.T
	source                 *RationalService
	armed, called, flushed bool
}

func (w *rationalBorrowTraceProbe) Flush() error {
	if w.armed && !w.flushed {
		w.flushed = true
		if !w.source.runtimeTransportExecuting() {
			w.t.Error("receiving trace flush did not protect the held source service lock")
		} else if _, e := w.source.Zero(); !errors.Is(e, arena.ErrInvalidSlot) {
			w.t.Error("trace flush entered source service", e)
		}
	}
	return nil
}

func (w *rationalBorrowTraceProbe) Write(p []byte) (int, error) {
	if !w.armed || w.called {
		return len(p), nil
	}
	w.called = true
	// Check before entering any service API: a missing callback guard would
	// otherwise deadlock this same goroutine on the already-held service lock.
	if !w.source.runtimeTransportExecuting() {
		w.t.Error("receiving chunk trace did not protect the held source service lock")
		return len(p), nil
	}
	for _, e := range []error{w.source.CloseChecked(), w.source.SourceFinalize()} {
		if !errors.Is(e, arena.ErrInvalidSlot) {
			w.t.Error("trace acquired source service", e)
		}
	}
	if _, e := w.source.Zero(); !errors.Is(e, arena.ErrInvalidSlot) {
		w.t.Error("trace entered source service", e)
	}
	return len(p), nil
}

func TestRationalNumericBorrowReceivingChunkTraceGuardsSourceLock(t *testing.T) {
	s, _ := serviceNew(t)
	w := rationalBorrowTraceProbe{t: t, source: s}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("memory")
	b := arena.NewBudget(1 << 24)
	z, e := NewIntegerManager(IntegerConfig{
		Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}},
		Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true, Windows: true}, Trace: tags,
	})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, z.CloseChecked())
		if b.Used() != 0 {
			t.Fatal("receiver charge", b.Used())
		}
	})
	values := make([]Int, 400)
	serviceMust(t, s.WithNumericBorrow(func(borrow *RationalNumericBorrow) error {
		return borrow.WithReceivingInteger(z, func() error {
			w.armed = true
			defer func() { w.armed = false }()
			for i := range values {
				if e := z.SetString(&values[i], "1267650600228229401496703205377"); e != nil {
					return e
				}
			}
			return nil
		})
	}))
	if w.called != tags.ShouldTrace("memory") || w.flushed != tags.ShouldTrace("memory") {
		t.Fatal("actual receiving chunk callbacks", w.called, w.flushed)
	}
	for i := range values {
		serviceMust(t, z.Del(&values[i]))
	}
	if s.runtimeTransportExecuting() {
		t.Fatal("completed borrow retained callback guard")
	}
	zero, e := s.Zero()
	serviceMust(t, e)
	serviceMust(t, s.Del(&zero))
}
