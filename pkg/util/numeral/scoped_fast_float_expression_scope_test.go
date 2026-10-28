package numeral

// Source: src/util/scoped_numeral.h:170-183; scoped_mpff return copy.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestScopedFastFloatExpressionBodyFailureEndsTemporary(t *testing.T) {
	m := fastManager(t, 3)
	a, e := NewScopedFastFloat(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, a.Cleanup()) })
	zero, e := NewScopedFastFloat(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, zero.Cleanup()) })
	serviceMust(t, a.SetInt32(7))
	baseline := m.config.Storage.Descriptors.Budget.Used()
	result, e := a.Div(zero)
	var pending *ScopedFastFloatExpressionCleanupError
	if result != nil || !errors.Is(e, ErrFastFloatDivisionByZero) || errors.As(e, &pending) || m.config.Storage.Descriptors.Budget.Used() != baseline {
		t.Fatal("body failure did not end source temporary", result, e)
	}
	fastAudit(t, m, &a.num, &zero.num)
}

func TestScopedFastFloatExpressionStoppedSourceRetainsReturn(t *testing.T) {
	m := fastManager(t, 3)
	a, e := NewScopedFastFloat(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, a.Cleanup()) })
	serviceMust(t, a.SetInt32(7))
	local, e := a.Clone()
	serviceMust(t, e)
	returned, e := local.Clone()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, local.Cleanup()); serviceMust(t, returned.Cleanup()) })
	m.powerWorkPending = true
	result := returned
	var cause error
	finishScopedFastFloatExpression(local, &result, &cause)
	var pending *ScopedFastFloatExpressionCleanupError
	if result != nil || !errors.As(cause, &pending) || !errors.Is(cause, ErrFastFloatPowerPending) || pending.local != local || pending.returned != returned || local.closed || returned.closed {
		t.Fatal("stopped source deletion lost result or temporary", result, cause)
	}
	m.powerWorkPending = false
	if e := local.Close(); !errors.Is(e, ErrFastFloatPowerPending) || local.closed {
		t.Fatal("stopped source destructor replayed", e)
	}
	serviceMust(t, pending.Cleanup())
	serviceMust(t, pending.Cleanup())
	if pending.local != nil || pending.returned != nil || !local.closed || !returned.closed {
		t.Fatal("owning cleanup retained expression owners")
	}
	fastAudit(t, m, &a.num)
}

func TestScopedFastFloatExpressionStoppedSourceRetainsFailedReturnCopy(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	cfg := DefaultFastFloatConfig(serviceConfig(budget).Integers.Words)
	cfg.InitialCapacity = 8
	m, e := NewFastFloatManager(cfg)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CloseChecked()) })
	a, e := NewScopedFastFloat(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, a.Cleanup()) })
	serviceMust(t, a.SetInt32(7))
	local, e := a.Clone()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, local.Cleanup()) })
	index := m.next
	if index >= m.capacity {
		t.Fatal("fixture did not reserve a fresh ID")
	}
	m.write(&m.metadata, 2*index+1, ^uint32(0))
	result, cause := local.Clone()
	var copy *ScopedFastFloatCopyCleanupError
	if result != nil || !errors.As(cause, &copy) || copy.owner == nil {
		t.Fatal("return copy did not retain failed constructor", result, cause)
	}
	m.powerWorkPending = true
	finishScopedFastFloatExpression(local, &result, &cause)
	var pending *ScopedFastFloatExpressionCleanupError
	if !errors.As(cause, &pending) || pending.local != local || pending.copy != copy || !errors.Is(cause, ErrFastFloatPowerPending) || !errors.Is(cause, ErrFastFloatOwner) {
		t.Fatal("source stop lost failed return copy", cause)
	}
	m.powerWorkPending = false
	serviceMust(t, pending.Cleanup())
	if pending.local != nil || pending.copy != nil || copy.owner != nil {
		t.Fatal("owning cleanup retained failed return copy")
	}
	fastAudit(t, m, &a.num)
}
