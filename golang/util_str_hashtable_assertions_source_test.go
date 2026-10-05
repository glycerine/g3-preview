// Source: src/util/hashtable.h:159-215,249,300-303,371,395,422,451,559,643-644;
// src/util/debug.h:60-65,76-89.
// Source author: Leonardo de Moura (2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/utilhash"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestStrHashtableBoundConstructorConditionBeforeEntryAllocation(t *testing.T) {
	for _, action := range []string{"C", "T"} {
		t.Run(action, func(t *testing.T) {
			p, d, _, in, s := permutationActionFixture(t, action)
			before := p.forward.column.raw.runtime.State().GlobalCount
			h, e := utilhash.NewStrHashtable(d.region, utilhash.StrHashtableConfig{Budget: d.budget, Runtime: p.forward.column.raw.runtime, InitialCapacity: 3, Assertions: d})
			if h != nil {
				t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
			}
			state, re := in.State()
			paramsMust(t, re)
			out, re := os.ReadFile(s.stderr.file.Name())
			paramsMust(t, re)
			if satStorageSourceDebug {
				if !strings.Contains(string(out), "File: src/util/hashtable.h\nLine: 249\nis_power_of_two(initial_capacity)\n") || state.Position != 1 {
					t.Fatal(string(out), state)
				}
				if action == "T" {
					var ex DefaultException
					if !errors.As(e, &ex) || p.forward.column.raw.runtime.State().GlobalCount != before+1 {
						t.Fatal("T before array", e, p.forward.column.raw.runtime.State())
					}
				} else if e != nil || h.Capacity() != 3 || p.forward.column.raw.runtime.State().GlobalCount != before+2 {
					t.Fatal("C resumes array publication", h, e, p.forward.column.raw.runtime.State())
				}
			} else if e != nil || state.Position != 0 || len(out) != 0 || h.Capacity() != 3 {
				t.Fatal("release macro erased", h, e, state, string(out))
			}
		})
	}
}

func TestStrHashtableBoundInvariantContinuesLiteralSites(t *testing.T) {
	p, d, _, in, s := permutationActionFixture(t, "CC")
	r := p.forward.column.raw.runtime
	h, e := utilhash.NewStrHashtable(d.region, utilhash.StrHashtableConfig{Budget: d.budget, Runtime: r, Assertions: d})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
	roots, e := h.RuntimeRoots()
	paramsMust(t, e)
	// Explicit malformed source count-domain fixture, not a valid solver state.
	paramsMust(t, r.WriteUint32(roots.Object, 12, 1))
	paramsMust(t, r.WriteUint32(roots.Object, 16, 1))
	yes, e := h.CheckInvariantChecked()
	if satStorageSourceDebug {
		paramsMust(t, e)
		if !yes {
			t.Fatal("source return true after C")
		}
		state, e := in.State()
		paramsMust(t, e)
		if state.Position != 2 {
			t.Fatal(state)
		}
		out, e := os.ReadFile(s.stderr.file.Name())
		paramsMust(t, e)
		want := "ASSERTION VIOLATION\nFile: src/util/hashtable.h\nLine: 643\nnum_deleted == m_num_deleted\n" + debugSourcePrompt + "ASSERTION VIOLATION\nFile: src/util/hashtable.h\nLine: 644\nnum_used == m_size\n" + debugSourcePrompt
		if string(out) != want {
			t.Fatal(string(out), want)
		}
	} else if e != nil || yes {
		t.Fatal("release inspection adapter", yes, e)
	}
}

func TestStrHashtableBoundDisabledNonPowerCapacityLinearProbe(t *testing.T) {
	p, d, _, in, s := permutationActionFixture(t, "T")
	paramsMust(t, d.EnableAssertions(false))
	h, e := utilhash.NewStrHashtable(d.region, utilhash.StrHashtableConfig{Budget: d.budget, Runtime: p.forward.column.raw.runtime, InitialCapacity: 3, Assertions: d})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
	// Same hash-start bucket exercises the source end/begin traversal at cap3;
	// &mask iteration would keep skipping the middle physical bucket.
	var keys []utilhash.StrKey
	for n := 0; len(keys) < 3; n++ {
		text := fmt.Sprintf("linear-%d", n)
		if utilhash.String(text, 17)&2 != 2 {
			continue
		}
		key := debugTag(t, d.region, text)
		keys = append(keys, key)
		paramsMust(t, h.Insert(key))
	}
	for _, key := range keys {
		found, e := h.Contains(key)
		paramsMust(t, e)
		if !found {
			t.Fatal(key)
		}
	}
	for _, key := range keys {
		paramsMust(t, h.Remove(key))
	}
	state, e := in.State()
	paramsMust(t, e)
	out, e := os.ReadFile(s.stderr.file.Name())
	paramsMust(t, e)
	if state.Position != 0 || len(out) != 0 {
		t.Fatal("erased source switch", state, string(out))
	}
}

