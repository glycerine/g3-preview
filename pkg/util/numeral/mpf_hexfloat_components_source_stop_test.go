package numeral

// Source: src/util/mpf.cpp:1608-1619; younger to_string NUM destruction.

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

type hexfloatStopText struct {
	*ArbitraryFloatFormatText
	hook func()
}

func (t *hexfloatStopText) AppendStreamString(s ArbitraryFloatFormatStream) error {
	if e := t.ArbitraryFloatFormatText.AppendStreamString(s); e != nil {
		return e
	}
	if t.hook != nil {
		hook := t.hook
		t.hook = nil
		hook()
	}
	return nil
}

func TestArbitraryFloatHexfloatComponentsYoungerSourceStop(t *testing.T) {
	for _, callerFirst := range []bool{false, true} {
		budget := arena.NewBudget(1 << 28)
		r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
		serviceMust(t, e)
		m, e := NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: budget, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
		serviceMust(t, e)
		input := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 259), big.NewInt(1))
		sig := NewInt(0)
		serviceMust(t, m.integer.SetString(&sig, input.String()))
		warm, e := NewArbitraryFloatFormatText(budget)
		serviceMust(t, e)
		_, e = m.ToStringHexfloatComponentsInto(true, 0, &sig, 8, 260, 0, warm)
		serviceMust(t, e)
		serviceMust(t, warm.Close())
		text, e := NewArbitraryFloatFormatText(budget)
		serviceMust(t, e)
		lane, e := r.Allocate(8)
		serviceMust(t, e)
		called := false
		var beforeQ ArbitraryFloat
		var beforeSig Int
		proxy := &hexfloatStopText{ArbitraryFloatFormatText: text, hook: func() {
			called = true
			s := &m.hexfloatComponents
			if !s.executing || !s.qLive || !s.sigLive || s.format != &text.Numeric || s.formatInvocation != text.Numeric.invocation || !text.Numeric.executing {
				t.Fatal("actual younger formatter missing")
			}
			beforeQ, beforeSig = s.q, s.sig
			contextTLSNearMinimum(t, r)
			delta := int64(-99950) - r.State().LocalSize
			if e := r.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
				t.Fatal("public TLS realloc", e)
			}
		}}
		_, body := m.ToStringHexfloatComponentsInto(true, 0, &sig, 8, 260, 0, proxy)
		s, f := &m.hexfloatComponents, &text.Numeric
		if !called || !errors.Is(body, arena.ErrBounds) || s.sourceCause == nil || s.sourceCause != f.sourceCause || !s.qLive || !s.sigLive || s.q != beforeQ || s.sig != beforeSig {
			t.Fatal("younger source stop did not protect parent", body, s.sourceCause, f.sourceCause)
		}
		state, used, cause := r.State(), budget.Used(), s.sourceCause
		if e := m.finishHexfloatComponentsSource(); e != cause {
			t.Fatal("unsticky component source", e)
		}
		if e := m.SourceFinalize(); e != cause {
			t.Fatal("class ignored format stop", e)
		}
		if r.State() != state || budget.Used() != used || s.q != beforeQ || s.sig != beforeSig || m.sourceEnded {
			t.Fatal("stopped source changed older values")
		}
		integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{lane}, &sig, &s.sig, &s.q.significand, &m.rational.NTmp)
		if budget.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes+text.StorageBytes() {
			t.Fatal("component source stop graph", budget.Used())
		}
		output, e := text.Result.View().String()
		serviceMust(t, e)
		recovery, e := r.Allocate(200000)
		serviceMust(t, e)
		if callerFirst {
			contextTLSFinish(t, r, text.Close)
		}
		contextTLSFinish(t, r, m.CleanupHexfloatComponents)
		if s.sourceCause != cause || s.qLive || s.sigLive || s.active || !s.formatOwned {
			t.Fatal("component owning join erased source history")
		}
		if !callerFirst {
			after, e := text.Result.View().String()
			serviceMust(t, e)
			if after != output {
				t.Fatal("parent cleanup changed borrowed result")
			}
			contextTLSFinish(t, r, text.Close)
		}
		contextTLSFinish(t, r, func() error { return m.integer.CleanupRawNumeral(&sig) })
		contextTLSFinish(t, r, m.Close)
		for _, block := range []*umemory.Block{&lane, &recovery} {
			contextTLSFinish(t, r, func() error {
				if *block == (umemory.Block{}) {
					return nil
				}
				return r.Deallocate(block)
			})
		}
		contextTLSFinish(t, r, r.Close)
		if budget.Used() != 0 {
			t.Fatal("component source stop refunds", budget.Used())
		}
	}
}

func TestArbitraryFloatHexfloatComponentsParentDeleteClearsView(t *testing.T) {
	m, _, budget := mpfNumeralHelpersFixture(t)
	var sig Int
	input := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 259), new(big.Int).Lsh(big.NewInt(1), 200))
	input.Add(input, big.NewInt(1))
	serviceMust(t, m.integer.SetString(&sig, input.String()))
	text, e := NewArbitraryFloatFormatText(budget)
	serviceMust(t, e)
	var cell NumericCellRef
	proxy := &hexfloatStopText{ArbitraryFloatFormatText: text, hook: func() {
		s := &m.hexfloatComponents
		if !s.sigLive || s.sig.Cell == 0 {
			return
		}
		cell = s.sig.Cell
		s.sig.Cell = ^NumericCellRef(0)
	}}
	view, body := m.ToStringHexfloatComponentsInto(true, 0, &sig, 8, 260, 0, proxy)
	s := &m.hexfloatComponents
	if cell == 0 {
		serviceMust(t, text.Close())
		serviceMust(t, m.integer.Del(&sig))
		t.Fatal("parent scoped integer missing after child text body", body, *s)
	}
	if !errors.Is(body, arena.ErrInvalidSlot) || view.view != nil || !s.active || !s.sigLive || s.sourceCause == nil || s.format == nil || !s.formatOwned {
		t.Fatal("failed parent scoped_mpz published a text view", body, view, *s)
	}
	if e := m.SourceFinalize(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("source finalization lost parent destructor failure", e)
	}
	s.sig.Cell = cell
	serviceMust(t, m.CleanupHexfloatComponents())
	serviceMust(t, text.Close())
	serviceMust(t, m.integer.Del(&sig))
}
