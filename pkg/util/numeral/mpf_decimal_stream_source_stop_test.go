package numeral

// Source: src/util/mpf.cpp:1578-1582; scoped q follows the younger display DT.

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	utext "github.com/glycerine/g3/pkg/util/text"
	"testing"
)

func TestArbitraryFloatDecimalStreamChildSourceStopProtectsScopedQ(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	sig := NewInt(1)
	var x ArbitraryFloat
	serviceMust(t, m.SetComponents(&x, 17, 4000, false, -160, &sig))
	var out utext.OwnedStringStream
	serviceMust(t, out.Construct(b))
	contextTLSNearMinimum(t, r)
	lane, e := r.Allocate(8)
	serviceMust(t, e)
	stream := decimalStopAtQuestionMark{inner: &out, runtime: r, lane: &lane, t: t}
	body := m.DisplayDecimalInto(&stream, &x, 0)
	s, child := &m.decimalStream, &m.rational.decimalStream
	if !errors.Is(body, arena.ErrBounds) || !stream.armed || s.sourceCause == nil || s.sourceCause != child.sourceCause || s.qCursor != 0 || !s.qLive || s.formatOwned {
		t.Fatal("younger decimal end-label DEL did not protect q", body, s.sourceCause, child.sourceCause, s.qCursor)
	}
	got, e := (utext.OwnedStringView{Owner: &out.Buffer.String, Generation: out.Buffer.String.Generation, Length: out.Buffer.Position}).String()
	serviceMust(t, e)
	if got != "0.?" {
		t.Fatal("completed decimal output missing", got)
	}
	before, q, used, cause := r.State(), s.q, b.Used(), s.sourceCause
	if e := m.finishDecimalStreamSource(); e != cause {
		t.Fatal("unsticky wrapper source", e)
	}
	if e := m.SourceFinalize(); e != cause {
		t.Fatal("older class source ignored younger stop", e)
	}
	if r.State() != before || s.q != q || b.Used() != used || !m.rationalReady || m.sourceEnded {
		t.Fatal("stopped source touched older owner")
	}
	integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{lane}, &x.significand, &s.q.Num, &s.q.Den, &child.values[0], &child.values[1], &child.values[2], &child.values[3], &m.rational.NTmp, &m.rationalConversion.a.significand, &m.rationalConversion.n, &m.rationalConversion.d)
	if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes+out.Buffer.String.StorageBytes() {
		t.Fatal("immediate wrapper source-stop graph", b.Used())
	}
	recovery, e := r.Allocate(200000)
	serviceMust(t, e)
	serviceMust(t, m.CleanupDecimalStream())
	if s.sourceCause != cause || s.qLive || s.qCursor != 2 || s.active || child.active {
		t.Fatal("owning join erased wrapper source history")
	}
	after, e := (utext.OwnedStringView{Owner: &out.Buffer.String, Generation: out.Buffer.String.Generation, Length: out.Buffer.Position}).String()
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