func TestStrHashtableBoundResetExactDeletedPrefixBeforeAssertion(t *testing.T) {
	for _, action := range []string{"CCC", "CT"} {
		t.Run(action, func(t *testing.T) {
			p, d, _, in, s := permutationActionFixture(t, action)
			r := p.forward.column.raw.runtime
			h, e := utilhash.NewStrHashtable(d.region, utilhash.StrHashtableConfig{Budget: d.budget, Runtime: r, InitialCapacity: 48, Assertions: d})
			paramsMust(t, e)
			t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
			paramsMust(t, h.Insert(debugTag(t, d.region, "kept-before-reset")))
			before, e := h.RuntimeRoots()
			paramsMust(t, e)
			count := r.State().GlobalCount
			e = h.Reset()
			roots, re := h.RuntimeRoots()
			paramsMust(t, re)
			if satStorageSourceDebug && action == "CT" {
				var ex DefaultException
				if !errors.As(e, &ex) || h.Capacity() != 48 || h.Size() != 1 || roots.Table != (memory.Block{}) || r.State().GlobalCount != count {
					t.Fatal("stop after table deletion before halve", e, h.Stats(), roots)
				}
			} else {
				paramsMust(t, e)
				if h.Capacity() != 24 || h.Size() != 0 || roots.Table == (memory.Block{}) || r.State().GlobalCount != count+1 {
					t.Fatal("source resumed replacement", h.Stats(), roots)
				}
			}
			if _, e = r.Size(before.Table); e == nil {
				t.Fatal("source old table remains live")
			}
			state, e := in.State()
			paramsMust(t, e)
			out, e := os.ReadFile(s.stderr.file.Name())
			paramsMust(t, e)
			if satStorageSourceDebug {
				want := uint64(3)
				if action == "CT" {
					want = 2
				}
				if state.Position != want || !strings.Contains(string(out), "Line: 301\nis_power_of_two(m_capacity)\n") {
					t.Fatal(state, string(out))
				}
			} else if state.Position != 0 || len(out) != 0 {
				t.Fatal(state, string(out))
			}
		})
	}
}

func TestStrHashtableBoundClosedProviderReadOnlyAtReachedMacro(t *testing.T) {
	p, d, _, _, _ := permutationActionFixture(t, "T")
	r := p.forward.column.raw.runtime
	h, e := utilhash.NewStrHashtable(d.region, utilhash.StrHashtableConfig{Budget: d.budget, Runtime: r, Assertions: d})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
	paramsMust(t, d.Close())
	// Actual empty Reset returns before every source SASSERT.
	paramsMust(t, h.Reset())
	before := r.State().GlobalCount
	next, e := utilhash.NewStrHashtable(d.region, utilhash.StrHashtableConfig{Budget: d.budget, Runtime: r, Assertions: d})
	if next != nil {
		t.Cleanup(func() { paramsMust(t, next.CloseChecked()) })
	}
	if satStorageSourceDebug {
		if !errors.Is(e, arena.ErrClosed) || next != nil || r.State().GlobalCount != before+1 {
			t.Fatal("wrapper before closed flag, no array", next, e, r.State())
		}
	} else {
		paramsMust(t, e)
		if next == nil || r.State().GlobalCount != before+2 {
			t.Fatal(next, r.State())
		}
	}
}

