// Source: src/util/mpf.cpp:1519-1569; src/ast/ast_smt2_pp.cpp:262-268.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package util

import (
	"errors"
	"math/big"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatToStringLiteralSpecialBranches(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	for _, v := range []struct {
		sign bool
		exp  int64
		sig  int32
		want string
	}{
		{false, 128, 1, "NaN"}, {true, 128, 1, "NaN"}, {false, 128, 0, "+oo"}, {true, 128, 0, "-oo"}, {false, -127, 0, "+zero"}, {true, -127, 0, "-zero"},
	} {
		var a ArbitraryFloat
		n := NewInt(v.sig)
		serviceMust(t, m.SetComponents(&a, 8, 24, v.sign, v.exp, &n))
		first := &formatTestText{}
		view, e := m.ToStringInto(&a, first)
		got := formatTestString(t, view)
		used := b.Used()
		b.FailAfter(0)
		second := &formatTestText{}
		v2, repeat := m.ToStringInto(&a, second)
		again := formatTestString(t, v2)
		b.FailAfter(-1)
		if e != nil || repeat != nil || got != v.want || again != v.want || b.Used() != used {
			t.Fatal(v, got, e, b.Used(), used)
		}
		serviceMust(t, m.CleanupFormatInvocation(&first.scope, true))
		serviceMust(t, m.EndFormatContext(&first.scope))
		serviceMust(t, m.CleanupFormatInvocation(&second.scope, true))
		serviceMust(t, m.EndFormatContext(&second.scope))
		serviceMust(t, m.Del(&a))
	}
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

// Special-route oracle does not implement the finite stream route; production
// strings use the actual root owned-text provider, tested in package g3.
type formatTestText struct {
	hook    func(int32)
	scope   ArbitraryFloatFormatInvocation
	literal string
}

func (t *formatTestText) NumericInvocation() *ArbitraryFloatFormatInvocation { return &t.scope }
func (t *formatTestText) AssignLiteral(s string) error                       { t.literal = s; return nil }
func (t *formatTestText) NewArbitraryFloatFormatStream() (ArbitraryFloatFormatStream, error) {
	return &formatTestStream{hook: t.hook}, nil
}
func (t *formatTestText) AppendStreamString(stream ArbitraryFloatFormatStream) error {
	t.literal += string(stream.(*formatTestStream).bytes)
	return nil
}
func (t *formatTestText) CString() (ArbitraryFloatFormattedCString, error) {
	return BorrowArbitraryFloatFormattedCString(t), nil
}
func (t *formatTestText) Byte(i uint64) (byte, error) { return t.literal[i], nil }
func (t *formatTestText) Length() (uint64, error)     { return uint64(len(t.literal)), nil }
func (t *formatTestText) SourceFinalize() error       { return nil }
func (t *formatTestText) Close() error                { return nil }
func formatTestString(t *testing.T, v ArbitraryFloatFormattedCString) string {
	t.Helper()
	n, e := v.Length()
	serviceMust(t, e)
	b := make([]byte, n)
	for i := range b {
		b[i], e = v.Byte(uint64(i))
		serviceMust(t, e)
	}
	return string(b)
}

func TestArbitraryFloatToStringLiteralFiniteBranches(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	for _, item := range []struct {
		sign bool
		exp  int64
		sig  int32
		want string
	}{
		{false, 3, 4, "1.5 3"}, {true, 3, 4, "-1.5 3"}, {false, 0, 0, "1.0 0"}, {false, -127, 4, "0.5 -126"},
	} {
		a := ArbitraryFloat{}
		sig := NewInt(item.sig)
		serviceMust(t, m.SetComponents(&a, 8, 4, item.sign, item.exp, &sig))
		text := &formatTestText{}
		v, e := m.ToStringInto(&a, text)
		serviceMust(t, e)
		if got := formatTestString(t, v); got != item.want {
			t.Fatal(got, item.want)
		}
		serviceMust(t, m.CleanupFormatInvocation(&text.scope, false))
		serviceMust(t, m.CleanupFormatInvocation(&text.scope, true))
		serviceMust(t, m.EndFormatContext(&text.scope))
		serviceMust(t, m.Del(&a))
	}
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

type formatTestStream struct {
	bytes []byte
	hook  func(int32)
}

func (s *formatTestStream) AppendInt32(v int32) error {
	if s.hook != nil {
		s.hook(v)
	}
	s.bytes = strconv.AppendInt(s.bytes, int64(v), 10)
	return nil
}
func (s *formatTestStream) AppendInt64(v int64) error {
	s.bytes = strconv.AppendInt(s.bytes, v, 10)
	return nil
}
func (s *formatTestStream) AppendCString(v string) error { s.bytes = append(s.bytes, v...); return nil }
func (s *formatTestStream) AppendNumeralCString(v NumeralCStringView) error {
	for i := uint64(0); ; i++ {
		b, e := v.Byte(i)
		if e != nil {
			return e
		}
		if b == 0 {
			return nil
		}
		s.bytes = append(s.bytes, b)
	}
}
func (s *formatTestStream) SourceFinalize() error { return nil }
func (s *formatTestStream) Close() error          { return nil }

func TestArbitraryFloatToStringEveryNumericalReservationPrefix(t *testing.T) {
	for ordinal := int64(0); ordinal < 400; ordinal++ {
		b := arena.NewBudget(1 << 28)
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64}})
		serviceMust(t, e)
		var a ArbitraryFloat
		sig := NewInt(1)
		serviceMust(t, m.SetComponents(&a, 8, 260, false, 0, &sig))
		_, e = m.IsNaN(&a)
		serviceMust(t, e)
		_, e = m.IsZero(&a)
		serviceMust(t, e)
		text := &formatTestText{}
		b.FailAfter(ordinal)
		view, body := m.ToStringInto(&a, text)
		integerPhysicalAuditWithRaw(t, m.integer, nil, &a.significand, &m.rational.NTmp)
		var maps uint64
		for i := range m.powers {
			maps += m.powers[i].StorageBytes()
		}
		if want := m.header + m.ValueCopyHistoryBytes() + m.powerWrapperBytes + m.rational.HeaderBytes + maps + integerActualOwnedBytes(m.integer); b.Used() != want {
			t.Fatal("numerical cold graph before fault lift", ordinal, b.Used(), want)
		}
		if body != nil && !numericPhysicalReservationRefusal(body) {
			t.Fatal("reached numerical refusal", ordinal, body)
		}
		b.FailAfter(-1)
		capturedBody, capturedSource, capturedOwn := m.FormatInvocationCauses(&text.scope)
		serviceMust(t, m.CleanupFormatInvocation(&text.scope, false))
		serviceMust(t, m.CleanupFormatInvocation(&text.scope, true))
		if b2, s2, o2 := m.FormatInvocationCauses(&text.scope); b2 != capturedBody || s2 != capturedSource || (capturedOwn != nil && o2 != capturedOwn) {
			t.Fatal("own ACK erased history", ordinal)
		}
		serviceMust(t, m.EndFormatContext(&text.scope))
		serviceMust(t, m.Del(&a))
		if e := m.SourceFinalize(); e != nil {
			if !errors.Is(e, arena.ErrInvalidSlot) || !m.integer.sourceGuard || m.integer.SourceFinalizeStage < 2 || m.integer.SourceFinalizeStage > 5 {
				t.Fatal("literal owner source stop", ordinal, e, m.integer.SourceFinalizeStage)
			}
			index := int(m.integer.SourceFinalizeStage-2) / 2
			ref := m.integer.Tmp[index]
			if m.integer.SourceFinalizeStage%2 == 1 {
				ref = m.integer.Arg[index]
			}
			if ref != 0 {
				t.Fatal("unproved scratch-hole source guard", ordinal, ref)
			}
		}
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal("own end leak", ordinal, b.Used())
		}
		if body == nil {
			den := new(big.Int).Lsh(big.NewInt(1), 259)
			num := new(big.Int).Add(new(big.Int).Set(den), big.NewInt(1))
			want := new(big.Rat).SetFrac(num, den).FloatString(259) + " 0"
			if got := formatTestString(t, view); got != want {
				t.Fatal("independent full finite decimal", got, want)
			}
			t.Log("first complete numerical prefix", ordinal)
			return
		}
	}
	t.Fatal("no complete numerical prefix")
}

