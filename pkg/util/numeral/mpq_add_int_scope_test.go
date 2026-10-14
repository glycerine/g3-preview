package numeral

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSynchronizedAddIntRetainsScratchAfterSourceBodyStops(t *testing.T) {
	m, _, _ := rawArithmeticGuardOwner(t)
	a, out := NewRational(), NewRational()
	var b Int
	serviceMust(t, m.SetString(&a, "3/2"))
	serviceMust(t, m.Z.SetString(&b, "18446744073709551617"))
	serviceMust(t, m.SetString(&out, "1/18446744073709551619"))
	rawArithmeticCorrupt(t, m.Z, &out.Den)

	e := m.AddInt(&a, &b, &out)
	if !errors.Is(e, arena.ErrBounds) || !m.addIntPending || m.addIntTemporary.Cell == 0 {
		t.Fatalf("source body failure lost scratch: %v, pending=%v, scratch=%+v", e, m.addIntPending, m.addIntTemporary)
	}
	if e := m.SourceFinalize(); !errors.Is(e, ErrRationalLexicalPending) {
		t.Fatalf("source finalization accepted retained scratch: %v", e)
	}
	if e := m.AddInt(&a, &b, &out); !errors.Is(e, ErrRationalLexicalPending) {
		t.Fatalf("second invocation accepted retained scratch: %v", e)
	}
	serviceMust(t, m.CleanupAddIntTemporary())
	if m.addIntPending || m.addIntTemporary.Cell != 0 {
		t.Fatal("owning cleanup did not release scratch")
	}
	serviceMust(t, m.Z.Del(&b))
	serviceMust(t, m.Del(&a))
}

func TestSynchronizedAddIntDeletesScratchAfterSuccess(t *testing.T) {
	m, _, _ := rawArithmeticGuardOwner(t)
	a, out := NewRational(), NewRational()
	var b Int
	serviceMust(t, m.SetString(&a, "3/2"))
	serviceMust(t, m.Z.SetString(&b, "18446744073709551617"))
	serviceMust(t, m.AddInt(&a, &b, &out))
	if m.addIntPending || m.addIntTemporary.Cell != 0 {
		t.Fatal("successful source del retained scratch")
	}
	var shown bytes.Buffer
	if e := m.DisplayTo(&shown, &out); e != nil || shown.String() != "36893488147419103237/2" {
		t.Fatalf("unexpected result: %q, %v", shown.String(), e)
	}
	serviceMust(t, m.Del(&out))
	serviceMust(t, m.Z.Del(&b))
	serviceMust(t, m.Del(&a))
}
