// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package g3

import "github.com/glycerine/g3/golang/mpnum"

func (m *ASTManager) parameterOfKind(p ASTParameter, kind ASTParameterKind) (*astParameterRecord, error) {
	r, e := m.parameter(p)
	if e != nil {
		return nil, e
	}
	if r.kind != kind {
		return nil, ErrASTDomain
	}
	return r, nil
}
func (m *ASTManager) ParameterInt(p ASTParameter) (int32, error) {
	if e := m.lock(); e != nil {
		return 0, e
	}
	defer m.mu.Unlock()
	r, e := m.parameterOfKind(p, ASTParameterInt)
	if e != nil {
		return 0, e
	}
	return r.integer, nil
}
func (m *ASTManager) ParameterAST(p ASTParameter) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	r, e := m.parameterOfKind(p, ASTParameterAST)
	if e != nil {
		return ASTHandle{}, e
	}
	return m.external(r.ast), nil
}
func (m *ASTManager) ParameterSymbol(p ASTParameter) (Symbol, error) {
	if e := m.lock(); e != nil {
		return Symbol{}, e
	}
	defer m.mu.Unlock()
	r, e := m.parameterOfKind(p, ASTParameterSymbol)
	if e != nil {
		return Symbol{}, e
	}
	return r.symbol, nil
}
func (m *ASTManager) ParameterDouble(p ASTParameter) (float64, error) {
	if e := m.lock(); e != nil {
		return 0, e
	}
	defer m.mu.Unlock()
	r, e := m.parameterOfKind(p, ASTParameterDouble)
	if e != nil {
		return 0, e
	}
	return r.double, nil
}
func (m *ASTManager) ParameterExternal(p ASTParameter) (uint32, error) {
	if e := m.lock(); e != nil {
		return 0, e
	}
	defer m.mu.Unlock()
	r, e := m.parameterOfKind(p, ASTParameterExternal)
	if e != nil {
		return 0, e
	}
	return r.external, nil
}

// ParameterRationalCopy is the source is_rational(rational&) overload:
// a kind mismatch leaves the caller's owner-confined numeral untouched.
// The C++ const-reference getter is not exposed as a second owning Go token.
func (m *ASTManager) ParameterRationalCopy(p ASTParameter, out *mpnum.OwnedRational) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	r, e := m.parameter(p)
	if e != nil {
		return false, e
	}
	if r.kind != ASTParameterRational {
		return false, nil
	}
	if e = m.numerals.Copy(out, &r.rational); e != nil {
		return false, e
	}
	return true, nil
}
