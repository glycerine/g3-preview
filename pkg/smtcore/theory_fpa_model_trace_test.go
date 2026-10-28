package smtcore

// Source: src/smt/theory_fpa.cpp:129-132,194-198,206-209,228.

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type smtFPAModelRefusingTraceWriter struct{}

func (smtFPAModelRefusingTraceWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func smtFPAModelTraceFixture(t *testing.T, tags *diagnostic.TraceTags) (*ast.ASTManager, *SMTModelGenerator) {
	t.Helper()
	m, _ := smtModelGeneratorFixture(t)
	_, err := m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	_, err = m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	g, err := NewSMTModelGenerator(m)
	astTestMust(t, err)
	g.trace = tags
	t.Cleanup(func() { astTestMust(t, g.Close()) })
	return m, g
}

func TestSMTFPAModelValueSourceTrace(t *testing.T) {
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("t_fpa_detail")
	tags.Enable("t_fpa")
	m, generator := smtFPAModelTraceFixture(t, tags)
	values := make([]ast.ASTHandle, 3)
	for i, field := range []uint64{1, 2, 6} {
		var err error
		values[i], err = m.BitVectorNumeralUint64(field, []uint32{1, 3, 3}[i])
		astTestMust(t, err)
	}
	procedure := &smtFPAValueProcedure{
		manager:      m,
		dependencies: make([]SMTModelValueDependency, 3),
		ebits:        3,
		sbits:        4,
	}
	value, err := procedure.MakeValue(generator, values)
	astTestMust(t, err)
	astTestPin(t, m, value)
	if tags.ShouldTrace("t_fpa") {
		trace := output.String()
		for _, want := range []string{
			"-------- [t_fpa_detail] mk_value src/smt/theory_fpa.cpp:129 ---------\n",
			"value[0] = #b1\nvalue[1] = #b010\nvalue[2] = #b110\n",
			"-------- [t_fpa] mk_value src/smt/theory_fpa.cpp:194 ---------\n",
			"result: [1,2,6] --> ",
		} {
			if !strings.Contains(trace, want) {
				t.Fatalf("missing source FPA trace %q in %q", want, trace)
			}
		}
	} else if output.Len() != 0 {
		t.Fatal("compiled-out FPA trace executed", output.String())
	}
}

func TestSMTFPAModelPackedAndRoundingSourceTrace(t *testing.T) {
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("t_fpa_detail")
	tags.Enable("t_fpa")
	m, generator := smtFPAModelTraceFixture(t, tags)
	packed, err := m.BitVectorNumeralUint64(86, 7)
	astTestMust(t, err)
	fp := &smtFPAValueProcedure{manager: m, dependencies: make([]SMTModelValueDependency, 1), ebits: 3, sbits: 4}
	value, err := fp.MakeValue(generator, []ast.ASTHandle{packed})
	astTestMust(t, err)
	astTestPin(t, m, value)
	roundingBits, err := m.BitVectorNumeralUint64(3, 3)
	astTestMust(t, err)
	rounding := &smtFPAValueProcedure{manager: m, dependencies: make([]SMTModelValueDependency, 1), rounding: true}
	roundingValue, err := rounding.MakeValue(generator, []ast.ASTHandle{roundingBits})
	astTestMust(t, err)
	astTestPin(t, m, roundingValue)
	if tags.ShouldTrace("t_fpa") {
		trace := output.String()
		for _, want := range []string{
			"result: [1,2,6] --> ",
			"-------- [t_fpa_detail] mk_value src/smt/theory_fpa.cpp:206 ---------\n",
			"value[0] = #b011\n",
			"-------- [t_fpa] mk_value src/smt/theory_fpa.cpp:228 ---------\n",
			"result: roundTowardNegative\n",
		} {
			if !strings.Contains(trace, want) {
				t.Fatalf("missing packed/RM trace %q in %q", want, trace)
			}
		}
	} else if output.Len() != 0 {
		t.Fatal("compiled-out FPA trace executed", output.String())
	}
}

func TestSMTFPAModelTraceStreamRefusalDoesNotStopValue(t *testing.T) {
	tags := diagnostic.NewTraceTagsWithWriter(smtFPAModelRefusingTraceWriter{})
	tags.Enable("t_fpa_detail")
	tags.Enable("t_fpa")
	m, generator := smtFPAModelTraceFixture(t, tags)
	packed, err := m.BitVectorNumeralUint64(1, 7)
	astTestMust(t, err)
	procedure := &smtFPAValueProcedure{manager: m, dependencies: make([]SMTModelValueDependency, 1), ebits: 3, sbits: 4}
	value, err := procedure.MakeValue(generator, []ast.ASTHandle{packed})
	astTestMust(t, err)
	if value.IsNull() {
		t.Fatal("source ostream badbit stopped FPA model construction")
	}
	astTestPin(t, m, value)
}
