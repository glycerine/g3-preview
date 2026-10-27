package smtcore

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func caseSplitFactoryParams(t *testing.T, strategy, relevancy uint32, auto bool) (*ast.ASTManager, *uparams.ParamsContext, *uparams.GlobalParamsService, *uparams.ParamsRef) {
	t.Helper()
	m, cfg := smtModelGeneratorFixture(t)
	p, err := cfg.Context.NewValue()
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, cfg.Context.Del(&p)) })
	key := "case_split"
	astTestMust(t, cfg.Context.SetUIntName(&p, &key, strategy))
	key = "relevancy"
	astTestMust(t, cfg.Context.SetUIntName(&p, &key, relevancy))
	key = "auto_config"
	astTestMust(t, cfg.Context.SetBoolName(&p, &key, auto))
	return m, cfg.Context, cfg.Globals, &p
}

func TestSMTCaseSplitFactoryWarningsSequentialAndSolverUpdate(t *testing.T) {
	for _, strategy := range []uint32{0, 1, 2, 3, 4, 5, 6, 7, 61, ^uint32(0)} {
		for _, auto := range []bool{false, true} {
			for _, relevancy := range []uint32{1, 2} {
				t.Run(fmt.Sprintf("%d/%t/%d", strategy, auto, relevancy), func(t *testing.T) {
					m, c, g, p := caseSplitFactoryParams(t, strategy, relevancy, auto)
					k := smtTestKernel(t, m)
					var out bytes.Buffer
					k.queueWarnings = diagnostic.NewWarningService(&out)
					astTestMust(t, k.ConfigurePreprocessing(c, g, p))
					selected := strategy
					if strategy > 6 {
						selected = 0
					}
					want := ""
					if strategy >= 3 && strategy <= 5 {
						if relevancy < 2 {
							selected = 0
							want = "WARNING: relevancy must be enabled to use option CASE_SPLIT=3, 4 or 5\n"
						} else if auto {
							selected = 0
							want = "WARNING: auto configuration (option AUTO_CONFIG) must be disabled to use option CASE_SPLIT=3, 4 or 5\n"
						}
					}
					state, err := k.smtParams.State()
					astTestMust(t, err)
					if k.caseSplitQueue.strategy != selected || k.internalizer.caseSplitStrategy != selected || state.CaseSplitStrategy != strategy || k.smtState.CaseSplitStrategy != strategy || out.String() != want {
						t.Fatal("factory/update ordering", k.caseSplitQueue.strategy, state.CaseSplitStrategy, out.String(), want)
					}
					q := k.caseSplitQueue
					astTestMust(t, k.UpdateParams(p))
					astTestMust(t, k.Push())
					astTestMust(t, k.Pop(1))
					if k.caseSplitQueue != q || out.String() != want {
						t.Fatal("updates/setup repeated factory", out.String())
					}
				})
			}
		}
	}
}

func TestSMTCaseSplitFactoryBorrowedMutationAndOwnedCopy(t *testing.T) {
	for _, borrow := range []bool{false, true} {
		m, c, g, p := caseSplitFactoryParams(t, 5, 1, true)
		source, err := NewSMTParams(c, p, g, m.Budget())
		astTestMust(t, err)
		t.Cleanup(func() { astTestMust(t, source.Close()) })
		k := smtTestKernel(t, m)
		var out bytes.Buffer
		k.queueWarnings = diagnostic.NewWarningService(&out)
		astTestMust(t, k.configurePreprocessingParams(c, g, p, source, borrow))
		state, err := source.State()
		astTestMust(t, err)
		wantSource := uint32(5)
		if borrow {
			wantSource = 0
		}
		if state.CaseSplitStrategy != wantSource || k.smtParams.state.CaseSplitStrategy != 0 || k.caseSplitQueue.strategy != 0 || k.smtState.CaseSplitStrategy != 0 {
			t.Fatal("factory did not mutate actual borrowed/copy owner", borrow, state.CaseSplitStrategy, k.smtParams.state.CaseSplitStrategy)
		}
		want := "WARNING: relevancy must be enabled to use option CASE_SPLIT=3, 4 or 5\n"
		if out.String() != want {
			t.Fatal(out.String(), want)
		}
		astTestMust(t, k.UpdateParams(p))
		if k.caseSplitQueue.strategy != 0 || k.smtParams.state.CaseSplitStrategy != 5 || out.String() != want {
			t.Fatal("later source update replaced class or repeated warning")
		}
	}
}

