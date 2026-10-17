// Source: src/util/params.cpp:559-610.
package params

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestParamsReferenceCopyEmptySourceDetachesSharedDestination(t *testing.T) {
	c, sy, _ := paramsTestOwner(t)
	key := internSymbol(t, sy, "retained")
	dst, err := c.NewValue()
	paramsMust(t, err)
	paramsMust(t, c.SetUInt(&dst, key, 7))
	alias, err := c.NewValue()
	paramsMust(t, err)
	paramsMust(t, c.Assign(&alias, &dst))
	empty := c.EmptyValue()
	nullDestination := c.EmptyValue()
	c.BudgetField.FailAfter(0)
	paramsMust(t, c.Copy(&nullDestination, &empty))
	if nullDestination != empty {
		t.Fatal("null destination must take the empty assignment branch", nullDestination, empty)
	}
	old := dst

	err = c.Copy(&dst, &empty)
	if !errors.Is(err, arena.ErrInjectedFailure) || dst.Slot == 0 || dst.Slot == old.Slot || c.Blocks[old.Slot-1].Refs != 2 || c.Blocks[dst.Slot-1].Size != 0 {
		t.Fatal("stopped COW copy must keep its published destination and old references", err, dst, old, c.Blocks[old.Slot-1].Refs)
	}
	c.BudgetField.FailAfter(-1)
	receipt, err := c.StoppedSource(&dst)
	paramsMust(t, err)
	paramsMust(t, c.CleanupSource(receipt))
	paramsMust(t, c.Del(&dst))
	dst, err = c.NewValue()
	paramsMust(t, err)
	paramsMust(t, c.Assign(&dst, &alias))
	paramsMust(t, c.Copy(&dst, &empty))
	if dst.Slot == old.Slot || c.Blocks[old.Slot-1].Refs != 1 || c.Blocks[dst.Slot-1].Refs != 1 || empty.Slot != 0 {
		t.Fatal("empty source must still detach a shared destination", dst, old, empty)
	}
	for _, value := range []*ParamsRef{&dst, &alias} {
		got, e := c.GetUInt(value, key, 0)
		if e != nil || got != 7 {
			t.Fatal("COW copy lost the existing entry", got, e)
		}
	}
	paramsMust(t, c.SetUInt(&dst, key, 9))
	if got, e := c.GetUInt(&alias, key, 0); e != nil || got != 7 {
		t.Fatal("detached destination changed its predecessor", got, e)
	}

	unique := dst
	c.BudgetField.FailAfter(0)
	paramsMust(t, c.Append(&dst, &empty))
	c.BudgetField.FailAfter(-1)
	if dst != unique {
		t.Fatal("empty-source append allocated a unique destination", dst, unique)
	}
	if got, e := c.GetUInt(&dst, key, 0); e != nil || got != 9 {
		t.Fatal("empty-source append changed the unique entry", got, e)
	}
	paramsMust(t, c.Del(&dst))
	paramsMust(t, c.Del(&alias))
}
