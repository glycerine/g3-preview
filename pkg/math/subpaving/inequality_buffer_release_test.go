package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingTemporaryInequalityBufferUnwind(t *testing.T) {
	t.Run("rational", func(t *testing.T) {
		c, m, value, x, _ := rationalClauseFixture(t)
		if err := m.SetInt64(value, 5); err != nil {
			t.Fatal(err)
		}
		a, err := c.NewInequality(x, value, true, false)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.AddClause([]*RationalInequality{a, nil}); !errors.Is(err, arena.ErrInvalidSlot) {
			t.Fatal(err)
		}
		if err := c.DiscardUnreferencedInequality(a); err != nil {
			t.Fatal(err)
		}
		if len(c.inequalities) != 0 || !a.closed {
			t.Fatal("unpublished rational atom retained numeral owner")
		}
		if err := c.DiscardUnreferencedInequality(a); !errors.Is(err, arena.ErrInvalidSlot) {
			t.Fatal("duplicate rational release", err)
		}
		kept, err := c.NewInequality(x, value, true, false)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.AddClause([]*RationalInequality{kept}); err != nil {
			t.Fatal(err)
		}
		if err := c.DiscardUnreferencedInequality(kept); err != nil {
			t.Fatal(err)
		}
		if len(c.inequalities) != 1 || kept.refs != 1 {
			t.Fatal("published rational atom lost its clause reference")
		}
	})
	t.Run("hardware", func(t *testing.T) {
		m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
		if err != nil {
			t.Fatal(err)
		}
		c, err := NewHardwareContext(m, &control.Reslimit{Limit: 100}, 0, 1, 8192)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if e := c.Close(); e != nil {
				t.Error(e)
			}
		})
		x, err := c.NewVariable(false)
		if err != nil {
			t.Fatal(err)
		}
		var value numeral.HardwareFloat
		if err := m.SetInt32(&value, 5); err != nil {
			t.Fatal(err)
		}
		a, err := c.NewInequality(x, &value, true, false)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.AddClause([]*HardwareInequality{a, nil}, false, true); !errors.Is(err, arena.ErrInvalidSlot) {
			t.Fatal(err)
		}
		if err := c.DiscardUnreferencedInequality(a); err != nil {
			t.Fatal(err)
		}
		if len(c.inequalities) != 0 || a.context != nil {
			t.Fatal("unpublished hardware atom remained live")
		}
		if err := c.DiscardUnreferencedInequality(a); !errors.Is(err, arena.ErrInvalidSlot) {
			t.Fatal("duplicate hardware release", err)
		}
		kept, err := c.NewInequality(x, &value, true, false)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.AddClause([]*HardwareInequality{kept}, false, true); err != nil {
			t.Fatal(err)
		}
		if err := c.DiscardUnreferencedInequality(kept); err != nil {
			t.Fatal(err)
		}
		if len(c.inequalities) != 1 || kept.refs != 1 {
			t.Fatal("published hardware atom lost its clause reference")
		}
	})
	t.Run("fast", func(t *testing.T) {
		m, c, value, _ := fastMonomialFixture(t, []bool{false})
		if err := m.SetInt32(value, 5); err != nil {
			t.Fatal(err)
		}
		a, err := c.NewInequality(0, value, true, false)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.AddClause([]*FastInequality{a, nil}, false, true); !errors.Is(err, arena.ErrInvalidSlot) {
			t.Fatal(err)
		}
		if err := c.DiscardUnreferencedInequality(a); err != nil {
			t.Fatal(err)
		}
		if len(c.inequalities) != 0 || a.owner != nil || a.context != nil {
			t.Fatal("unpublished fast atom retained numeral owner")
		}
		if err := c.DiscardUnreferencedInequality(a); !errors.Is(err, arena.ErrInvalidSlot) {
			t.Fatal("duplicate fast release", err)
		}
		kept, err := c.NewInequality(0, value, true, false)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.AddClause([]*FastInequality{kept}, false, true); err != nil {
			t.Fatal(err)
		}
		if err := c.DiscardUnreferencedInequality(kept); err != nil {
			t.Fatal(err)
		}
		if len(c.inequalities) != 1 || kept.refs != 1 {
			t.Fatal("published fast atom lost its clause reference")
		}
	})
	t.Run("arbitrary", func(t *testing.T) {
		f := newArbitraryClauseFixture(t)
		c := f.context
		a, err := c.NewInequality(f.x, &f.five, true, false)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.AddClause([]*ArbitraryInequality{a, nil}); !errors.Is(err, arena.ErrInvalidSlot) {
			t.Fatal(err)
		}
		if err := c.DiscardUnreferencedInequality(a); err != nil {
			t.Fatal(err)
		}
		if len(c.inequalities) != 0 || a.context != nil {
			t.Fatal("unpublished arbitrary atom retained numeral owner")
		}
		if err := c.DiscardUnreferencedInequality(a); !errors.Is(err, arena.ErrInvalidSlot) {
			t.Fatal("duplicate arbitrary release", err)
		}
		kept, err := c.NewInequality(f.x, &f.five, true, false)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.AddClause([]*ArbitraryInequality{kept}); err != nil {
			t.Fatal(err)
		}
		if err := c.DiscardUnreferencedInequality(kept); err != nil {
			t.Fatal(err)
		}
		if len(c.inequalities) != 1 || kept.refs != 1 {
			t.Fatal("published arbitrary atom lost its clause reference")
		}
	})
}
