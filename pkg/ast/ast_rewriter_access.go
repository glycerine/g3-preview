// Source: src/ast/ast.h; src/ast/rewriter/rewriter_def.h.
package ast

func (m *ASTManager) RewriterMakeApp(d ASTHandle, args AppArguments) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	decl, e := m.requireKind(d, ASTFuncDecl)
	if e != nil {
		return ASTHandle{}, e
	}
	h, e := m.mkAppArguments(decl, args)
	return m.external(h), e
}

func (m *ASTManager) RewriterAssociative(d ASTHandle) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	decl, e := m.requireKind(d, ASTFuncDecl)
	if e != nil {
		return false, e
	}
	flags, e := m.funcFlags(decl)
	return flags.LeftAssociative && flags.RightAssociative, e
}

func (m *ASTManager) RewriterVectorArguments(v ASTRefVector) (AppArguments, error) {
	if e := m.lock(); e != nil {
		return AppArguments{}, e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v)
	return AppArguments{refVector: true, vector: s, formatOwner: m, count: s.size}, e
}

func (m *ASTManager) RewriterPattern(t ASTHandle) (bool, error) {
	kind, e := m.GetKind(t)
	if e != nil || kind != ASTApp {
		return false, e
	}
	fid, e := m.GetFamilyID(t)
	if e != nil || fid != 2 {
		return false, e
	}
	op, e := m.GetDeclKind(t)
	return op == 0, e
}

func (m *ASTManager) MkOrVector(v ASTRefVector) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	s, e := m.vectorState(v)
	if e != nil {
		return ASTHandle{}, e
	}
	if s.size == 0 {
		return m.external(m.falseApp), nil
	}
	if s.size == 1 {
		r, e := m.vectorValue(s, 0)
		return m.external(r), e
	}
	if !m.pluginAt(0).present {
		return ASTHandle{}, nil
	}
	r, e := m.mkAppArguments(m.basic.declarations[ASTOpOr], AppArguments{refVector: true, vector: s, formatOwner: m, count: s.size})
	return m.external(r), e
}
