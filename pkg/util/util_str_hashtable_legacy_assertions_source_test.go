// Source: src/util/hashtable.h:159-215,249,371,395,422,451,559,643-644;
// src/util/debug.h:60-65,76-89.
// Source author: Leonardo de Moura (2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestStrHashtableLegacyBoundConstructorAndLinearSourceContinuation(t *testing.T) {
	for _, action := range []string{"C", "T"} {
		t.Run(action, func(t *testing.T) {
			_, d, _, in, streams := permutationActionFixture(t, action)
			before := d.BudgetField.Used()
			h, e := NewStrHashtable(d.region, StrHashtableConfig{Budget: d.BudgetField, InitialCapacity: 3, Assertions: d})
			if h != nil {
				t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
			}
			state, re := in.State()
			paramsMust(t, re)
			out, re := os.ReadFile(streams.StderrField.File.Name())
			paramsMust(t, re)
			if sourceDebug {
				if !strings.Contains(string(out), "Line: 249\nis_power_of_two(initial_capacity)\n") || state.Position != 1 {
					t.Fatal("actual source assertion before entry allocation", e, state, string(out))
				}
				if action == "T" {
					var ex DefaultException
					if !errors.As(e, &ex) || d.BudgetField.Used() != before {
						t.Fatal("throw constructor owning prefix", e, d.BudgetField.Used(), before)
					}
					return
				}
			} else if state.Position != 0 || len(out) != 0 {
				t.Fatal("release erased provider", state, string(out))
			}
			paramsMust(t, e)
			if h == nil || h.Capacity() != 3 {
				t.Fatal("C/release actual array continuation", h, e)
			}
			paramsMust(t, d.EnableAssertions(false))
			var keys []StrKey
			for n := 0; len(keys) < 3; n++ {
				text := fmt.Sprintf("legacy-linear-%d", n)
				if String(text, 17)&2 != 2 {
					continue
				}
				key := debugTag(t, d.region, text)
				keys = append(keys, key)
				paramsMust(t, h.Insert(key))
			}
			for _, key := range keys {
				yes, e := h.Contains(key)
				paramsMust(t, e)
				if !yes {
					t.Fatal("linear end/begin probe", key)
				}
				text, e := h.Text(key)
				paramsMust(t, e)
				if _, yes, e := h.FindCString(text); e != nil || !yes {
					t.Fatal("CString linear probe", text, e)
				}
				if _, yes, e := h.FindCStringBytes([]byte(text)); e != nil || !yes {
					t.Fatal("byte CString linear probe", text, e)
				}
			}
			copy, e := h.Clone()
			paramsMust(t, e)
			for _, key := range keys {
				yes, e := copy.Contains(key)
				paramsMust(t, e)
				if !yes {
					t.Fatal("copy source linear traversal")
				}
			}
			paramsMust(t, copy.CloseChecked())
			for _, key := range keys {
				paramsMust(t, h.Remove(key))
			}
			if h.Size() != 0 {
				t.Fatal("remove wrap pointer rather than mask", h.Size())
			}
		})
	}
}

func TestStrHashtableLegacyClosedAssertionProviderAtCopyAndInvariantSites(t *testing.T) {
	_, d, _, _, _ := permutationActionFixture(t, "T")
	h, e := NewStrHashtable(d.region, StrHashtableConfig{Budget: d.BudgetField, Assertions: d})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
	paramsMust(t, d.Close())
	// reset's actual early scalar branch does not read the assertion owner.
	paramsMust(t, h.Reset())
	_, e = h.CheckInvariantChecked()
	if sourceDebug {
		if !errors.Is(e, arena.ErrClosed) {
			t.Fatal("invariant reached source flag", e)
		}
	} else {
		paramsMust(t, e)
	}
	before := h.Stats()
	copy, e := h.Clone()
	if copy != nil {
		t.Cleanup(func() { paramsMust(t, copy.CloseChecked()) })
	}
	if sourceDebug {
		if !errors.Is(e, arena.ErrClosed) || copy == nil || copy.Stats().TableBytes != before.TableBytes {
			t.Fatal("copy table allocation precedes source flag", copy, e)
		}
	} else {
		paramsMust(t, e)
		if copy == nil {
			t.Fatal("release copy erased provider")
		}
	}
}

