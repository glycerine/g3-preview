// Source: src/util/scoped_numeral_vector.h:25-72; src/util/vector.h:59-100,423-435;
// src/ast/fpa_decl_plugin.cpp:51-65.
package ast

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTFloatingPointScopedValueActualDefaultReserveGeometry(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	v := &m.floatingPointPlugin.values
	astTestMust(t, v.reserve(8))
	n, e := v.size()
	astTestMust(t, e)
	capacity, e := v.raw.Capacity()
	astTestMust(t, e)
	extent, e := v.raw.Runtime.Size(v.raw.Block)
	astTestMust(t, e)
	if n != 8 || capacity != 8 || extent != 8+32*8 || v.initialized != 8 {
		t.Fatal("actual 32-byte source vector extent/prefix", n, capacity, extent, v.initialized)
	}
	for i := uint32(0); i < n; i++ {
		a, e := v.at(i)
		astTestMust(t, e)
		if a.EBits() != 0 || a.SBits() != 0 || a.Sign() || a.Exponent() != 0 {
			t.Fatal("actual ascending default MPF", i, a)
		}
		for k := uint64(0); k < 6; k++ {
			word, e := v.raw.Runtime.ReadUint32(v.raw.Block, 8+32*uint64(i)+4*k)
			astTestMust(t, e)
			if word != 0 {
				t.Fatal("default codec", i, k, word)
			}
		}
		if _, e := v.raw.Runtime.Read(v.raw.Block, 8+32*uint64(i)+24); e == nil {
			t.Fatal("unused source padding incorrectly initialized")
		}
	}
	raw := v.raw.Block
	astTestMust(t, m.SourceFinalize())
	if !v.sourceDone || v.raw.Block != (util.Block{}) || v.initialized != 0 {
		t.Fatal("actual scoped DT before older manager", v)
	}
	if _, e := v.raw.Runtime.Size(raw); e == nil {
		t.Fatal("source vector backing survived member DT")
	}
}

func TestASTFloatingPointScopedValueEveryActualReservePrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		m := astTestManager(t)
		_, e := m.RegisterArithmeticPlugin()
		astTestMust(t, e)
		_, e = m.RegisterBitVectorPlugin()
		astTestMust(t, e)
		_, e = m.RegisterFloatingPointPlugin()
		astTestMust(t, e)
		p := m.floatingPointPlugin
		v := &p.values
		before := m.BudgetField.Used() - m.Stats().AllocatedBytes - m.numerals.RuntimeOwner().Stats().TotalBytes
		m.BudgetField.FailAfter(ordinal)
		call := v.reserve(8)
		astFloatingPointConstructorRawGraph(t, m.numerals, p)
		if got := m.BudgetField.Used(); got != before+m.Stats().AllocatedBytes+m.numerals.RuntimeOwner().Stats().TotalBytes {
			t.Fatal("actual immediate reserve cold graph", ordinal, got, before, m.Stats())
		}
		if call != nil && v.bodyCause != call {
			t.Fatal("first actual vector body cause", ordinal, call, v.bodyCause)
		}
		if call == nil {
			success = true
			if v.initialized != 8 {
				t.Fatal("actual placement prefix", v.initialized)
			}
		}
		m.BudgetField.FailAfter(-1)
		astTestMust(t, m.Close())
		if success {
			t.Log("first successful actual vector growth prefix", ordinal)
			break
		}
	}
	if !success {
		t.Fatal("no reached reserve success")
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
	if !equal || n != 1 || v.initialized != 1 {
		t.Fatal("actual default-before-SET publication", got, source, n, v.initialized)
	}
	astFloatingPointConstructorRawGraph(t, m.numerals, p)
	astTestMust(t, p.numeric.Del(&source))

	astTestMust(t, m.Close())
}

