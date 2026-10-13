package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestScopedArbitraryFloatNumeralSourceLifetime(t *testing.T) {
	core, _, _ := mpfNumeralHelpersFixture(t)
	m, err := NewArbitraryFloatNumeralManager(core, 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	}()
	s, err := NewScopedArbitraryFloatNumeral(m)
	if err != nil {
		t.Fatal(err)
	}
	x, err := s.Get()
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(x, 7); err != nil {
		t.Fatal(err)
	}
	copyOfOwner := *s
	if _, err := copyOfOwner.Get(); !errors.Is(err, ErrScopedCopy) {
		t.Fatal("copied owner", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed scope", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal("idempotent close", err)
	}
}