type caseSplitFactoryWarningWriter struct {
	output         bytes.Buffer
	owner          *SMTParams
	kernel         *SMTKernel
	calls, flushes int
	failAllocation bool
	refuse         bool
}

func (w *caseSplitFactoryWarningWriter) Write(p []byte) (int, error) {
	if w.calls == 0 {
		if w.kernel.internalizer.random != *z3rand.New(17) {
			return 0, fmt.Errorf("factory warning preceded RNG construction")
		}
		w.kernel.internalizer.random.Next()
		if w.owner.state.CaseSplitStrategy != 3 || w.kernel.caseSplitQueue != nil || w.kernel.internalizer.basicActivity != nil {
			return 0, fmt.Errorf("factory warning moved after owner mutation/queue allocation")
		}
		if w.failAllocation {
			w.owner.Budget.FailAfter(0)
		}
	}
	w.calls++
	if w.refuse {
		return 0, io.ErrClosedPipe
	}
	return w.output.Write(p)
}
func (w *caseSplitFactoryWarningWriter) Flush() error { w.flushes++; return nil }

func TestSMTCaseSplitFactoryWarningBeforeMutationAndAllocationRefusal(t *testing.T) {
	for _, fail := range []bool{false, true} {
		m, c, g, p := caseSplitFactoryParams(t, 3, 1, true)
		key := "random_seed"
		astTestMust(t, c.SetUIntName(p, &key, 17))
		source, err := NewSMTParams(c, p, g, m.Budget())
		astTestMust(t, err)
		t.Cleanup(func() { astTestMust(t, source.Close()) })
		k := smtTestKernel(t, m)
		w := &caseSplitFactoryWarningWriter{owner: source, kernel: k, failAllocation: fail}
		warnings := diagnostic.NewWarningService(w)
		astTestMust(t, warnings.SetWarningStream(w))
		k.queueWarnings = warnings
		err = k.configurePreprocessingParams(c, g, p, source, true)
		m.Budget().FailAfter(-1)
		rng := z3rand.New(17)
		rng.Next()
		if k.internalizer.random != *rng {
			t.Fatal("factory reseeded after warning callback")
		}
		if fail {
			if !errors.Is(err, arena.ErrInjectedFailure) || k.caseSplitQueue != nil {
				t.Fatal("missing queue constructor refusal", err)
			}
		} else {
			astTestMust(t, err)
		}
		if source.state.CaseSplitStrategy != 0 || w.calls != 3 || w.flushes != 1 || w.output.String() != "WARNING: relevancy must be enabled to use option CASE_SPLIT=3, 4 or 5\n" {
			t.Fatal("source warning/mutation prefix", fail, source.state.CaseSplitStrategy, w.calls, w.flushes, w.output.String())
		}
	}
}

func TestSMTCaseSplitFactoryWarningDisabledOrRefusedStillMutates(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		m, c, g, p := caseSplitFactoryParams(t, 3, 2, true)
		source, err := NewSMTParams(c, p, g, m.Budget())
		astTestMust(t, err)
		t.Cleanup(func() { astTestMust(t, source.Close()) })
		k := smtTestKernel(t, m)
		w := &smtLRAOutputPrefix{remaining: 0}
		warnings := diagnostic.NewWarningService(w)
		astTestMust(t, warnings.SetWarningStream(w))
		astTestMust(t, warnings.Enable(!disabled))
		k.queueWarnings = warnings
		astTestMust(t, k.configurePreprocessingParams(c, g, p, source, true))
		if source.state.CaseSplitStrategy != 0 || k.caseSplitQueue.strategy != 0 || w.text.Len() != 0 {
			t.Fatal("warning settings changed factory mutation")
		}
	}
}

func TestSMTCaseSplitFactoryDoesNotReplaceAlreadyConstructedRawContext(t *testing.T) {
	m, c, g, p := caseSplitFactoryParams(t, 3, 1, true)
	k := smtTestKernel(t, m)
	_, err := k.Check()
	astTestMust(t, err)
	queue := k.caseSplitQueue
	var out bytes.Buffer
	k.queueWarnings = diagnostic.NewWarningService(&out)
	astTestMust(t, k.ConfigurePreprocessing(c, g, p))
	if k.caseSplitQueue != queue || out.Len() != 0 || k.smtParams.state.CaseSplitStrategy != 3 {
		t.Fatal("late parameter services reconstructed queue", out.String())
	}
}
