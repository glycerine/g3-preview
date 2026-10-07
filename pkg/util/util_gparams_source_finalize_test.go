// Source: src/util/gparams.cpp:81-123,633-644; src/util/params.cpp:59-86.
package util

import (
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// No registered gparams destructor ENSURE campaign exists. These fixtures test
// the literal source destruction frontier and separately named owning abort.
func TestGlobalParamsSourceFinalizeSparseShrinkRefusal(t *testing.T) {
	ctx, sy, _ := paramsTestOwner(t)
	b := arena.NewBudget(^uint64(0))
	g, e := NewGlobalParams(sy, ctx, gparamsTestConfig(b))
	paramsMust(t, e)
	for i := 0; i < 26; i++ {
		k, e := sy.Intern("source-module-" + strconv.Itoa(i))
		paramsMust(t, e)
		_, e = g.GetParams(k)
		paramsMust(t, e)
	}
	if len(g.ModuleParams.Entries) != 64 {
		t.Fatal("actual source growth", len(g.ModuleParams.Entries))
	}
	paramsMust(t, g.Reset())
	if len(g.ModuleParams.Entries) != 64 {
		t.Fatal("actual first reset retains its dense table", len(g.ModuleParams.Entries))
	}
	k, e := sy.Intern("source-sparse-module")
	paramsMust(t, e)
	_, e = g.GetParams(k)
	paramsMust(t, e)
	if b.Used() != gparamsGraph(g) {
		t.Fatal("before refusal graph", b.Used(), gparamsGraph(g))
	}
	b.FailAfter(0)
	e = g.SourceFinalize()
	if !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("actual shrinking allocation", e)
	}
	if len(g.ModuleParams.Entries) != 0 {
		t.Fatal("source old table freed before allocation", len(g.ModuleParams.Entries))
	}
	for _, cell := range g.Cells {
		if cell != nil {
			t.Fatal("completed ascending deletion retained cell", cell)
		}
	}
	if b.Used() != gparamsGraph(g) {
		t.Fatal("immediate stopped graph", b.Used(), gparamsGraph(g))
	}
	before := b.Used()
	if again := g.SourceFinalize(); again != e {
		t.Fatal("sticky original source stop", again, e)
	}
	if e := g.Reset(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("ordinary reset after destruction started", e)
	}
	if b.Used() != before {
		t.Fatal("source replay changed ownership", b.Used(), before)
	}
	b.FailAfter(-1)
	paramsMust(t, g.CleanupStopped())
	if g.sourceCompleted || !g.closed || g.sourceStage != 10 {
		t.Fatal("owning abort must not claim source completion", g.sourceCompleted, g.closed, g.sourceStage)
	}
	if b.Used() != 0 {
		t.Fatal("owning suffix refund", b.Used())
	}
	if again := g.SourceFinalize(); again != e {
		t.Fatal("owning abort erased source history", again, e)
	}
	paramsMust(t, g.Close())
}

func TestGlobalParamsSourceFinalizeFiniteNamedModuleOwners(t *testing.T) {
	for n := 0; n <= 20; n++ {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			ctx, sy, _ := paramsTestOwner(t)
			b := arena.NewBudget(^uint64(0))
			g, e := NewGlobalParams(sy, ctx, gparamsTestConfig(b))
			paramsMust(t, e)
			var snapshots []ParamsRef
			var keys []Symbol
			for i := 0; i < n; i++ {
				module, e := sy.Intern("finite-module-" + strconv.Itoa(i))
				paramsMust(t, e)
				key, e := sy.Intern("finite-key")
				paramsMust(t, e)
				ref, e := g.GetParams(module)
				paramsMust(t, e)
				paramsMust(t, ctx.SetUInt(ref, key, uint32(100+i)))
				snap, e := g.GetModule(module)
				paramsMust(t, e)
				snapshots = append(snapshots, snap)
				keys = append(keys, key)
				d, e := NewParamDescrs(sy, ParamDescrsConfig{Budget: b})
				paramsMust(t, e)
				paramsMust(t, d.InsertName("finite-key", ParamUInt, pdLiteral("source descriptor text"), nil, nil))
				paramsMust(t, g.RegisterModule("finite-module-"+strconv.Itoa(i), d))
			}
			if b.Used() != gparamsGraph(g) {
				t.Fatal("named graph", b.Used(), gparamsGraph(g))
			}
			paramsMust(t, g.SourceFinalize())
			if !g.sourceCompleted || g.sourceCause != nil || g.sourceStage != 10 || b.Used() != 0 {
				t.Fatal("completed source teardown", g.sourceStage, g.sourceCause, b.Used())
			}
			for i := range snapshots {
				v, e := ctx.GetUInt(&snapshots[i], keys[i], 0)
				paramsMust(t, e)
				if v != uint32(100+i) {
					t.Fatal("borrowed output surviving owned source destruction", i, v)
				}
				paramsMust(t, ctx.Del(&snapshots[i]))
			}
			paramsMust(t, g.Close())
		})
	}
}

