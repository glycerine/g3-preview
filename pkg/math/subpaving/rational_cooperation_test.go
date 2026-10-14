package subpaving

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/cooperate"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestRationalContextCheckpointAndRootBorrowCooperation(t *testing.T) {
	c, m, value, x, _ := rationalClauseFixture(t)
	b := arena.NewBudget(1 << 20)
	owner, err := cooperate.NewCooperation(b)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := owner.Close(); err != nil {
			t.Error(err)
		}
		if b.Used() != 0 {
			t.Error("retained rational cooperation owner", b.Used())
		}
	})
	if err := c.BindCooperation(nil); !errors.Is(err, arena.ErrConfig) {
		t.Fatal("nil binding", err)
	}
	if err := c.BindCooperation(owner); err != nil {
		t.Fatal(err)
	}
	if err := owner.BeginSection(); err != nil {
		t.Fatal(err)
	}
	var trace bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&trace)
	tags.Enable("cooperate_detail")
	if err := owner.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(value, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AddInequality(x, value, true, false, true); err != nil {
		t.Fatal(err)
	}
	c.limit.Limit, c.limit.Steps = 1, 1
	if _, err := c.AssertRootUnits(); !errors.Is(err, ErrRationalContextCanceled) || owner.OwnerThread != -1 || trace.Len() != 0 {
		t.Fatal("resource refusal preceded cooperation", err, trace.String())
	}
	c.limit.Limit = 0
	if _, err := c.AssertRootUnits(); err != nil || owner.OwnerThread != 0 {
		t.Fatal("unit checkpoint omitted cooperation", err)
	}
	c.maxDepth = 0
	if err := c.Search(); err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(trace.String(), "subpaving, tid: 0\n"); tags.ShouldTrace("cooperate_detail") && count != 3 {
		t.Fatal("unit, queue, selection checkpoints", count, trace.String())
	}
	if err := owner.EndTask(); err != nil {
		t.Fatal(err)
	}
	a, err := NewRationalInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	out, err := NewRationalInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := out.Close(); err != nil {
			t.Error(err)
		}
		if err := a.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := m.SetInt32(value, 2); err != nil {
		t.Fatal(err)
	}
	if err := a.SetFinite(value, value, false, false); err != nil {
		t.Fatal(err)
	}
	if err := c.XnEqY(out, a, 2); err != nil || owner.OwnerThread != 0 || tags.ShouldTrace("cooperate_detail") && !strings.Contains(trace.String(), "interval, tid: 0\n") {
		t.Fatal("inverse root omitted interval cooperation", err, trace.String())
	}
	if err := owner.EndSection(); err != nil {
		t.Fatal(err)
	}
}
