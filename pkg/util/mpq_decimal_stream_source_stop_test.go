// Source: src/util/mpq.cpp:183-185; explicit successful-body end-label DEL.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

type decimalStopAtQuestionMark struct {
	inner   *OwnedStringStream
	runtime *Runtime
	lane    *Block
	t       *testing.T
	armed   bool
}

func (s *decimalStopAtQuestionMark) AppendInt32(v int32) error { return s.inner.AppendInt32(v) }
func (s *decimalStopAtQuestionMark) AppendNumeralText(v string) error {
	return s.inner.AppendNumeralText(v)
}
func (s *decimalStopAtQuestionMark) AppendCString(v string) error {
	if e := s.inner.AppendCString(v); e != nil {
		return e
	}
	if v == "?" && !s.armed {
		s.armed = true
		delta := int64(-99950) - s.runtime.State().LocalSize
		if e := s.runtime.Reallocate(s.lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
			s.t.Fatal("actual failed TLS reallocation", e)
		}
	}
	return nil
}

func TestRationalDecimalStreamReachedEndLabelSourceStop(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	a := Rational{Num: NewInt(1), Den: NewInt(1)}
	serviceMust(t, m.integer.Mul2kInPlace(&a.Den, 4000))
	var out OwnedStringStream
	serviceMust(t, out.Construct(b))
	contextTLSNearMinimum(t, r)
	lane, e := r.Allocate(8)
	serviceMust(t, e)
	stream := decimalStopAtQuestionMark{inner: &out, runtime: r, lane: &lane, t: t}
	body := m.rational.DisplayDecimalInto(&stream, &a, 0, false)
	s := &m.rational.decimalStream
	if !errors.Is(body, arena.ErrBounds) || !stream.armed || !s.bodySucceeded || s.sourceCause == nil || s.sourceCursor != 2 || s.ownCursor != 0 {
		t.Fatal("actual d1 end-label source stop missing", body, s.sourceCursor)
	}
	got, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
	serviceMust(t, e)
	if got != "0.?" {
		t.Fatal("completed source output missing", got)
	}
	before, used, values, cause := r.State(), b.Used(), s.values, s.sourceCause
	if e := m.rational.finishDecimalStreamSource(); e != cause {
		t.Fatal("unsticky decimal end label", e)
	}
	if e := m.rational.SourceFinalize(); e != cause {
		t.Fatal("older class source ignored end-label stop", e)
	}
	if r.State() != before || b.Used() != used || s.values != values || m.rational.SourceFinalized {
		t.Fatal("source stop changed unread older owner")
	}
	integerPhysicalAuditWithRaw(t, m.integer, []Block{lane}, &a.Num, &a.Den, &s.values[0], &s.values[1], &s.values[2], &s.values[3])
	if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes+out.Buffer.String.StorageBytes() {
		t.Fatal("immediate decimal end-label graph", b.Used())
	}
	recovery, e := r.Allocate(200000)
	serviceMust(t, e)
	serviceMust(t, m.rational.CleanupDecimalStream())
	if s.sourceCause != cause || s.sourceCursor != 2 || s.ownCursor != 4 || s.active {
		t.Fatal("owning cleanup erased or replayed source end label")
	}
	after, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
	serviceMust(t, e)
	if after != got {
		t.Fatal("owning cleanup changed borrowed output")
	}
	out.Buffer.OwnAbort()
	contextTLSFinish(t, r, func() error { return m.integer.CleanupRawNumeral(&a.Num) })
	contextTLSFinish(t, r, func() error { return m.integer.CleanupRawNumeral(&a.Den) })
	contextTLSFinish(t, r, m.Close)
	for _, v := range []*Block{&lane, &recovery} {
		contextTLSFinish(t, r, func() error {
			if *v == (Block{}) {
				return nil
			}
			return r.Deallocate(v)
		})
	}
	contextTLSFinish(t, r, r.Close)
	if b.Used() != 0 {
		t.Fatal("end-label stop refunds", b.Used())
	}
}
