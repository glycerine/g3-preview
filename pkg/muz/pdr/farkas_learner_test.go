package pdr_test

// Source: src/muz/pdr/pdr_farkas_learner.cpp Farkas-lemma recognition and
// combine_constraints coefficient and literal modes.

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/muz/pdr"
)

func farkasFixture(t *testing.T) (*cmd_context.Context, *ast.ASTManager, *pdr.FarkasLearner) {
	t.Helper()
	command, err := cmd_context.NewStandaloneContext()
	pdrMust(t, err)
	manager, err := command.Manager()
	pdrMust(t, err)
	pdrMust(t, manager.ToggleProofMode(ast.ASTProofEnabled))
	globals, err := command.GlobalParameters()
	pdrMust(t, err)
	learner, err := pdr.NewFarkasLearner(manager, globals.Context, globals)
	pdrMust(t, err)
	return command, manager, learner
}

func TestFarkasLearnerRecognizesArithmeticTheoryLemma(t *testing.T) {
	command, manager, learner := farkasFixture(t)
	arithmetic, err := manager.RegisterArithmeticPlugin()
	pdrMust(t, err)
	integer, err := manager.MkFamilySort(arithmetic, 1, nil)
	pdrMust(t, err)
	xName, err := manager.Symbols.Intern("farkas-x")
	pdrMust(t, err)
	x, err := manager.MkConst(xName, integer)
	pdrMust(t, err)
	zero, err := manager.ArithmeticIntegerNumeralText("0")
	pdrMust(t, err)
	one, err := manager.ArithmeticIntegerNumeralText("1")
	pdrMust(t, err)
	le, err := manager.MkFamilyApp(arithmetic, 2, nil, []ast.ASTHandle{x, zero})
	pdrMust(t, err)
	ge, err := manager.MkFamilyApp(arithmetic, 3, nil, []ast.ASTHandle{x, one})
	pdrMust(t, err)
	premise0, err := manager.MkProofAsserted(le)
	pdrMust(t, err)
	premise1, err := manager.MkProofAsserted(ge)
	pdrMust(t, err)
	falsity, err := manager.MkFalse()
	pdrMust(t, err)
	farkasName, err := manager.Symbols.Intern("farkas")
	pdrMust(t, err)
	farkas, err := manager.NewParameterSymbol(farkasName)
	pdrMust(t, err)
	coefficientValue, err := manager.NumeralService().NewValue()
	pdrMust(t, err)
	pdrMust(t, manager.NumeralService().SetString(&coefficientValue, "-3/2"))
	coefficient0, err := manager.NewParameterRational(&coefficientValue)
	pdrMust(t, err)
	pdrMust(t, manager.NumeralService().SetString(&coefficientValue, "5"))
	coefficient1, err := manager.NewParameterRational(&coefficientValue)
	pdrMust(t, err)
	pdrMust(t, manager.NumeralService().SetString(&coefficientValue, "7"))
	conclusionCoefficient, err := manager.NewParameterRational(&coefficientValue)
	pdrMust(t, err)
	pdrMust(t, manager.NumeralService().Del(&coefficientValue))
	lemma, err := manager.MkProofTheoryLemma(arithmetic, falsity, []ast.ASTHandle{premise0, premise1}, []ast.ASTParameter{farkas, coefficient0, coefficient1})
	pdrMust(t, err)
	lemmaWithConclusion, err := manager.MkProofTheoryLemma(arithmetic, le, []ast.ASTHandle{premise0, premise1}, []ast.ASTParameter{farkas, coefficient0, coefficient1, conclusionCoefficient})
	pdrMust(t, err)
	pdrMust(t, manager.DelParameter(&conclusionCoefficient))
	pdrMust(t, manager.DelParameter(&coefficient1))
	pdrMust(t, manager.DelParameter(&coefficient0))
	pdrMust(t, manager.DelParameter(&farkas))

	matched, err := learner.IsFarkasLemma(lemma)
	pdrMust(t, err)
	if !matched {
		t.Fatal("arithmetic Farkas theory lemma was not recognized")
	}
	facts, coefficients, matched, err := learner.FarkasPremises(lemma)
	pdrMust(t, err)
	if !matched || len(facts) != 2 || facts[0] != le || facts[1] != ge {
		t.Fatal("Farkas parent facts were not extracted in proof order", matched, facts)
	}
	if coefficients[0].Cmp(big.NewRat(3, 2)) != 0 || coefficients[1].Cmp(big.NewRat(5, 1)) != 0 {
		t.Fatal("Farkas coefficients were not extracted and normalized", coefficients)
	}
	combined, produced, err := learner.CombineFarkasLemma(lemma, []bool{true, false})
	pdrMust(t, err)
	if !produced {
		t.Fatal("B-pure Farkas parent did not produce a combined lemma")
	}
	combinedHandle, err := manager.RefGet(combined)
	pdrMust(t, err)
	if combinedHandle.IsNull() {
		t.Fatal("Farkas theory-lemma reduction returned a null lemma")
	}
	pdrMust(t, manager.DelRef(&combined))
	combined, produced, err = learner.CombineFarkasLemma(lemma, []bool{false, false})
	pdrMust(t, err)
	if produced || combined != (ast.ASTRef{}) {
		t.Fatal("Farkas theory lemma without a B-pure parent produced output")
	}
	combined, produced, err = learner.CombineFarkasLemma(lemmaWithConclusion, []bool{true, false})
	pdrMust(t, err)
	if !produced {
		t.Fatal("Farkas conclusion coefficient did not produce a combined lemma")
	}
	pdrMust(t, manager.DelRef(&combined))
	matched, err = learner.IsFarkasLemma(premise0)
	pdrMust(t, err)
	if matched {
		t.Fatal("asserted proof was recognized as a Farkas lemma")
	}

	pdrMust(t, learner.Close())
	pdrMust(t, command.Close())
}

