// Source: src/util/union_find.h:25-241.
package backtrack

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type unionFindRecordingContext struct {
	trail  *TrailStack
	events []unionFindEvent
}

func TestUnionFindTraceBeforeSameRootReturnAndUnmerge(t *testing.T) {
	context := NewUnionFindDefaultContext()
	u, err := NewUnionFind(context)
	trailMust(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("union_find")
	compiled := tags.ShouldTrace("union_find")
	u.BindTrace(tags)
	context.GetTrailStack().PushScope()
	u.MkVar()
	u.MkVar()
	u.Merge(0, 1)
	u.Merge(0, 1)
	trailMust(t, context.GetTrailStack().PopScope(1))
	if !compiled {
		if output.Len() != 0 {
			t.Fatal("erased union-find TRACE emitted", output.String())
		}
		return
	}
	want := "-------- [union_find] merge src/util/union_find.h:122 ---------\nmerging 0 1\n------------------------------------------------\n" +
		"-------- [union_find] merge src/util/union_find.h:122 ---------\nmerging 1 1\n------------------------------------------------\n" +
		"-------- [union_find] unmerge src/util/union_find.h:79 ---------\nunmerging 0 1\n------------------------------------------------\n"
	if output.String() != want {
		t.Fatal("source merge/unmerge TRACE order", output.String())
	}
}

type unionFindEvent struct {
	kind string
	args [4]uint32
}

func (c *unionFindRecordingContext) GetTrailStack() *TrailStack { return c.trail }
func (c *unionFindRecordingContext) UnmergeEH(root, child uint32) {
	c.events = append(c.events, unionFindEvent{kind: "unmerge", args: [4]uint32{root, child}})
}
func (c *unionFindRecordingContext) MergeEH(root, child, value2, value1 uint32) {
	c.events = append(c.events, unionFindEvent{kind: "merge", args: [4]uint32{root, child, value2, value1}})
}
func (c *unionFindRecordingContext) AfterMergeEH(root, child, value2, value1 uint32) {
	c.events = append(c.events, unionFindEvent{kind: "after", args: [4]uint32{root, child, value2, value1}})
}

func TestUnionFindBacktrackingCallbacksAndClassCycle(t *testing.T) {
	context := &unionFindRecordingContext{trail: NewTrailStack()}
	u, err := NewUnionFind(context)
	trailMust(t, err)
	context.trail.PushScope()
	for range 4 {
		u.MkVar()
	}
	u.Merge(0, 1)
	u.Merge(2, 3)
	context.trail.PushScope()
	u.Merge(0, 2)
	if u.Size(0) != 4 || u.Find(0) != u.Find(3) || !u.CheckInvariant() {
		t.Fatal("merged class", u.find, u.size, u.next)
	}
	seen := map[uint32]bool{}
	for value := uint32(0); !seen[value]; value = u.Next(value) {
		seen[value] = true
	}
	if len(seen) != 4 {
		t.Fatal("class cycle", seen, u.next)
	}
	trailMust(t, context.trail.PopScope(1))
	if u.Find(0) == u.Find(2) || u.Size(0) != 2 || u.Size(2) != 2 || !u.CheckInvariant() {
		t.Fatal("unmerge", u.find, u.size, u.next)
	}
	trailMust(t, context.trail.PopScope(1))
	if u.GetNumVars() != 0 {
		t.Fatal("mk_var trails", u.GetNumVars())
	}
	wantEvents := []unionFindEvent{
		{kind: "merge", args: [4]uint32{1, 0, 1, 0}},
		{kind: "after", args: [4]uint32{1, 0, 1, 0}},
		{kind: "merge", args: [4]uint32{3, 2, 3, 2}},
		{kind: "after", args: [4]uint32{3, 2, 3, 2}},
		{kind: "merge", args: [4]uint32{3, 1, 2, 0}},
		{kind: "after", args: [4]uint32{3, 1, 2, 0}},
		{kind: "unmerge", args: [4]uint32{3, 1}},
		{kind: "unmerge", args: [4]uint32{3, 2}},
		{kind: "unmerge", args: [4]uint32{1, 0}},
	}
	if !reflect.DeepEqual(context.events, wantEvents) {
		t.Fatal("callback sequence", context.events)
	}
}

func TestUnionFindDissolveEntireClassAndDisplay(t *testing.T) {
	u := NewDefaultUnionFind()
	for range 4 {
		u.MkVar()
	}
	u.Merge(0, 1)
	u.Merge(1, 2)
	u.Merge(2, 3)
	u.Dissolve(1)
	for value := uint32(0); value < 4; value++ {
		if u.Find(value) != value || u.Next(value) != value || u.Size(value) != 1 {
			t.Fatal("dissolved member", value, u.find, u.size, u.next)
		}
	}
	if !u.CheckInvariant() {
		t.Fatal("dissolved invariant")
	}
	var out bytes.Buffer
	trailMust(t, u.Display(&out))
	if want := "v0 --> v0 (1)\nv1 --> v1 (1)\nv2 --> v2 (1)\nv3 --> v3 (1)\n"; out.String() != want {
		t.Fatal("display", out.String(), want)
	}
}

func TestBasicUnionFindUnknownMergeCycleAndReset(t *testing.T) {
	u := NewBasicUnionFind()
	if u.Find(7) != 7 || u.Next(7) != 7 || !u.IsRoot(7) {
		t.Fatal("unknown singleton")
	}
	u.Merge(5, 2)
	if u.GetNumVars() != 6 || u.Find(5) != u.Find(2) || !u.IsRoot(u.Find(5)) {
		t.Fatal("automatic growth", u.GetNumVars(), u.find)
	}
	root := u.Find(5)
	other := u.Next(root)
	if other == root || u.Next(other) != root {
		t.Fatal("two-member cycle", u.next)
	}
	before := append([]uint32(nil), u.find...)
	u.Merge(5, 2)
	if !reflect.DeepEqual(u.find, before) {
		t.Fatal("same-class merge changed forest")
	}
	u.Reset()
	if u.GetNumVars() != 0 || len(u.size) != 0 || len(u.next) != 0 {
		t.Fatal("reset", u.find, u.size, u.next)
	}
}

func TestUnionFindContextValidationAndInvariantFailure(t *testing.T) {
	if u, err := NewUnionFind(nil); u != nil || !errors.Is(err, ErrUnionFindContext) {
		t.Fatal("nil context", u, err)
	}
	context := &unionFindRecordingContext{}
	if u, err := NewUnionFind(context); u != nil || !errors.Is(err, ErrUnionFindContext) {
		t.Fatal("nil trail", u, err)
	}
	u := NewDefaultUnionFind()
	u.MkVar()
	u.size[0] = 2
	if u.CheckInvariant() {
		t.Fatal("bad size accepted")
	}
}
