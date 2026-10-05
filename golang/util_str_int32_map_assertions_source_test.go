// Source: src/util/map.h:24-196; src/util/hashtable.h:159-215,249,371-451,559,643-644.
// src/util/debug.h:60-65,76-89; src/util/debug.cpp:37-68.
// Source author: Leonardo de Moura (2006-10-14; 2006-09-11).
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

func TestStrInt32MapBoundConstructorActualDefaultEntryContinueThrow(t *testing.T) {
	for _, physical := range []bool{false, true} {
		for _, action := range []string{"C", "T"} {
			t.Run(fmt.Sprintf("runtime%v-%s", physical, action), func(t *testing.T) {
				p, d, _, in, streams := permutationActionFixture(t, action)
				var r *memory.Runtime
				if physical {
					r = p.forward.column.raw.runtime
				}
				beforeCold := d.budget.Used()
				beforeCount := p.forward.column.raw.runtime.State().GlobalCount
				h, e := utilhash.NewStrInt32Map(d.region, utilhash.StrInt32MapConfig{Budget: d.budget, Runtime: r, Statistics: true, Assertions: d, InitialCapacity: 3})
				if h != nil {
					t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
				}
				state, re := in.State()
				paramsMust(t, re)
				out, re := os.ReadFile(streams.stderr.file.Name())
				paramsMust(t, re)
				if satStorageSourceDebug {
					if state.Position != 1 || !strings.Contains(string(out), "Line: 249\nis_power_of_two(initial_capacity)\n") {
						t.Fatal(state, e, string(out))
					}
					if action == "T" {
						var ex DefaultException
						if !errors.As(e, &ex) {
							t.Fatal(e)
						}
						if physical {
							if p.forward.column.raw.runtime.State().GlobalCount != beforeCount+1 {
								t.Fatal("wrapper allocated, array never reached")
							}
						} else if d.budget.Used() != beforeCold {
							t.Fatal("no acquired source array/cold root", d.budget.Used(), beforeCold)
						}
						return
					}
				} else if state.Position != 0 || len(out) != 0 {
					t.Fatal("release erasure", state, string(out))
				}
				paramsMust(t, e)
				if h.Capacity() != 3 || h.CollisionCount() != 0 {
					t.Fatal("C source publication/st_init")
				}
				paramsMust(t, d.EnableAssertions(false))
				var keys []utilhash.StrKey
				for n := 0; len(keys) < 3; n++ {
					text := fmt.Sprintf("map-linear-%d", n)
					if utilhash.String(text, 17)&2 != 2 {
						continue
					}
					key := debugTag(t, d.region, text)
					paramsMust(t, h.Insert(key, int32(97-len(keys)*13)))
					keys = append(keys, key)
				}
				for i, key := range keys {
					value, yes, e := h.Find(key)
					paramsMust(t, e)
					if !yes || value != int32(97-i*13) {
						t.Fatal("literal wrap and default int data", i, value, yes)
					}
				}
				copy, e := h.Clone()
				paramsMust(t, e)
				if copy.CollisionCount() != 0 {
					t.Fatal("source copy st_init")
				}
				for i, key := range keys {
					value, yes, e := copy.Find(key)
					paramsMust(t, e)
					if !yes || value != int32(97-i*13) {
						t.Fatal("deep copied actual value", i, value, yes)
					}
				}
				paramsMust(t, copy.CloseChecked())
			})
		}
	}
}
func TestStrInt32MapClosedAssertionOwnerCopyInvariantAndErasedReset(t *testing.T) {
	for _, physical := range []bool{false, true} {
		t.Run(fmt.Sprint(physical), func(t *testing.T) {
			p, d, _, _, _ := permutationActionFixture(t, "T")
			var r *memory.Runtime
			if physical {
				r = p.forward.column.raw.runtime
			}
			h, e := utilhash.NewStrInt32Map(d.region, utilhash.StrInt32MapConfig{Budget: d.budget, Runtime: r, Assertions: d, Statistics: true})
			paramsMust(t, e)
			t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
			paramsMust(t, d.Close())
			paramsMust(t, h.Reset()) // actual empty scalar shortcut
			_, e = h.CheckInvariantChecked()
			if satStorageSourceDebug {
				if !errors.Is(e, arena.ErrClosed) {
					t.Fatal("actual reached flag", e)
				}
			} else {
				paramsMust(t, e)
			}
			copy, e := h.Clone()
			if copy != nil {
				t.Cleanup(func() { paramsMust(t, copy.CloseChecked()) })
			}
			if satStorageSourceDebug {
				if !errors.Is(e, arena.ErrClosed) || copy == nil {
					t.Fatal("copy allocation reached before actual flag159", copy, e)
				}
			} else {
				paramsMust(t, e)
			}
		})
	}
}
func TestStrInt32MapRuntimeBoundInvariantLiteralContinue(t *testing.T) {
	p, d, _, in, streams := permutationActionFixture(t, "CC")
	r := p.forward.column.raw.runtime
	h, e := utilhash.NewStrInt32Map(d.region, utilhash.StrInt32MapConfig{Budget: d.budget, Runtime: r, Assertions: d})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
	roots, e := h.RuntimeRoots()
	paramsMust(t, e)
	// Malformed source scalar-count fixture; C proves literal assertion order.
	paramsMust(t, r.WriteUint32(roots.Object, 12, 1))
	paramsMust(t, r.WriteUint32(roots.Object, 16, 1))
	yes, e := h.CheckInvariantChecked()
	paramsMust(t, e)
	state, e := in.State()
	paramsMust(t, e)
	out, e := os.ReadFile(streams.stderr.file.Name())
	paramsMust(t, e)
	if satStorageSourceDebug {
		want := "ASSERTION VIOLATION\nFile: src/util/hashtable.h\nLine: 643\nnum_deleted == m_num_deleted\n" + debugSourcePrompt + "ASSERTION VIOLATION\nFile: src/util/hashtable.h\nLine: 644\nnum_used == m_size\n" + debugSourcePrompt
		if !yes || state.Position != 2 || string(out) != want {
			t.Fatal(yes, state, string(out), want)
		}
	} else if yes || state.Position != 0 || len(out) != 0 {
		t.Fatal("release inspection/erasure", yes, state, string(out))
	}
}

