//go:build g3checked && g3debug

// Source: src/util/mpf.cpp:1519-1569; src/util/mpq.cpp:156-185.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package g3

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/mpnum"
)

// This test-only proxy arms the supported signed-host TLS owner frontier at
// the reached integer-part insertion. Every text operation still uses the
// actual owned res/ss.str provider; no extra source NUM or allocation is added.
type mpfFormatStopText struct {
	*ArbitraryFloatFormatText
	hook func()
}
type mpfFormatStopStream struct {
	mpnum.ArbitraryFloatFormatStream
	hook func()
}

func (s *mpfFormatStopStream) AppendInt32(n int32) error {
	e := s.ArbitraryFloatFormatStream.AppendInt32(n)
	if e == nil && s.hook != nil {
		h := s.hook
		s.hook = nil
		h()
	}
	return e
}
func (t *mpfFormatStopText) NewArbitraryFloatFormatStream() (mpnum.ArbitraryFloatFormatStream, error) {
	s, e := t.ArbitraryFloatFormatText.NewArbitraryFloatFormatStream()
	if e != nil {
		return nil, e
	}
	return &mpfFormatStopStream{s, t.hook}, nil
}
func (t *mpfFormatStopText) AppendStreamString(s mpnum.ArbitraryFloatFormatStream) error {
	return t.ArbitraryFloatFormatText.AppendStreamString(s.(*mpfFormatStopStream).ArbitraryFloatFormatStream)
}

func TestMPFFormatTextYoungerNumeralStopProtectsNonemptyResult(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	astTestMust(t, e)
	config := mpnum.IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}
	q, e := mpnum.NewRationalService(mpnum.RationalServiceConfig{Integers: config})
	astTestMust(t, e)
	m, e := q.NewArbitraryFloatManager()
	astTestMust(t, e)
	var a mpnum.ArbitraryFloat
	n := mpnum.NewInt(1)
	astTestMust(t, m.SetComponents(&a, 8, 260, true, 0, &n))
	warm, e := NewArbitraryFloatFormatText(b)
	astTestMust(t, e)
	_, e = m.ToStringInto(&a, warm)
	astTestMust(t, e)
	astTestMust(t, warm.Close())
	external, e := rt.Allocate(200000)
	astTestMust(t, e)
	baseline := b.Used() - m.OwnedStorageBytes() - rt.Stats().TotalBytes
	text, e := NewArbitraryFloatFormatText(b)
	astTestMust(t, e)
	var release error
	proxy := &mpfFormatStopText{ArbitraryFloatFormatText: text, hook: func() {
		satHeapTLSNearMinimum(t, rt)
		release = rt.Deallocate(&external)
	}}
	_, body := m.ToStringInto(&a, proxy)
	if !errors.Is(release, arena.ErrBounds) || external != (memory.Block{}) {
		t.Fatal("actual accounting frontier", release)
	}
	cause := text.numeric.SourceCause()
	frame := reflect.ValueOf(&text.numeric).Elem()
	if !errors.Is(body, arena.ErrBounds) || cause == nil || frame.FieldByName("sourceCursor").Uint() != 1 {
		t.Fatal("actual younger q NUM DT", body, cause, frame.FieldByName("sourceCursor").Uint())
	}
	// Immediate physical and cold graph before fault recovery. The actual q
	// source release has happened; surviving NUM and owned res remain named.
	mpfFormatNumericRawGraph(t, q, m, nil, []*ArbitraryFloatFormatText{text}, &a)
	owned := text.header + uint64(len(text.result.external)+len(text.result.pending)+len(text.temporary.external)+len(text.temporary.pending))
	if text.stream.NumericTextStream != nil && !text.stream.closed {
		stats := text.stream.NumericTextStream.Stats()
		owned += stats.OwnerBytes + stats.ScratchBytes + stats.OutputOwnerBytes + stats.OutputOverflowBytes
	}
	if want := baseline + m.OwnedStorageBytes() + rt.Stats().TotalBytes + owned; b.Used() != want {
		t.Fatal("immediate actual numeric/result cold graph", b.Used(), want)
	}
	view, e := text.CString()
	astTestMust(t, e)
	ch, e := view.Byte(0)
	astTestMust(t, e)
	if ch != '-' || text.result.length != 1 {
		t.Fatal("nonempty older res", ch, text.result.length)
	}
	oldResult := text.result
	used, state := b.Used(), rt.State()
	if e = text.SourceFinalize(); e != cause {
		t.Fatal("older result source stop", e, cause)
	}
	if text.sourceEnded || !reflect.DeepEqual(text.result, oldResult) || b.Used() != used || rt.State() != state {
		t.Fatal("older result mutated across younger stop")
	}
	// The real text-owned format frame is younger than cached manager fields.
	managerBefore := *m
	if e = m.SourceFinalize(); e != cause {
		t.Fatal("manager crossed younger format source stop", e, cause)
	}
	if !reflect.DeepEqual(*m, managerBefore) || b.Used() != used || rt.State() != state {
		t.Fatal("manager source mutated across younger format stop")
	}
	if e = m.Close(); !errors.Is(e, mpnum.ErrPolynomialPending) {
		t.Fatal("manager crossed caller-owned format context", e)
	}
	if !reflect.DeepEqual(*m, managerBefore) || b.Used() != used || rt.State() != state {
		t.Fatal("manager own mutated caller-owned format context")
	}
	if got, err := view.Byte(0); err != nil || got != '-' {
		t.Fatal("manager invalidated borrowed result", got, err)
	}
	own := text.numeric.Cleanup(false)
	if !errors.Is(own, arena.ErrBounds) {
		t.Fatal("actual first own stop", own)
	}
	_, sourceHistory, ownHistory := m.FormatInvocationCauses(&text.numeric)
	if sourceHistory != cause || ownHistory != own {
		t.Fatal("first cause history", sourceHistory, ownHistory)
	}
	mpfFormatNumericRawGraph(t, q, m, nil, []*ArbitraryFloatFormatText{text}, &a)
	recovery, e := rt.Allocate(400000)
	astTestMust(t, e)
	finish := func(f func() error) {
		t.Helper()
		for {
			before := rt.Stats().LiveBlocks
			err := f()
			if err == nil {
				return
			}
			if !errors.Is(err, arena.ErrBounds) || rt.Stats().LiveBlocks >= before {
				t.Fatal("actual physical own progress", err, rt.State())
			}
		}
	}
	finish(func() error { return text.numeric.Cleanup(false) })
	finish(func() error { return text.numeric.Cleanup(true) })
	_, sourceHistory, ownHistory = m.FormatInvocationCauses(&text.numeric)
	if sourceHistory != cause || ownHistory != own || !errors.Is(body, cause) {
		t.Fatal("ACK changed immutable history", sourceHistory, ownHistory, body)
	}
	if e = text.SourceFinalize(); e != cause || text.sourceEnded || !reflect.DeepEqual(text.result, oldResult) {
		t.Fatal("ACK reinterpreted source stop", e)
	}
	finish(text.Close)
	if _, e = view.Byte(0); e == nil {
		t.Fatal("c_str survives actual owning end")
	}
	finish(func() error { return m.Del(&a) })
	finish(m.Close)
	finish(q.Close)
	if e = rt.Deallocate(&recovery); e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (memory.Block{})) {
		t.Fatal(e)
	}
	finish(rt.Close)
	if b.Used() != 0 {
		t.Fatal("owner graph retained", b.Used())
	}
}