func TestASTFloatingPointScopedValueActualNullGrowthRetryKeepsOlderOutcome(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	astTestMust(t, e)
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: r, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	astTestMust(t, e)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
	astTestMust(t, e)
	m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	astTestMust(t, e)
	defer func() {
		astTestMust(t, m.Close())
		astTestMust(t, sy.Close())
		astTestMust(t, q.Close())
		astTestMust(t, r.Close())
		if b.Used() != 0 {
			t.Error("real owner context end", b.Used())
		}
	}()
	_, e = m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	_, e = m.RegisterFloatingPointPlugin()
	astTestMust(t, e)
	v := &m.floatingPointPlugin.values
	padding, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	astTestMust(t, e)
	defer func() { astTestMust(t, r.Deallocate(&padding)) }()
	astTestMust(t, r.SetMaxSize(1))
	call := v.reserve(1)
	if call == nil || v.raw.Pending == (util.Block{}) || v.raw.Block != (util.Block{}) {
		t.Fatal("real postphysical unjoined outcome", call, v.raw)
	}
	captured := v.raw.Pending
	cause := v.bodyCause
	astFloatingPointRawGraphWithBlocks(t, q, m.floatingPointPlugin, []util.Block{padding})
	astTestMust(t, r.SetMaxSize(0))
	beforeCold := b.Used()
	beforeRuntime := r.Stats()
	b.FailAfter(0)
	refusal := v.reserve(1)
	if refusal == nil || v.raw.Pending != captured || v.history != nil || v.bodyCause != cause || b.Used() != beforeCold || r.Stats() != beforeRuntime {
		t.Fatal("actual archive reservation preserves exact older token", refusal, v.raw, v.history)
	}
	astFloatingPointRawGraphWithBlocks(t, q, m.floatingPointPlugin, []util.Block{padding})
	b.FailAfter(-1)
	// memory::allocate never consults the sticky global OOM flag; this NULL
	// source retry is defined and must not overwrite its old acquired token.
	retry := v.reserve(1)
	if retry != nil {
		t.Errorf("source-valid NULL growth retry: %v", retry)
		return
	}
	if v.raw.Block == (util.Block{}) || v.raw.Block == captured || v.bodyCause != cause {
		t.Fatal("retry source pointer/history", v.raw, cause, v.bodyCause)
	}
	astFloatingPointRawGraphWithBlocks(t, q, m.floatingPointPlugin, []util.Block{padding})
	a := v.history
	if a == nil || a.original != captured || a.active != captured || a.bodyCause != cause || a.acknowledged || a.header == 0 {
		t.Fatal("actual unjoined acquisition tuple", a)
	}
	astTestMust(t, m.SourceFinalize())
	if !v.sourceDone || v.raw.Block != (util.Block{}) || a.active != captured || a.acknowledged || a.header == 0 || v.bodyCause != cause {
		t.Fatal("source DT protects own-only old acquisition", v, a)
	}
	astFloatingPointRawGraphWithBlocks(t, q, m.floatingPointPlugin, []util.Block{padding})
	astTestMust(t, m.Close())
	if !a.acknowledged || a.active != (util.Block{}) || a.original != captured || a.bodyCause != cause || a.header != 0 {
		t.Fatal("exact ACK then enclosing owner end", a)
	}
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
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64})
	astTestMust(t, e)
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: r, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
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
	z, e := util.NewIntegerManager(util.IntegerConfig{Memory: r, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: m.BudgetField}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: m.BudgetField}}})
	astTestMust(t, e)
	sig, two := util.NewInt(0), util.NewInt(2)
	astTestMust(t, z.Power(&two, 100, &sig))
	astTestMust(t, v.reserve(3))
	var initial util.ArbitraryFloatValueCopyReceipt
	astTestMust(t, p.numeric.SetComponentsToRuntimeFromOwnerWithReceipt(v.raw.Block, 8, 8, 120, false, 3, z, &sig, &initial))
	astTestMust(t, p.numeric.CleanupValueCopy(&initial, false))
	astTestMust(t, p.numeric.CleanupValueCopy(&initial, true))
	source, e := p.numeric.CaptureRuntimeValue(v.raw.Block, 8)
	astTestMust(t, e)
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
	if !v.copyFalseVisited || !v.copyTrueAcknowledged || !v.sourceDone || v.bodyCause != cause || v.destructorIndex != 3 {
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
