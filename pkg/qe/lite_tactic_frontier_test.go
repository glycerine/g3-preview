package qe

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/memory"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestLiteTacticAllocationFrontiersTraceReportAndCountedClose(t *testing.T) {
	for _, phase := range []string{"constructor", "apply", "cleanup", "translate", "counted-close", "trace", "closed-trace", "report"} {
		t.Run(phase, func(t *testing.T) {
			success := false
			for stop := int64(0); stop < 4096 && !success; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					f := newFMFixture(t)
					m := f.r.manager
					g := f.goal(false, false)
					result := f.result()
					fmMust(t, g.Assert(f.quantifiedDefinition()))
					var trace bytes.Buffer
					if phase == "trace" || phase == "closed-trace" {
						var writer io.Writer = &trace
						if phase == "closed-trace" {
							writer = fmClosedTrace{}
						}
						tags := diagnostic.NewTraceTagsWithWriter(writer)
						tags.Enable("qe")
						t.Cleanup(tags.Finalize)
						f.r.config.Trace = tags
					}
					var report *utext.NumericTextStream
					if phase == "report" {
						level, err := diagnostic.ProcessVerbosityLevel()
						fmMust(t, err)
						previous, err := diagnostic.BorrowProcessVerboseStream()
						fmMust(t, err)
						report, err = utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: m.Budget()})
						fmMust(t, err)
						fmMust(t, diagnostic.SetProcessVerboseStream(report))
						fmMust(t, diagnostic.SetProcessVerbosityLevel(10))
						t.Cleanup(func() {
							fmMust(t, diagnostic.SetProcessVerboseStream(previous))
							fmMust(t, diagnostic.SetProcessVerbosityLevel(level))
							fmMust(t, report.Close())
						})
					}
					var tt *LiteTactic
					var translated tactic.Tactic
					var err error
					var target *ast.ASTManager
					if phase != "constructor" {
						tt = f.liteTactic()
					}
					if phase == "counted-close" {
						fmMust(t, tt.Apply(g, result))
						fmMust(t, tt.IncRef())
					}
					if phase == "translate" {
						target, err = ast.NewASTManager(ast.ASTManagerConfig{Symbols: m.Symbols, Numerals: m.NumeralService(), Budget: m.Budget()})
						fmMust(t, err)
						t.Cleanup(func() { fmMust(t, target.Close()) })
					}
					m.Budget().FailAfter(stop)
					switch phase {
					case "constructor":
						tt, err = NewLiteTactic(m, nil, f.r.config)
					case "cleanup":
						err = tt.Cleanup()
					case "translate":
						translated, err = tt.Translate(target)
					case "counted-close":
						err = tt.DecRef()
					default:
						err = tt.Apply(g, result)
					}
					m.Budget().FailAfter(-1)
					var failure memory.ConstructionFailure
					if errors.As(err, &failure) {
						fmMust(t, failure.Cleanup())
					}
					if phase == "constructor" && tt != nil {
						fmMust(t, tt.Close())
					}
					if translated != nil {
						fmMust(t, translated.(*LiteTactic).Close())
					}
					if err == nil {
						success = true
						if phase == "trace" && f.r.config.Trace.ShouldTrace("qe") {
							for _, text := range []string{"src/qe/qe_lite.cpp:2589", "src/qe/qe_lite.cpp:2595", "(tactic-p 3)"} {
								if !strings.Contains(trace.String(), text) {
									t.Fatal("missing source trace", text, trace.String())
								}
							}
						}
						if phase == "report" {
							text, err := report.String()
							fmMust(t, err)
							if !strings.HasPrefix(text, "(qe-lite :num-exprs ") {
								t.Fatal("missing tactic report", text)
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
				t.Fatal("no successful frontier")
			}
		})
	}
}
