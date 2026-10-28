package subpaving

// Source: src/math/subpaving/subpaving_t_def.h:461-467,1806-1843.

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/cooperate"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingCheckpointCooperatesAfterResourceAndMemory(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	owner, err := cooperate.NewCooperation(b)
	if err != nil {
		t.Fatal(err)
	}
	process := memory.NewProcessAccounting()
	runtime, err := memory.NewRuntime(memory.RuntimeConfig{Budget: b, Process: process, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := owner.Close(); err != nil {
			t.Error(err)
		}
		if err := runtime.Close(); err != nil {
			t.Error(err)
		}
		if b.Used() != 0 {
			t.Error("retained checkpoint owners", b.Used())
		}
	})
	if err := owner.BeginSection(); err != nil {
		t.Fatal(err)
	}
	var traceOutput bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&traceOutput)
	tags.Enable("cooperate_detail")
	if err := owner.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	limit := &control.Reslimit{Limit: 1, Steps: 1}
	if err := sourceCheckpoint(limit, nil, process, 0, ErrFastContextCanceled, owner); !errors.Is(err, ErrFastContextCanceled) || owner.OwnerThread != -1 {
		t.Fatal("resource cancellation must precede cooperation", err, owner.OwnerThread)
	}
	limit.Limit = 100
	block, err := runtime.Allocate(1)
	if err != nil {
		t.Fatal(err)
	}
	if err := sourceCheckpoint(limit, nil, process, 0, ErrFastContextCanceled, owner); !errors.Is(err, ErrSubpavingMaxMemory) || owner.OwnerThread != -1 {
		t.Fatal("memory cancellation must precede cooperation", err, owner.OwnerThread)
	}
	if traceOutput.Len() != 0 {
		t.Fatal("failed checkpoint cooperated", traceOutput.String())
	}
	if err := runtime.Deallocate(&block); err != nil {
		t.Fatal(err)
	}
	if err := sourceCheckpoint(limit, nil, process, math.MaxUint64, ErrFastContextCanceled, owner); err != nil || owner.OwnerThread != 0 {
		t.Fatal("successful checkpoint must cooperate", err, owner.OwnerThread)
	}
	if tags.ShouldTrace("cooperate_detail") && !strings.Contains(traceOutput.String(), "subpaving, tid: 0\n") {
		t.Fatal("wrong source cooperation task", traceOutput.String())
	}
	if err := owner.EndSection(); err != nil {
		t.Fatal(err)
	}
}

func TestFastSearchUsesBoundCooperation(t *testing.T) {
	c, _, _ := fastProcessFixture(t)
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
			t.Error("retained cooperation owner", b.Used())
		}
	})
	if err := c.BindCooperation(nil); !errors.Is(err, arena.ErrConfig) {
		t.Fatal("nil cooperation binding", err)
	}
	copy := *owner
	if err := c.BindCooperation(&copy); !errors.Is(err, cooperate.ErrCooperationOwner) {
		t.Fatal("copied cooperation binding", err)
	}
	if err := c.BindCooperation(owner); err != nil {
		t.Fatal(err)
	}
	if err := owner.BeginSection(); err != nil {
		t.Fatal(err)
	}
	c.maxDepth = 0
	if err := c.Search(); err != nil || owner.OwnerThread != 0 {
		t.Fatal("source search checkpoint did not cooperate", err, owner.OwnerThread)
	}
	if err := owner.EndSection(); err != nil {
		t.Fatal(err)
	}
}

func TestFixedContextRootUsesBoundCooperation(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	owner, err := cooperate.NewCooperation(b)
	if err != nil {
		t.Fatal(err)
	}
	m, err := numeral.NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewFixedContext(m, &control.Reslimit{Limit: 10000}, nil, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	src, err := NewFixedInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	dst, err := NewFixedInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	lo, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	hi, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, close := range []func() error{dst.Close, src.Close, lo.Close, hi.Close, c.Close, m.Close, owner.Close} {
			if err := close(); err != nil {
				t.Error(err)
			}
		}
		if b.Used() != 0 {
			t.Error("retained root cooperation owner", b.Used())
		}
	})
	if err := c.BindCooperation(owner); err != nil {
		t.Fatal(err)
	}
	if err := owner.BeginSection(); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt64(lo, 2); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt64(hi, 4); err != nil {
		t.Fatal(err)
	}
	if err := src.SetFinite(lo, hi, false, false); err != nil {
		t.Fatal(err)
	}
	if err := c.XnEqY(dst, src, 3); err != nil || owner.OwnerThread != 0 {
		t.Fatal("context inverse root omitted borrowed cooperation", err, owner.OwnerThread)
	}
	if err := owner.EndSection(); err != nil {
		t.Fatal(err)
	}
}
