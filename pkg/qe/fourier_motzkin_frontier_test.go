package qe

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
)

func TestFourierMotzkinAllocationFrontiersAndClosedTrace(t *testing.T) {
	for _, phase := range []string{"constructor", "real", "real-large", "integer-unit", "integer-nonunit", "literal-resolution", "inconsistent", "tautology", "repeat-boolean", "trace", "closed-trace"} {
		t.Run(phase, func(t *testing.T) {
			success := false
			for stop := int64(0); stop < 4096 && !success; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					f := newFMFixture(t)
					mode := phase
					if phase == "constructor" || phase == "trace" || phase == "closed-trace" {
						mode = "real"
					}
					if phase == "repeat-boolean" {
						mode = "boolean-clause"
					}
					forms, _, _, _ := fmDriverCase(f, mode)
					conjs := f.conjunctions(forms...)
					var trace bytes.Buffer
					if phase == "trace" || phase == "closed-trace" {
						var writer io.Writer = &trace
						if phase == "closed-trace" {
							writer = fmClosedTrace{}
						}
						tags := diagnostic.NewTraceTagsWithWriter(writer)
						tags.Enable("qe_lite")
						t.Cleanup(tags.Finalize)
						f.r.config.Trace = tags
					}
					var d *FourierMotzkin
					var err error
					if phase != "constructor" {
						d = f.eliminator()
					}
					if phase == "repeat-boolean" {
						fmMust(t, d.Apply(conjs))
						conjs = f.conjunctions(forms...)
					}
					f.r.manager.Budget().FailAfter(stop)
					if phase == "constructor" {
						d, err = NewFourierMotzkin(f.r.manager, f.r.selection, f.r.config)
					} else {
						err = d.Apply(conjs)
					}
					f.r.manager.Budget().FailAfter(-1)
					var failure memory.ConstructionFailure
					if errors.As(err, &failure) {
						fmMust(t, failure.Cleanup())
					}
					if phase == "constructor" && d != nil {
						fmMust(t, d.Close())
					}
					if err == nil {
						success = true
						if phase == "trace" && f.r.config.Trace.ShouldTrace("qe_lite") {
							for _, text := range []string{"src/qe/qe_lite.cpp:2262", "src/qe/qe_lite.cpp:1878", "src/qe/qe_lite.cpp:2177", "src/qe/qe_lite.cpp:2136", "new_dep: 0\n", "src/qe/qe_lite.cpp:2212"} {
								if !strings.Contains(trace.String(), text) {
									t.Fatal("missing source trace", text, trace.String())
								}
							}
						}
					} else if !errors.Is(err, arena.ErrInjectedFailure) {
						var memout memory.Failure
						if !errors.As(err, &memout) || memout.Kind != memory.FailureMemout {
							t.Fatal(err)
						}
					}
				})
				if t.Failed() {
					t.Fatalf("%s allocation frontier %d", phase, stop)
				}
			}
			if !success {
				t.Fatal("no successful allocation frontier")
			}
		})
	}
}
