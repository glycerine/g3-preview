// Source: src/ast/arith_decl_plugin.cpp:511-559.
package ast

import "github.com/glycerine/g3/pkg/util"

func (m *ASTManager) appendArithmeticName(names *ASTBuiltinNames, text string, kind int32) error {
	s, e := m.Symbols.Intern(text)
	if e != nil {
		return e
	}
	return names.push(m.BudgetField, s, kind)
}

func (m *ASTManager) GetArithmeticSortNames(names *ASTBuiltinNames, logic util.Symbol) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if names == nil || m.arithmeticPlugin == nil {
		return ErrASTHandle
	}
	converted := false
	for _, text := range []string{"NRA", "QF_NRA", "QF_UFNRA"} {
		yes, e := m.Symbols.EqualString(logic, &text)
		if e != nil {
			return e
		}
		if yes {
			converted = true
			break
		}
	}
	if converted {
		m.arithmeticPlugin.convertIntNumerals = true
	} else {
		if e := m.appendArithmeticName(names, "Int", 1); e != nil {
			return e
		}
	}
	return m.appendArithmeticName(names, "Real", 0)
}

func (m *ASTManager) GetArithmeticOpNames(names *ASTBuiltinNames, logic util.Symbol) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if names == nil || m.arithmeticPlugin == nil {
		return ErrASTHandle
	}
	texts := [...]string{"<=", ">=", "<", ">", "+", "-", "~", "*", "/", "div", "rem", "mod", "to_real", "to_int", "is_int", "abs", "^", "sin", "cos", "tan", "asin", "acos", "atan", "sinh", "cosh", "tanh", "asinh", "acosh", "atanh", "pi", "euler"}
	for i := 0; i < 16; i++ {
		if e := m.appendArithmeticName(names, texts[i], int32(i+2)); e != nil {
			return e
		}
	}
	if !logic.IsNull() {
		all, e := m.Symbols.Intern("ALL")
		if e != nil {
			return e
		}
		if logic != all {
			return nil
		}
	}
	for i := 16; i < len(texts); i++ {
		if e := m.appendArithmeticName(names, texts[i], int32(i+2)); e != nil {
			return e
		}
	}
	return nil
}
