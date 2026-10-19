// Source: algebraic_numbers.cpp:2998-3039; mpq.cpp power normal END.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicOwnedBoundBodyFailureFinishesOlderTemporary(t *testing.T) {
	c, _ := algebraicCellSourceFixtureProfile(t, true)
	service, _ := serviceNew(t)
	root := algebraicTinyBinaryBoundsRoot(t, c)
	out, e := service.NewValue()
	serviceMust(t, e)
	serviceMust(t, service.SetInt64(&out, 77))
	pad := algebraicCellSourceTLSBoundary(t, c)
	body := c.GetUpperOwnedRational(&root, service, &out)
	if body == nil || c.bounds.sourceStop != nil || !c.toOwnedRational.tmpDone {
		t.Fatal("RAW body failure protected completed older temporary", body)
	}
	runtime := c.config.Rationals.Integers().RuntimeOwner()
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	serviceMust(t, c.CleanupToOwnedRational())
	text, e := service.String(&out)
	serviceMust(t, e)
	if text != "77" {
		t.Fatal("failed body published global output", text)
	}
	serviceMust(t, service.Del(&out))
	serviceMust(t, c.Del(&root))
}

func TestAlgebraicOwnedBoundChildSourceStopProtectsOuterTemporary(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	service, _ := serviceNew(t)
	root := algebraicTinyBinaryBoundsRoot(t, c)
	out, e := service.NewValue()
	serviceMust(t, e)
	serviceMust(t, service.SetInt64(&out, 77))
	writer := &ownedBoundRawEndGuard{core: c}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("mpq")
	serviceMust(t, c.config.Rationals.BindTrace(tags))
	body := c.GetUpperOwnedRational(&root, service, &out)
	serviceMust(t, c.config.Rationals.BindTrace(nil))
	if tags.ShouldTrace("mpq") {
		if writer.ref == 0 || !errors.Is(body, arena.ErrBounds) || c.bounds.conversion.stop == nil || c.bounds.remaining == 0 {
			t.Fatal("actual bound source END guard not reached", body)
		}
		if c.toOwnedRational.sourceCause != c.bounds.sourceStop || c.toOwnedRational.tmpDone || c.toOwnedRational.tmp != writer.tmp {
			t.Error("younger bound source stop failed to protect older raw rational")
		}
		c.config.Rationals.Integers().Store(writer.ref, writer.original)
	} else if body != nil || writer.ref != 0 {
		t.Fatal("erased MPQ trace changed bound", body)
	}
	serviceMust(t, c.CleanupToOwnedRational())
	if tags.ShouldTrace("mpq") {
		text, e := service.String(&out)
		serviceMust(t, e)
		if text != "77" || !errors.Is(body, c.bounds.sourceStop) {
			t.Fatal("owning acknowledgement lost output or source history", text, body)
		}
	}
	serviceMust(t, service.Del(&out))
	serviceMust(t, c.Del(&root))
}

type ownedBoundRawEndGuard struct {
	core     *AlgebraicRootCore
	ref      unumeral.NumericCellRef
	original unumeral.NumericCell
	tmp      unumeral.Rational
}

func (w *ownedBoundRawEndGuard) Write(p []byte) (int, error) {
	m := w.core.config.Rationals
	z := m.Integers()
	r := &m.RawArithmetic
	if w.ref == 0 && r.PowerActive && r.Power.Num.Cell != 0 {
		ref := r.Power.Num.Cell
		cell := z.Cell(ref)
		if cell.Size == 3 && z.Digit(ref, 0) == 0 && z.Digit(ref, 1) == 0 && z.Digit(ref, 2) == 1 {
			w.ref = ref
			w.original = cell
			w.tmp = w.core.toOwnedRational.tmp
			// Qualified requested-size guard on the reached final unused square,
			// before its actual source END deletion; no fabricated RAW scope.
			guard := cell
			guard.Backing.Requested = uint64(^uint32(0)) + 1
			z.Store(ref, guard)
		}
	}
	return len(p), nil
}
