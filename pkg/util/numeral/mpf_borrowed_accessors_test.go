package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatBorrowedFPAComponentsAndManagerLifetime(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var value ArbitraryFloat
	if err := m.SetComponentsUint64(&value, 8, 24, true, -2, 1<<22); err != nil {
		t.Fatal(err)
	}
	if !value.Sign() || value.Exponent() != -2 || value.EBits() != 8 || value.SBits() != 24 {
		t.Fatal("source scalar component projections")
	}
	integer, err := m.BorrowIntegerManager()
	if err != nil {
		t.Fatal(err)
	}
	rational, err := m.BorrowRationalManager()
	if err != nil {
		t.Fatal(err)
	}
	if integer != rational.Z {
		t.Fatal("mpf_manager integer reference must be its rational child")
	}
	sig, err := m.BorrowSignificand(&value)
	if err != nil {
		t.Fatal(err)
	}
	if sig != &value.significand {
		t.Fatal("source sig returns the stored numeral reference")
	}
	got, err := integer.Int64(sig)
	if err != nil || got != 1<<22 {
		t.Fatalf("borrowed significand %d: %v", got, err)
	}
	if _, err := m.BorrowSignificand(nil); err == nil {
		t.Fatal("nil value must be refused")
	}
	if err := m.Del(&value); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := m.BorrowIntegerManager(); !errors.Is(err, arena.ErrClosed) {
		t.Fatalf("closed integer borrow: %v", err)
	}
	if _, err := m.BorrowRationalManager(); !errors.Is(err, arena.ErrClosed) {
		t.Fatalf("closed rational borrow: %v", err)
	}
	if _, err := m.BorrowSignificand(&value); !errors.Is(err, arena.ErrClosed) {
		t.Fatalf("closed significand borrow: %v", err)
	}
}
