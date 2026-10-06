// Source: src/ast/ast.h:1662,2033-2063; src/ast/ast.cpp:2503-2527.
// Licensed under the MIT license; source authors: Leonardo de Moura; Nikolaj Bjorner.
package g3

func (m *ASTManager) mkEquality(lhs, rhs ASTHandle) (astNodeRef, error) {
	r, e := m.requireExpr(lhs)
	if e != nil {
		return astNodeRef{}, e
	}
	s, e := m.sortOf(r)
	if e != nil {
		return astNodeRef{}, e
	}
	k := int32(ASTOpEq)
	if s == m.basic.boolSort {
		k = ASTOpIff
	}
	args := [2]ASTHandle{lhs, rhs}
	return m.mkFamilyApp(0, k, nil, args[:])
}
func (m *ASTManager) MkEq(lhs, rhs ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	r, e := m.mkEquality(lhs, rhs)
	return m.external(r), e
}
func (m *ASTManager) MkIff(lhs, rhs ASTHandle) (ASTHandle, error) {
	args := [2]ASTHandle{lhs, rhs}
	return m.MkFamilyApp(0, ASTOpIff, nil, args[:])
}
func (m *ASTManager) MkOEq(lhs, rhs ASTHandle) (ASTHandle, error) {
	args := [2]ASTHandle{lhs, rhs}
	return m.MkFamilyApp(0, ASTOpOEq, nil, args[:])
}
func (m *ASTManager) MkXor(lhs, rhs ASTHandle) (ASTHandle, error) {
	args := [2]ASTHandle{lhs, rhs}
	return m.MkFamilyApp(0, ASTOpXor, nil, args[:])
}
func (m *ASTManager) MkIte(c, t, e ASTHandle) (ASTHandle, error) {
	args := [3]ASTHandle{c, t, e}
	return m.MkFamilyApp(0, ASTOpIte, nil, args[:])
}
func (m *ASTManager) MkImplies(lhs, rhs ASTHandle) (ASTHandle, error) {
	args := [2]ASTHandle{lhs, rhs}
	return m.MkFamilyApp(0, ASTOpImplies, nil, args[:])
}
func (m *ASTManager) MkInterp(a ASTHandle) (ASTHandle, error) {
	args := [1]ASTHandle{a}
	return m.MkFamilyApp(0, ASTOpInterp, nil, args[:])
}
func (m *ASTManager) MkAnd3(a, b, c ASTHandle) (ASTHandle, error) {
	args := [3]ASTHandle{a, b, c}
	return m.MkFamilyApp(0, ASTOpAnd, nil, args[:])
}
func (m *ASTManager) MkOr3(a, b, c ASTHandle) (ASTHandle, error) {
	args := [3]ASTHandle{a, b, c}
	return m.MkFamilyApp(0, ASTOpOr, nil, args[:])
}

// These ast_manager array overloads always construct an application. They
// differ from ast_util's separately ported singleton/empty MkOrArray shortcut.
func (m *ASTManager) MkXorArgs(args []ASTHandle) (ASTHandle, error) {
	return m.MkFamilyApp(0, ASTOpXor, nil, args)
}
func (m *ASTManager) MkAndArgs(args []ASTHandle) (ASTHandle, error) {
	return m.MkFamilyApp(0, ASTOpAnd, nil, args)
}
func (m *ASTManager) MkOrArgs(args []ASTHandle) (ASTHandle, error) {
	return m.MkFamilyApp(0, ASTOpOr, nil, args)
}
func (m *ASTManager) MkBoolVal(b bool) (ASTHandle, error) {
	if b {
		return m.MkTrue()
	}
	return m.MkFalse()
}
func (m *ASTManager) MkDistinct(args []ASTHandle) (ASTHandle, error) {
	return m.MkFamilyApp(0, ASTOpDistinct, nil, args)
}
func (m *ASTManager) MkDistinctExpanded(args []ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	if len(args) < 2 {
		return m.external(m.trueApp), nil
	}
	if len(args) == 2 {
		eq, e := m.mkEquality(args[0], args[1])
		if e != nil {
			return ASTHandle{}, e
		}
		unary := [1]ASTHandle{m.external(eq)}
		r, e := m.mkFamilyApp(0, ASTOpNot, nil, unary[:])
		return m.external(r), e
	}
	// The literal ptr_buffer<expr> has sixteen inline non-owning slots. Reuse
	// the existing concrete storage engine without adding reference increments.
	var expanded astProofArguments
	defer expanded.close(m)
	for i := 0; i < len(args)-1; i++ {
		a1 := args[i]
		for j := i + 1; j < len(args); j++ {
			eq, e := m.mkEquality(a1, args[j])
			if e != nil {
				return ASTHandle{}, e
			}
			unary := [1]ASTHandle{m.external(eq)}
			neg, e := m.mkFamilyApp(0, ASTOpNot, nil, unary[:])
			if e != nil {
				return ASTHandle{}, e
			}
			if e = expanded.push(m, m.external(neg)); e != nil {
				return ASTHandle{}, e
			}
		}
	}
	r, e := m.mkFamilyApp(0, ASTOpAnd, nil, expanded.slice())
	return m.external(r), e
}
func (m *ASTManager) basicBooleanDeclaration(k int32, n uint32) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	domain := [2]ASTHandle{m.external(m.basic.boolSort), m.external(m.basic.boolSort)}
	r, e := m.basicDecl(k, nil, domain[:n])
	return m.external(r), e
}
func (m *ASTManager) MkAndDecl() (ASTHandle, error) { return m.basicBooleanDeclaration(ASTOpAnd, 2) }
func (m *ASTManager) MkOrDecl() (ASTHandle, error)  { return m.basicBooleanDeclaration(ASTOpOr, 2) }
func (m *ASTManager) MkNotDecl() (ASTHandle, error) { return m.basicBooleanDeclaration(ASTOpNot, 1) }
