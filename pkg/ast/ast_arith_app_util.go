// Source: src/ast/arith_decl_plugin.h:81-145.
package ast

import "errors"

func (m *ASTManager) arithmeticApp(kind int32, args ...ASTHandle) (ASTHandle, error) {
	family, err := m.RegisterArithmeticPlugin()
	if err != nil {
		return ASTHandle{}, err
	}
	return m.MkFamilyApp(family, kind, nil, args)
}

func (m *ASTManager) MkArithmeticAdd(left, right ASTHandle) (ASTHandle, error) {
	return m.arithmeticApp(6, left, right)
}

func (m *ASTManager) MkArithmeticMul(left, right ASTHandle) (ASTHandle, error) {
	return m.arithmeticApp(9, left, right)
}

func (m *ASTManager) MkArithmeticDiv(left, right ASTHandle) (ASTHandle, error) {
	return m.arithmeticApp(10, left, right)
}

func (m *ASTManager) MkArithmeticNeg(value ASTHandle) (ASTHandle, error) {
	return m.arithmeticApp(8, value)
}

func (m *ASTManager) MkArithmeticPower(base, exponent ASTHandle) (ASTHandle, error) {
	return m.arithmeticApp(18, base, exponent)
}

// ArithmeticRealNumeralText constructs a real numeral from source-compatible
// rational text.
func (m *ASTManager) ArithmeticRealNumeralText(text string) (result ASTHandle, resultErr error) {
	service := m.NumeralService()
	value, err := service.NewValue()
	if err != nil {
		return ASTHandle{}, err
	}
	defer func() { resultErr = errors.Join(resultErr, service.Del(&value)) }()
	if err = service.SetString(&value, text); err != nil {
		return ASTHandle{}, err
	}
	return m.ArithmeticNumeral(&value, false)
}
