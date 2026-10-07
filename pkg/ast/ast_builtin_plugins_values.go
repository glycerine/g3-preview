// Source: src/ast/ast.cpp:909-915,1127-1141,1246-1254,1589-1632,2556-2592; src/ast/ast.h:991-1020.
package ast

func (m *ASTManager) pluginValue(fid FamilyID, r astNodeRef) (bool, error) {
	if m.nodeAt(r.slot).kind != ASTApp {
		return false, nil
	}
	decl := m.appAt(r.slot).decl
	if fid == 0 {
		return decl == m.basic.declarations[ASTOpTrue] || decl == m.basic.declarations[ASTOpFalse], nil
	}
	if fid == 3 {
		f, k, e := m.declIdentity(decl)
		return f == 3 && k == 0, e
	}
	return false, nil
}
func (m *ASTManager) isValue(r astNodeRef) (bool, error) {
	if m.nodeAt(r.slot).kind != ASTApp {
		return false, nil
	}
	fid, _, e := m.declIdentity(m.appAt(r.slot).decl)
	if e != nil {
		return false, e
	}
	if fid < 0 || !m.pluginAt(fid).present {
		return false, nil
	}
	return m.pluginValue(fid, r)
}
func (m *ASTManager) IsValue(h ASTHandle) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	r, e := m.internal(h)
	if e != nil {
		return false, e
	}
	return m.isValue(r)
}

// All five actual base plugins use the same is_value/is_unique_value routes.
func (m *ASTManager) IsUniqueValue(h ASTHandle) (bool, error) { return m.IsValue(h) }
func (m *ASTManager) AreEqual(a, b ASTHandle) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	x, e := m.internal(a)
	if e != nil {
		return false, e
	}
	y, e := m.internal(b)
	if e != nil {
		return false, e
	}
	if x == y {
		return true, nil
	}
	return false, nil
}
func (m *ASTManager) AreDistinct(a, b ASTHandle) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	x, e := m.internal(a)
	if e != nil {
		return false, e
	}
	y, e := m.internal(b)
	if e != nil {
		return false, e
	}
	if m.nodeAt(x.slot).kind != ASTApp || m.nodeAt(y.slot).kind != ASTApp {
		return false, nil
	}
	fid, _, e := m.declIdentity(m.appAt(x.slot).decl)
	if e != nil {
		return false, e
	}
	if fid < 0 || !m.pluginAt(fid).present {
		fid, _, e = m.declIdentity(m.appAt(y.slot).decl)
		if e != nil {
			return false, e
		}
	}
	if fid < 0 || !m.pluginAt(fid).present || x == y {
		return false, nil
	}
	first, e := m.pluginValue(fid, x)
	if e != nil || !first {
		return false, e
	}
	return m.pluginValue(fid, y)
}
func (m *ASTManager) GetSomeValue(sort ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	r, e := m.requireKind(sort, ASTSort)
	if e != nil {
		return ASTHandle{}, e
	}
	fid, _, e := m.declIdentity(r)
	if e != nil {
		return ASTHandle{}, e
	}
	if fid == 0 && r == m.basic.boolSort {
		v, e := m.mkFamilyApp(0, ASTOpFalse, nil, nil)
		return m.external(v), e
	}
	// Actual manager fallback constructs two lexical parameters; AST-valued
	// parameter construction doesn't retain until its declaration info attaches.
	index, e := m.newParameterInt(0)
	if e != nil {
		return ASTHandle{}, e
	}
	defer m.dropParameter(index)
	value, e := m.allocParameter()
	if e != nil {
		return ASTHandle{}, e
	}
	m.parameterAt(value.slot).Kind = ASTParameterAST
	m.parameterAt(value.slot).ast = r
	defer m.dropParameter(value)
	ps := [2]ASTParameter{index, value}
	v, e := m.mkFamilyApp(3, 0, ps[:], nil)
	return m.external(v), e
}
func (m *ASTManager) IsFullyInterpreted(sort ASTHandle) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	r, e := m.requireKind(sort, ASTSort)
	if e != nil {
		return false, e
	}
	fid, _, e := m.declIdentity(r)
	if e != nil {
		return false, e
	}
	if fid == NullFamilyID || fid == 4 {
		return false, nil
	}
	return fid >= 0 && m.pluginAt(fid).present, nil
}
