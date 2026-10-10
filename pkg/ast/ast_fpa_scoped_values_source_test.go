// Source: src/util/scoped_numeral_vector.h:25-72; src/util/vector.h:59-100,423-435;
// src/ast/fpa_decl_plugin.cpp:51-65.
package ast

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTFloatingPointScopedValueNativeGrowthAndDefaults(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	v := &m.floatingPointPlugin.values
	before := m.BudgetField.Used()
	physical := m.numerals.RuntimeOwner().Stats()
	m.BudgetField.FailAfter(0)
	astTestMust(t, v.reserve(257))
	n, e := v.size()
	astTestMust(t, e)
	if n != 257 || m.BudgetField.Used() != before || m.numerals.RuntimeOwner().Stats() != physical {
		t.Fatal("Go storage charged numerical arena", n)
	}
	for i := uint32(0); i < n; i++ {
		a, e := v.at(i)
		astTestMust(t, e)
		if a != (util.ArbitraryFloat{}) {
			t.Fatal("default numeral", i, a)
		}
	}
	// Smaller reserve preserves values and size, matching source reserve usage.
	astTestMust(t, v.reserve(8))
	if len(v.values) != 257 {
		t.Fatal("reserve shrank values")
	}
	m.BudgetField.FailAfter(-1)
	astTestMust(t, m.SourceFinalize())
	if !v.sourceDone || v.values != nil || v.destructorIndex != 257 {
		t.Fatal("ordered member destruction", v)
	}
}

func TestASTFloatingPointScopedValueActualInlinePushPublication(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	v := &p.values
	sig := util.NewInt(-19)

	var source util.ArbitraryFloat
	astTestMust(t, p.numeric.SetComponents(&source, 8, 24, true, 7, &sig))
	astTestMust(t, v.push(&source))
	got, e := v.at(0)
	astTestMust(t, e)
	equal, e := p.numeric.EqCore(&source, &got)
	astTestMust(t, e)
	n, e := v.size()
	astTestMust(t, e)
	if !equal || n != 1 || len(v.values) != 1 {
		t.Fatal("actual default-before-SET publication", got, source, n, len(v.values))
	}
	// Native growth relocates scalar handles without copying their numerical
	// ownership or charging the numeric arena. The original copied row survives.
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	astTestMust(t, v.reserve(257))
	got, e = v.at(0)
	astTestMust(t, e)
	equal, e = p.numeric.EqCore(&source, &got)
	astTestMust(t, e)
	if !equal || m.BudgetField.Used() != before {
		t.Fatal("native growth lost owned value", got)
	}
	m.BudgetField.FailAfter(-1)
	astFloatingPointConstructorRawGraph(t, m.numerals, p)
	astTestMust(t, p.numeric.Del(&source))

	astTestMust(t, m.Close())
}

func TestASTFloatingPointScopedValueInlineErasesCopyAcquisition(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	v := &p.values
	astTestMust(t, v.reserve(8))
	// A prepared row is copied directly, without another vector growth action.
	var source util.ArbitraryFloat
	sig := util.NewInt(31)
	astTestMust(t, p.numeric.SetComponents(&source, 8, 24, false, 11, &sig))
	before := m.BudgetField.Used()
	beforeHistory := p.numeric.ValueCopyHistoryBytes()
	m.BudgetField.FailAfter(0)
	astTestMust(t, v.set(3, &source))
	if v.copyReceipt != (util.ArbitraryFloatValueCopyReceipt{}) || p.numeric.ValueCopyHistoryBytes() != beforeHistory || m.BudgetField.Used() != before {
		t.Fatal("inline actual SET acquired child/history", v.copyReceipt, m.BudgetField.Used(), before)
	}
	got, e := v.at(3)
	astTestMust(t, e)
	same, e := p.numeric.EqCore(&source, &got)
	astTestMust(t, e)
	if !same {
		t.Fatal("actual direct row publication", got, source)
	}
	astFloatingPointConstructorRawGraph(t, m.numerals, p)
	m.BudgetField.FailAfter(-1)
	astTestMust(t, m.Close())
}

