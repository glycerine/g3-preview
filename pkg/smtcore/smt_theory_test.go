package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
)

type smtInternalizerTestTheory struct {
	*SMTTheoryBase
	atoms, equalities, terms, sortConstraints int
}

func (t *smtInternalizerTestTheory) DefaultInternalizer() bool { return false }

func (t *smtInternalizerTestTheory) InternalizeAtom(expression ast.ASTHandle, gateContext bool) (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	count, err := state.manager.GetNumArgs(expression)
	if err != nil {
		return false, err
	}
	for index := uint32(0); index < count; index++ {
		argument, err := state.manager.GetArg(expression, index)
		if err != nil {
			return false, err
		}
		if err = state.Internalize(argument, false); err != nil {
			return false, err
		}
	}
	if _, err = state.MkBoolVar(expression); err != nil {
		return false, err
	}
	t.atoms++
	return true, nil
}

func (t *smtInternalizerTestTheory) InternalizeEquality(ast.ASTHandle, int32) error {
	t.equalities++
	return nil
}

func (t *smtInternalizerTestTheory) InternalizeTerm(expression ast.ASTHandle) (bool, error) {
	state, err := t.State()
	if err != nil {
		return false, err
	}
	node, err := state.MkEnode(expression, true, false, false)
	if err != nil {
		return false, err
	}
	if _, err = t.CreateVariable(node); err != nil {
		return false, err
	}
	t.terms++
	return true, nil
}

func (t *smtInternalizerTestTheory) ApplySortConstraint(*SMTEnode, ast.ASTHandle) error {
	t.sortConstraints++
	return nil
}

func TestSMTTheoryInternalizationCallbacksVariablesAndScopes(t *testing.T) {
	m := astTestManager(t)
	family, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	base, err := NewSMTTheoryBase(family)
	astTestMust(t, err)
	theory := &smtInternalizerTestTheory{SMTTheoryBase: base}
	_, _, state := smtInternalizerFixture(t, m, nil)
	astTestMust(t, state.RegisterTheory(theory))

	integer, err := m.MkFamilySort(family, 1, nil)
	astTestMust(t, err)
	boolean, err := m.MkBoolSort()
	astTestMust(t, err)
	termInfo, err := m.NewFuncDeclInfo(family, 9001, nil, ast.ASTFuncFlags{}, false)
	astTestMust(t, err)
	termDecl, err := m.MkFuncDeclWithInfo(astTestSymbol(t, m, "theory-term"), nil, integer, termInfo)
	astTestMust(t, err)
	astTestMust(t, m.DelFuncDeclInfo(&termInfo))
	term, err := m.MkApp(termDecl, nil)
	astTestMust(t, err)
	atomInfo, err := m.NewFuncDeclInfo(family, 9002, nil, ast.ASTFuncFlags{}, false)
	astTestMust(t, err)
	atomDecl, err := m.MkFuncDeclWithInfo(astTestSymbol(t, m, "theory-atom"), []ast.ASTHandle{integer}, boolean, atomInfo)
	astTestMust(t, err)
	astTestMust(t, m.DelFuncDeclInfo(&atomInfo))
	atom, err := m.MkApp(atomDecl, []ast.ASTHandle{term})
	astTestMust(t, err)

	astTestMust(t, state.PushScope())
	astTestMust(t, state.Internalize(atom, true))
	node := state.context.enodes[term]
	if node == nil || theory.atoms != 1 || theory.terms != 1 || theory.sortConstraints != 1 || theory.NumVariables() != 1 || !theory.IsAttachedToVariable(node) {
		t.Fatal("theory atom/term internalization", theory.atoms, theory.terms, theory.sortConstraints, theory.NumVariables(), node)
	}
	equality, err := m.MkEq(term, term)
	astTestMust(t, err)
	astTestMust(t, state.Internalize(equality, true))
	if theory.equalities != 1 {
		t.Fatal("theory equality callback", theory.equalities)
	}
	astTestMust(t, state.PopScopes(1))
	if theory.NumVariables() != 0 || node.TheoryVar(family) != SMTNullTheoryVar || state.context.enodes[term] != nil {
		t.Fatal("theory scope rollback", theory.NumVariables(), node.TheoryVar(family), state.context.enodes[term])
	}
}