func TestStrInt32MapLegacyGrowthSourceAssertionStopRetainsNewAndOld(t *testing.T) {
	_, d, _, _, _ := permutationActionFixture(t, "T")
	h, e := utilhash.NewStrInt32Map(d.region, utilhash.StrInt32MapConfig{Budget: d.budget, Assertions: d})
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
	for n := 0; n < 7; n++ {
		paramsMust(t, h.Insert(debugTag(t, d.region, fmt.Sprintf("growth-%d", n)), int32(-n)))
	}
	key := debugTag(t, d.region, "growth-new")
	before := h.Stats()
	paramsMust(t, d.Close())
	used := d.budget.Used()
	// Source NEW is allocated before move_table's first assertion flag read.
	// Closing the real provider is a Go lifetime guard-domain fixture.
	e = h.Insert(key, -199)
	if satStorageSourceDebug {
		if !errors.Is(e, arena.ErrClosed) || h.Stats().TableBytes != before.TableBytes || h.Size() != before.Size || h.Stats().RetainedBytes <= 16*40 {
			t.Fatal("stopped move lost actual new/old owning ranges", e, h.Stats(), before)
		}
		ownedNew := mapBoundPendingGraph(t, h)
		if ownedNew != h.Stats().RetainedBytes || d.budget.Used() != used+ownedNew {
			t.Fatal("immediate independent NEW/OLD cold graph", ownedNew, h.Stats(), d.budget.Used(), used)
		}
		e = h.Insert(key, -199)
		if !errors.Is(e, arena.ErrClosed) || h.Stats().RetainedBytes != ownedNew*2 {
			t.Fatal("legal repeated growth overwrote earlier acquired array", e, h.Stats())
		}
		if mapBoundPendingGraph(t, h) != ownedNew*2 || d.budget.Used() != used+ownedNew*2 {
			t.Fatal("independent repeated acquisition graph")
		}
		paramsMust(t, h.SourceFinalize())
		if mapBoundPendingGraph(t, h) != ownedNew*2 || h.Stats().TableBytes != 0 || d.budget.Used() != used+ownedNew*2-before.TableBytes {
			t.Fatal("source destructor must leave source-invisible NEW nodes for owning abort")
		}
		paramsMust(t, h.CloseChecked())
		paramsMust(t, h.CloseChecked())
		if d.budget.Used() != used-before.OwnerBytes-before.TableBytes-before.DirectoryBytes {
			t.Fatal("once-only exact owned headers/arrays refund")
		}
	} else {
		paramsMust(t, e)
		if h.Capacity() != 16 || h.Stats().RetainedBytes != 0 {
			t.Fatal("release erased source flag", h.Stats())
		}
	}
}

