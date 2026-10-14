package params

import "testing"

// src/util/params.cpp:677-689,799-828: reset on an allocated-empty shared
// backing still finalizes its vector, without detaching either reference.
func TestParamsSharedAllocatedEmptyResetFinalizesBacking(t *testing.T) {
	c, symbols, _ := paramsTestOwner(t)
	a, err := c.NewValue()
	paramsMust(t, err)
	b, err := c.NewValue()
	paramsMust(t, err)
	t.Cleanup(func() {
		paramsMust(t, c.Del(&a))
		paramsMust(t, c.Del(&b))
	})
	key := internSymbol(t, symbols, "key")
	paramsMust(t, c.SetUInt(&a, key, 7))
	paramsMust(t, c.Assign(&b, &a))
	paramsMust(t, c.ResetKey(&a, key))
	block := &c.Blocks[a.Slot-1]
	if a != b || block.Refs != 2 || block.Size != 0 || block.capacity == 0 {
		t.Fatal("fixture lost shared allocated-empty backing", a, b, *block)
	}
	paramsMust(t, c.Reset(&a))
	if a != b || !block.Live || block.Refs != 2 || block.Size != 0 || block.capacity != 0 || block.entries != (paramsEntriesRef{}) {
		t.Fatal("source reset did not finalize shared empty vector", a, b, *block)
	}
	paramsMust(t, c.SetUInt(&a, key, 9))
	if a == b {
		t.Fatal("setter failed to detach after shared reset")
	}
	if found, err := c.Contains(&b, key); err != nil || found {
		t.Fatal("older alias changed after detach", found, err)
	}
}
