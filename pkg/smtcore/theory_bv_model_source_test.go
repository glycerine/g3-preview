package smtcore

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func bvModelSourceFixture(t *testing.T) (*ast.ASTManager, *SMTInternalizerState, *SMTBitVectorTheory, *SMTModelGenerator) {
	t.Helper()
	m, cfg := smtModelGeneratorFixture(t)
	_, _, s := smtInternalizerFixture(t, m, nil)
	family, err := m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	bv, err := NewSMTBitVectorTheory(family)
	astTestMust(t, err)
	astTestMust(t, s.RegisterTheory(bv))
	g, err := NewSMTModelGenerator(m)
	astTestMust(t, err)
	astTestMust(t, g.SetContext(s.context))
	astTestMust(t, g.InitializeModel(cfg))
	t.Cleanup(func() { astTestMust(t, g.Close()) })
	return m, s, bv, g
}

func TestSMTBVModelValueNativePrefixAndDebugFixedness(t *testing.T) {
	for _, tc := range []struct {
		name  string
		bits  []lbool.LBool
		want  string
		fixed bool
	}{
		{"first-undefined", []lbool.LBool{lbool.Undef, lbool.True, lbool.True, lbool.True}, "0", false},
		{"middle-undefined", []lbool.LBool{lbool.True, lbool.Undef, lbool.True, lbool.True}, "1", false},
		{"last-undefined", []lbool.LBool{lbool.True, lbool.False, lbool.True, lbool.Undef}, "5", false},
		{"fully-fixed", []lbool.LBool{lbool.True, lbool.False, lbool.False, lbool.True}, "9", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, s, bv, g := bvModelSourceFixture(t)
			node, v := bvFixedSymbolic(t, m, s, bv, "native-model-"+tc.name, 4)
			for index, value := range tc.bits {
				if value == lbool.Undef {
					continue
				}
				literal := bv.bitLiterals[v][index]
				if value == lbool.False {
					literal = literal.Negated()
				}
				astTestMust(t, s.AssignLiteral(literal, NewSMTBooleanAxiom()))
			}
			astTestMust(t, bv.InitializeModel(g))
			prefix, fixed, err := bv.fixedLiteralValue(v)
			astTestMust(t, err)
			if prefix.String() != tc.want || fixed != tc.fixed {
				t.Fatal("source get_fixed_value prefix", prefix, fixed, tc.want, tc.fixed)
			}
			before := m.Stats()
			if smtcoreSourceDebug && !tc.fixed {
				m.Budget().FailAfter(0)
			}
			procedure, err := bv.ModelValue(node, g)
			m.Budget().FailAfter(-1)
			if smtcoreSourceDebug && !tc.fixed {
				if !errors.Is(err, ast.ErrASTDomain) || procedure != nil || m.Stats() != before {
					t.Fatal("source fixedness assertion must precede factory construction", err, procedure)
				}
				return
			}
			astTestMust(t, err)
			value, err := procedure.MakeValue(g, nil)
			astTestMust(t, err)
			text, yes, err := m.BitVectorNumeralText(value)
			astTestMust(t, err)
			width, err := m.BitVectorSize(value)
			astTestMust(t, err)
			if !yes || text != tc.want || width != 4 || procedure.IsFresh() || len(procedure.Dependencies()) != 0 {
				t.Fatal("native BV expression-wrapper model", text, width, tc.want)
			}
		})
	}
}

func TestSMTBVModelValueSignedLiteralsAndFactoryPipeline(t *testing.T) {
	m, s, bv, g := bvModelSourceFixture(t)
	p, q := smtRelevancyBool(t, m, "model-signed-p"), smtRelevancyBool(t, m, "model-signed-q")
	notQ, err := m.MkNot(q)
	astTestMust(t, err)
	truth, err := m.MkTrue()
	astTestMust(t, err)
	falsity, err := m.MkFalse()
	astTestMust(t, err)
	node, _ := bvNativeWord(t, m, s, bv, p, notQ, truth, falsity)
	for _, expr := range []ast.ASTHandle{p, notQ} {
		literal, err := s.Literal(expr)
		astTestMust(t, err)
		astTestMust(t, s.AssignLiteral(literal, NewSMTBooleanAxiom()))
	}
	astTestMust(t, s.context.SetRelevant(node.owner, true))
	// The real generator initializes the factory and invokes the registered
	// theory's ModelValue callback, then pins the returned expression.
	astTestMust(t, g.BuildStateValues(s))
	value, err := g.Value(node)
	astTestMust(t, err)
	text, yes, err := m.BitVectorNumeralText(value)
	astTestMust(t, err)
	width, err := m.BitVectorSize(value)
	astTestMust(t, err)
	if !yes || text != "7" || width != 4 {
		t.Fatal("native signed and constant model bits", text, width)
	}
}
