package smtcore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/symbol"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestSMTAssertedWellSortedConditionalErasure(t *testing.T) {
	m, s := smtAssertedNNFFixture(t, false)
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	p, err := m.MkConst(astTestSymbol(t, m, "well-sorted-asserted-p"), b)
	astTestMust(t, err)
	astTestMust(t, s.Assert(p))
	d := diagnostic.BorrowProcessDebugContext()
	previousTag, err := d.IsDebugEnabled("well_sorted")
	astTestMust(t, err)
	previousAssertions, err := d.AssertionsEnabled()
	astTestMust(t, err)
	defer func() {
		if previousTag {
			astTestMust(t, d.EnableDebug("well_sorted"))
		} else {
			astTestMust(t, d.DisableDebug("well_sorted"))
		}
		astTestMust(t, d.EnableAssertions(previousAssertions))
	}()
	// Make evaluation detectable without changing the owned expression: restore
	// its borrowed row slot before the kernel's registered cleanup runs.
	original := s.kernel.assertions[0].Formula
	defer func() { s.kernel.assertions[0].Formula = original }()
	s.kernel.assertions[0].Formula = ast.ASTHandle{}
	astTestMust(t, d.EnableAssertions(true))
	astTestMust(t, d.DisableDebug("well_sorted"))
	astTestMust(t, s.kernel.checkAssertedWellSorted(239))
	astTestMust(t, d.EnableDebug("well_sorted"))
	err = s.kernel.checkAssertedWellSorted(239)
	if diagnostic.DebugBuild {
		if !errors.Is(err, ast.ErrASTHandle) {
			t.Fatal("enabled source condition did not inspect the row", err)
		}
	} else {
		astTestMust(t, err)
	}
	astTestMust(t, d.EnableAssertions(false))
	astTestMust(t, s.kernel.checkAssertedWellSorted(239))
}

func TestSMTAssertedVerbosePassSequenceCPPReference(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to the prebuilt C++ executable")
	}
	for _, quantified := range []bool{false, true} {
		t.Run(fmt.Sprint(quantified), func(t *testing.T) {
			m, s := smtAssertedNNFFixture(t, false)
			smtDisableAutoConfig(t, s)
			b, err := m.MkBoolSort()
			astTestMust(t, err)
			p, err := m.MkConst(astTestSymbol(t, m, "verbose-source-p"), b)
			astTestMust(t, err)
			script := "(declare-const verbose-source-p Bool)(assert verbose-source-p)"
			if quantified {
				params, err := s.kernel.paramContext.NewValue()
				astTestMust(t, err)
				name := "macro_finder"
				astTestMust(t, s.kernel.paramContext.SetBoolName(&params, &name, true))
				astTestMust(t, s.UpdateParams(&params))
				astTestMust(t, s.kernel.paramContext.Del(&params))
				one, err := m.ArithmeticIntegerNumeralText("1")
				astTestMust(t, err)
				integer, err := m.GetSort(one)
				astTestMust(t, err)
				x, err := m.MkVar(0, integer)
				astTestMust(t, err)
				f, err := m.MkFuncDecl(astTestSymbol(t, m, "verbose-source-f"), []ast.ASTHandle{integer}, integer, ast.ASTFuncDeclInfo{})
				astTestMust(t, err)
				fx, err := m.MkApp(f, []ast.ASTHandle{x})
				astTestMust(t, err)
				body, err := m.MkEq(fx, x)
				astTestMust(t, err)
				p, err = m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{integer}, Names: []symbol.Symbol{symbol.Numerical(0)}, Body: body})
				astTestMust(t, err)
				script = "(declare-fun verbose-source-f (Int) Int)(assert (forall ((x Int)) (= (verbose-source-f x) x)))"
			}
			astTestMust(t, s.Assert(p))
			out := smtAssertedVerboseFixture(t, 10000)
			ready, err := s.kernel.reduceAssertedPending()
			astTestMust(t, err)
			if !ready {
				t.Fatal("native preprocessing stopped")
			}
			native, err := out.String()
			astTestMust(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, reference, "-in")
			prefix := "(set-option :auto_config false)(set-option :verbose 10000)"
			if quantified {
				prefix += "(set-option :smt.macro_finder true)"
			}
			cmd.Stdin = strings.NewReader(prefix + "(set-logic ALL)" + script + "(check-sat-using smt)")
			var cppOut, cppVerbose bytes.Buffer
			cmd.Stdout, cmd.Stderr = &cppOut, &cppVerbose
			astTestMust(t, cmd.Run())
			if strings.TrimSpace(cppOut.String()) != "sat" {
				t.Fatal("C++ ground assertion result", cppOut.String(), cppVerbose.String())
			}
			// Compare the asserted manager's diagnostics; the surrounding C++ tactic
			// and search engine emit additional verbose messages of their own.
			selectPasses := func(text string) string {
				var selected []string
				for _, line := range strings.Split(text, "\n") {
					if line == "(smt.simplifier-done)" || line == "(smt.reducing)" || strings.HasPrefix(line, "total size: ") {
						selected = append(selected, line)
						continue
					}
					for p := assertedPropagate; p <= assertedBVSharing; p++ {
						if line == "(smt."+p.sourceName()+")" {
							selected = append(selected, line)
							break
						}
					}
				}
				return strings.Join(selected, "\n")
			}
			got, want := selectPasses(native), selectPasses(cppVerbose.String())
			if !strings.Contains(want, "(smt.simplifier-done)") || got != want {
				t.Fatalf("asserted pass order differs\nGo:\n%s\nC++:\n%s\nfull C++ diagnostics:\n%s", got, want, cppVerbose.String())
			}
		})
	}
}
func smtAssertedVerboseFixture(t *testing.T, level uint32) *utext.NumericTextStream {
	t.Helper()
	previous, err := diagnostic.BorrowProcessVerboseStream()
	astTestMust(t, err)
	previousLevel, err := diagnostic.ProcessVerbosityLevel()
	astTestMust(t, err)
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{})
	astTestMust(t, err)
	astTestMust(t, diagnostic.SetProcessVerboseStream(out))
	astTestMust(t, diagnostic.SetProcessVerbosityLevel(level))
	t.Cleanup(func() {
		astTestMust(t, diagnostic.SetProcessVerboseStream(previous))
		astTestMust(t, diagnostic.SetProcessVerbosityLevel(previousLevel))
		astTestMust(t, out.Close())
	})
	return out
}

