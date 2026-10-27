package smtcore

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSMTFPANativeModelProcedureCanonicalComponents(t *testing.T) {
	m, cfg := smtModelGeneratorFixture(t)
	s, err := NewSMTSolver(m, nil, "QF_FPBV", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	fp := s.kernel.internalizer.theories[family].(*SMTFloatingPointTheory)
	g, err := NewSMTModelGenerator(m)
	astTestMust(t, err)
	astTestMust(t, g.SetContext(s.kernel.internalizer.context))
	astTestMust(t, g.InitializeModel(cfg))
	t.Cleanup(func() { astTestMust(t, g.Close()) })
	// A real fp constructor supplies the procedure's three actual dependencies.
	var arguments []ast.ASTHandle
	for i, width := range []uint32{1, 3, 3} {
		sort, err := m.BitVectorSort(width)
		astTestMust(t, err)
		arg, err := m.MkConst(astTestSymbol(t, m, []string{"native-model-sign", "native-model-exp", "native-model-sig"}[i]), sort)
		astTestMust(t, err)
		astTestMust(t, s.kernel.internalizer.Internalize(arg, false))
		arguments = append(arguments, arg)
	}
	term, err := m.MkFamilyApp(family, ast.FPAOpFP, nil, arguments)
	astTestMust(t, err)
	node, err := s.kernel.internalizer.MkEnode(term, false, false, true)
	astTestMust(t, err)
	astTestMust(t, fp.ApplySortConstraint(node, mustFPASort(t, m, term)))
	procedure, err := fp.ModelValue(node, g)
	astTestMust(t, err)
	if len(procedure.Dependencies()) != 3 || procedure.IsFresh() {
		t.Fatal("native FP dependency shape")
	}
	for _, tc := range []struct {
		sign, exp, sig uint64
		kind           int32
	}{
		{0, 0, 0, ast.FPAOpPositiveZero}, {1, 0, 0, ast.FPAOpNegativeZero},
		{0, 7, 0, ast.FPAOpPositiveInf}, {1, 7, 0, ast.FPAOpNegativeInf},
		{0, 7, 1, ast.FPAOpNaN}, {1, 7, 7, ast.FPAOpNaN},
		{0, 3, 0, ast.FPAOpNumeral}, {1, 0, 1, ast.FPAOpNumeral},
	} {
		fields := []ast.ASTHandle{}
		for i, n := range []uint64{tc.sign, tc.exp, tc.sig} {
			v, err := m.BitVectorNumeralUint64(n, []uint32{1, 3, 3}[i])
			astTestMust(t, err)
			fields = append(fields, v)
		}
		value, err := procedure.MakeValue(g, fields)
		astTestMust(t, err)
		kind, err := m.GetDeclKind(value)
		astTestMust(t, err)
		if kind != tc.kind {
			t.Fatal("model returned an unnormalized FP constructor", tc, kind)
		}
		// Canonical registration makes repeated dependency values identical.
		held, err := m.NewRef(value)
		astTestMust(t, err)
		again, err := procedure.MakeValue(g, fields)
		astTestMust(t, err)
		borrowedGeneratorValue, e := procedure.MakeValue(&SMTModelGenerator{}, fields)
		astTestMust(t, e)
		if borrowedGeneratorValue != value {
			t.Fatal("source procedure must use its theory manager")
		}
		if value != again {
			t.Fatal("model numeral was not canonical")
		}
		astTestMust(t, m.DelRef(&held))
	}
}

func TestSMTFPAModelWidthAssertionsAndActions(t *testing.T) {
	for _, tc := range []struct {
		name  string
		field int
		line  int
		rm    bool
	}{
		{"packed", -1, 147, false}, {"sign", 0, 169, false}, {"exponent", 1, 171, false}, {"significand", 2, 173, false}, {"rounding", -1, 216, true},
	} {
		for _, enabled := range []bool{false, true} {
			for _, action := range []string{"T", "C"} {
				t.Run(fmt.Sprintf("%s/%t/%s", tc.name, enabled, action), func(t *testing.T) {
					d, _ := newLRASourceAssertionContext(t)
					report := bvDebugActionInput(t, d, strings.Repeat(action, 32))
					m, _ := smtModelGeneratorFixture(t)
					_, err := m.RegisterBitVectorPlugin()
					astTestMust(t, err)
					_, err = m.RegisterFloatingPointPlugin()
					astTestMust(t, err)
					widths := []uint32{1, 3, 3}
					if tc.field >= 0 {
						widths[tc.field]++
					} else if tc.rm {
						widths = []uint32{4}
					} else {
						widths = []uint32{8}
					}
					fields := make([]ast.ASTHandle, len(widths))
					for i, w := range widths {
						fields[i], err = m.BitVectorNumeralUint64(0, w)
						astTestMust(t, err)
					}
					astTestMust(t, d.EnableAssertions(enabled))
					var value ast.ASTHandle
					if tc.rm {
						value, err = m.MkFPARoundingModeModelValue(fields)
					} else {
						value, err = m.MkFPAModelValue(3, 4, fields)
					}
					if diagnostic.DebugBuild && enabled && action == "T" {
						var ex z3exception.DefaultException
						if !errors.As(err, &ex) || ex.Message() != "assertion violation" || value != (ast.ASTHandle{}) {
							t.Fatal("actual source width throw", value, err)
						}
					} else {
						astTestMust(t, err)
						if value == (ast.ASTHandle{}) {
							t.Fatal("continued/erased assertion lost value")
						}
					}
					data, e := os.ReadFile(report)
					astTestMust(t, e)
					want := fmt.Sprintf("File: src/smt/theory_fpa.cpp\nLine: %d\n", tc.line)
					if diagnostic.DebugBuild && enabled {
						if !strings.Contains(string(data), want) {
							t.Fatal("missing width assertion", string(data), want)
						}
					} else if len(data) != 0 {
						t.Fatal("disabled assertion evaluated", string(data))
					}
				})
			}
		}
	}
}

func TestSMTFPAModelNumeralVerification(t *testing.T) {
	for _, rm := range []bool{false, true} {
		for _, enabled := range []bool{false, true} {
			for _, action := range []string{"T", "C"} {
				t.Run(fmt.Sprintf("%t/%t/%s", rm, enabled, action), func(t *testing.T) {
					d, _ := newLRASourceAssertionContext(t)
					report := bvDebugActionInput(t, d, strings.Repeat(action, 32))
					m, _ := smtModelGeneratorFixture(t)
					_, err := m.RegisterBitVectorPlugin()
					astTestMust(t, err)
					_, err = m.RegisterFloatingPointPlugin()
					astTestMust(t, err)
					width := uint32(7)
					line := 152
					if rm {
						width = 3
						line = 215
					}
					sort, err := m.BitVectorSort(width)
					astTestMust(t, err)
					symbolic, err := m.MkConst(astTestSymbol(t, m, "non_numeral_fpa_model"), sort)
					astTestMust(t, err)
					astTestMust(t, d.EnableAssertions(enabled))
					if rm {
						_, err = m.MkFPARoundingModeModelValue([]ast.ASTHandle{symbolic})
					} else {
						_, err = m.MkFPAModelValue(3, 4, []ast.ASTHandle{symbolic})
					}
					if !diagnostic.DebugBuild {
						if !errors.Is(err, diagnostic.ErrDebugNativeUnreachable) {
							t.Fatal("release VERIFY", err)
						}
					} else if action == "T" {
						var ex z3exception.DefaultException
						if !errors.As(err, &ex) || ex.Message() != "assertion violation" {
							t.Fatal("VERIFY throw", err)
						}
					} else if enabled {
						if !errors.Is(err, ast.ErrASTDomain) {
							t.Fatal("continued VERIFY width is uninitialized", err)
						}
					} else {
						astTestMust(t, err)
					}
					data, e := os.ReadFile(report)
					astTestMust(t, e)
					if !strings.Contains(string(data), "Failed to verify:") {
						t.Fatal("VERIFY did not execute", string(data))
					}
					if diagnostic.DebugBuild && !strings.Contains(string(data), fmt.Sprintf("File: src/smt/theory_fpa.cpp\nLine: %d\n", line)) {
						t.Fatal("VERIFY source location", string(data))
					}
				})
			}
		}
	}
}
