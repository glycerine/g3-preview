//go:build g3checked && g3debug

// Source: src/util/mpf.cpp:1519-1569; src/ast/fpa_decl_plugin.cpp.
package ast

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"math"
	"reflect"
	"testing"
)

// This test-only proxy arms the supported signed-host TLS owner frontier at
// the reached integer-part insertion. Every text operation still uses the
// actual owned res/ss.str provider; no extra source NUM or allocation is added.
type mpfFormatStopText struct {
	*util.ArbitraryFloatFormatText
	hook func()
}
type mpfFormatStopStream struct {
	util.ArbitraryFloatFormatStream
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
func (t *mpfFormatStopText) NewArbitraryFloatFormatStream() (util.ArbitraryFloatFormatStream, error) {
	s, e := t.ArbitraryFloatFormatText.NewArbitraryFloatFormatStream()
	if e != nil {
		return nil, e
	}
	return &mpfFormatStopStream{s, t.hook}, nil
}
func (t *mpfFormatStopText) AppendStreamString(s util.ArbitraryFloatFormatStream) error {
	return t.ArbitraryFloatFormatText.AppendStreamString(s.(*mpfFormatStopStream).ArbitraryFloatFormatStream)
}

func TestASTCheckpointMPFFormatTextYoungerNumeralStopProtectsNonemptyResult(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	astTestMust(t, e)
	config := util.IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}}
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: config})
	astTestMust(t, e)
	symbols, e := util.NewSymbolManager(util.SymbolConfig{Budget: b})
	astTestMust(t, e)
	parent, e := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: q, Budget: b})
	astTestMust(t, e)
	_, e = parent.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = parent.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = parent.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := parent.floatingPointPlugin
	m := p.numeric
	var a util.ArbitraryFloat
	n := util.NewInt(1)
	astTestMust(t, m.SetComponents(&a, 8, 260, true, 0, &n))
	warm, e := util.NewArbitraryFloatFormatText(b)
	astTestMust(t, e)
	_, e = m.ToStringInto(&a, warm)
	astTestMust(t, e)
	astTestMust(t, warm.Close())
	external, e := rt.Allocate(200000)
	astTestMust(t, e)
	baseline := b.Used() - parent.Stats().AllocatedBytes - rt.Stats().TotalBytes
	text, e := util.NewArbitraryFloatFormatText(b)
	astTestMust(t, e)
	var release error
	proxy := &mpfFormatStopText{ArbitraryFloatFormatText: text, hook: func() {
		astMpfFormatTLSNearMinimum(t, rt)
		release = rt.Deallocate(&external)
	}}
	_, body := m.ToStringInto(&a, proxy)
	if !errors.Is(release, arena.ErrBounds) || external != (util.Block{}) {
		t.Fatal("actual accounting frontier", release)
	}
	cause := text.Numeric.SourceCause()
	frame := reflect.ValueOf(&text.Numeric).Elem()
	if !errors.Is(body, arena.ErrBounds) || cause == nil || frame.FieldByName("sourceCursor").Uint() != 1 {
		t.Fatal("actual younger q NUM DT", body, cause, frame.FieldByName("sourceCursor").Uint())
	}
	// Immediate physical and cold graph before fault recovery. The actual q
	// source release has happened; surviving NUM and owned res remain named.
	astFloatingPointFormatRawGraph(t, q, p, text)
	owned := text.Header + uint64(text.Result.StorageBytes()+text.Temporary.StorageBytes())
	if text.Stream.NumericTextStream != nil && !text.Stream.ClosedField {
		stats := text.Stream.NumericTextStream.Stats()
		owned += stats.OwnerBytes + stats.ScratchBytes + stats.OutputOwnerBytes + stats.OutputOverflowBytes
	}
	if want := baseline + parent.Stats().AllocatedBytes + rt.Stats().TotalBytes + owned; b.Used() != want {
		t.Fatal("immediate actual numeric/result cold graph", b.Used(), want)
	}
	view, e := text.CString()
	astTestMust(t, e)
	ch, e := view.Byte(0)
	astTestMust(t, e)
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
	own := text.Numeric.Cleanup(false)
	if !errors.Is(own, arena.ErrBounds) {
		t.Fatal("actual first own stop", own)
	}
	_, sourceHistory, ownHistory := m.FormatInvocationCauses(&text.Numeric)
	if sourceHistory != cause || ownHistory != own {
		t.Fatal("first cause history", sourceHistory, ownHistory)
	}
	astFloatingPointFormatRawGraph(t, q, p, text)
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
	finish(parent.Close)
	finish(q.Close)
	finish(symbols.Close)
	if e = rt.Deallocate(&recovery); e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (util.Block{})) {
		t.Fatal(e)
	}
	finish(rt.Close)
	if b.Used() != 0 {
		t.Fatal("owner graph retained", b.Used())
	}
}

func astMpfFormatTLSNearMinimum(t *testing.T, r *util.Runtime) {
	t.Helper()
	first, e := r.Allocate(8)
	astTestMust(t, e)
	second, e := r.Allocate(8)
	astTestMust(t, e)
	delta := int64(-1000000)
	request := uint64(16) + uint64(int64(delta)) - 8
	if e = r.Reallocate(&first, request); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	astTestMust(t, r.Deallocate(&first))
	state := r.State()
	if state.LocalSize != 0 || state.GlobalSize >= 0 {
		t.Fatal("first real TLS synchronization", state)
	}
	target := int64(math.MinInt64) + 1000 - state.GlobalSize + 16
	request = uint64(target) + 16 - 8
	if e = r.Reallocate(&second, request); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	astTestMust(t, r.Deallocate(&second))
	if state = r.State(); state.GlobalSize != math.MinInt64+1000 || state.LocalSize != 0 {
		t.Fatal("public TLS fixture", state)
	}
}