func TestGlobalParamsSourceFinalizeEveryActualShrinkReservation(t *testing.T) {
	refusals, successes := 0, 0
	for fault := int64(0); fault < 4; fault++ {
		t.Run(strconv.FormatInt(fault, 10), func(t *testing.T) {
			ctx, sy, _ := paramsTestOwner(t)
			b := arena.NewBudget(^uint64(0))
			g, e := NewGlobalParams(sy, ctx, gparamsTestConfig(b))
			paramsMust(t, e)
			for i := 0; i < 26; i++ {
				k, e := sy.Intern("campaign-module-" + strconv.Itoa(i))
				paramsMust(t, e)
				_, e = g.GetParams(k)
				paramsMust(t, e)
			}
			paramsMust(t, g.Reset())
			k, e := sy.Intern("campaign-sparse")
			paramsMust(t, e)
			_, e = g.GetParams(k)
			paramsMust(t, e)
			b.FailAfter(fault)
			e = g.SourceFinalize()
			if b.Used() != gparamsGraph(g) {
				t.Fatal("immediate actual graph before lifting refusal", b.Used(), gparamsGraph(g))
			}
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal(e)
				}
				refusals++
				if g.sourceStage != 2 || g.sourceIndex != 0 || g.sourceCompleted {
					t.Fatal("exact failed source shrink cursor", g.sourceStage, g.sourceIndex)
				}
			} else {
				successes++
			}
			b.FailAfter(-1)
			paramsMust(t, g.Close())
			if b.Used() != 0 {
				t.Fatal("actual once-only refund", b.Used())
			}
		})
		if successes != 0 {
			t.Logf("actual destructor reservation refusals=%d firstsuccess=%d", refusals, fault)
			break
		}
	}
	if refusals != 1 || successes != 1 {
		t.Fatal("complete actual reservation horizon", refusals, successes)
	}
}

func TestGlobalParamsSourceFinalizeCopiedOwnerBeforeMutex(t *testing.T) {
	ctx, sy, _ := paramsTestOwner(t)
	b := arena.NewBudget(^uint64(0))
	g, e := NewGlobalParams(sy, ctx, gparamsTestConfig(b))
	paramsMust(t, e)
	copyOwner := new(GlobalParams)
	reflect.ValueOf(copyOwner).Elem().Set(reflect.ValueOf(g).Elem())
	before := b.Used()
	if e := copyOwner.SourceFinalize(); !errors.Is(e, ErrGlobalParamsOwner) {
		t.Fatal(e)
	}
	if e := copyOwner.CleanupStopped(); !errors.Is(e, ErrGlobalParamsOwner) {
		t.Fatal(e)
	}
	if b.Used() != before || g.sourceStarted {
		t.Fatal("copied owner changed actual source owner")
	}
	paramsMust(t, g.SourceFinalize())
}

func TestGlobalParamsSourceFinalizeEmptyMembersAndGuards(t *testing.T) {
	ctx, sy, _ := paramsTestOwner(t)
	b := arena.NewBudget(^uint64(0))
	g, e := NewGlobalParams(sy, ctx, gparamsTestConfig(b))
	paramsMust(t, e)
	b.FailAfter(0)
	paramsMust(t, g.SourceFinalize())
	if b.Used() != 0 {
		t.Fatal("all actual source members and cold metadata", b.Used())
	}
	paramsMust(t, g.SourceFinalize())
	paramsMust(t, g.Close())
	if e := g.Reset(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("consumed owner", e)
	}
}
