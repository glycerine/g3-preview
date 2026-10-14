// Source: src/ast/ast_smt2_pp.cpp:235-268.
package ast

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTSMT2FloatRealLiteralUsesActualNumeralTextAndFormat(t *testing.T) {
	m := astTestManager(t)
	defer func() { astTestMust(t, m.Close()) }()
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	for _, tc := range []struct {
		sig  int32
		exp  int64
		sign bool
		want string
	}{
		{0, -127, false, "(_ +zero 8 4)"}, {0, 128, true, "(_ -oo 8 4)"},
		{1, 128, false, "(_ NaN 8 4)"},
		{4, 3, false, "((_ to_fp 8 4) RTZ 1.5 3)"},
	} {
		sig := unumeral.NewInt(tc.sig)
		var v unumeral.ArbitraryFloat
		astTestMust(t, p.numeric.SetComponents(&v, 8, 4, tc.sign, tc.exp, &sig))
		node, e := p.mkNumeral(&v, nil)
		astTestMust(t, e)
		got, e := m.ppFloatRealLiteral(m.external(node), nil)
		astTestMust(t, e)
		if p.literalFrames != nil || !p.literalPrimary.sourceDone || !p.literalPrimary.retired || !p.literalPrimary.valueConsumed {
			t.Fatal("float printer returned before its scoped MPF destructor", p.literalFrames, p.literalPrimary)
		}
		want, e := m.FormatString(tc.want)
		astTestMust(t, e)
		if got != want {
			t.Fatal("literal source result", tc, got, want)
		}
		astTestMust(t, p.numeric.Del(&v))
	}
}

// Actual returning transport at numerical allocation, not a source debugger
// callback. A bounded child turns an accidental held-lock reentry into failure.
func TestASTSMT2FloatRealLiteralActualCallbackRetainsParent(t *testing.T) {
	if os.Getenv("G3_FLOAT_REAL_CALLBACK_CHILD") != "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestASTSMT2FloatRealLiteralActualCallbackRetainsParent$", "-test.count=1")
		cmd.Env = append(os.Environ(), "G3_FLOAT_REAL_CALLBACK_CHILD=1")
		output, e := cmd.CombinedOutput()
		if e != nil {
			t.Fatalf("actual callback child: %v; %s", e, output)
		}
		return
	}
	b := arena.NewBudget(1 << 28)
	var m *ASTManager
	active, calls := false, 0
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if active {
			calls++
			if e := m.Close(); !errors.Is(e, ErrASTPending) {
				t.Fatal("own callback mutated parent", e)
			}
			if e := m.SourceFinalize(); !errors.Is(e, ErrASTPending) {
				t.Fatal("source callback mutated parent", e)
			}
			if m.floatingPointPlugin.literalFrames == nil || m.floatingPointPlugin.literalFrames.owner != m.floatingPointPlugin {
				t.Fatal("actual frame detached")
			}
		}
		return f
	}})
	astTestMust(t, e)
	q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	astTestMust(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
	astTestMust(t, e)
	m, e = NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	astTestMust(t, e)
	_, e = m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	sig := unumeral.NewInt(0)
	var v unumeral.ArbitraryFloat
	astTestMust(t, p.numeric.SetComponents(&v, 8, 120, false, 3, &sig))
	node, e := p.mkNumeral(&v, nil)
	astTestMust(t, e)
	pad := int64(99990) - r.State().LocalSize - 8
	if pad <= 0 {
		t.Fatal("actual TLS setup frontier", r.State())
	}
	padding, e := r.Allocate(uint64(pad))
	astTestMust(t, e)
	astTestMust(t, r.SetMaxSize(1))
	active = true
	_, call := m.ppFloatRealLiteral(m.external(node), nil)
	active = false
	if call == nil || calls == 0 {
		t.Fatal("numerical transport not reached", call, calls)
	}
	f := p.literalFrames
	var texts []*unumeral.ArbitraryFloatFormatText
	if f.text != nil {
		texts = append(texts, f.text)
	}
	astFloatingPointRawGraphSourceOwners(t, q, p, []umemory.Block{padding}, texts, nil)
	if f.bodyCause == nil || p.header == 0 || f.self != f || f != &p.literalPrimary {
		t.Fatal("actual stopped call history", f)
	}
	astTestMust(t, r.SetMaxSize(0))
	astTestMust(t, p.numeric.Del(&v))
	astTestMust(t, m.Close())
	astTestMust(t, sy.Close())
	astTestMust(t, q.Close())
	astTestMust(t, r.Deallocate(&padding))
	astTestMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("exact context end", b.Used())
	}
}

