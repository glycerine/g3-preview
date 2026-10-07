//go:build g3checked && g3debug

// Source: src/util/mpf.cpp:1519-1569; src/util/mpq.cpp:156-185.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"reflect"
	"testing"
)

// This test-only proxy arms the supported signed-host TLS owner frontier at
// the reached integer-part insertion. Every text operation still uses the
// actual owned res/ss.str provider; no extra source NUM or allocation is added.
type mpfFormatStopText struct {
	*ArbitraryFloatFormatText
	hook func()
}
type mpfFormatStopStream struct {
	ArbitraryFloatFormatStream
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
func (t *mpfFormatStopText) NewArbitraryFloatFormatStream() (ArbitraryFloatFormatStream, error) {
	s, e := t.ArbitraryFloatFormatText.NewArbitraryFloatFormatStream()
	if e != nil {
		return nil, e
	}
	return &mpfFormatStopStream{s, t.hook}, nil
}
func (t *mpfFormatStopText) AppendStreamString(s ArbitraryFloatFormatStream) error {
	return t.ArbitraryFloatFormatText.AppendStreamString(s.(*mpfFormatStopStream).ArbitraryFloatFormatStream)
}

func TestMPFFormatTextYoungerNumeralStopProtectsNonemptyResult(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	rt, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	config := IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}}
	q, e := NewRationalService(RationalServiceConfig{Integers: config})
	serviceMust(t, e)
	m, e := q.NewArbitraryFloatManager()
	serviceMust(t, e)
	var a ArbitraryFloat
	n := NewInt(1)
	serviceMust(t, m.SetComponents(&a, 8, 260, true, 0, &n))
	warm, e := NewArbitraryFloatFormatText(b)
	serviceMust(t, e)
	_, e = m.ToStringInto(&a, warm)
	serviceMust(t, e)
	serviceMust(t, warm.Close())
	external, e := rt.Allocate(200000)
	serviceMust(t, e)
	baseline := b.Used() - m.OwnedStorageBytes() - rt.Stats().TotalBytes
	text, e := NewArbitraryFloatFormatText(b)
	serviceMust(t, e)
	var release error
	proxy := &mpfFormatStopText{ArbitraryFloatFormatText: text, hook: func() {
		contextTLSNearMinimum(t, rt)
		release = rt.Deallocate(&external)
	}}
	_, body := m.ToStringInto(&a, proxy)
	if !errors.Is(release, arena.ErrBounds) || external != (Block{}) {
		t.Fatal("actual accounting frontier", release)
	}
	cause := text.Numeric.SourceCause()
	frame := reflect.ValueOf(&text.Numeric).Elem()
	if !errors.Is(body, arena.ErrBounds) || cause == nil || frame.FieldByName("sourceCursor").Uint() != 1 {
		t.Fatal("actual younger q NUM DT", body, cause, frame.FieldByName("sourceCursor").Uint())
	}
	// Immediate physical and cold graph before fault recovery. The actual q
	// source release has happened; surviving NUM and owned res remain named.
	mpfFormatNumericRawGraph(t, q, m, nil, []*ArbitraryFloatFormatText{text}, &a)
	owned := text.Header + uint64(len(text.Result.External)+len(text.Result.Pending)+len(text.Temporary.External)+len(text.Temporary.Pending))
	if text.Stream.NumericTextStream != nil && !text.Stream.ClosedField {
		stats := text.Stream.NumericTextStream.Stats()
		owned += stats.OwnerBytes + stats.ScratchBytes + stats.OutputOwnerBytes + stats.OutputOverflowBytes
	}
	if want := baseline + m.OwnedStorageBytes() + rt.Stats().TotalBytes + owned; b.Used() != want {
		t.Fatal("immediate actual numeric/result cold graph", b.Used(), want)
	}
	view, e := text.CString()
	serviceMust(t, e)
	ch, e := view.Byte(0)
	serviceMust(t, e)
	if ch != '-' || text.Result.Length != 1 {
		t.Fatal("nonempty older res", ch, text.Result.Length)
	}
	oldResult := text.Result
	used, state := b.Used(), rt.State()
	if e = text.SourceFinalize(); e != cause {
		t.Fatal("older result source stop", e, cause)
	}
	if text.SourceEndedField || !reflect.DeepEqual(text.Result, oldResult) || b.Used() != used || rt.State() != state {
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
	if e = m.Close(); !errors.Is(e, ErrNumericPending) {
		t.Fatal("manager crossed caller-owned format context", e)
	}
	if !reflect.DeepEqual(*m, managerBefore) || b.Used() != used || rt.State() != state {
		t.Fatal("manager own mutated caller-owned format context")
	}
	if got, err := view.Byte(0); err != nil || got != '-' {
		t.Fatal("manager invalidated borrowed result", got, err)
	}
	own := text.Numeric.Cleanup(false)
	if !errors.Is(own, arena.ErrBounds) {
		t.Fatal("actual first own stop", own)
	}
	_, sourceHistory, ownHistory := m.FormatInvocationCauses(&text.Numeric)
	if sourceHistory != cause || ownHistory != own {
		t.Fatal("first cause history", sourceHistory, ownHistory)
	}
	mpfFormatNumericRawGraph(t, q, m, nil, []*ArbitraryFloatFormatText{text}, &a)
	recovery, e := rt.Allocate(400000)
	serviceMust(t, e)
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
	finish(func() error { return text.Numeric.Cleanup(false) })
	finish(func() error { return text.Numeric.Cleanup(true) })
	_, sourceHistory, ownHistory = m.FormatInvocationCauses(&text.Numeric)
	if sourceHistory != cause || ownHistory != own || !errors.Is(body, cause) {
		t.Fatal("ACK changed immutable history", sourceHistory, ownHistory, body)
	}
	if e = text.SourceFinalize(); e != cause || text.SourceEndedField || !reflect.DeepEqual(text.Result, oldResult) {
		t.Fatal("ACK reinterpreted source stop", e)
	}
	finish(text.Close)
	if _, e = view.Byte(0); e == nil {
		t.Fatal("c_str survives actual owning end")
	}
	finish(func() error { return m.Del(&a) })
	finish(m.Close)
	finish(q.Close)
	if e = rt.Deallocate(&recovery); e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (Block{})) {
		t.Fatal(e)
	}
	finish(rt.Close)
	if b.Used() != 0 {
		t.Fatal("owner graph retained", b.Used())
	}
}
