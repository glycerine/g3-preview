// Source: src/util/mpz.cpp:1415-1423; literal negative scoped-numeral DT.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

// The real signed-host TLS guard is reached by deallocating the temporary,
// after all negative-branch insertions, before the older rational continuation.
type smt2StopBeforeTemporaryDT struct {
	inner   *OwnedStringStream
	runtime *Runtime
	lane    *Block
	t       *testing.T
	armed   bool
}

func (s *smt2StopBeforeTemporaryDT) AppendInt32(v int32) error { return s.inner.AppendInt32(v) }
func (s *smt2StopBeforeTemporaryDT) AppendNumeralCString(v NumeralCStringView) error {
	return s.inner.AppendNumeralCString(v)
}
func (s *smt2StopBeforeTemporaryDT) AppendCString(v string) error {
	if e := s.inner.AppendCString(v); e != nil {
		return e
	}
	if v == ")" && !s.armed {
		s.armed = true
		delta := int64(-99950) - s.runtime.State().LocalSize
		if e := s.runtime.Reallocate(s.lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
			s.t.Fatal("actual TLS failed reallocation", e)
		}
	}
	return nil
}

func TestRationalSMT2StreamNegativeSourceStopProtectsContinuation(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	a := Rational{Num: NewInt(-1), Den: NewInt(3)}
	serviceMust(t, m.integer.Mul2kInPlace(&a.Num, 4000))
	var out OwnedStringStream
	serviceMust(t, out.Construct(b))
	contextTLSNearMinimum(t, r)
	lane, e := r.Allocate(8)
	serviceMust(t, e)
	stream := smt2StopBeforeTemporaryDT{inner: &out, runtime: r, lane: &lane, t: t}
	body := m.rational.DisplaySMT2Into(&stream, &a, false)
	s, p := &m.integer.smt2Stream, &m.rational.smt2Stream
	if !errors.Is(body, arena.ErrBounds) || !stream.armed || s.sourceCause == nil || p.sourceCause != s.sourceCause || !s.tmpLive || p.childOwned {
		t.Fatal("negative DT source stop missing", body, s.sourceCause, p.sourceCause, s.tmpLive)
	}
	got, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
	serviceMust(t, e)
	if len(got) < 5 || got[len(got)-1] != ')' {
		t.Fatal("negative branch output missing", got)
	}
	state, used, old, child, cause := r.State(), b.Used(), a, p.child, s.sourceCause
	if e := m.rational.finishSMT2StreamSource(); e != cause {
		t.Fatal("unsticky parent source", e)
	}
	if e := m.rational.SourceFinalize(); e != cause {
		t.Fatal("older rational owner source ignored child", e)
	}
	if r.State() != state || b.Used() != used || a != old || p.child != child || m.rational.SourceFinalized {
		t.Fatal("stopped source changed older owner")
	}
	integerPhysicalAuditWithRaw(t, m.integer, []Block{lane}, &a.Num, &a.Den, &s.tmp)
	if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes+out.Buffer.String.StorageBytes() {
		t.Fatal("immediate source-stop graph", b.Used())
	}
	recovery, e := r.Allocate(200000)
	serviceMust(t, e)
	serviceMust(t, m.rational.CleanupSMT2Stream())
	if s.sourceCause != cause || p.sourceCause != cause || s.tmpLive || s.active || p.active {
		t.Fatal("owning join erased source history")
	}
	after, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
	serviceMust(t, e)
	if after != got {
		t.Fatal("owning join changed borrowed output")
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
		t.Fatal("source-stop refunds", b.Used())
	}
}
