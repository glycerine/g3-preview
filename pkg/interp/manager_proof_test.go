package interp

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

func interpProofManagerFixture(t *testing.T) (*ast.ASTManager, *Manager) {
	t.Helper()
	m := interpASTFixture(t)
	interpMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	interpMust(t, m.RegisterDeclarationPlugins())
	w, err := NewManager(m)
	interpMust(t, err)
	t.Cleanup(func() { interpMust(t, w.Close()) })
	return m, w
}

func interpTheoryLemma(t *testing.T, m *ast.ASTManager, family ast.FamilyID, kind string, conclusion ast.ASTHandle, premises []ast.ASTHandle, coefficients ...string) ast.ASTHandle {
	t.Helper()
	kindParameter, err := m.NewParameterString(kind)
	interpMust(t, err)
	parameters := []ast.ASTParameter{kindParameter}
	for _, text := range coefficients {
		value, e := m.NumeralService().NewValue()
		interpMust(t, e)
		interpMust(t, m.NumeralService().SetString(&value, text))
		parameter, e := m.NewParameterRational(&value)
		interpMust(t, e)
		interpMust(t, m.NumeralService().Del(&value))
		parameters = append(parameters, parameter)
	}
	lemma, err := m.MkProofTheoryLemma(family, conclusion, premises, parameters)
	interpMust(t, err)
	for i := range parameters {
		interpMust(t, m.DelParameter(&parameters[i]))
	}
	return interpPin(t, m, lemma)
}

func TestInterpolationManagerProofMetadataAndTheoryKinds(t *testing.T) {
	m, w := interpProofManagerFixture(t)
	boolSort, err := w.BoolType()
	interpMust(t, err)
	p, err := w.MakeVariable("proof-p", boolSort)
	interpMust(t, err)
	p = interpPin(t, m, p)
	premise, err := m.MkProofAsserted(p)
	interpMust(t, err)
	premise = interpPin(t, m, premise)
	falsehood, err := w.Make(OperatorFalse)
	interpMust(t, err)
	arithmetic, err := m.ArithmeticFamilyID()
	interpMust(t, err)
	lemma := interpTheoryLemma(t, m, arithmetic, "farkas", falsehood, []ast.ASTHandle{premise}, "-1/2", "3/4")

	if rule, e := w.ProofRule(lemma); e != nil || rule != ast.ASTPrTheoryLemma {
		t.Fatal("proof rule", rule, e)
	}
	if count, e := w.NumPremises(lemma); e != nil || count != 1 {
		t.Fatal("premise count", count, e)
	}
	if got, e := w.Premise(lemma, 0); e != nil || got != premise {
		t.Fatal("premise", got, e)
	}
	if got, e := w.Conclusion(lemma); e != nil || got != falsehood {
		t.Fatal("conclusion", got, e)
	}
	if theory, e := w.TheoryLemmaTheory(lemma); e != nil || theory != ArithTheory {
		t.Fatal("theory", theory, e)
	}
	if kind, e := w.TheoryLemmaKind(lemma); e != nil || kind != FarkasKind {
		t.Fatal("kind", kind, e)
	}
	if negative, e := w.IsFarkasCoefficientNegative(lemma, 0); e != nil || !negative {
		t.Fatal("negative coefficient", negative, e)
	}
	if _, e := w.Premise(lemma, 1); !errors.Is(e, ast.ErrASTDomain) {
		t.Fatal("premise bounds", e)
	}

	array, err := m.ArrayFamilyID()
	interpMust(t, err)
	mystery := interpTheoryLemma(t, m, array, "", falsehood, nil)
	if theory, e := w.TheoryLemmaTheory(mystery); e != nil || theory != ArrayTheory {
		t.Fatal("array theory", theory, e)
	}
}

func TestInterpolationManagerTheoryCoefficientNormalization(t *testing.T) {
	m, w := interpProofManagerFixture(t)
	falsehood, err := w.Make(OperatorFalse)
	interpMust(t, err)
	arithmetic, err := m.ArithmeticFamilyID()
	interpMust(t, err)
	lemma := interpTheoryLemma(t, m, arithmetic, "farkas", falsehood, nil, "-1/2", "3/4", "-5")

	check := func(name string, got []*big.Rat, err error, want []string) {
		t.Helper()
		interpMust(t, err)
		if len(got) != len(want) {
			t.Fatalf("%s length = %d, want %d", name, len(got), len(want))
		}
		for i := range got {
			if got[i].RatString() != want[i] {
				t.Fatalf("%s[%d] = %s, want %s", name, i, got[i], want[i])
			}
		}
	}
	farkas, err := w.FarkasCoefficients(lemma)
	check("farkas", farkas, err, []string{"2", "3", "20"})
	gomory, err := w.GomoryCutCoefficients(lemma)
	check("gomory", gomory, err, []string{"2", "3", "20"})
	brokenGCD, err := w.BrokenGCDTestCoefficients(lemma)
	check("broken gcd", brokenGCD, err, []string{"-2", "3", "-20"})
}

func TestInterpolationManagerTriangleEqualityKinds(t *testing.T) {
	m, w := interpProofManagerFixture(t)
	boolSort, err := w.BoolType()
	interpMust(t, err)
	p, err := w.MakeVariable("triangle-p", boolSort)
	interpMust(t, err)
	p = interpPin(t, m, p)
	q, err := w.MakeVariable("triangle-q", boolSort)
	interpMust(t, err)
	q = interpPin(t, m, q)
	notP, err := w.Make(OperatorNot, p)
	interpMust(t, err)
	arithmetic, err := m.ArithmeticFamilyID()
	interpMust(t, err)

	for _, test := range []struct {
		name  string
		first ast.ASTHandle
		want  LemmaKind
	}{{"leq-to-eq", p, Leq2EqKind}, {"eq-to-leq", notP, Eq2LeqKind}} {
		t.Run(test.name, func(t *testing.T) {
			conclusion, e := w.Make(OperatorOr, test.first, q)
			interpMust(t, e)
			lemma := interpTheoryLemma(t, m, arithmetic, "triangle-eq", conclusion, nil)
			if kind, e := w.TheoryLemmaKind(lemma); e != nil || kind != test.want {
				t.Fatal(kind, e)
			}
		})
	}
}
