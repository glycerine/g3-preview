// Source: src/util/trace.cpp:25-62; src/util/trace.h:38-49.
// Licensed under the MIT license. See LICENSE.txt.
// Source author: Leonardo de Moura (leonardo), 2006.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/utilhash"
	"reflect"
	"strconv"
	"testing"
)

func traceRegion(t *testing.T, b *arena.Budget) *arena.BytesRegion {
	t.Helper()
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 4, Budget: b}})
	smallObjectMust(t, e)
	return r
}
func traceKey(t *testing.T, r *arena.BytesRegion, s string) utilhash.StrKey {
	t.Helper()
	id := smallObjectID(t, r, s)
	k, e := utilhash.BorrowCString(r, id.Ref, 0)
	smallObjectMust(t, e)
	return k
}
func tracePhysical(t *testing.T, b *arena.Budget, c *TraceContext, regions ...*arena.BytesRegion) {
	t.Helper()
	var want uint64
	for _, r := range regions {
		rs := r.Stats()
		ds := rs.Descriptors
		want += rs.OwnerBytes + rs.PayloadBytes + rs.PageHeaderBytes + rs.DirectoryBytes + rs.LogBytes + rs.ScopeBytes + ds.OwnerBytes + ds.RecordBytes + ds.MetadataBytes + ds.DirectoryBytes
	}
	st := c.Stats()
	want += st.OwnerBytes + st.Table.OwnerBytes + st.Table.TableBytes + st.Table.DirectoryBytes
	if want != b.Used() {
		t.Fatal("named trace graph", want, b.Used())
	}
}
func TestTraceContextShortCircuitAndSourceFinalize(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r := traceRegion(t, b)
	defer r.Close()
	k := traceKey(t, r, "tag")
	c, e := NewTraceContext(r, b)
	smallObjectMust(t, e)
	defer c.Close()
	if c.tags != nil || c.all || c.Stats().OwnerBytes != uint64(reflect.TypeOf(TraceContext{}).Size()) {
		t.Fatal("source initial state")
	}
	smallObjectMust(t, r.Reset())
	before := b.Used()
	ok, e := c.IsTraceEnabled(k)
	smallObjectMust(t, e)
	if ok || c.tags != nil || before != b.Used() {
		t.Fatal("absent table must not read stale key or allocate")
	}
	smallObjectMust(t, c.EnableAllTrace(true))
	ok, e = c.IsTraceEnabled(k)
	smallObjectMust(t, e)
	if !ok || c.tags != nil {
		t.Fatal("true first operand")
	}
	smallObjectMust(t, c.EnableAllTrace(false))
	k = traceKey(t, r, "again")
	smallObjectMust(t, c.EnableTrace(k))
	ok, e = c.IsTraceEnabled(k)
	smallObjectMust(t, e)
	if !ok {
		t.Fatal("inserted")
	}
	smallObjectMust(t, c.EnableAllTrace(true))
	smallObjectMust(t, c.FinalizeTrace())
	if c.tags != nil || !c.all {
		t.Fatal("finalizer changed persistent all flag")
	}
	smallObjectMust(t, r.Reset())
	ok, e = c.IsTraceEnabled(k)
	smallObjectMust(t, e)
	if !ok {
		t.Fatal("post-finalize shortcut")
	}
	smallObjectMust(t, c.EnableAllTrace(false))
	ok, e = c.IsTraceEnabled(k)
	smallObjectMust(t, e)
	if ok {
		t.Fatal("no lazy table on query")
	}
	tracePhysical(t, b, c, r)
}
func TestTraceContextIndependentBoundedSetModel(t *testing.T) {
	for code := 0; code < 256; code++ {
		b := arena.NewBudget(1 << 20)
		r := traceRegion(t, b)
		keys := []utilhash.StrKey{traceKey(t, r, "a"), traceKey(t, r, "b")}
		c, e := NewTraceContext(r, b)
		smallObjectMust(t, e)
		mask := 0
		all := false
		n := code
		for step := 0; step < 4; step++ {
			action := n % 4
			n /= 4
			switch action {
			case 0:
				smallObjectMust(t, c.EnableTrace(keys[step%2]))
				mask |= 1 << (step % 2)
			case 1:
				smallObjectMust(t, c.DisableTrace(keys[step%2]))
				mask &^= 1 << (step % 2)
			case 2:
				all = !all
				smallObjectMust(t, c.EnableAllTrace(all))
			case 3:
				smallObjectMust(t, c.FinalizeTrace())
				mask = 0
			}
			for i, k := range keys {
				got, e := c.IsTraceEnabled(k)
				smallObjectMust(t, e)
				if got != (all || mask&(1<<i) != 0) {
					t.Fatal(code, step, i, got, mask, all)
				}
			}
			tracePhysical(t, b, c, r)
		}
		smallObjectMust(t, c.Close())
		smallObjectMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
}
func TestTraceContextEveryRealLazyConstructorRefusal(t *testing.T) {
	failures, success := 0, 0
	for ordinal := int64(0); ordinal < 8; ordinal++ {
		b := arena.NewBudget(1 << 20)
		r := traceRegion(t, b)
		k := traceKey(t, r, "lazy")
		c, e := NewTraceContext(r, b)
		smallObjectMust(t, e)
		before := b.Used()
		b.FailAfter(ordinal)
		e = c.EnableTrace(k)
		tracePhysical(t, b, c, r)
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) || c.tags != nil || b.Used() != before {
				t.Fatal(ordinal, e, b.Used(), before)
			}
			failures++
		} else {
			success++
			if c.tags == nil || c.tags.Size() != 1 {
				t.Fatal("publication")
			}
		}
		b.FailAfter(-1)
		smallObjectMust(t, c.Close())
		smallObjectMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
		if success != 0 {
			break
		}
	}
	if failures != 2 || success != 1 {
		t.Fatal(failures, success)
	}
}
func TestTraceContextForeignBorrowAndOwnerGuards(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r := traceRegion(t, b)
	foreign := traceRegion(t, b)
	defer r.Close()
	defer foreign.Close()
	k := traceKey(t, foreign, "foreign")
	c, e := NewTraceContext(r, b)
	smallObjectMust(t, e)
	defer c.Close()
	smallObjectMust(t, c.EnableTraceFrom(foreign, k))
	ok, e := c.IsTraceEnabledFrom(foreign, k)
	smallObjectMust(t, e)
	if !ok {
		t.Fatal("foreign source pointer")
	}
	tracePhysical(t, b, c, r, foreign)
	smallObjectMust(t, foreign.Reset())
	_, e = c.IsTraceEnabledFrom(foreign, k)
	if e == nil {
		t.Fatal("actual reached dangling tag")
	}
	smallObjectMust(t, c.EnableAllTrace(true))
	ok, e = c.IsTraceEnabledFrom(nil, k)
	smallObjectMust(t, e)
	if !ok {
		t.Fatal("shortcut read ignored region")
	}
	copy := *c
	if e = copy.EnableAllTrace(false); !errors.Is(e, ErrTraceContextOwner) {
		t.Fatal(e)
	}
	smallObjectMust(t, c.Close())
	if _, e = c.IsTraceEnabled(k); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestTraceContextCStringCurrentReadsAndWarmQueries(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r := traceRegion(t, b)
	defer r.Close()
	c, e := NewTraceContext(r, b)
	smallObjectMust(t, e)
	defer c.Close()
	ok, e := c.IsTraceEnabledCString("absent")
	smallObjectMust(t, e)
	if ok || c.tags != nil {
		t.Fatal("query created table")
	}
	k := traceKey(t, r, "tag")
	smallObjectMust(t, c.EnableTrace(k))
	before := b.Used()
	b.FailAfter(0)
	ok, e = c.IsTraceEnabledCString("tag\x00ignored")
	smallObjectMust(t, e)
	if !ok || b.Used() != before {
		t.Fatal(ok, b.Used(), before)
	}
	if n := testing.AllocsPerRun(100, func() {
		got, e := c.IsTraceEnabledCString("tag")
		if e != nil || !got {
			panic("actual lookup")
		}
	}); n != 0 {
		t.Fatal(n)
	}
	b.FailAfter(-1)
}
func TestTraceContextForeignPublicationAndGrowthRefusalPrefixes(t *testing.T) {
	for _, foreignRoute := range []bool{false, true} {
		fail, success := 0, 0
		for fault := int64(0); fault < 8; fault++ {
			b := arena.NewBudget(1 << 20)
			r := traceRegion(t, b)
			other := traceRegion(t, b)
			c, e := NewTraceContext(r, b)
			smallObjectMust(t, e)
			keys := make([]utilhash.StrKey, 8)
			for i := range keys {
				keys[i] = traceKey(t, r, string(rune('a'+i)))
			}
			if !foreignRoute {
				for _, k := range keys[:7] {
					smallObjectMust(t, c.EnableTrace(k))
				}
				if c.tags.Capacity() != 8 {
					t.Fatal(c.tags.Stats())
				}
			}
			extra := traceKey(t, other, "foreign")
			b.FailAfter(fault)
			if foreignRoute {
				e = c.EnableTraceFrom(other, extra)
			} else {
				e = c.EnableTrace(keys[7])
			}
			tracePhysical(t, b, c, r, other)
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal(e)
				}
				fail++
				if foreignRoute {
					if fault < 2 && c.tags != nil {
						t.Fatal("incomplete table published")
					}
					if fault == 2 && (c.tags == nil || c.tags.Size() != 0) {
						t.Fatal("completed table prefix lost")
					}
				} else {
					if c.tags.Size() != 7 {
						t.Fatal("new key prematurely published")
					}
					for _, k := range keys[:7] {
						ok, err := c.IsTraceEnabled(k)
						smallObjectMust(t, err)
						if !ok {
							t.Fatal("old key lost")
						}
					}
				}
			} else {
				success++
			}
			b.FailAfter(-1)
			smallObjectMust(t, c.Close())
			smallObjectMust(t, other.Close())
			smallObjectMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
			if success != 0 {
				break
			}
		}
		// Legacy growth charges its concrete acquisition header and actual NEW
		// array separately. Both refusal prefixes preserve every old key above.
		want := 2
		if foreignRoute {
			want = 3
		}
		if fail != want || success != 1 {
			t.Fatal(foreignRoute, fail, success)
		}
	}
}
func TestTraceContextEraseRehashCompletedPrefix(t *testing.T) {
	fail, success := 0, 0
	for fault := int64(0); fault < 16; fault++ {
		b := arena.NewBudget(1 << 22)
		r := traceRegion(t, b)
		c, e := NewTraceContext(r, b)
		smallObjectMust(t, e)
		keys := make([]utilhash.StrKey, 300)
		// Select genuine colliding spellings with the existing source hash. This
		// input selection does not edit table state or synthesize a refusal.
		chosen := 0
		for candidate := 0; chosen < len(keys); candidate++ {
			spelling := "collision" + strconv.Itoa(candidate)
			if utilhash.String(spelling, 17)&511 != 0 {
				continue
			}
			keys[chosen] = traceKey(t, r, spelling)
			smallObjectMust(t, c.EnableTrace(keys[chosen]))
			chosen++
		}
		if c.tags.Capacity() != 512 {
			t.Fatal(c.tags.Stats())
		}
		for _, k := range keys {
			h, e := c.tags.Hash(k)
			smallObjectMust(t, e)
			if h&511 != 0 {
				t.Fatal("fixture hash route", h)
			}
		}
		removed := 0
		b.FailAfter(fault)
		for _, k := range keys {
			e = c.DisableTrace(k)
			removed++
			if e != nil {
				break
			}
		}
		tracePhysical(t, b, c, r)
		if e != nil {
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
			fail++
		} else {
			success++
		}
		if c.tags.Size() != uint32(len(keys)-removed) {
			t.Fatal("source removal precedes rehash", c.tags.Size(), removed)
		}
		for i, k := range keys {
			ok, err := c.IsTraceEnabled(k)
			smallObjectMust(t, err)
			if ok != (i >= removed) {
				t.Fatal("live publication prefix", fault, i, removed, ok)
			}
		}
		b.FailAfter(-1)
		smallObjectMust(t, c.Close())
		smallObjectMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
		if success != 0 {
			break
		}
	}
	if fail == 0 || success != 1 {
		t.Fatal(fail, success)
	}
	t.Log("actual erase-rehash refusal horizon", fail)
}