func TestArbitraryFloatToStringActualAllocationCallbackGuards(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var m *ArbitraryFloatManager
	var rt *Runtime
	var text formatTestText
	var a ArbitraryFloat
	active := false
	calls := 0
	rt, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f Failure) error {
		if active {
			calls++
			used := b.Used()
			stage := m.integer.SourceFinalizeStage
			own := m.ownCause
			source := m.sourceCause
			if e := m.Close(); e != ErrNumericPending {
				t.Fatal("MPF own reentry", e)
			}
			if e := m.SourceFinalize(); e != ErrNumericPending {
				t.Fatal("MPF source reentry", e)
			}
			if e := m.integer.CloseChecked(); e != ErrNumericPending {
				t.Fatal("Integer own reentry", e)
			}
			if e := m.integer.SourceFinalize(); e != ErrNumericPending {
				t.Fatal("Integer source reentry", e)
			}
			if e := m.rational.CloseChecked(); e != ErrNumericPending {
				t.Fatal("Rational own reentry", e)
			}
			if e := m.rational.SourceFinalize(); e != ErrNumericPending {
				t.Fatal("Rational source reentry", e)
			}
			if e := m.CleanupFormatInvocation(&text.scope, false); e != ErrNumericPending {
				t.Fatal("false child reentry", e)
			}
			if e := m.CleanupFormatInvocation(&text.scope, true); e != ErrNumericPending {
				t.Fatal("true child reentry", e)
			}
			other := &formatTestText{}
			if _, e := m.ToStringInto(&a, other); e != ErrNumericPending {
				t.Fatal("same manager format reentry", e)
			}
			if b.Used() != used || m.integer.SourceFinalizeStage != stage || m.ownCause != own || m.sourceCause != source || text.scope.ownCause != nil {
				t.Fatal("executing cleanup changed older owner")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	sig := NewInt(1)
	serviceMust(t, m.SetComponents(&a, 8, 40000, false, 0, &sig))
	_, e = m.IsNaN(&a)
	serviceMust(t, e)
	_, e = m.IsZero(&a)
	serviceMust(t, e)
	pad, e := rt.Allocate(uint64(99990 - rt.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	active = true
	_, body := m.ToStringInto(&a, &text)
	active = false
	if !errors.Is(body, Failure{Kind: FailureMemout}) || calls == 0 {
		t.Fatal("actual callback leaf not reached", body, calls)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []Block{pad}, &a.significand, &m.rational.NTmp)
	serviceMust(t, rt.SetMaxSize(0))
	serviceMust(t, m.CleanupFormatInvocation(&text.scope, false))
	serviceMust(t, m.CleanupFormatInvocation(&text.scope, true))
	if text.scope.bodyCause != body || text.scope.ownCause != nil {
		t.Fatal("callback failure history", text.scope.bodyCause, text.scope.ownCause)
	}
	serviceMust(t, m.EndFormatContext(&text.scope))
	serviceMust(t, m.Del(&a))
	serviceMust(t, rt.Deallocate(&pad))
	serviceMust(t, m.Close())
	serviceMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

// Qualified signed-host TLS fixture at actual integer-part insertion. The
// reached body refusal is followed by q's NUM destructor; it protects denom,
// num and the caller-owned result. The earlier guessed verification-DT recipe
// is retained in diagnostic logs and receives no youngest-child credit.
func TestArbitraryFloatToStringScopedRationalSourceDTStop(t *testing.T) {
	if !modularSourceDebug {
		return
	}
	b := arena.NewBudget(1 << 28)
	rt, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewArbitraryFloatManager(IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	var a ArbitraryFloat
	sig := NewInt(1)
	serviceMust(t, m.SetComponents(&a, 8, 260, false, 0, &sig))
	warm := &formatTestText{}
	_, e = m.ToStringInto(&a, warm)
	serviceMust(t, e)
	serviceMust(t, warm.scope.Cleanup(false))
	serviceMust(t, warm.scope.Cleanup(true))
	serviceMust(t, warm.scope.EndContext())
	external, e := rt.Allocate(200000)
	serviceMust(t, e)
	text := &formatTestText{}
	armed := true
	var release error
	var older [4]Int
	text.hook = func(v int32) {
		if armed {
			armed = false
			if text.scope.Constructed != 3 || !text.scope.Decimal.executing {
				t.Fatal("unreached caller declarations")
			}
			older = [4]Int{text.scope.Q.Num, text.scope.Q.Den, text.scope.Denom, text.scope.Num}
			contextTLSNearMinimum(t, rt)
			release = rt.Deallocate(&external)
		}
	}
	_, body := m.ToStringInto(&a, text)
	integerPhysicalAuditWithRaw(t, m.integer, nil, &a.significand, &m.rational.NTmp)
	if !errors.Is(release, arena.ErrBounds) || external != (Block{}) {
		t.Fatal("actual accounting frontier", release)
	}
	if !errors.Is(body, arena.ErrBounds) || text.scope.sourceCause == nil || text.scope.sourceCursor != 1 || older != ([4]Int{text.scope.Q.Num, text.scope.Q.Den, text.scope.Denom, text.scope.Num}) {
		t.Fatal("reached q NUM DT did not protect older fields", body, text.scope.sourceCause, text.scope.sourceCursor)
	}
	if text.scope.Q.Num.Cell == 0 || !m.integer.Cell(text.scope.Q.Num.Cell).Released {
		t.Fatal("q NUM physical release not reached")
	}
	if text.scope.SourceReady() != text.scope.sourceCause {
		t.Fatal("caller res source DT did not observe younger stop")
	}

	captured := body
	cause := text.scope.sourceCause
	state := rt.State()
	if e := m.finishFormatSource(&text.scope); e != cause || rt.State() != state {
		t.Fatal("sticky source retry", e)
	}
	ownStop := text.scope.Cleanup(false)
	if !errors.Is(ownStop, arena.ErrBounds) || text.scope.ownCause != ownStop {
		t.Fatal("actual first own stop", ownStop, text.scope.ownCause)
	}
	firstOwn := text.scope.ownCause
	integerPhysicalAuditWithRaw(t, m.integer, nil, &a.significand, &m.rational.NTmp)
	recovery, e := rt.Allocate(400000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, func() error { return text.scope.Cleanup(false) })
	contextTLSFinish(t, rt, func() error { return text.scope.Cleanup(true) })
	if text.scope.sourceCause != cause || text.scope.ownCause != firstOwn || !errors.Is(captured, cause) {
		t.Fatal("own ACK lost source error")
	}
	serviceMust(t, text.scope.EndContext())
	contextTLSFinish(t, rt, func() error { return m.Del(&a) })
	contextTLSFinish(t, rt, m.Close)
	if e := rt.Deallocate(&recovery); e != nil && (!errors.Is(e, arena.ErrBounds) || recovery != (Block{})) {
		t.Fatal(e)
	}
	contextTLSFinish(t, rt, rt.Close)
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestArbitraryFloatToStringCopiedScopeAndRetainedHistory(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	var a ArbitraryFloat
	sig := NewInt(1)
	serviceMust(t, m.SetComponents(&a, 8, 260, false, 0, &sig))
	_, e = m.IsNaN(&a)
	serviceMust(t, e)
	_, e = m.IsZero(&a)
	serviceMust(t, e)
	first := &formatTestText{}
	b.FailAfter(3)
	_, body := m.ToStringInto(&a, first)
	if !numericPhysicalReservationRefusal(body) {
		t.Fatal("real body refusal", body)
	}
	integerPhysicalAuditWithRaw(t, m.integer, nil, &a.significand, &m.rational.NTmp)
	old := first.scope.allocationContext()
	bodyCause, sourceCause, ownCause := m.FormatInvocationCauses(&first.scope)
	originalCell, originalPending := first.scope.originalCell, first.scope.originalPending
	copied := first.scope
	used := b.Used()
	state := m.memory.State()
	if e := m.CleanupFormatInvocation(&copied, true); e != ErrRuntimeOwner {
		t.Fatal("copied actual frame accepted", e)
	}
	if e := copied.SourceReady(); e != ErrRuntimeOwner {
		t.Fatal("copied history accepted", e)
	}
	if used != b.Used() || state != m.memory.State() {
		t.Fatal("copied refusal changed owner")
	}
	b.FailAfter(-1)
	serviceMust(t, first.scope.Cleanup(false))
	serviceMust(t, first.scope.Cleanup(true))
	// An inline later call is source-valid even though prior source scratch is
	// absent. It creates no numerical allocation and does not replace old history.
	var zero ArbitraryFloat
	serviceMust(t, m.SetComponents(&zero, 8, 24, false, -127, &Int{}))
	later := &formatTestText{}
	b.FailAfter(0)
	v, e := m.ToStringInto(&zero, later)
	if e != nil || formatTestString(t, v) != "+zero" {
		t.Fatal("later inline call", e)
	}
	if old.roots(m.integer) != nil {
		t.Fatal("old allocation context retargeted")
	}
	used = b.Used()
	fresh := later.scope
	serviceMust(t, first.scope.Cleanup(false))
	serviceMust(t, first.scope.Cleanup(true))
	if later.scope.invocation != fresh.invocation || later.scope.Num != fresh.Num || used != b.Used() {
		t.Fatal("old ACK adopted later child")
	}
	if bc, sc, oc := m.FormatInvocationCauses(&first.scope); bc != bodyCause || sc != sourceCause || oc != ownCause || first.scope.originalCell != originalCell || first.scope.originalPending != originalPending {
		t.Fatal("old source/own/acquisition history erased")
	}
	b.FailAfter(-1)
	serviceMust(t, later.scope.Cleanup(true))
	serviceMust(t, later.scope.EndContext())
	serviceMust(t, first.scope.EndContext())
	serviceMust(t, m.Del(&zero))
	serviceMust(t, m.Del(&a))
	if e := m.SourceFinalize(); e != nil && !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestArbitraryFloatToStringOldScopePreservesLaterAllocatingCall(t *testing.T) {
	for ordinal := int64(0); ordinal < 30; ordinal++ {
		b := arena.NewBudget(1 << 28)
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		var a ArbitraryFloat
		sig := NewInt(1)
		serviceMust(t, m.SetComponents(&a, 8, 260, false, 0, &sig))
		_, e = m.IsNaN(&a)
		serviceMust(t, e)
		_, e = m.IsZero(&a)
		serviceMust(t, e)
		first := &formatTestText{}
		b.FailAfter(ordinal)
		_, body := m.ToStringInto(&a, first)
		integerPhysicalAuditWithRaw(t, m.integer, nil, &a.significand, &m.rational.NTmp)
		// Count3 is the reached GetNum/GetDen Copy suffix before ten's constructor.
		// The copied RAW holder remains owned even when malloc refuses before a token
		// is issued. No descriptor-failure credit is inferred from this fixture.
		target := body != nil && first.scope.Decimal.Parent != nil && first.scope.Decimal.Count == 3
		b.FailAfter(-1)
		serviceMust(t, first.scope.Cleanup(false))
		serviceMust(t, first.scope.Cleanup(true))
		if target {
			oldBody, oldSource, oldOwn := m.FormatInvocationCauses(&first.scope)
			child := first.scope.Decimal.verification
			original := first.scope.Decimal.originalCell
			later := &formatTestText{}
			view, e := m.ToStringInto(&a, later)
			serviceMust(t, e)
			used := b.Used()
			fresh := later.scope.invocation
			value := formatTestString(t, view)
			serviceMust(t, first.scope.Cleanup(false))
			serviceMust(t, first.scope.Cleanup(true))
			if used != b.Used() || later.scope.invocation != fresh || formatTestString(t, view) != value {
				t.Fatal("old ACK adopted later caller")
			}
			if bc, sc, oc := m.FormatInvocationCauses(&first.scope); bc != oldBody || sc != oldSource || oc != oldOwn || first.scope.Decimal.verification != child || first.scope.Decimal.originalCell != original {
				t.Fatal("retained exact history changed")
			}
			integerPhysicalAuditWithRaw(t, m.integer, nil, &a.significand, &m.rational.NTmp)
			serviceMust(t, later.scope.Cleanup(true))
			serviceMust(t, later.scope.EndContext())
			serviceMust(t, first.scope.EndContext())
			serviceMust(t, m.Del(&a))
			serviceMust(t, m.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
			t.Log("actual reached decimal Copy body refusal", ordinal)
			return
		}
		serviceMust(t, first.scope.EndContext())
		serviceMust(t, m.Del(&a))
		if e := m.SourceFinalize(); e != nil && !errors.Is(e, arena.ErrInvalidSlot) {
			t.Fatal(e)
		}
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
	t.Fatal("decimal Copy body refusal not reached")
}
