// Source: src/util/params.cpp:550-595,960-1035.
package params

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestParamsReferenceCopyAllKindsSharedTargetOrderedMerge(t *testing.T) {
	c, sy, q := paramsTestOwner(t)
	src, e := c.NewValue()
	paramsMust(t, e)
	dst, e := c.NewValue()
	paramsMust(t, e)
	shadow, e := c.NewValue()
	paramsMust(t, e)
	b := internSymbol(t, sy, "b")
	u := internSymbol(t, sy, "u")
	d := internSymbol(t, sy, "d")
	s := internSymbol(t, sy, "s")
	z := internSymbol(t, sy, "z")
	r := internSymbol(t, sy, "r")
	old := internSymbol(t, sy, "old")
	sym := internSymbol(t, sy, "sym")
	paramsMust(t, c.SetBool(&src, b, true))
	paramsMust(t, c.SetUInt(&src, u, 7))
	paramsMust(t, c.SetDouble(&src, d, 1.5))
	text := "hello"
	paramsMust(t, c.SetString(&src, s, &text))
	paramsMust(t, c.SetSymbol(&src, z, sym))
	value, e := q.NewValue()
	paramsMust(t, e)
	paramsMust(t, q.SetFractionSigned(&value, -7, 3))
	paramsMust(t, c.SetRat(&src, r, &value))
	paramsMust(t, q.Del(&value))
	paramsMust(t, c.SetUInt(&dst, u, 99))
	before := "before"
	paramsMust(t, c.SetString(&dst, old, &before))
	paramsMust(t, c.Assign(&shadow, &dst))
	paramsMust(t, c.Copy(&dst, &src))
	paramsMust(t, c.Del(&src))
	got, e := c.Display(&dst)
	paramsMust(t, e)
	if want := "(params u 7 old before b true d 1.5 s hello z sym r -7/3)"; got != want {
		t.Fatalf("source ordered merge: %q", got)
	}
	unchanged, e := c.Display(&shadow)
	paramsMust(t, e)
	if unchanged != "(params u 99 old before)" {
		t.Fatalf("shared predecessor changed: %q", unchanged)
	}
	paramsMust(t, c.Del(&dst))
	paramsMust(t, c.Del(&shadow))
}

func TestParamsReferenceCopyAllKindsSharedTargetRefusalPrefixes(t *testing.T) {
	const predecessor = "(params u 99 old before)"
	const source = "(params b true u 7 d 1.5 s hello z sym r -7/3)"
	const complete = "(params u 7 old before b true d 1.5 s hello z sym r -7/3)"
	// Each successful set_core step may replace an existing key or append a
	// new one. A stopped copy retains precisely the completed source prefix.
	merged := []string{
		predecessor,
		"(params u 99 old before b true)",
		"(params u 7 old before b true)",
		"(params u 7 old before b true d 1.5)",
		"(params u 7 old before b true d 1.5 s hello)",
		"(params u 7 old before b true d 1.5 s hello z sym)",
		complete,
	}
	seenDetachedStop := false
	seenSuccess := false
	maxStoppedPrefix := -1
	for fault := int64(0); fault < 64 && !seenSuccess; fault++ {
		t.Run(strconv.FormatInt(fault, 10), func(t *testing.T) {
			c, sy, q := paramsTestOwner(t)
			src, e := c.NewValue()
			paramsMust(t, e)
			dst, e := c.NewValue()
			paramsMust(t, e)
			shadow, e := c.NewValue()
			paramsMust(t, e)
			b := internSymbol(t, sy, "b")
			u := internSymbol(t, sy, "u")
			d := internSymbol(t, sy, "d")
			s := internSymbol(t, sy, "s")
			z := internSymbol(t, sy, "z")
			r := internSymbol(t, sy, "r")
			old := internSymbol(t, sy, "old")
			sym := internSymbol(t, sy, "sym")
			paramsMust(t, c.SetBool(&src, b, true))
			paramsMust(t, c.SetUInt(&src, u, 7))
			paramsMust(t, c.SetDouble(&src, d, 1.5))
			text := "hello"
			paramsMust(t, c.SetString(&src, s, &text))
			paramsMust(t, c.SetSymbol(&src, z, sym))
			value, e := q.NewValue()
			paramsMust(t, e)
			paramsMust(t, q.SetFractionSigned(&value, -7, 3))
			paramsMust(t, c.SetRat(&src, r, &value))
			paramsMust(t, q.Del(&value))
			paramsMust(t, c.SetUInt(&dst, u, 99))
			before := "before"
			paramsMust(t, c.SetString(&dst, old, &before))
			paramsMust(t, c.Assign(&shadow, &dst))
			oldSlot := dst.Slot
			c.BudgetField.FailAfter(fault)
			e = c.Copy(&dst, &src)
			c.BudgetField.FailAfter(-1)
			if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
			unchanged, err := c.Display(&shadow)
			paramsMust(t, err)
			if unchanged != predecessor {
				t.Fatalf("shared predecessor changed: %q", unchanged)
			}
			unchanged, err = c.Display(&src)
			paramsMust(t, err)
			if unchanged != source {
				t.Fatalf("copy source changed: %q", unchanged)
			}
			if e != nil && dst.Slot != oldSlot {
				if receipt, err := c.StoppedSource(&dst); err == nil {
					paramsMust(t, c.CleanupSource(receipt))
				} else if !errors.Is(err, ErrParamsPrecondition) {
					t.Fatal(err)
				} else {
					seenDetachedStop = true
					got, err := c.Display(&dst)
					paramsMust(t, err)
					found := -1
					for i, want := range merged {
						if got == want {
							found = i
						}
					}
					if found < 0 {
						t.Fatalf("copy stopped outside a completed source prefix: %q", got)
					}
					if found > maxStoppedPrefix {
						maxStoppedPrefix = found
					}
				}
			}
			if e == nil {
				seenSuccess = true
				got, err := c.Display(&dst)
				paramsMust(t, err)
				if got != complete {
					t.Fatalf("completed copy: %q", got)
				}
			}
			paramsMust(t, c.Del(&src))
			paramsMust(t, c.Del(&dst))
			paramsMust(t, c.Del(&shadow))
		})
	}
	if !seenDetachedStop || maxStoppedPrefix < 2 || !seenSuccess {
		t.Fatalf("refusal sweep missed copied prefix or success: stop=%v prefix=%d success=%v", seenDetachedStop, maxStoppedPrefix, seenSuccess)
	}
}

