package subpaving

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingHardwareContextBoundPolicyAndRootDispatch(t *testing.T) {
	m, src, out, values := hardwareIntervalFixture(t)
	limit := &control.Reslimit{Limit: 1000}
	c, err := NewHardwareContext(m, limit, 20, 10, 8192)
	if err != nil {
		t.Fatal(err)
	}
	if c.zeroEpsilon || m.ToFloat64(&c.maxBound) != 1e10 || m.ToFloat64(&c.minusMaxBound) != -1e10 || m.ToFloat64(&c.rootPrecision) != 1.0/8192.0 {
		t.Fatal("hardware context policy numerals")
	}
	epsilon := new(big.Rat).SetFloat64(m.ToFloat64(&c.epsilon))
	if epsilon.Cmp(big.NewRat(1, 20)) < 0 || epsilon.Cmp(big.NewRat(1, 19)) > 0 {
		t.Fatal("directed epsilon", epsilon)
	}
	if err := m.SetInt32(&values[0], 0); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(&values[1], 1); err != nil {
		t.Fatal(err)
	}
	if err := src.SetFinite(&values[0], &values[1], false, true); err != nil {
		t.Fatal(err)
	}
	if err := c.XnEqY(out, src, 2); err != nil {
		t.Fatal(err)
	}
	lo, hi, err := out.endpoints()
	if err != nil {
		t.Fatal(err)
	}
	if m.ToFloat64(lo) != -1 || m.ToFloat64(hi) != 1 || !out.lowerOpen || !out.upperOpen {
		t.Fatal("context inverse-image dispatch")
	}
	limit.Limit, limit.Steps = 1, 0
	if err := m.SetInt32(&values[1], 2); err != nil {
		t.Fatal(err)
	}
	if err := src.SetFinite(&values[0], &values[1], false, false); err != nil {
		t.Fatal(err)
	}
	if err := c.XnEqY(out, src, 2); !errors.Is(err, interval.ErrHardwareRootCanceled) {
		t.Fatal("context checkpoint", err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.XnEqY(out, src, 2); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed context", err)
	}
	zero, err := NewHardwareContext(m, limit, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !zero.zeroEpsilon || !m.IsZero(&zero.epsilon) || m.ToFloat64(&zero.maxBound) != 1 || m.ToFloat64(&zero.rootPrecision) != 1 {
		t.Fatal("zero parameter substitutions")
	}
	if err := zero.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := NewHardwareContext((*numeral.HardwareFloatNumeralManager)(nil), limit, 20, 10, 8192); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("missing manager", err)
	}
}
