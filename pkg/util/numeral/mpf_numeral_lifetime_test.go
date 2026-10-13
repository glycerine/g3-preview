package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatNumeralSourceSwapAndDel(t *testing.T) {
	core, _, _ := mpfNumeralHelpersFixture(t)
	m, err := NewArbitraryFloatNumeralManager(core, 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	if m.Precise() || !m.Field() {
		t.Fatal("f2n interval traits")
	}
	var x, y ArbitraryFloat
	if err := m.SetDouble(&x, math.Ldexp(1, 100)); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(&y, 3); err != nil {
		t.Fatal(err)
	}
	if err := m.Swap(&x, &y); err != nil {
		t.Fatal(err)
	}
	gotX, err := core.ToDouble(&x)
	if err != nil || gotX != 3 {
		t.Fatal("swapped x", gotX, err)
	}
	gotY, err := core.ToDouble(&y)
	if err != nil || gotY != math.Ldexp(1, 100) {
		t.Fatal("swapped y", gotY, err)
	}
	if err := m.Swap(&x, &x); err != nil {
		t.Fatal(err)
	}
	if err := m.SetUnsigned(&x, math.MaxUint32); err != nil {
		t.Fatal(err)
	}
	gotUnsigned, err := core.ToDouble(&x)
	if err != nil || gotUnsigned != math.MaxUint32 {
		t.Fatal("unsigned via double", gotUnsigned, err)
	}
	if err := m.SetInt32(&x, 8); err != nil {
		t.Fatal(err)
	}
	previous, err := m.PrevPowerOfTwo(&x)
	if err != nil || previous != 55 {
		t.Fatal("source previous-power formula", previous, err)
	}
	if err := m.Del(&x); err != nil {
		t.Fatal(err)
	}
	if err := m.Del(&y); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if err := m.Swap(&x, &y); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed swap", err)
	}
}
