package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/symbol"
)

type smtAntecedentCollector struct {
	justifications []SMTJustification
	literals       []SMTLiteral
	equalities     []SMTEnodePair
	justified      []SMTEnodePair
	nodeLiteral    SMTLiteral
}

func (c *smtAntecedentCollector) MarkJustification(j SMTJustification) error {
	c.justifications = append(c.justifications, j)
	return nil
}
func (c *smtAntecedentCollector) MarkLiteral(l SMTLiteral) error {
	c.literals = append(c.literals, l)
	return nil
}
func (c *smtAntecedentCollector) MarkEquality(a, b *SMTEnode) error {
	c.equalities = append(c.equalities, SMTEnodePair{a, b})
	return nil
}
func (c *smtAntecedentCollector) MarkJustifiedEquality(a, b *SMTEnode, _ SMTEqualityJustification) error {
	c.justified = append(c.justified, SMTEnodePair{a, b})
	return nil
}
func (c *smtAntecedentCollector) LiteralForENode(*SMTEnode) (SMTLiteral, error) {
	return c.nodeLiteral, nil
}

func TestSMTJustificationBaseUnitAndSimpleAntecedents(t *testing.T) {
	base := &SMTJustificationBase{}
	if base.IsMarked() || !base.InRegion() || base.Name() != "unknown" {
		t.Fatal("base defaults")
	}
	if err := base.SetMarked(true); err != nil || !base.IsMarked() {
		t.Fatal("mark", err)
	}
	child := &SMTSimpleJustification{SMTJustificationBase: NewSMTJustificationBase(false)}
	unit, err := NewSMTUnitResolutionJustification(false, child, []SMTLiteral{NewSMTLiteral(3, false), NewSMTLiteral(4, true)})
	if err != nil {
		t.Fatal(err)
	}
	collector := &smtAntecedentCollector{}
	if err = unit.MarkAntecedents(collector); err != nil {
		t.Fatal(err)
	}
	if len(collector.justifications) != 1 || collector.justifications[0] != child || len(collector.literals) != 2 {
		t.Fatal("antecedents", collector)
	}
}

func TestSMTJustificationEqualityAntecedentRecords(t *testing.T) {
	leftRoot, rightRoot := &SMTEnode{interpreted: true}, &SMTEnode{interpreted: true}
	leftRoot.root, rightRoot.root = leftRoot, rightRoot
	left, right := &SMTEnode{root: leftRoot}, &SMTEnode{root: rightRoot}
	collector := &smtAntecedentCollector{nodeLiteral: NewSMTLiteral(9, true)}
	conflict, err := NewSMTEqualityConflictJustification(left, right, NewSMTEqualityAxiom())
	if err != nil {
		t.Fatal(err)
	}
	if err = conflict.MarkAntecedents(collector); err != nil {
		t.Fatal(err)
	}
	root, err := NewSMTEqualityRootJustification(left)
	if err != nil {
		t.Fatal(err)
	}
	if err = root.MarkAntecedents(collector); err != nil {
		t.Fatal(err)
	}
	propagation, err := NewSMTEqualityPropagationJustification(left, right)
	if err != nil {
		t.Fatal(err)
	}
	if err = propagation.MarkAntecedents(collector); err != nil {
		t.Fatal(err)
	}
	modusPonens, err := NewSMTModusPonensIffJustification(left, right)
	if err != nil {
		t.Fatal(err)
	}
	if err = modusPonens.MarkAntecedents(collector); err != nil {
		t.Fatal(err)
	}
	if len(collector.equalities) != 5 || len(collector.justified) != 1 || len(collector.literals) != 1 || collector.literals[0] != collector.nodeLiteral {
		t.Fatal("equality antecedents", collector)
	}
	sameRoot := &SMTEnode{root: leftRoot}
	extended, err := NewSMTExtSimpleJustification(true, []SMTLiteral{SMTTrueLiteral}, []SMTEnodePair{{left, sameRoot}})
	if err != nil {
		t.Fatal(err)
	}
	if extended.Name() != "ext-simple" || extended.FromTheory() != SMTNullTheoryID {
		t.Fatal("extended metadata")
	}
	if err = extended.MarkAntecedents(collector); err != nil {
		t.Fatal(err)
	}
	if len(collector.equalities) != 6 || len(collector.literals) != 2 {
		t.Fatal("extended antecedents", collector)
	}
}

func TestSMTTheoryJustificationParameterExpressionAndProofOwnership(t *testing.T) {
	manager := astTestManager(t)
	parameter, err := manager.NewParameterInt(37)
	if err != nil {
		t.Fatal(err)
	}
	theory, err := NewSMTTheoryPropagationJustification(manager, 11, []SMTLiteral{NewSMTLiteral(2, false)}, NewSMTLiteral(3, true), []ast.ASTParameter{parameter})
	if err != nil {
		t.Fatal(err)
	}
	if err = manager.DelParameter(&parameter); err != nil {
		t.Fatal(err)
	}
	owned := theory.Parameters()
	if len(owned) != 1 || theory.FromTheory() != 11 || theory.Name() != "theory-propagation" {
		t.Fatal("theory metadata")
	}
	if kind, err := manager.ParameterKind(owned[0]); err != nil || kind != ast.ASTParameterInt {
		t.Fatal("copied parameter", kind, err)
	}
	collector := &smtAntecedentCollector{}
	if err = theory.MarkAntecedents(collector); err != nil || len(collector.literals) != 1 {
		t.Fatal("theory antecedents", collector, err)
	}
	if err = theory.Release(manager); err != nil {
		t.Fatal(err)
	}
	if _, err = manager.ParameterKind(owned[0]); err == nil {
		t.Fatal("released parameter remained live")
	}

	boolean, err := manager.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	decl, err := manager.MkFuncDecl(symbol.Numerical(2701), nil, boolean, ast.ASTFuncDeclInfo{})
	if err != nil {
		t.Fatal(err)
	}
	atom, err := manager.MkApp(decl, nil)
	if err != nil {
		t.Fatal(err)
	}
	lemma, err := NewSMTTheoryLemmaJustification(manager, 12, []SMTLiteral{NewSMTLiteral(1, true)}, []ast.ASTHandle{{}, atom}, nil)
	if err != nil {
		t.Fatal(err)
	}
	expressions, signs, err := lemma.Expressions()
	if err != nil {
		t.Fatal(err)
	}
	if len(expressions) != 1 || expressions[0] != atom || len(signs) != 1 || !signs[0] || lemma.InRegion() {
		t.Fatal("theory lemma payload")
	}
	wrapper, err := NewSMTJustificationProofWrapper(manager, atom, true)
	if err != nil {
		t.Fatal(err)
	}
	if err = lemma.Release(manager); err != nil {
		t.Fatal(err)
	}
	if proof, err := wrapper.Proof(); err != nil || proof != atom {
		t.Fatal("proof wrapper", proof, err)
	}
	if err = wrapper.Release(manager); err != nil {
		t.Fatal(err)
	}
	if _, err = wrapper.Proof(); err == nil {
		t.Fatal("released proof wrapper")
	}
}
