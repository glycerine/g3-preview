// Source: src/util/gparams.cpp:81-123,633-644; src/util/params.cpp:59-86.
package util

import (
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

// No registered gparams destructor ENSURE campaign exists. These fixtures test
// the literal source destruction frontier and separately named owning abort.
func TestGlobalParamsSourceFinalizeFiniteNamedModuleOwners(t *testing.T) {
	for n := 0; n <= 20; n++ {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			ctx, sy, _ := paramsTestOwner(t)
			b := arena.NewBudget(^uint64(0))
			g, e := NewGlobalParams(sy, ctx, gparamsTestConfig(b))
			paramsMust(t, e)
			var snapshots []ParamsRef
			var keys []usymbol.Symbol
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