func TestStrHashtableLegacyGrowthSourceAssertionStopRetainsNewAndOld(t *testing.T) {
	_, d, _, _, _ := permutationActionFixture(t, "T")
	h, e := NewStrHashtable(d.region, StrHashtableConfig{Budget: d.BudgetField, Assertions: d})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
	for n := 0; n < 7; n++ {
		paramsMust(t, h.Insert(debugTag(t, d.region, fmt.Sprintf("growth-%d", n))))
	}
	key := debugTag(t, d.region, "growth-new")
	before := h.Stats()
	paramsMust(t, d.Close())
	used := d.BudgetField.Used()
	// Source NEW is allocated before move_table's first assertion flag read.
	// Closing the real provider is a Go lifetime guard-domain fixture.
	e = h.Insert(key)
	if sourceDebug {
		if !errors.Is(e, arena.ErrClosed) || h.Stats().TableBytes != before.TableBytes || h.Size() != before.Size || h.Stats().RetainedBytes <= 16*32 {
			t.Fatal("stopped move lost actual new/old owning ranges", e, h.Stats(), before)
		}
		ownedNew := legacyBoundPendingGraph(t, h)
		if ownedNew != h.Stats().RetainedBytes || d.BudgetField.Used() != used+ownedNew {
			t.Fatal("immediate independent NEW/OLD cold graph", ownedNew, h.Stats(), d.BudgetField.Used(), used)
		}
		e = h.Insert(key)
		if !errors.Is(e, arena.ErrClosed) || h.Stats().RetainedBytes != ownedNew*2 {
			t.Fatal("legal repeated growth overwrote earlier acquired array", e, h.Stats())
		}
		if legacyBoundPendingGraph(t, h) != ownedNew*2 || d.BudgetField.Used() != used+ownedNew*2 {
			t.Fatal("independent repeated acquisition graph")
		}
		paramsMust(t, h.SourceFinalize())
		if legacyBoundPendingGraph(t, h) != ownedNew*2 || h.Stats().TableBytes != 0 || d.BudgetField.Used() != used+ownedNew*2-before.TableBytes {
			t.Fatal("source destructor must leave source-invisible NEW nodes for owning abort")
		}
		paramsMust(t, h.CloseChecked())
		paramsMust(t, h.CloseChecked())
		if d.BudgetField.Used() != used-before.OwnerBytes-before.TableBytes-before.DirectoryBytes {
			t.Fatal("once-only exact owned headers/arrays refund")
		}
	} else {
		paramsMust(t, e)
		if h.Capacity() != 16 || h.Stats().RetainedBytes != 0 {
			t.Fatal("release erased source flag", h.Stats())
		}
	}
}

func TestStrHashtableLegacyClosedFlagAtActualDeletedInsertAndFullRemove(t *testing.T) {
	for _, core := range []bool{false, true} {
		t.Run(fmt.Sprint(core), func(t *testing.T) {
			_, d, _, _, _ := permutationActionFixture(t, "T")
			h, e := NewStrHashtable(d.region, StrHashtableConfig{Budget: d.BudgetField, Assertions: d})
			paramsMust(t, e)
			t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
			var keys []StrKey
			for n := 0; len(keys) < 3; n++ {
				text := fmt.Sprintf("legacy-delete-%d", n)
				if String(text, 17)&7 == 0 {
					keys = append(keys, debugTag(t, d.region, text))
				}
			}
			paramsMust(t, h.Insert(keys[0]))
			paramsMust(t, h.Insert(keys[1]))
			paramsMust(t, h.Remove(keys[0]))
			before := h.Stats()
			paramsMust(t, d.Close())
			if core {
				_, _, e = h.InsertIfNotThereCore(keys[2])
			} else {
				e = h.Insert(keys[2])
			}
			if sourceDebug {
				if !errors.Is(e, arena.ErrClosed) || h.Stats() != before {
					t.Fatal("source flag before deleted-entry publication", core, e, h.Stats(), before)
				}
			} else {
				paramsMust(t, e)
				if h.Size() != 2 || h.Deleted() != 0 {
					t.Fatal("erased flag source body", h.Stats())
				}
			}
		})
	}
	_, d, _, _, _ := permutationActionFixture(t, "T")
	paramsMust(t, d.EnableAssertions(false))
	h, e := NewStrHashtable(d.region, StrHashtableConfig{Budget: d.BudgetField, InitialCapacity: 3, Assertions: d})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
	for i := 0; i < 3; i++ {
		paramsMust(t, h.Insert(debugTag(t, d.region, fmt.Sprintf("legacy-full-%d", i))))
	}
	absent := debugTag(t, d.region, "absent-from-actual-full")
	before := h.Stats()
	paramsMust(t, d.Close())
	e = h.Remove(absent)
	if sourceDebug {
		if !errors.Is(e, arena.ErrClosed) || h.Stats() != before {
			t.Fatal("full-miss reached SASSERT flag before second contains", e, h.Stats())
		}
	} else {
		paramsMust(t, e)
		if h.Stats() != before {
			t.Fatal("erased missing remove mutation")
		}
	}
}

