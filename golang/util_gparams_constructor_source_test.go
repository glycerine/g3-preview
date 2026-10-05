// Source: src/util/gparams.cpp:81-105,633-644; src/util/params.cpp:55-77,210-217.
// Author: Leonardo de Moura (2012-11-29; params: 2011-05-09).
// Licensed under the MIT license; see LICENSE.txt in the source distribution.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"strconv"
	"testing"
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
					if g.constructorPendingDescriptor != nil && g.globalDescriptors != nil {
						t.Fatal("failed descriptor published as completed member")
					}
					if g.constructorUnwind != 0 || len(g.moduleDescriptors.entries) != 0 || len(g.moduleDescriptions.entries) != 0 || len(g.moduleParams.entries) != 0 || g.globalDescriptors != nil {
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
				if !g.constructorReady || g.constructorCompleted != 5 || g.globalParams.slot != 0 {
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
	if g == nil || g.self != g || g.budget != b {
		t.Fatal("actual constructor owner identity")
	}
	n := gparamsGraph(g)
	if d := g.constructorPendingDescriptor; d != nil {
		if d.self != d || d.budget != b || d.constructorReady || d.closed || d == g.globalDescriptors {
			t.Fatal("actual unfinished child identity")
		}
		n += d.ownerBytes + pdEntryBytes(len(d.entries)) + pdNameBytes(cap(d.names))
		if d.texts != nil {
			n += gparamsVectorGraph(d.texts.Stats())
		}
	}
	if n != b.Used() {
		t.Fatal("named constructor physical/cold graph", n, b.Used())
	}
	if g.ownerBytes != uint64(reflect.TypeOf(GlobalParams{}).Size()) && !g.closed {
		t.Fatal("actual charged header")
	}
}
