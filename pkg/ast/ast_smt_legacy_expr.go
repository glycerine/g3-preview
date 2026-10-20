// Source: src/ast/ast_smt_pp.cpp:176-206,292-417,564-607.
package ast

import (
	"io"
	"strings"
)

func legacySMTRational(text string, integerSort bool) string {
	negative := strings.HasPrefix(text, "-")
	if negative {
		text = text[1:]
	}
	formatted := text
	if slash := strings.IndexByte(text, '/'); slash >= 0 {
		numerator, denominator := text[:slash], text[slash+1:]
		if !integerSort {
			numerator += ".0"
			denominator += ".0"
		}
		formatted = "(/ " + numerator + " " + denominator + ")"
	} else if !integerSort {
		formatted += ".0"
	}
	if negative {
		return "(- " + formatted + ")"
	}
	return formatted
}

func (p *LegacySMTPrinter) declarationText(decl ASTHandle) (string, error) {
	family, err := p.manager.GetFamilyID(decl)
	if err != nil {
		return "", err
	}
	op, err := p.manager.GetDeclKind(decl)
	if err != nil {
		return "", err
	}
	if family == p.manager.BasicFamilyID() {
		switch op {
		case ASTOpIte:
			return "ite", nil
		case ASTOpIff:
			return "=", nil
		case ASTOpImplies:
			return "=>", nil
		}
	}
	if family == p.manager.arithmeticPluginFamily() && op == 8 {
		return "-", nil
	}
	name, err := p.manager.GetASTName(decl)
	if err != nil {
		return "", err
	}
	return p.renamedText(name)
}

func (m *ASTManager) arithmeticPluginFamily() FamilyID {
	if m.arithmeticPlugin == nil {
		return NullFamilyID
	}
	return m.arithmeticPlugin.family
}

func (p *LegacySMTPrinter) exprText(h ASTHandle, variableNames []string) (string, error) {
	kind, err := p.manager.GetKind(h)
	if err != nil {
		return "", err
	}
	switch kind {
	case ASTVar:
		index, e := p.manager.GetVarIndex(h)
		if e != nil {
			return "", e
		}
		if int(index) < len(variableNames) {
			return variableNames[len(variableNames)-int(index)-1], nil
		}
		return "?" + uintText(index), nil
	case ASTQuantifier:
		return "", ErrASTDomain
	case ASTApp:
	default:
		return "", ErrASTDomain
	}
	if value, integer, matched, e := p.manager.ArithmeticNumeralText(h); e != nil {
		return "", e
	} else if matched {
		return legacySMTRational(value, integer), nil
	}
	decl, err := p.manager.GetDecl(h)
	if err != nil {
		return "", err
	}
	name, err := p.declarationText(decl)
	if err != nil {
		return "", err
	}
	n, err := p.manager.GetNumArgs(h)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return name, nil
	}
	var out strings.Builder
	out.WriteByte('(')
	out.WriteString(name)
	for i := uint32(0); i < n; i++ {
		arg, e := p.manager.GetArg(h, i)
		if e != nil {
			return "", e
		}
		text, e := p.exprText(arg, variableNames)
		if e != nil {
			return "", e
		}
		out.WriteByte(' ')
		out.WriteString(text)
	}
	out.WriteByte(')')
	return out.String(), nil
}

func uintText(value uint32) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	var buffer [10]byte
	i := len(buffer)
	for value != 0 {
		i--
		buffer[i] = digits[value%10]
		value /= 10
	}
	return string(buffer[i:])
}

func (p *LegacySMTPrinter) DisplayExpr(out io.Writer, expr ASTHandle, variableNames []string) error {
	if p == nil || p.manager == nil || p.renaming == nil || out == nil {
		return ErrASTHandle
	}
	text, err := p.exprText(expr, variableNames)
	if err != nil {
		return err
	}
	n, err := io.WriteString(out, text)
	if err == nil && n != len(text) {
		return io.ErrShortWrite
	}
	return err
}