func TestSMTAssertedSetupMutatesSharedSourcePolicies(t *testing.T) {
	for _, mode := range []uint32{smtLiftITENone, smtLiftITEConservative, smtLiftITEFull} {
		for _, nonground := range []uint32{smtLiftITENone, smtLiftITEConservative, smtLiftITEFull} {
			for _, level := range []uint32{0, 2} {
				_, s := smtAssertedNNFFixture(t, false)
				smtDisableAutoConfig(t, s)
				k := s.kernel
				k.preprocessState.LiftITE = mode
				k.preprocessState.NonGroundLiftITE = nonground
				k.smtState.RelevancyLvl = level
				k.smtState.RelevancyLemma = true
				astTestMust(t, k.setupAssertedBounds(false))
				want := nonground
				if mode == smtLiftITEFull || mode == smtLiftITEConservative && nonground == smtLiftITEConservative {
					want = smtLiftITENone
				}
				if k.preprocessState.NonGroundLiftITE != want || k.smtParams.preprocess.state.NonGroundLiftITE != want {
					t.Fatal("source setup did not publish redundant-pass suppression", mode, nonground)
				}
				if k.smtState.RelevancyLemma != (level != 0) || k.smtParams.state.RelevancyLemma != (level != 0) {
					t.Fatal("source setup did not publish the relevancy-lemma guard", level)
				}
			}
		}
	}
}

func TestSMTAssertedVerbosePassNamesSizeAndSkippedGuards(t *testing.T) {
	m, s := smtAssertedNNFFixture(t, false)
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	p, err := m.MkConst(astTestSymbol(t, m, "verbose-asserted-p"), b)
	astTestMust(t, err)
	astTestMust(t, s.Assert(p))
	astTestMust(t, s.Assert(p))
	out := smtAssertedVerboseFixture(t, 10000)
	proceed, canceled, err := s.kernel.invokeAssertedPass(assertedReduce)
	astTestMust(t, err)
	if !proceed || canceled {
		t.Fatal("ordinary reduction unexpectedly stopped")
	}
	got, err := out.String()
	astTestMust(t, err)
	if got != "(smt.reduce-asserted)\ntotal size: 1\n" {
		t.Fatal("source pass label/shared-root total", got)
	}
	s.kernel.assertedHasQuantifiers = false
	s.kernel.preprocessState.MacroFinder = true
	_, _, err = s.kernel.invokeAssertedPass(assertedMacros)
	astTestMust(t, err)
	after, err := out.String()
	astTestMust(t, err)
	if after != got {
		t.Fatal("skipped source pass emitted diagnostics")
	}
}

func TestSMTAssertedLowLevelDisplayUsesSharedDefinitions(t *testing.T) {
	m, s := smtAssertedNNFFixture(t, false)
	b, err := m.MkBoolSort()
	astTestMust(t, err)
	p, err := m.MkConst(astTestSymbol(t, m, "low-level-asserted-p"), b)
	astTestMust(t, err)
	astTestMust(t, s.Assert(p))
	astTestMust(t, s.Assert(p))
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: m.Budget()})
	astTestMust(t, err)
	defer func() { astTestMust(t, out.Close()) }()
	astTestMust(t, s.kernel.DisplayAssertedFormulasLowLevel(out))
	id, err := m.GetASTID(p)
	astTestMust(t, err)
	got, err := out.String()
	astTestMust(t, err)
	want := fmt.Sprintf("#%d := low-level-asserted-p\nasserted formulas:\n#%d #%d \n", id, id, id)
	if got != want {
		t.Fatalf("low-level asserted display: got %q, want %q", got, want)
	}
}