func TestStrHashtableBoundUnreachableIgnoresSwitchAndContinuesSource(t *testing.T) {
	for _, core := range []bool{false, true} {
		t.Run(fmt.Sprint(core), func(t *testing.T) {
			p, d, _, in, s := permutationActionFixture(t, "C")
			r := p.forward.column.raw.runtime
			paramsMust(t, d.EnableAssertions(false))
			h, e := utilhash.NewStrHashtable(d.region, utilhash.StrHashtableConfig{Budget: d.budget, Runtime: r, InitialCapacity: 3, Assertions: d})
			paramsMust(t, e)
			t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
			for i := 0; i < 3; i++ {
				paramsMust(t, h.Insert(debugTag(t, d.region, fmt.Sprintf("full-%d", i))))
			}
			roots, e := h.RuntimeRoots()
			paramsMust(t, e)
			// Malformed native count domain prevents source expansion and reaches the
			// actual exhausted probe branch. No iteration counter or fake callback.
			paramsMust(t, r.WriteUint32(roots.Object, 12, 0))
			key := debugTag(t, d.region, "absent-from-full")
			if core {
				_, _, e = h.InsertIfNotThereCore(key)
			} else {
				e = h.Insert(key)
			}
			state, re := in.State()
			paramsMust(t, re)
			out, re := os.ReadFile(s.stderr.file.Name())
			paramsMust(t, re)
			if satStorageSourceDebug {
				paramsMust(t, e)
				line := 395
				if core {
					line = 451
				}
				want := fmt.Sprintf("ASSERTION VIOLATION\nFile: src/util/hashtable.h\nLine: %d\nUNREACHABLE CODE WAS REACHED.\n", line) + debugSourcePrompt
				if string(out) != want || state.Position != 1 {
					t.Fatal(string(out), want, state)
				}
			} else if !errors.Is(e, utilhash.ErrStrHashtableUnreachable) || state.Position != 0 || len(out) != 0 {
				t.Fatal("GNU undefined-domain typed guard erases provider", e, state, string(out))
			}
		})
	}
}

func strHashtableBindingPhysical(t *testing.T, p *RuntimePermutation, d *DebugContext, a *DebugActionContext, in *NumericCharInput, s *MemoryStandardStreams, owners ...*utilhash.StrHashtable) {
	t.Helper()
	tokens := []memory.Block{p.inverse.column.raw.block}
	cold := uint64(0)
	for _, h := range owners {
		if h == nil {
			continue
		}
		roots, e := h.RuntimeRoots()
		paramsMust(t, e)
		for _, token := range append([]memory.Block{roots.Object, roots.Table}, roots.Acquisitions...) {
			if token != (memory.Block{}) {
				tokens = append(tokens, token)
			}
		}
		v := reflect.ValueOf(h).Elem()
		cold += v.FieldByName("rootBytes").Uint() + v.FieldByName("directoryBytes").Uint() + roots.AcquisitionBytes
	}
	satAssignmentAuditBlocks(t, &p.forward.column.raw, tokens...)
	r := p.forward.column.raw.runtime
	x := d.region.Stats()
	z := x.Descriptors
	cold += p.header + unsignedCopyCold(t, &p.forward) + unsignedCopyCold(t, &p.inverse) + d.ownerBytes + debugTagColdBytes(d) + x.OwnerBytes + x.PayloadBytes + x.PageHeaderBytes + x.DirectoryBytes + x.LogBytes + x.ScopeBytes + z.OwnerBytes + z.RecordBytes + z.MetadataBytes + z.DirectoryBytes + a.header + in.header + s.header + s.stdout.Stats().OwnerBytes + s.stderr.Stats().OwnerBytes
	if p.budget.Used() != r.Stats().TotalBytes+cold {
		t.Fatal("exact named class assertion graph", p.budget.Used(), r.Stats().TotalBytes, cold)
	}
}

func TestStrHashtableBoundMoveAndCopyFlagAfterActualNewTable(t *testing.T) {
	for _, copying := range []bool{false, true} {
		t.Run(fmt.Sprint(copying), func(t *testing.T) {
			p, d, a, in, s := permutationActionFixture(t, "T")
			r := p.forward.column.raw.runtime
			h, e := utilhash.NewStrHashtable(d.region, utilhash.StrHashtableConfig{Budget: d.budget, Runtime: r, Assertions: d})
			paramsMust(t, e)
			t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
			for i := 0; i < 7; i++ {
				paramsMust(t, h.Insert(debugTag(t, d.region, fmt.Sprintf("move-%d", i))))
			}
			key := debugTag(t, d.region, "eighth")
			paramsMust(t, d.Close())
			before := r.State().GlobalCount
			var out *utilhash.StrHashtable
			if copying {
				out, e = h.Clone()
				if out != nil {
					t.Cleanup(func() { paramsMust(t, out.CloseChecked()) })
				}
			} else {
				e = h.Insert(key)
			}
			if satStorageSourceDebug {
				if !errors.Is(e, arena.ErrClosed) || r.State().GlobalCount != before+map[bool]int64{false: 1, true: 2}[copying] {
					t.Fatal("new array before actual source switch", e, r.State())
				}
				if copying {
					roots, re := out.RuntimeRoots()
					paramsMust(t, re)
					if roots.Object != (memory.Block{}) || roots.Table == (memory.Block{}) {
						t.Fatal("uncompleted copy new-expression unwind", roots)
					}
				} else {
					roots, re := h.RuntimeRoots()
					paramsMust(t, re)
					if len(roots.Acquisitions) != 1 || h.Size() != 7 || h.Capacity() != 8 {
						t.Fatal("unjoined NEW before move assertion", roots, h.Stats())
					}
				}
			} else {
				paramsMust(t, e)
			}
			strHashtableBindingPhysical(t, p, d, a, in, s, h, out)
		})
	}
}

