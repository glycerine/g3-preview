// Source: src/ast/arith_decl_plugin.cpp:561-577,627-630.
package ast

func (m *ASTManager) arithmeticValue(fid FamilyID, r astNodeRef, unique bool) (bool, error) {
	if m.nodeAt(r.slot).kind != ASTApp {
		return false, nil
	}
	f, kind, e := m.declIdentity(m.appAt(r.slot).decl)
	if e != nil || f != fid {
		return false, e
	}
	return kind == 0 || kind == 31 || kind == 32 || !unique && kind == 1, nil
}

func (m *ASTManager) arithmeticSomeValue(sort astNodeRef) (ASTHandle, error) {
	p := m.arithmeticPlugin
	if sort != p.fields[0] && sort != p.fields[1] {
		return ASTHandle{}, ErrASTDomain
	}
	previous := m.formatExecuting
	m.formatExecuting = true
	defer func() { m.formatExecuting = previous }()
	r, e := m.mkArithmeticNumeral(nil, sort == p.fields[1], true)
	return m.external(r), e
}
