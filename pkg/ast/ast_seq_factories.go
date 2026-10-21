// Source: src/ast/seq_decl_plugin.cpp:620-672,755-767;
// src/ast/seq_decl_plugin.h:234-244.
package ast

// MkSequenceEmpty constructs the canonical empty value for a sequence sort.
// String sorts use the source empty-string specialization.
func (m *ASTManager) MkSequenceEmpty(sort ASTHandle) (ASTHandle, error) {
	if err := m.prepareSequencePlugin(); err != nil {
		return ASTHandle{}, err
	}
	if err := m.lock(); err != nil {
		return ASTHandle{}, err
	}
	defer m.mu.Unlock()
	r, err := m.requireKind(sort, ASTSort)
	if err != nil {
		return ASTHandle{}, err
	}
	value, err := m.mkSequenceEmpty(r)
	return m.external(value), err
}

// MkSequenceUnit constructs seq.unit with its result sort inferred from value.
func (m *ASTManager) MkSequenceUnit(value ASTHandle) (ASTHandle, error) {
	if err := m.prepareSequencePlugin(); err != nil {
		return ASTHandle{}, err
	}
	if err := m.lock(); err != nil {
		return ASTHandle{}, err
	}
	defer m.mu.Unlock()
	r, err := m.requireExpr(value)
	if err != nil {
		return ASTHandle{}, err
	}
	elementSort, err := m.sortOf(r)
	if err != nil {
		return ASTHandle{}, err
	}
	parameter := astParameterRecord{Kind: ASTParameterAST, ast: elementSort}
	sequenceSort, err := m.sequenceSort(m.sequencePlugin, 0, &parameter)
	if err != nil {
		return ASTHandle{}, err
	}
	decl, err := m.makePluginDeclaration(m.sequencePlugin.family, "seq.unit", ASTSequenceOpUnit, []astNodeRef{elementSort}, sequenceSort, ASTFuncFlags{})
	if err != nil {
		return ASTHandle{}, err
	}
	app, err := m.mkApp(decl, []ASTHandle{value})
	return m.external(app), err
}

// MkSequenceConcat constructs the source binary associative concatenation.
func (m *ASTManager) MkSequenceConcat(left, right ASTHandle) (ASTHandle, error) {
	if err := m.prepareSequencePlugin(); err != nil {
		return ASTHandle{}, err
	}
	if err := m.lock(); err != nil {
		return ASTHandle{}, err
	}
	defer m.mu.Unlock()
	l, err := m.requireExpr(left)
	if err != nil {
		return ASTHandle{}, err
	}
	r, err := m.requireExpr(right)
	if err != nil {
		return ASTHandle{}, err
	}
	ls, err := m.sortOf(l)
	if err != nil {
		return ASTHandle{}, err
	}
	rs, err := m.sortOf(r)
	if err != nil || ls != rs {
		return ASTHandle{}, ErrASTDomain
	}
	family, kind, err := m.declIdentity(ls)
	if err != nil || family != m.sequencePlugin.family || kind != 0 {
		return ASTHandle{}, ErrASTDomain
	}
	name := "seq.++"
	if ls == m.sequencePlugin.stringSort {
		name = "str.++"
	}
	decl, err := m.makePluginDeclaration(m.sequencePlugin.family, name, ASTSequenceOpConcat, []astNodeRef{ls, ls}, ls, ASTFuncFlags{RightAssociative: true})
	if err != nil {
		return ASTHandle{}, err
	}
	app, err := m.mkApp(decl, []ASTHandle{left, right})
	return m.external(app), err
}