func mapBoundPendingGraph(t *testing.T, h *utilhash.StrInt32Map) uint64 {
	t.Helper()
	owner := reflect.ValueOf(h).Elem()
	current := owner.FieldByName("table").Pointer()
	seen := map[uintptr]bool{current: true}
	var total uint64
	for link := owner.FieldByName("raw").FieldByName("legacyAcquisitions"); !link.IsNil(); link = link.Elem().FieldByName("next") {
		node := link.Elem()
		table := node.FieldByName("table")
		pointer := table.Pointer()
		if pointer == 0 || seen[pointer] {
			t.Fatal("NEW/OLD owning ranges overlap")
		}
		seen[pointer] = true
		bytes := node.FieldByName("bytes").Uint()
		tableBytes := node.FieldByName("tableBytes").Uint()
		if bytes != uint64(node.Type().Size()) || tableBytes != uint64(table.Len())*uint64(table.Type().Elem().Size()) {
			t.Fatal("physical Go header/backing geometry")
		}
		total += bytes + tableBytes
	}
	return total
}

func TestStrInt32MapReferenceFindLiteralAssertionAndValueLvalue(t *testing.T) {
	for _, physical := range []bool{false, true} {
		for _, constant := range []bool{false, true} {
			for _, action := range []string{"C", "T"} {
				t.Run(fmt.Sprintf("runtime%v-const%v-%s", physical, constant, action), func(t *testing.T) {
					p, d, _, in, streams := permutationActionFixture(t, action)
					var r *memory.Runtime
					if physical {
						r = p.forward.column.raw.runtime
					}
					h, e := utilhash.NewStrInt32Map(d.region, utilhash.StrInt32MapConfig{Budget: d.budget, Runtime: r, Assertions: d})
					paramsMust(t, e)
					t.Cleanup(func() { paramsMust(t, h.CloseChecked()) })
					key := debugTag(t, d.region, "missing-reference")
					var ref utilhash.StrInt32MapEntryRef
					if constant {
						ref, e = h.AtConst(key)
					} else {
						ref, e = h.At(key)
					}
					line := 157
					if constant {
						line = 151
					}
					out, re := os.ReadFile(streams.stderr.file.Name())
					paramsMust(t, re)
					state, re := in.State()
					paramsMust(t, re)
					if satStorageSourceDebug {
						if state.Position != 1 || !strings.Contains(string(out), fmt.Sprintf("File: src/util/map.h\nLine: %d\ne\n", line)) {
							t.Fatal(e, state, string(out))
						}
						if action == "T" {
							var ex DefaultException
							if !errors.As(e, &ex) {
								t.Fatal(e)
							}
						} else if !errors.Is(e, arena.ErrBounds) {
							t.Fatal("C reaches null-entry guard domain", e)
						}
					} else if state.Position != 0 || len(out) != 0 || !errors.Is(e, arena.ErrBounds) {
						t.Fatal("release erased assertion, actual null-entry guard", e, state, string(out))
					}
					_ = ref
					paramsMust(t, d.EnableAssertions(false))
					paramsMust(t, h.Insert(key, -711))
					if constant {
						ref, e = h.AtConst(key)
					} else {
						ref, e = h.At(key)
					}
					paramsMust(t, e)
					value, e := ref.Value()
					paramsMust(t, e)
					if value != -711 {
						t.Fatal(value)
					}
					if !constant {
						paramsMust(t, ref.SetValue(314))
						value, ok, e := h.Find(key)
						paramsMust(t, e)
						if !ok || value != 314 {
							t.Fatal("actual mutable source lvalue", value, ok)
						}
					}
				})
			}
		}
	}
}
