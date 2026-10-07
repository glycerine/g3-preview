package util

import (
	"errors"
	"testing"
)

type numericGuardTestOwner struct{ executing bool }

func (g *numericGuardTestOwner) NumericExecuting() bool { return g.executing }
func TestNumericCallerBindingsPreserveNestedOwnersAndRejectCopiedHandles(t *testing.T) {
	m, _ := integerManager(t)
	outer, inner := numericGuardTestOwner{true}, numericGuardTestOwner{false}
	var a, b NumericCallerBinding
	if e := m.BindCaller(&a, &outer, 1); e != nil {
		t.Fatal(e)
	}
	if e := m.BindCaller(&b, &inner, 2); e != nil {
		t.Fatal(e)
	}
	if !m.numericCallerExecuting() {
		t.Fatal("inner idle caller erased active outer owner")
	}
	copy := b
	if e := copy.Unbind(); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("copied binding accepted", e)
	}
	if e := a.Unbind(); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("non-LIFO unbind accepted", e)
	}
	if m.CallerBindings() != &b || b.Previous() != &a {
		t.Fatal("failed unbind changed live chain")
	}
	if e := b.Unbind(); e != nil {
		t.Fatal(e)
	}
	if m.CallerBindings() != &a || !m.numericCallerExecuting() {
		t.Fatal("outer owner not restored")
	}
	if e := a.Unbind(); e != nil {
		t.Fatal(e)
	}
	if m.CallerBindings() != nil || m.numericCallerExecuting() {
		t.Fatal("caller binding leaked")
	}
	if e := b.Unbind(); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("retired binding accepted", e)
	}
}
func TestNumericCallerBindingsDoNotAllocateAfterConstruction(t *testing.T) {
	m, _ := integerManager(t)
	owner := numericGuardTestOwner{true}
	var binding NumericCallerBinding
	n := testing.AllocsPerRun(100, func() {
		if e := m.BindCaller(&binding, &owner, 1); e != nil {
			panic(e)
		}
		if e := binding.Unbind(); e != nil {
			panic(e)
		}
	})
	if n != 0 {
		t.Fatal("embedded source binding allocated", n)
	}
}

type numericPolicyTestOwner struct {
	manager *IntegerManager
	roots   NumericAllocationRoots
	active  bool
	calls   uint32
}

func (p *numericPolicyTestOwner) NumericRoots(m *IntegerManager, local *NumericAllocationRoots) *NumericAllocationRoots {
	p.calls++
	if !p.active || m != p.manager {
		return nil
	}
	if local != nil {
		return local
	}
	return &p.roots
}
func TestNumericPolicyDispatchPreservesArithmeticAndRetiredCaller(t *testing.T) {
	m, _ := integerManager(t)
	p := numericPolicyTestOwner{manager: m, active: true}
	h := NumericInvocationHooks{Policy: &p}
	op := m.WithAllocationContext(NumericAllocationContext{}.WithCaller(&h))
	a, b, out := NewInt(2), NewInt(3), NewInt(0)
	n := testing.AllocsPerRun(100, func() {
		if e := op.Add(&a, &b, &out); e != nil {
			panic(e)
		}
	})
	if n != 0 {
		t.Fatal("policy dispatch allocated warmed source calls", n)
	}
	if p.calls == 0 || out != NewInt(5) {
		t.Fatal("caller policy or arithmetic bypassed", p.calls, out)
	}
	p.active = false
	if e := op.Add(&a, &b, &out); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("expired caller accepted", e)
	}
	if out != NewInt(5) {
		t.Fatal("expired caller modified output")
	}
}

type numericScalarTestPolicy struct {
	numericPolicyTestOwner
	enabled     bool
	scalar      IntegerScalarRawScope
	scalarCalls uint32
}

func (p *numericScalarTestPolicy) NumericScalarRawEnabled() bool { return p.enabled }
func (p *numericScalarTestPolicy) NumericBound(bool) bool        { return p.active }
func (p *numericScalarTestPolicy) NumericScalarRaw(m *IntegerManager, event string) (*IntegerScalarRawScope, error) {
	p.scalarCalls++
	if m != p.manager || event != "division" {
		return nil, ErrNumericOwner
	}
	return &p.scalar, nil
}
func TestNumericScalarPolicyUsesOnlySelectedCallerScope(t *testing.T) {
	m, _ := integerManager(t)
	p := numericScalarTestPolicy{numericPolicyTestOwner: numericPolicyTestOwner{manager: m, active: true}, enabled: true}
	hooks := NumericInvocationHooks{Policy: &p}
	op := m.WithAllocationContext(NumericAllocationContext{}.WithCaller(&hooks))
	a, b, out := NewInt(-7), NewInt(3), Int{}
	if e := op.Div(&a, &b, &out); e != nil {
		t.Fatal(e)
	}
	got, e := m.Int64(&out)
	if e != nil || got != -3 || p.scalarCalls != 1 || !p.scalar.ended || p.scalar.Active() {
		t.Fatal("selected source RAW scope", got, e, p.scalarCalls)
	}
	if m.DivisionRaw.Invocation != 0 {
		t.Fatal("caller scope also constructed ambient RAW")
	}
	copied := p.scalar
	if e := copied.Cleanup(true); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("copied scalar scope accepted", e)
	}
	p.enabled = false
	if e := op.Div(&a, &b, &out); e != nil {
		t.Fatal(e)
	}
	if p.scalarCalls != 1 || m.DivisionRaw.Invocation == 0 {
		t.Fatal("disabled specialization changed ordinary RAW scope", p.scalarCalls)
	}
	if e := m.Del(&out); e != nil {
		t.Fatal(e)
	}
}
