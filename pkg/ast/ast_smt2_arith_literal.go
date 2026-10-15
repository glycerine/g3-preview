// Source: src/ast/ast_smt2_pp.cpp:296-345. Default decimal=false profile.
package ast

import "math/big"

// smt2PPArithLiteralDefault returns matched=false for expressions outside the
// concrete rational numeral specialization reached by the AIG SMT2 printer.
func (m *ASTManager) smt2PPArithLiteralDefault(h ASTHandle) (format ASTHandle, matched bool, err error) {
	value, integer, matched, err := m.ArithmeticNumeralText(h)
	if err != nil || !matched {
		return ASTHandle{}, matched, err
	}
	var rational big.Rat
	if _, ok := rational.SetString(value); !ok {
		return ASTHandle{}, true, ErrASTDomain
	}
	negative := rational.Sign() < 0
	if negative {
		rational.Neg(&rational)
	}
	if integer {
		if !rational.IsInt() {
			return ASTHandle{}, true, ErrASTDomain
		}
		format, err = m.FormatString(rational.Num().String())
	} else if rational.IsInt() {
		format, err = m.FormatString(rational.Num().String() + ".0")
	} else {
		numerator, e := m.FormatString(rational.Num().String() + ".0")
		if e != nil {
			return ASTHandle{}, true, e
		}
		denominator, e := m.FormatString(rational.Denom().String() + ".0")
		if e != nil {
			return ASTHandle{}, true, e
		}
		format, err = m.FormatSeq1("/", []ASTHandle{numerator, denominator})
	}
	if err != nil {
		return ASTHandle{}, true, err
	}
	if negative {
		format, err = m.FormatSeq1("-", []ASTHandle{format})
	}
	return format, true, err
}
