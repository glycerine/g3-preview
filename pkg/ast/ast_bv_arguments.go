// Source: src/ast/bv_decl_plugin.cpp:571-630; src/ast/bv_decl_plugin.h:406;
// src/ast/ast.cpp:1863-1881,2064-2211.
package ast

// MkBitVectorOrArguments borrows the actual operand storage through the
// original cached declaration and application factories, without a clone.
func (m *ASTManager) MkBitVectorOrArguments(args AppArguments) (ASTHandle, error) {
	return m.mkBitVectorArguments(31, args)
}

func (m *ASTManager) mkBitVectorArguments(kind int32, args AppArguments) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	return m.bitVectorArgumentsCore(kind, args)
}

func (m *ASTManager) bitVectorArgumentsCore(kind int32, args AppArguments) (ASTHandle, error) {
	if m.bitVectorPlugin == nil {
		return ASTHandle{}, ErrASTDomain
	}
	d, handled, e := m.bitVectorExprDeclArguments(kind, nil, args)
	if e != nil {
		return ASTHandle{}, e
	}
	if !handled {
		return ASTHandle{}, ErrASTDomain
	}
	if d.slot == 0 {
		return ASTHandle{}, nil
	}
	r, e := m.mkAppArguments(d, args)
	return m.external(r), e
}