func TestStrHashtableBoundRemoveExhaustionFlagAndEarlyFreeErasure(t *testing.T) {
	for _, full := range []bool{false, true} {
		t.Run(fmt.Sprint(full), func(t *testing.T) {
			p, d, a, in, s := permutationActionFixture(t, "T")
			r := p.forward.column.raw.runtime
			paramsMust(t, d.EnableAssertions(false))
			h, e := utilhash.NewStrHashtable(d.region, utilhash.StrHashtableConfig{Budget: d.budget, Runtime: r, InitialCapacity: 3, Assertions: d})
			paramsMust(t, e)
			t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
			if full {
				for i := 0; i < 3; i++ {
					paramsMust(t, h.Insert(debugTag(t, d.region, fmt.Sprintf("remove-full-%d", i))))
				}
			}
			key := debugTag(t, d.region, "absent-remove")
			paramsMust(t, d.EnableAssertions(true))
			paramsMust(t, d.Close())
			e = h.Remove(key)
			if satStorageSourceDebug && full {
				if !errors.Is(e, arena.ErrClosed) {
					t.Fatal("exhausted remove reaches source559 flag", e)
				}
			} else {
				paramsMust(t, e)
			}
			strHashtableBindingPhysical(t, p, d, a, in, s, h)
		})
	}
}

func TestStrHashtableBoundDisabledInvariantAndUnusedTransport(t *testing.T) {
	p, d, a, in, s := permutationActionFixture(t, "T")
	r := p.forward.column.raw.runtime
	h, e := utilhash.NewStrHashtable(d.region, utilhash.StrHashtableConfig{Budget: d.budget, Runtime: r, Assertions: d})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
	roots, e := h.RuntimeRoots()
	paramsMust(t, e)
	paramsMust(t, r.WriteUint32(roots.Object, 12, 19))
	paramsMust(t, r.WriteUint32(roots.Object, 16, 29))
	paramsMust(t, d.EnableAssertions(false))
	paramsMust(t, a.Close())
	paramsMust(t, s.Close())
	yes, e := h.CheckInvariantChecked()
	paramsMust(t, e)
	if satStorageSourceDebug && !yes {
		t.Fatal("disabled conditions erased")
	}
	state, e := in.State()
	paramsMust(t, e)
	if state.Position != 0 {
		t.Fatal(state)
	}
}

func TestStrHashtableBoundBorrowBudgetValidationBeforeReservation(t *testing.T) {
	p, d, _, _, _ := permutationActionFixture(t, "T")
	foreign, _, _ := debugFixture(t)
	before := p.budget.Used()
	p.budget.FailAfter(0)
	h, e := utilhash.NewStrHashtable(d.region, utilhash.StrHashtableConfig{Budget: p.budget, Runtime: p.forward.column.raw.runtime, Assertions: foreign})
	if h != nil || !errors.Is(e, arena.ErrConfig) || p.budget.Used() != before {
		t.Fatal("foreign cold binding before reservation", h, e)
	}
	p.budget.FailAfter(-1)
}

func TestStrHashtableBoundEveryConstructorReservationImmediateGraph(t *testing.T) {
	first := int64(-1)
	for ordinal := int64(0); ordinal < 24; ordinal++ {
		p, d, a, in, s := permutationActionFixture(t, "C")
		r := p.forward.column.raw.runtime
		p.budget.FailAfter(ordinal)
		h, e := utilhash.NewStrHashtable(d.region, utilhash.StrHashtableConfig{Budget: d.budget, Runtime: r, InitialCapacity: 3, Assertions: d})
		strHashtableBindingPhysical(t, p, d, a, in, s, h)
		if e == nil {
			first = ordinal
			if h.Capacity() != 3 {
				t.Fatal(h.Capacity())
			}
		}
		p.budget.FailAfter(-1)
		if h != nil {
			paramsMust(t, h.CloseChecked())
		}
		if first >= 0 {
			break
		}
	}
	if first < 0 {
		t.Fatal("real reservation horizon missing")
	}
	t.Log("first bound constructor success", first)
}
