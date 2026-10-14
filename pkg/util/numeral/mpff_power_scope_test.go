package numeral

import (
	"errors"
	"testing"
)

func TestFastFloatPowerTemporariesRetainSourceCleanupOrder(t *testing.T) {
	m := fastManager(t, 2)
	serviceMust(t, m.SetInt32(&m.powerWork, 3))
	serviceMust(t, m.SetInt32(&m.powerTrace, 5))
	m.powerWorkPending, m.powerTracePending = true, true
	work, trace := m.powerWork.bits, m.powerTrace.bits
	m.powerWork.bits = ffPack(m.capacity, 0, false)
	if e := m.CleanupPowerTemporaries(); !errors.Is(e, ErrFastFloatOwner) || !m.powerWorkPending || !m.powerTracePending || m.powerTrace.bits != trace {
		t.Fatal("work refusal crossed trace snapshot", e)
	}
	if e := m.open(); !errors.Is(e, ErrFastFloatPowerPending) {
		t.Fatal("retained power work allowed reentry", e)
	}
	m.powerWork.bits = work
	m.powerTrace.bits = ffPack(m.capacity, 0, false)
	if e := m.CleanupPowerTemporaries(); !errors.Is(e, ErrFastFloatOwner) || m.powerWorkPending || m.powerWork.bits != 0 || !m.powerTracePending {
		t.Fatal("retry replayed work or lost trace snapshot", e)
	}
	m.powerTrace.bits = trace
	serviceMust(t, m.CleanupPowerTemporaries())
	if m.powerWorkPending || m.powerTracePending || m.powerTrace.bits != 0 {
		t.Fatal("owning retry retained power temporaries")
	}
	serviceMust(t, m.open())
}
