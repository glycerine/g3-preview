// Source: src/ast/ast_smt_pp.cpp:176-206,292-417,564-607.
package ast

import (
	"errors"
	"io"
	"strings"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func (p *LegacySMTPrinter) bitVectorNumeralText(h ASTHandle) (text string, matched bool, result error) {
	value, err := p.manager.numerals.NewValue()
	if err != nil {
		return "", false, err
	}
	defer func() { result = errors.Join(result, p.manager.numerals.Del(&value)) }()
	var width uint32
	matched, result = p.manager.BitVectorNumeralValue(h, &value, &width)
	if result != nil || !matched {
		return "", matched, result
	}
	valueText, err := p.manager.numerals.String(&value)
	if err != nil {
		return "", false, err
	}
	return "(_ bv" + valueText + " " + uintText(width) + ")", true, nil
}

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

type legacySMTExprContext struct {
	freeNames   []string
	quantifiers [][]usymbol.Symbol
}

func (p *LegacySMTPrinter) exprText(h ASTHandle, variableNames []string) (string, error) {
	ctx := legacySMTExprContext{freeNames: variableNames}
	return p.exprTextContext(h, &ctx)
}

func (p *LegacySMTPrinter) variableText(index uint32, ctx *legacySMTExprContext) (string, error) {
	remaining := index
	for i := len(ctx.quantifiers); i > 0; i-- {
		names := ctx.quantifiers[i-1]
		if uint64(remaining) < uint64(len(names)) {
			return p.renamedText(names[len(names)-int(remaining)-1])
		}
		remaining -= uint32(len(names))
	}
	if int(remaining) < len(ctx.freeNames) {
		return ctx.freeNames[len(ctx.freeNames)-int(remaining)-1], nil
	}
	return "?" + uintText(remaining), nil
}

func (p *LegacySMTPrinter) isLegacySkolemHackPattern(pattern ASTHandle) (bool, error) {
	n, err := p.manager.GetNumArgs(pattern)
	if err != nil || n != 1 {
		return false, err
	}
	arg, err := p.manager.GetArg(pattern, 0)
	if err != nil {
		return false, err
	}
	kind, err := p.manager.GetKind(arg)
	if err != nil || kind != ASTApp {
		return false, err
	}
	n, err = p.manager.GetNumArgs(arg)
	if err != nil || n != 1 {
		return false, err
	}
	decl, err := p.manager.GetDecl(arg)
	if err != nil {
		return false, err
	}
	name, err := p.manager.GetASTName(decl)
	if err != nil || name.IsNumerical() || name.IsNull() {
		return false, err
	}
	text, present, err := p.manager.Symbols.BareString(name)
	return present && text == "sk_hack", err
}

func (p *LegacySMTPrinter) quantifierText(q ASTHandle, ctx *legacySMTExprContext) (string, error) {
	forall, err := p.manager.IsForall(q)
	if err != nil {
		return "", err
	}
	n, err := p.manager.GetNumDeclarations(q)
	if err != nil {
		return "", err
	}
	names := make([]usymbol.Symbol, n)
	var out strings.Builder
	out.WriteByte('(')
	if forall {
		out.WriteString("forall ")
	} else {
		out.WriteString("exists ")
	}
	out.WriteByte('(')
	for i := uint32(0); i < n; i++ {
		name, e := p.manager.GetDeclarationName(q, i)
		if e != nil {
			return "", e
		}
		names[i] = name
		nameText, e := p.renamedText(name)
		if e != nil {
			return "", e
		}
		sort, e := p.manager.GetDeclarationSort(q, i)
		if e != nil {
			return "", e
		}
		sortText, e := p.sortText(sort)
		if e != nil {
			return "", e
		}
		out.WriteByte('(')
		out.WriteString(nameText)
		out.WriteByte(' ')
		out.WriteString(sortText)
		out.WriteString(") ")
	}
	out.WriteByte(')')

	patternCount, err := p.manager.GetNumPatterns(q)
	if err != nil {
		return "", err
	}
	qid, _, err := p.manager.GetQuantifierIDs(q)
	if err != nil {
		return "", err
	}
	annotated := patternCount > 0 || !qid.IsNull()
	if annotated {
		out.WriteString("(! ")
	}
	ctx.quantifiers = append(ctx.quantifiers, names)
	defer func() { ctx.quantifiers = ctx.quantifiers[:len(ctx.quantifiers)-1] }()
	body, err := p.manager.GetQuantifierBody(q)
	if err != nil {
		return "", err
	}
	bodyText, err := p.exprTextContext(body, ctx)
	if err != nil {
		return "", err
	}
	out.WriteString(bodyText)
	for i := uint32(0); i < patternCount; i++ {
		pattern, e := p.manager.GetPattern(q, i)
		if e != nil {
			return "", e
		}
		skip, e := p.isLegacySkolemHackPattern(pattern)
		if e != nil {
			return "", e
		}
		if skip {
			continue
		}
		args, e := p.manager.GetNumArgs(pattern)
		if e != nil {
			return "", e
		}
		out.WriteString(" :pattern ( ")
		for j := uint32(0); j < args; j++ {
			arg, e := p.manager.GetArg(pattern, j)
			if e != nil {
				return "", e
			}
			argText, e := p.exprTextContext(arg, ctx)
			if e != nil {
				return "", e
			}
			out.WriteString(argText)
			out.WriteByte(' ')
		}
		out.WriteByte(')')
	}
	if !qid.IsNull() {
		qidText, e := p.manager.Symbols.Display(qid)
		if e != nil {
			return "", e
		}
		out.WriteString(" :qid ")
		out.WriteString(qidText)
	}
	if annotated {
		out.WriteByte(')')
	}
	out.WriteString(")\n")
	return out.String(), nil
}

func (p *LegacySMTPrinter) exprTextContext(h ASTHandle, ctx *legacySMTExprContext) (string, error) {
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
		return p.variableText(index, ctx)
	case ASTQuantifier:
		return p.quantifierText(h, ctx)
	case ASTApp:
	default:
		return "", ErrASTDomain
	}
	if value, integer, matched, e := p.manager.ArithmeticNumeralText(h); e != nil {
		return "", e
	} else if matched {
		return legacySMTRational(value, integer), nil
	}
	if value, matched, e := p.bitVectorNumeralText(h); e != nil {
		return "", e
	} else if matched {
		return value, nil
	}
	var labels []usymbol.Symbol
	positive := false
	if matched, e := p.manager.IsLabel(h, &positive, &labels); e != nil {
		return "", e
	} else if matched && len(labels) > 0 {
		body, e := p.manager.GetArg(h, 0)
		if e != nil {
			return "", e
		}
		bodyText, e := p.exprTextContext(body, ctx)
		if e != nil {
			return "", e
		}
		label, e := p.renamedText(labels[0])
		if e != nil {
			return "", e
		}
		attribute := ":lblneg"
		if positive {
			attribute = ":lblpos"
		}
		return "(! " + bodyText + attribute + " " + label + ")", nil
	}
	labels = labels[:0]
	if matched, e := p.manager.IsLabelLit(h, &labels); e != nil {
		return "", e
	} else if matched && len(labels) > 0 {
		label, e := p.renamedText(labels[0])
		if e != nil {
			return "", e
		}
		return "(! true :lblpos " + label + ")", nil
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
		text, e := p.exprTextContext(arg, ctx)
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
