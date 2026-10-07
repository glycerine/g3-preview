// Source: src/util/sorting_network.h:39-50,97-115; src/util/vector.h:303-329; src/util/debug.h:60-65.
// Source author: Nikolaj Bjorner (2013-11-07); Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestUnsignedSortingNetworkLiteralExchangeAndSortActions(t *testing.T) {
	for _, action := range []string{"C", "T"} {
		for _, site := range []string{"exchange", "sort"} {
			t.Run(action+site, func(t *testing.T) {
				p, d, _, in, streams := permutationActionFixture(t, action)
				sn, e := NewRuntimeUnsignedSortingNetwork(p.forward.Column.Raw.Runtime, d.BudgetField, d)
				paramsMust(t, e)
				t.Cleanup(func() { paramsMust(t, sn.Close()) })
				line, text := 40, "i <= j"
				if site == "exchange" {
					e = sn.exchange(9, 3, nil)
				} else {
					line, text = 50, "is_power_of2(k) && k > 0"
					e = sn.sort(0, &p.forward)
				}
				state, re := in.State()
				paramsMust(t, re)
				output, re := os.ReadFile(streams.StderrField.File.Name())
				paramsMust(t, re)
				if sourceDebug {
					if state.Position != 1 || !strings.Contains(string(output), "File: src/util/sorting_network.h") || !strings.Contains(string(output), text) {
						t.Fatal(e, line, state, string(output))
					}
					if action == "T" {
						var ex DefaultException
						if !errors.As(e, &ex) {
							t.Fatal(e)
						}
					} else if site == "exchange" {
						paramsMust(t, e)
					} else if !errors.Is(e, arena.ErrBounds) {
						t.Fatal("C reaches unsigned division-by-zero guard domain", e)
					}
				} else {
					if state.Position != 0 || len(output) != 0 {
						t.Fatal("erased source assertion", state, string(output))
					}
					if site == "exchange" {
						paramsMust(t, e)
					} else if !errors.Is(e, arena.ErrBounds) {
						t.Fatal(e)
					}
				}
			})
		}
	}
}
func TestUnsignedSortingNetworkClosedAssertionProviderAndUnusedBorrow(t *testing.T) {
	p, d, _, _, _ := permutationActionFixture(t, "T")
	sn, e := NewRuntimeUnsignedSortingNetwork(p.forward.Column.Raw.Runtime, d.BudgetField, d)
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, sn.Close()) })
	empty, e := NewRuntimeUnsignedVector(p.forward.Column.Raw.Runtime, d.BudgetField)
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, empty.Close()) })
	paramsMust(t, d.EnableAssertions(false))
	// Disabled condition and equal indices erase all unused output reads.
	paramsMust(t, sn.exchange(^uint32(0), ^uint32(0), nil))
	paramsMust(t, d.Close())
	e = sn.exchange(0, 0, nil)
	if sourceDebug {
		if !errors.Is(e, arena.ErrClosed) {
			t.Fatal("switch lookup reached before true condition", e)
		}
	} else {
		paramsMust(t, e)
	}
	// The actual zero input early return precedes every network/assertion site.
	paramsMust(t, sn.Sort(empty, empty))
	size, e := sn.currentv.Size()
	paramsMust(t, e)
	if size != 0 {
		t.Fatal(size)
	}
}
func TestUnsignedSortingNetworkCapturedExchangeOperandsAndSetSourceOrder(t *testing.T) {
	p, d, _, in, streams := permutationActionFixture(t, "T")
	sn, e := NewRuntimeUnsignedSortingNetwork(p.forward.Column.Raw.Runtime, d.BudgetField, d)
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, sn.Close()) })
	out, e := NewRuntimeUnsignedVector(p.forward.Column.Raw.Runtime, d.BudgetField)
	paramsMust(t, e)
	t.Cleanup(func() { paramsMust(t, out.Close()) })
	paramsMust(t, out.Push(17))
	paramsMust(t, out.Push(3))
	paramsMust(t, sn.exchange(0, 1, out))
	x, e := out.At(0)
	paramsMust(t, e)
	y, e := out.At(1)
	paramsMust(t, e)
	if x != 3 || y != 17 {
		t.Fatal("both source ei/ej captured before first write", x, y)
	}
	e = sn.exchange(0, 2, out)
	if sourceDebug {
		var ex DefaultException
		if !errors.As(e, &ex) {
			t.Fatal(e)
		}
		state, re := in.State()
		paramsMust(t, re)
		bytes, re := os.ReadFile(streams.StderrField.File.Name())
		paramsMust(t, re)
		if state.Position != 1 || !strings.Contains(string(bytes), "Line: 314\nidx < size()\n") {
			t.Fatal("source nonconst get precedes writes", e, state, string(bytes))
		}
	}
	if !sourceDebug && e == nil {
		t.Fatal("release reached invalid second get must guard")
	}
	x, re := out.At(0)
	paramsMust(t, re)
	y, re = out.At(1)
	paramsMust(t, re)
	if x != 3 || y != 17 {
		t.Fatal("failed second get must not publish first set", x, y)
	}
}
