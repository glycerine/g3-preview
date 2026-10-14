package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalParserRationalScopesRetainReverseDeletionOrder(t *testing.T) {
	m, _, _ := rawArithmeticGuardOwner(t)
	for _, slot := range []*rationalParserScopedValue{&m.parseB, &m.parseExp, &m.parseTen} {
		slot.value = NewRational()
		slot.pending = true
		serviceMust(t, m.Z.SetLimbs(&slot.value.Num, []uint32{17, 0, 1}))
	}
	ten, exp, b := m.parseTen.value.Num.Cell, m.parseExp.value.Num.Cell, m.parseB.value.Num.Cell
	original := m.Z.Cell(ten)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	m.Z.Store(ten, bad)
	a := NewRational()
	if e := m.SetString(&a, "1.25e2"); !errors.Is(e, ErrRationalLexicalPending) || a != NewRational() {
		t.Fatal("pending rational parser scope changed caller", e, a)
	}
	if e := m.SourceFinalize(); !errors.Is(e, ErrRationalLexicalPending) {
		t.Fatal("source finalization discarded parser rational roots", e)
	}
	if e := m.CleanupParserTemporaries(); !errors.Is(e, arena.ErrBounds) || !m.parseTen.pending || !m.parseExp.pending || !m.parseB.pending || m.parseTen.value.Num.Cell != ten || m.parseExp.value.Num.Cell != exp || m.parseB.value.Num.Cell != b {
		t.Fatal("_ten refusal must precede _exp and b cleanup", e, m.parseTen, m.parseExp, m.parseB)
	}
	m.Z.Store(ten, original)
	serviceMust(t, m.CleanupParserTemporaries())
	if m.parseTen.pending || m.parseExp.pending || m.parseB.pending || m.parseTen != (rationalParserScopedValue{}) || m.parseExp != (rationalParserScopedValue{}) || m.parseB != (rationalParserScopedValue{}) {
		t.Fatal("owning cleanup did not drain parser rational scopes")
	}
	serviceMust(t, m.SetString(&a, "1.25e2"))
	if got, e := m.String(&a); e != nil || got != "125" {
		t.Fatal("fresh parser after rational scope cleanup", got, e)
	}
	serviceMust(t, m.Del(&a))
}

func TestRationalParserRationalScopeDenominatorStopKeepsComponentCursor(t *testing.T) {
	m, _, _ := rawArithmeticGuardOwner(t)
	value := NewRational()
	serviceMust(t, m.Z.SetLimbs(&value.Num, []uint32{19, 0, 1}))
	serviceMust(t, m.Z.SetLimbs(&value.Den, []uint32{23, 0, 1}))
	num, den := value.Num.Cell, value.Den.Cell
	original := m.Z.Cell(den)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	m.Z.Store(den, bad)
	e := m.finishParserRationalSource(&value, &m.parseB)
	if !errors.Is(e, arena.ErrBounds) || !m.parseB.pending || m.parseB.stage != 1 || m.parseB.value.Num.Cell != 0 || m.parseB.value.Den.Cell != den || ValidNumericRef(m.Z.Cells, num) || !ValidNumericRef(m.Z.Cells, den) {
		t.Fatal("source numerator deletion must precede stopped denominator", e, m.parseB)
	}
	m.Z.Store(den, original)
	serviceMust(t, m.CleanupParserTemporaries())
	if m.parseB != (rationalParserScopedValue{}) {
		t.Fatal("owning continuation did not retire stopped denominator")
	}
}