func TestFarkasLearnerCombinesCoefficientsAndNegatedLiterals(t *testing.T) {
	command, manager, learner := farkasFixture(t)
	arithmetic, err := manager.RegisterArithmeticPlugin()
	pdrMust(t, err)
	integer, err := manager.MkFamilySort(arithmetic, 1, nil)
	pdrMust(t, err)
	xName, err := manager.Symbols.Intern("combine-x")
	pdrMust(t, err)
	x, err := manager.MkConst(xName, integer)
	pdrMust(t, err)
	zero, err := manager.ArithmeticIntegerNumeralText("0")
	pdrMust(t, err)
	one, err := manager.ArithmeticIntegerNumeralText("1")
	pdrMust(t, err)
	le, err := manager.MkFamilyApp(arithmetic, 2, nil, []ast.ASTHandle{x, zero})
	pdrMust(t, err)
	ge, err := manager.MkFamilyApp(arithmetic, 3, nil, []ast.ASTHandle{x, one})
	pdrMust(t, err)
	combined, err := learner.CombineConstraints([]ast.ASTHandle{le, ge}, []*big.Rat{big.NewRat(1, 1), big.NewRat(1, 1)})
	pdrMust(t, err)
	combinedHandle, err := manager.RefGet(combined)
	pdrMust(t, err)
	if combinedHandle.IsNull() {
		t.Fatal("coefficient combination returned a null constraint")
	}
	pdrMust(t, manager.DelRef(&combined))

	pdrMust(t, learner.SetCombineCoefficients(false))
	boolean, err := manager.MkBoolSort()
	pdrMust(t, err)
	pName, err := manager.Symbols.Intern("combine-p")
	pdrMust(t, err)
	qName, err := manager.Symbols.Intern("combine-q")
	pdrMust(t, err)
	p, err := manager.MkConst(pName, boolean)
	pdrMust(t, err)
	q, err := manager.MkConst(qName, boolean)
	pdrMust(t, err)
	negated, err := learner.CombineConstraints([]ast.ASTHandle{p, q}, []*big.Rat{big.NewRat(1, 1), big.NewRat(1, 1)})
	pdrMust(t, err)
	negatedHandle, err := manager.RefGet(negated)
	pdrMust(t, err)
	isNot, _, err := manager.IsNot(negatedHandle)
	pdrMust(t, err)
	if !isNot {
		t.Fatal("literal combination did not produce not(or(literals))")
	}
	pdrMust(t, manager.DelRef(&negated))

	pdrMust(t, learner.Close())
	pdrMust(t, command.Close())
}