type assertedRefusingTrace struct{}

func (assertedRefusingTrace) Write([]byte) (int, error) { return 0, os.ErrClosed }

func TestSMTAssertedDriverTraceMetadataAndRefusals(t *testing.T) {
	for _, closed := range []bool{false, true} {
		m, s := smtAssertedNNFFixture(t, false)
		var out bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&out)
		if closed {
			tags = diagnostic.NewTraceTagsWithWriter(assertedRefusingTrace{})
		}
		for _, tag := range []string{"before_reduce", "reduce_step_ll", "after_reduce", "after_reduce_ll", "asserted_formulas", "macros"} {
			tags.Enable(tag)
		}
		s.kernel.modelConfig.Trace = tags
		b, err := m.MkBoolSort()
		astTestMust(t, err)
		p, err := m.MkConst(astTestSymbol(t, m, "trace-asserted-p"), b)
		astTestMust(t, err)
		astTestMust(t, s.Assert(p))
		ready, err := s.kernel.reduceAssertedPending()
		astTestMust(t, err)
		if !ready {
			t.Fatal("trace refusal stopped preprocessing")
		}
		if !closed && tags.ShouldTrace("before_reduce") {
			for _, want := range []string{"src/smt/asserted_formulas.cpp:238", "src/smt/asserted_formulas.cpp:285", "src/smt/asserted_formulas.cpp:263", "src/smt/asserted_formulas.cpp:264", "src/smt/asserted_formulas.cpp:265", "inconsistent: 0"} {
				if !strings.Contains(out.String(), want) {
					t.Fatal("missing source trace metadata/body", want, out.String())
				}
			}
		} else if out.Len() != 0 {
			t.Fatal("erased or refused trace emitted output")
		}
	}
}

func TestSMTAssertedMacroTraceDisplaysHeadAndDefinition(t *testing.T) {
	m, s, _ := smtAssertedMacroFixture(t, false, false)
	one, err := m.ArithmeticIntegerNumeralText("1")
	astTestMust(t, err)
	integer, err := m.GetSort(one)
	astTestMust(t, err)
	x, err := m.MkVar(0, integer)
	astTestMust(t, err)
	f, err := m.MkFuncDecl(astTestSymbol(t, m, "trace-macro-f"), []ast.ASTHandle{integer}, integer, ast.ASTFuncDeclInfo{})
	astTestMust(t, err)
	fx, err := m.MkApp(f, []ast.ASTHandle{x})
	astTestMust(t, err)
	body, err := m.MkEq(fx, x)
	astTestMust(t, err)
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{integer}, Names: []symbol.Symbol{symbol.Numerical(0)}, Body: body})
	astTestMust(t, err)
	astTestMust(t, s.Assert(q))
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("macros")
	s.kernel.modelConfig.Trace = tags
	ready, err := s.kernel.reduceAssertedPending()
	astTestMust(t, err)
	if !ready || s.kernel.assertedMacros.NumMacros() != 1 {
		t.Fatal("macro extraction did not complete")
	}
	if tags.ShouldTrace("macros") {
		if !strings.Contains(out.String(), "trace-macro-f") || !strings.Contains(out.String(), " ->\n") {
			t.Fatal("macro trace omitted its head/definition", out.String())
		}
	} else if out.Len() != 0 {
		t.Fatal("erased macro trace emitted output")
	}
}

func TestSMTAssertedTraceDefinitionAllocationFrontiers(t *testing.T) {
	for _, closed := range []bool{false, true} {
		succeeded := false
		for stop := int64(0); stop < 512 && !succeeded; stop++ {
			t.Run(fmt.Sprint(closed, "/", stop), func(t *testing.T) {
				m, s := smtAssertedNNFFixture(t, false)
				var out bytes.Buffer
				tags := diagnostic.NewTraceTagsWithWriter(&out)
				if closed {
					tags = diagnostic.NewTraceTagsWithWriter(assertedRefusingTrace{})
				}
				tags.Enable("reduce_step_ll")
				s.kernel.modelConfig.Trace = tags
				b, err := m.MkBoolSort()
				astTestMust(t, err)
				p, err := m.MkConst(astTestSymbol(t, m, "frontier-trace-p"), b)
				astTestMust(t, err)
				astTestMust(t, s.Assert(p))
				m.Budget().FailAfter(stop)
				err = s.kernel.traceAssertedState("reduce_step_ll", "invoke", 285, true)
				m.Budget().FailAfter(-1)
				if err == nil {
					succeeded = true
				} else if !errors.Is(err, arena.ErrInjectedFailure) {
					t.Fatal(err)
				}
			})
			if t.Failed() {
				t.Fatalf("trace allocation frontier %d", stop)
			}
		}
		if !succeeded {
			t.Fatal("no successful trace allocation frontier")
		}
	}
}