func TestStrHashtableLegacyNonPowerResetRealSourceContinueAndThrow(t *testing.T) {
	for _, actions := range []string{"CCC", "CT"} {
		t.Run(actions, func(t *testing.T) {
			_, d, _, in, streams := permutationActionFixture(t, actions)
			h, e := NewStrHashtable(d.region, StrHashtableConfig{Budget: d.BudgetField, InitialCapacity: 48, Assertions: d})
			paramsMust(t, e)
			t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
			paramsMust(t, h.Insert(debugTag(t, d.region, "before-reset")))
			e = h.Reset()
			state, re := in.State()
			paramsMust(t, re)
			output, re := os.ReadFile(streams.StderrField.File.Name())
			paramsMust(t, re)
			if sourceDebug {
				want := uint64(3)
				if actions == "CT" {
					want = 2
					var ex DefaultException
					if !errors.As(e, &ex) || h.Capacity() != 48 || h.Size() != 1 || h.Stats().TableBytes != 0 {
						t.Fatal("source throw after OLD delete before halve", e, h.Stats())
					}
				} else {
					paramsMust(t, e)
					if h.Capacity() != 24 || h.Size() != 0 || h.Stats().TableBytes != 24*32 {
						t.Fatal("source continue into alloc_table(24)", h.Stats())
					}
				}
				if state.Position != want || !strings.Contains(string(output), "Line: 301\nis_power_of_two(m_capacity)\n") {
					t.Fatal(state, string(output))
				}
			} else {
				paramsMust(t, e)
				if state.Position != 0 || len(output) != 0 || h.Capacity() != 24 || h.Size() != 0 {
					t.Fatal("release reset erasure", state, string(output), h.Stats())
				}
			}
		})
	}
}

// The census follows only this exact owner's declared pending fields. Reflection
// reads concrete cold scalar/slice metadata; no unsafe access or provider scan.
func legacyBoundPendingGraph(t *testing.T, h *StrHashtable) uint64 {
	t.Helper()
	owner := reflect.ValueOf(h).Elem()
	current := testFieldBySourceName(owner, "table").Pointer()
	seen := map[uintptr]bool{current: true}
	var total uint64
	for link := testFieldBySourceName(testFieldBySourceName(owner, "raw"), "legacyAcquisitions"); !link.IsNil(); link = testFieldBySourceName(link.Elem(), "next") {
		node := link.Elem()
		table := testFieldBySourceName(node, "table")
		pointer := table.Pointer()
		if pointer == 0 || seen[pointer] {
			t.Fatal("NEW/OLD owning ranges overlap")
		}
		seen[pointer] = true
		bytes := testFieldBySourceName(node, "bytes").Uint()
		tableBytes := testFieldBySourceName(node, "tableBytes").Uint()
		if bytes != uint64(node.Type().Size()) || tableBytes != uint64(table.Len())*uint64(table.Type().Elem().Size()) {
			t.Fatal("physical Go header/backing geometry")
		}
		total += bytes + tableBytes
	}
	return total
}