func TestParamsReferenceCopyFromSharedSourceStoppedDetach(t *testing.T) {
	const want = "(params r -7/3 s live b true)"
	seenPublishedStop := false
	seenSuccess := false
	for fault := int64(0); fault < 48 && !seenSuccess; fault++ {
		t.Run(strconv.FormatInt(fault, 10), func(t *testing.T) {
			c, sy, q := paramsTestOwner(t)
			src, e := c.NewValue()
			paramsMust(t, e)
			dst, e := c.NewValue()
			paramsMust(t, e)
			r := internSymbol(t, sy, "r")
			s := internSymbol(t, sy, "s")
			b := internSymbol(t, sy, "b")
			value, e := q.NewValue()
			paramsMust(t, e)
			paramsMust(t, q.SetFractionSigned(&value, -7, 3))
			paramsMust(t, c.SetRat(&src, r, &value))
			paramsMust(t, q.Del(&value))
			text := "live"
			paramsMust(t, c.SetString(&src, s, &text))
			paramsMust(t, c.SetBool(&src, b, true))
			paramsMust(t, c.Assign(&dst, &src))
			oldSlot := src.Slot
			if dst.Slot != oldSlot || c.Blocks[oldSlot-1].Refs != 2 {
				t.Fatal("source alias not established")
			}
			c.BudgetField.FailAfter(fault)
			e = c.Copy(&dst, &src)
			c.BudgetField.FailAfter(-1)
			if e != nil && !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
			if e != nil && dst.Slot != oldSlot {
				seenPublishedStop = true
				receipt, err := c.StoppedSource(&dst)
				paramsMust(t, err)
				if receipt.Invocation() == 0 {
					t.Fatal("missing actual detach invocation")
				}
				paramsMust(t, c.CleanupSource(receipt))
			}
			if got, err := c.Display(&src); err != nil || got != want {
				t.Fatalf("source changed during detach: %q, %v", got, err)
			}
			if e == nil {
				seenSuccess = true
				if got, err := c.Display(&dst); err != nil || got != want || dst.Slot == oldSlot {
					t.Fatalf("completed detach: %q, %v, slot=%d", got, err, dst.Slot)
				}
			}
			if dst.Slot != oldSlot && c.Blocks[oldSlot-1].Refs != 1 {
				t.Fatal("cleanup consumed source alias or retained destination alias")
			}
			paramsMust(t, c.Del(&dst))
			paramsMust(t, c.Del(&src))
		})
	}
	if !seenPublishedStop || !seenSuccess {
		t.Fatalf("refusal sweep missed published stop or success: stop=%v success=%v", seenPublishedStop, seenSuccess)
	}
}