// Source: src/util/scoped_numeral_vector.h:25-65; src/util/mpf.cpp:316-333.
func TestASTFloatingPointScopedValueTwoRetainedAllocationChildren(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64})
	astTestMust(t, e)
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	astTestMust(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
	astTestMust(t, e)
	m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	astTestMust(t, e)
	_, e = m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	p := m.floatingPointPlugin
	v := &p.values
	z, e := util.NewIntegerManager(util.IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: m.BudgetField}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: m.BudgetField}}})
	astTestMust(t, e)
	sig, two := util.NewInt(0), util.NewInt(2)
	astTestMust(t, z.Power(&two, 100, &sig))
	astTestMust(t, v.reserve(3))
	var initial util.ArbitraryFloatValueCopyReceipt
	// The existing foreign-owner input adapter constructs one owned numeral.
	// Transfer that numeral out of its temporary input row into the native slice.
	row, e := r.Allocate(32)
	astTestMust(t, e)
	for k := uint64(0); k < 6; k++ {
		astTestMust(t, r.WriteUint32(row, 4*k, 0))
	}
	astTestMust(t, p.numeric.SetComponentsToRuntimeFromOwnerWithReceipt(row, 0, 8, 120, false, 3, z, &sig, &initial))
	astTestMust(t, p.numeric.CleanupValueCopy(&initial, false))
	astTestMust(t, p.numeric.CleanupValueCopy(&initial, true))
	source, e := p.numeric.CaptureRuntimeValue(row, 0)
	astTestMust(t, e)
	v.values[0] = source
	astTestMust(t, r.Deallocate(&row))
	// Relocate a slice containing a real BIG numeral while all numeric arena
	// reservations are refused. Native growth must preserve its sole owner.
	beforeGrowth, physical := b.Used(), r.Stats()
	b.FailAfter(0)
	astTestMust(t, v.reserve(257))
	b.FailAfter(-1)
	relocated, e := v.at(0)
	astTestMust(t, e)
	same, e := p.numeric.EqCore(&source, &relocated)
	astTestMust(t, e)
	if !same || b.Used() != beforeGrowth || r.Stats() != physical {
		t.Fatal("native growth changed BIG numeral ownership")
	}
	baseCold := b.Used() - m.Stats().AllocatedBytes - r.Stats().TotalBytes
	astTestMust(t, r.SetMaxSize(1))
	first := v.set(1, &source)
	firstTuple := v.copyReceipt
	second := v.set(2, &source)
	secondTuple := v.copyReceipt
	if first == nil || second == nil || firstTuple == (util.ArbitraryFloatValueCopyReceipt{}) || secondTuple == firstTuple {
		t.Fatal("two actual distinct stopped copy children", first, second, firstTuple, secondTuple)
	}
	cause := v.bodyCause
	if b.Used() != baseCold+m.Stats().AllocatedBytes+r.Stats().TotalBytes {
		t.Fatal("before-lift actual parent cold graph")
	}
	astFloatingPointRawGraphSourceOwners(t, m.numerals, p, nil, nil, []astFloatBorrowedInteger{{z, &sig}})
	astTestMust(t, r.SetMaxSize(0))
	astTestMust(t, v.destroy(true))
	if !v.copyFalseVisited || !v.copyTrueAcknowledged || !v.sourceDone || v.bodyCause != cause || v.destructorIndex != 257 {
		t.Fatal("false child ACK, ascending row DT, true child ACK", v)
	}
	chain := reflect.ValueOf(&v.copyChain).Elem()
	for call := testFieldBySourceName(chain, "head"); !call.IsNil(); call = testFieldBySourceName(call.Elem(), "callerNext") {
		frame := call.Elem()
		if !testFieldBySourceName(frame, "callerFalseAck").Bool() || !testFieldBySourceName(frame, "callerTrueAck").Bool() || testFieldBySourceName(testFieldBySourceName(frame, "childFrame"), "active").Bool() {
			t.Fatal("exact older copy child left active after vector ACK")
		}
	}
	// The separately supplied significand owner is borrowed and survives.
	if sig.IsSmall() {
		t.Fatal("foreign source changed")
	}
	astFloatingPointRawGraphSourceOwners(t, m.numerals, p, nil, nil, []astFloatBorrowedInteger{{z, &sig}})
	astTestMust(t, z.Del(&sig))
	astTestMust(t, z.CloseChecked())
	astTestMust(t, m.Close())
	astTestMust(t, sy.Close())
	astTestMust(t, q.Close())
	astTestMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("actual enclosing owner end", b.Used())
	}
}
