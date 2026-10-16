// Source: src/util/params.cpp:559-565,598-613. In a shared self-copy,
// init's clone stops before old->dec_ref when copying an entry fails.
package params

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestParamsSelfCopyStoppedDetachRetainsSharedPredecessor(t *testing.T) {
	const want = "(params a 11 b 22 c 33 d 44 e 55)"
	seenStopped := false
	seenSuccess := false
	for fault := int64(0); fault < 48 && !seenSuccess; fault++ {
		t.Run(strconv.FormatInt(fault, 10), func(t *testing.T) {
			c, sy, _ := paramsTestOwner(t)
			v, e := c.NewValue()
			paramsMust(t, e)
			shadow, e := c.NewValue()
			paramsMust(t, e)
			for i, name := range []string{"a", "b", "c", "d", "e"} {
				paramsMust(t, c.SetUInt(&v, internSymbol(t, sy, name), uint32(11*(i+1))))
			}
			paramsMust(t, c.Assign(&shadow, &v))
			oldSlot := v.Slot
			c.BudgetField.FailAfter(fault)
			e = c.Copy(&v, &v)
			c.BudgetField.FailAfter(-1)
			if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
			if e == nil {
				seenSuccess = true
				if v.Slot == oldSlot {
					t.Fatal("successful self-copy retained shared backing")
				}
			} else if v.Slot != oldSlot {
				seenStopped = true
				if c.Blocks[oldSlot-1].Refs != 2 {
					t.Fatal("failed init consumed the old shared reference")
				}
				receipt, err := c.StoppedSource(&v)
				paramsMust(t, err)
				paramsMust(t, c.CleanupSource(receipt))
				if c.Blocks[oldSlot-1].Refs != 1 {
					t.Fatal("cleanup did not release exactly one held reference")
				}
			} else if c.Blocks[oldSlot-1].Refs != 2 {
				t.Fatal("pre-publication refusal changed shared references")
			}
			got, err := c.Display(&shadow)
			paramsMust(t, err)
			if got != want {
				t.Fatalf("shared predecessor changed: %q", got)
			}
			paramsMust(t, c.Del(&v))
			paramsMust(t, c.Del(&shadow))
		})
	}
	if !seenStopped || !seenSuccess {
		t.Fatalf("missing stopped detach or completion: stopped=%v success=%v", seenStopped, seenSuccess)
	}
}
