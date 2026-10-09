// Source: src/util/debug.cpp:28-79; src/util/debug.h:63-65.
package util

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func debugTag(_ *testing.T, _ *arena.BytesRegion, s string) string { return s }
func debugTagColdBytes(_ *DebugContext) uint64                     { return 0 }

func debugAudit(t *testing.T, d *DebugContext, r *arena.BytesRegion, b *arena.Budget) {
	t.Helper()
	x := r.Stats()
	a := x.Descriptors
	want := x.OwnerBytes + x.PayloadBytes + x.PageHeaderBytes + x.DirectoryBytes + x.LogBytes + x.ScopeBytes + a.OwnerBytes + a.RecordBytes + a.MetadataBytes + a.DirectoryBytes
	if !d.closed {
		want += uint64(reflect.TypeOf(DebugContext{}).Size())
	}
	if b.Used() != want {
		t.Fatal("actual named debug graph", b.Used(), want)
	}
}

func debugFixture(t *testing.T) (*DebugContext, *arena.BytesRegion, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	serviceMust(t, e)
	d, e := NewDebugContext(r, b)
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, d.Close())
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("remaining debug graph", b.Used())
		}
	})
	return d, r, b
}

func TestDebugContextUsesOwnedGoStrings(t *testing.T) {
	d, _, _ := debugFixture(t)
	serviceMust(t, d.EnableDebug("sat"))
	ok, e := d.IsDebugEnabled("sat\x00ignored")
	serviceMust(t, e)
	if !ok || len(d.Tags) != 1 {
		t.Fatal("enabled name missing")
	}
	serviceMust(t, d.DisableDebug("sat"))
	ok, e = d.IsDebugEnabled("sat")
	serviceMust(t, e)
	if ok {
		t.Fatal("disabled name remains")
	}
}

func TestDebugContextAssertionsFinalizeAndOwner(t *testing.T) {
	d, _, _ := debugFixture(t)
	serviceMust(t, d.EnableDebug("heap"))
	ok, e := d.ConditionalAssertionEnabled("heap")
	serviceMust(t, e)
	if !ok {
		t.Fatal("enabled conditional assertion")
	}
	serviceMust(t, d.EnableAssertions(false))
	ok, e = d.ConditionalAssertionEnabled("heap")
	serviceMust(t, e)
	if ok {
		t.Fatal("disabled assertion evaluated tag")
	}
	serviceMust(t, d.FinalizeDebug())
	if d.Tags != nil {
		t.Fatal("finalize retained tags")
	}
	copy := *d
	if e = copy.EnableDebug("x"); !errors.Is(e, ErrDebugContextOwner) {
		t.Fatal(e)
	}
}
