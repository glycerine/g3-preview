package smtcore

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type smtModelGeneratorRefusingTrace struct{}

func (smtModelGeneratorRefusingTrace) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestSMTModelGeneratorTraceMetadataAndRefusals(t *testing.T) {
	for _, closed := range []bool{false, true} {
		t.Run(map[bool]string{false: "open", true: "closed"}[closed], func(t *testing.T) {
			m, config := smtModelGeneratorFixture(t)
			var output bytes.Buffer
			var writer io.Writer = &output
			if closed {
				writer = smtModelGeneratorRefusingTrace{}
			}
			tags := diagnostic.NewTraceTagsWithWriter(writer)
			for _, tag := range []string{"model", "mg_top_sort", "sorted_sources", "model_fresh_bug"} {
				tags.Enable(tag)
			}
			config.Trace = tags
			sort, err := m.MkUninterpretedSort(astTestSymbol(t, m, "model-trace-sort"), nil)
			astTestMust(t, err)
			expression, err := m.MkConst(astTestSymbol(t, m, "model-trace-value"), sort)
			astTestMust(t, err)
			context, err := NewSMTCheckerContext(m)
			astTestMust(t, err)
			state, err := NewSMTInternalizerState(m, context, nil)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, state.Close()) })
			_, err = state.MkEnode(expression, false, false, true)
			astTestMust(t, err)
			astTestMust(t, context.SetRelevant(expression, true))
			generator, err := NewSMTModelGenerator(m)
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, generator.Close()) })
			astTestMust(t, generator.SetContext(context))
			astTestMust(t, generator.InitializeModel(config))
			astTestMust(t, generator.BuildStateValues(state))
			astTestMust(t, generator.registerAssertedMacros(nil))
			if closed {
				if output.Len() != 0 {
					t.Fatal("refused trace retained output", output.String())
				}
				return
			}
			text := output.String()
			if !tags.ShouldTrace("mg_top_sort") {
				if text != "" {
					t.Fatal("erased trace emitted output", text)
				}
				return
			}
			for _, want := range []string{
				"[mg_top_sort] process_source src/smt/smt_model_generator.cpp:224",
				"[mg_top_sort] process_source src/smt/smt_model_generator.cpp:236",
				"[mg_top_sort] process_source src/smt/smt_model_generator.cpp:257",
				"[sorted_sources] mk_values src/smt/smt_model_generator.cpp:310",
				"[mg_top_sort] mk_values src/smt/smt_model_generator.cpp:345",
				"[model] mk_model src/smt/smt_model_generator.cpp:525",
				"[model] register_macros src/smt/smt_model_generator.cpp:512",
				"model-trace-sort",
			} {
				if !strings.Contains(text, want) {
					t.Fatal("missing model-generator trace", want, text)
				}
			}
		})
	}
}

func TestSMTModelGeneratorErasedTraceSkipsOperands(t *testing.T) {
	tags := diagnostic.NewTraceTagsWithWriter(smtModelGeneratorRefusingTrace{})
	g := &SMTModelGenerator{trace: tags}
	reached := false
	g.traceModel("model", "mk_model", 525, func() bool {
		reached = true
		return true
	}, func(io.Writer) error {
		reached = true
		return nil
	})
	if reached {
		t.Fatal("disabled TRACE evaluated condition or body")
	}
}