func TestASTSMT2FloatRealLiteralEveryActualReservationPrefix(t *testing.T) {
	completed := false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		m := astTestManager(t)
		_, e := m.RegisterArithmeticPlugin()
		astTestMust(t, e)
		_, e = m.RegisterBitVectorPlugin()
		astTestMust(t, e)
		_, e = m.RegisterFloatingPointPlugin()
		astTestMust(t, e)
		p := m.floatingPointPlugin
		sig := unumeral.NewInt(4)
		var v unumeral.ArbitraryFloat
		astTestMust(t, p.numeric.SetComponents(&v, 8, 4, true, 3, &sig))
		node, e := p.mkNumeral(&v, nil)
		astTestMust(t, e)
		base := m.BudgetField.Used() - m.Stats().AllocatedBytes - m.numerals.RuntimeOwner().Stats().TotalBytes
		m.BudgetField.FailAfter(ordinal)
		out, call := m.ppFloatRealLiteral(m.external(node), nil)
		var texts []*unumeral.ArbitraryFloatFormatText
		for f := p.literalFrames; f != nil; f = f.next {
			if f.text != nil {
				texts = append(texts, f.text)
			}
		}
		astFloatingPointFormatRawGraph(t, m.numerals, p, texts...)
		var captured *astFloatLiteralFrame
		var copyTuple unumeral.ArbitraryFloatValueCopyReceipt
		var bodyTuple, sourceTuple error
		if captured = p.literalFrames; captured != nil {
			copyTuple, bodyTuple, sourceTuple = captured.copy, captured.bodyCause, captured.sourceCause
			if captured.text != nil && !captured.text.SourceEndedField && captured.sourceCause != nil {
				if captured.buffer.Len() == 0 || captured.valueConsumed {
					t.Fatal("younger formatting source stop consumed older lexical members", ordinal, captured)
				}
			}
		}
		if m.BudgetField.Used() != base+m.Stats().AllocatedBytes+m.numerals.RuntimeOwner().Stats().TotalBytes {
			t.Fatal("before-lift actual printer cold graph", ordinal, call, m.BudgetField.Used(), base+m.Stats().AllocatedBytes+m.numerals.RuntimeOwner().Stats().TotalBytes)
		}
		if call == nil {
			completed = true
			if out.IsNull() {
				t.Fatal("actual format publication")
			}
		}
		m.BudgetField.FailAfter(-1)
		astTestMust(t, p.numeric.Del(&v))
		astTestMust(t, m.Close())
		if captured != nil && (captured.copy != copyTuple || captured.bodyCause != bodyTuple || captured.sourceCause != sourceTuple || captured.header != 0 || !captured.ownDone) {
			t.Fatal("actual caller history changed at owning context end", ordinal, captured)
		}
		if completed {
			t.Log("actual first successful printer prefix", ordinal)
			break
		}
	}
	if !completed {
		t.Fatal("actual printer acquisition horizon")
	}
}

func TestASTSMT2FloatRealLiteralWarmSmallNaNErasesFrameAllocation(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	var value unumeral.ArbitraryFloat
	var receipt unumeral.ArbitraryFloatValueCopyReceipt
	astTestMust(t, p.numeric.MkNaN(8, 4, &value, &receipt))
	node, e := p.mkNumeral(&value, nil)
	astTestMust(t, e)
	warm, e := m.ppFloatRealLiteral(m.external(node), nil)
	astTestMust(t, e)
	fm, e := m.GetFormatManager()
	astTestMust(t, e)
	astTestMust(t, fm.IncRef(warm))
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	again, call := m.ppFloatRealLiteral(m.external(node), nil)
	astFloatingPointConstructorRawGraph(t, m.numerals, p)
	if call != nil || again != warm || m.BudgetField.Used() != before {
		t.Fatal("literal inline/reused NaN branches must not reserve frame history", call, again, warm, m.BudgetField.Used(), before)
	}
	m.BudgetField.FailAfter(-1)
	astTestMust(t, fm.DecRef(warm))
	astTestMust(t, p.numeric.Del(&value))
	astTestMust(t, m.Close())
}
