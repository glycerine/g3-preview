// Source: src/util/mpf.cpp:1584-1588; scoped q follows the younger display DT.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func TestArbitraryFloatSMT2StreamChildSourceStopProtectsScopedQ(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	sig := NewInt(1)
	var x ArbitraryFloat
	serviceMust(t, m.SetComponents(&x, 17, 4000, true, 160, &sig))
	var out OwnedStringStream
	serviceMust(t, out.Construct(b))
	contextTLSNearMinimum(t, r)
	lane, e := r.Allocate(8)
	serviceMust(t, e)
	stream := smt2StopBeforeTemporaryDT{inner: &out, runtime: r, lane: &lane, t: t}
	body := m.DisplaySMT2Into(&stream, &x, false)
	s, child := &m.smt2Stream, &m.integer.smt2Stream
	if !errors.Is(body, arena.ErrBounds) || !stream.armed || s.sourceCause == nil || s.sourceCause != child.sourceCause || s.qCursor != 0 || !s.qLive || s.formatOwned {
		t.Fatal("younger negative DT did not protect q", body, s.sourceCause, child.sourceCause, s.qCursor)
	}
	got, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
	serviceMust(t, e)
	if len(got) < 5 || got[len(got)-1] != ')' {
		t.Fatal("negative numerator output missing", got)
	}
	before, q, used, cause := r.State(), s.q, b.Used(), s.sourceCause
	if e := m.finishSMT2StreamSource(); e != cause {
		t.Fatal("unsticky wrapper source", e)
	}
	if e := m.SourceFinalize(); e != cause {
		t.Fatal("older class source ignored younger stop", e)
	}
	if r.State() != before || s.q != q || b.Used() != used || !m.rationalReady || m.sourceEnded {
		t.Fatal("stopped source touched older owner")
	}
	integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{lane}, &x.significand, &s.q.Num, &s.q.Den, &child.tmp, &m.rational.NTmp, &m.rationalConversion.a.significand, &m.rationalConversion.n, &m.rationalConversion.d)
	if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes+out.Buffer.String.StorageBytes() {
		t.Fatal("immediate wrapper source-stop graph", b.Used())
	}
	recovery, e := r.Allocate(200000)
	serviceMust(t, e)
	serviceMust(t, m.CleanupSMT2Stream())
	if s.sourceCause != cause || s.qLive || s.qCursor != 2 || s.active || child.active {
		t.Fatal("owning join erased wrapper source history")
	}
	after, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
	serviceMust(t, e)
	if after != got {
		t.Fatal("owning join changed borrowed output")
	}
	out.Buffer.OwnAbort()
	contextTLSFinish(t, r, func() error { return m.CleanupValue(&x) })
	contextTLSFinish(t, r, m.Close)
	for _, v := range []*umemory.Block{&lane, &recovery} {
		contextTLSFinish(t, r, func() error {
			if *v == (umemory.Block{}) {
				return nil
			}
			return r.Deallocate(v)
		})
	}
	contextTLSFinish(t, r, r.Close)
	if b.Used() != 0 {
		t.Fatal("wrapper source-stop refunds", b.Used())
	}
}
