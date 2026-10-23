// Source: src/smt/theory_array_full.cpp instantiate_select_const_axiom,
// instantiate_select_map_axiom, instantiate_select_as_array_axiom,
// try_assign_eq, and pop_scope_eh.
package smtcore

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
)

type smtArrayExpressionPair struct {
	left, right ast.ASTHandle
}

// tryAssignArrayEquality caches ordered expression pairs, independently of
// the canonical ordering used to construct the equality's Boolean atom.
func (t *SMTArrayTheory) tryAssignArrayEquality(left, right ast.ASTHandle) (bool, error) {
	pair := smtArrayExpressionPair{left: left, right: right}
	if t.equalityAxioms[pair] {
		return false, nil
	}
	t.equalityAxioms[pair] = true
	literal, err := t.MkEqualityLiteral(left, right, true)
	if err != nil {
		return false, err
	}
	if err = t.markArrayEqualityRelevant(literal); err != nil {
		return false, err
	}
	if err = t.state.addTheoryAxiom(t.FamilyID(), []SMTLiteral{literal}); err != nil {
		return false, err
	}
	return true, nil
}

func (t *SMTArrayTheory) instantiateConstructorRead(observer, constructor ast.ASTHandle) (added bool, err error) {
	s, err := t.State()
	if err != nil {
		return false, err
	}
	op, array, err := t.arrayOperation(constructor)
	if err != nil || !array {
		return false, err
	}
	if op != ast.ASTArrayOpConst && op != ast.ASTArrayOpMap && op != ast.ASTArrayOpAsArray {
		return false, nil
	}
	read := s.context.enodes[observer]
	if read == nil || len(read.arguments) < 2 {
		return false, ast.ErrASTDomain
	}
	if !t.insertArrayReadFingerprint(constructor, read.arguments[1:]) {
		return false, nil
	}
	arguments := []ast.ASTHandle{constructor}
	for _, index := range read.arguments[1:] {
		arguments = append(arguments, index.owner)
	}
	selected, err := s.manager.MkFamilyApp(t.FamilyID(), ast.ASTArrayOpSelect, nil, arguments)
	if err != nil {
		return false, err
	}
	var value ast.ASTHandle
	switch op {
	case ast.ASTArrayOpConst:
		t.statistics.selectConst++
		value, err = s.manager.GetArg(constructor, 0)
	case ast.ASTArrayOpAsArray:
		t.statistics.selectAsArray++
		var function ast.ASTHandle
		function, err = s.manager.AsArrayFunction(constructor)
		if err == nil {
			value, err = s.manager.MkApp(function, arguments[1:])
		}
	case ast.ASTArrayOpMap:
		t.statistics.mapAxiom++
		value, err = t.mappedReadValue(constructor, arguments[1:])
	}
	if err != nil {
		return false, err
	}
	// The full SMT context owns a rewriter even when assertion preprocessing
	// is disabled. Standalone internalizer fixtures can attach one explicitly.
	if op == ast.ASTArrayOpMap && s.theoryRewriter != nil {
		ref, refErr := s.manager.NewRef(value)
		if refErr != nil {
			return false, refErr
		}
		defer func() { err = errors.Join(err, s.manager.DelRef(&ref)) }()
		if err = s.theoryRewriter.Rewrite(value, ref); err != nil {
			return false, err
		}
		value, err = s.manager.RefGet(ref)
		if err != nil {
			return false, err
		}
	}
	if err = s.Internalize(selected, false); err != nil {
		return false, err
	}
	if err = s.Internalize(value, false); err != nil {
		return false, err
	}
	return t.tryAssignArrayEquality(selected, value)
}

func (t *SMTArrayTheory) mappedReadValue(constructor ast.ASTHandle, indices []ast.ASTHandle) (ast.ASTHandle, error) {
	m := t.state.manager
	declaration, err := m.GetDecl(constructor)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	function, err := m.ArrayMapFunction(declaration)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	count, err := m.GetNumArgs(constructor)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	values := make([]ast.ASTHandle, 0, count)
	for i := uint32(0); i < count; i++ {
		input, err := m.GetArg(constructor, i)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		arguments := append([]ast.ASTHandle{input}, indices...)
		value, err := m.MkFamilyApp(t.FamilyID(), ast.ASTArrayOpSelect, nil, arguments)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		values = append(values, value)
	}
	return m.MkApp(function, values)
}
