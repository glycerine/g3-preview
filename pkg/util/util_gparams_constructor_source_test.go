// Source: src/util/gparams.cpp:81-105,633-644; src/util/params.cpp:55-77,210-217.
package util

import (
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestGlobalParamsSourceConstructorAllActualNestedPrefixes(t *testing.T) {
	ctx, sy, _ := paramsTestOwner(t)
	success := false
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			b := arena.NewBudget(1 << 26)
			b.FailAfter(ordinal)
			g, e := newSourceGlobalParams(sy, ctx, gparamsTestConfig(b))
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal("actual constructor reservation", e)
				}
				if g != nil {
					if g.constructorReady || g.sourceStarted || g.sourceStage != 0 || g.constructorCause == nil {
						t.Fatal("incomplete source ctor ran imp destructor body")
					}
					if g.constructorPendingDescriptor != nil && g.GlobalDescriptors != nil {
						t.Fatal("failed descriptor published as completed member")
					}
					if g.constructorUnwind != 0 || g.ModuleDescriptors.size() != 0 || g.ModuleDescriptions.size() != 0 || g.ModuleParams.size() != 0 || g.GlobalDescriptors != nil {
						t.Fatal("source completed members did not unwind before failed ctor return", g.constructorCompleted, g.constructorUnwind)
					}
					t.Logf("failed ctor source completed=%d, unfinished descriptor=%v", g.constructorCompleted, g.constructorPendingDescriptor != nil)
					gparamsConstructorAudit(t, g, b)
					if e := g.SourceFinalize(); !errors.Is(e, ErrGlobalParamsUnconstructed) || g.sourceStarted {
						t.Fatal("full destructor reached before constructor completion", e)
					}
					if _, e := g.GetModuleName("unused"); !errors.Is(e, ErrGlobalParamsUnconstructed) {
						t.Fatal("ordinary cold owner looked up incomplete members", e)
					}
					if d := g.constructorPendingDescriptor; d != nil {
						if _, e := d.Size(); !errors.Is(e, ErrParamDescrsPrecondition) {
							t.Fatal("failed inner descriptor ordinary access", e)
						}
					}
					// Refusal stays active while explicit owning abort acknowledges real roots.
					paramsMust(t, g.abortConstructor())
				}
			} else {
				success = true
				if !g.constructorReady || g.constructorCompleted != 5 || g.globalParams.Slot != 0 {
					t.Fatal("source member completion")
				}
				gparamsConstructorAudit(t, g, b)
				paramsMust(t, g.SourceFinalize())
			}
			if b.Used() != 0 {
				t.Fatal("actual constructor roots not drained", b.Used())
			}
		})
		if success {
			t.Logf("first actual constructor success %d", ordinal)
			break
		}
	}
	if !success {
		t.Fatal("constructor campaign did not reach success")
	}
}

func gparamsConstructorAudit(t *testing.T, g *GlobalParams, b *arena.Budget) {
	t.Helper()
	if g == nil || g.self != g || g.Budget != b {
		t.Fatal("actual constructor owner identity")
	}
	n := gparamsGraph(g)
	if d := g.constructorPendingDescriptor; d != nil {
		if d.Self != d || d.Budget != b || d.constructorReady || d.closed || d == g.GlobalDescriptors {
			t.Fatal("actual unfinished child identity")
		}
		n += d.OwnerBytes
		if d.Texts != nil {
			n += gparamsVectorGraph(d.Texts.Stats())
		}
	}
	if n != b.Used() {
		t.Fatal("named constructor physical/cold graph", n, b.Used())
	}
	if g.OwnerBytes != uint64(reflect.TypeOf(GlobalParams{}).Size()) && !g.closed {
		t.Fatal("actual charged header")
	}
}
