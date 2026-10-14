package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalParserIntegerScopesRetainSuffixBeforeOuterCleanup(t *testing.T) {
	m, _, _ := rawArithmeticGuardOwner(t)
	serviceMust(t, m.Z.SetLimbs(&m.parseOuterTemporary, []uint32{11, 0, 1}))
	serviceMust(t, m.Z.SetLimbs(&m.parseSuffixTemporary, []uint32{13, 0, 1}))
	m.parseOuterPending, m.parseSuffixPending = true, true
	suffix := m.parseSuffixTemporary.Cell
	outer := m.parseOuterTemporary.Cell
	original := m.Z.Cell(suffix)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	m.Z.Store(suffix, bad)
	a := NewRational()
	if e := m.SetString(&a, "42"); !errors.Is(e, ErrRationalLexicalPending) || a.Num != NewInt(0) {
		t.Fatal("pending parser scope changed caller", e, a)
	}
	if e := m.SourceFinalize(); !errors.Is(e, ErrRationalLexicalPending) {
		t.Fatal("source finalization discarded parser roots", e)
	}
	if e := m.CleanupParserTemporaries(); !errors.Is(e, arena.ErrBounds) || !m.parseSuffixPending || !m.parseOuterPending || m.parseSuffixTemporary.Cell != suffix || m.parseOuterTemporary.Cell != outer {
		t.Fatal("suffix refusal must precede outer cleanup", e, m.parseSuffixTemporary, m.parseOuterTemporary)
	}
	m.Z.Store(suffix, original)
	serviceMust(t, m.CleanupParserTemporaries())
	if m.parseSuffixPending || m.parseOuterPending || m.parseSuffixTemporary != (Int{}) || m.parseOuterTemporary != (Int{}) {
		t.Fatal("parser scope owning cleanup did not retire both roots")
	}
	serviceMust(t, m.SetString(&a, "42.5"))
	if got, e := m.String(&a); e != nil || got != "85/2" {
		t.Fatal("fresh parser after owning cleanup", got, e)
	}
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.Z.SetLimbs(&m.parseOuterTemporary, []uint32{17, 0, 1}))
	m.parseOuterPending = true
	serviceMust(t, m.CloseChecked())
	if !m.closed || m.parseOuterPending {
		t.Fatal("manager close skipped retained parser integer")
	}
}
