// Source: src/ast/ast_smt_pp.cpp:756-850.
package ast

import (
	"io"
	"strings"
)

func (p *LegacySMTPrinter) sortDeclarationText(sort ASTHandle) (string, error) {
	view, err := p.manager.smt2SortView(sort)
	if err != nil {
		return "", err
	}
	if view.datatypeSort {
		return "", ErrASTDomain
	}
	text, err := p.sortText(sort)
	if err != nil {
		return "", err
	}
	return "(declare-sort " + text + ")\n", nil
}

func (p *LegacySMTPrinter) funcDeclarationText(decl ASTHandle) (string, error) {
	ctx := legacySMTExprContext{}
	name, err := p.declarationText(decl, &ctx, true)
	if err != nil {
		return "", err
	}
	arity, err := p.manager.GetArity(decl)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	out.WriteString("(declare-fun ")
	out.WriteString(name)
	out.WriteString(" (")
	for i := uint32(0); i < arity; i++ {
		if i != 0 {
			out.WriteByte(' ')
		}
		sort, e := p.manager.GetDomain(decl, i)
		if e != nil {
			return "", e
		}
		text, e := p.sortText(sort)
		if e != nil {
			return "", e
		}
		out.WriteString(text)
	}
	rangeSort, err := p.manager.GetRange(decl)
	if err != nil {
		return "", err
	}
	rangeText, err := p.sortText(rangeSort)
	if err != nil {
		return "", err
	}
	out.WriteString(") ")
	out.WriteString(rangeText)
	out.WriteByte(')')
	return out.String(), nil
}

func legacySMTWriteText(out io.Writer, body string) error {
	if out == nil {
		return ErrASTHandle
	}
	n, err := io.WriteString(out, body)
	if err == nil && n != len(body) {
		return io.ErrShortWrite
	}
	return err
}

func (p *LegacySMTPrinter) DisplaySortDeclaration(out io.Writer, sort ASTHandle) error {
	if p == nil || p.manager == nil || p.renaming == nil {
		return ErrASTHandle
	}
	body, err := p.sortDeclarationText(sort)
	if err != nil {
		return err
	}
	return legacySMTWriteText(out, body)
}

func (p *LegacySMTPrinter) DisplayFuncDeclaration(out io.Writer, decl ASTHandle) error {
	if p == nil || p.manager == nil || p.renaming == nil {
		return ErrASTHandle
	}
	body, err := p.funcDeclarationText(decl)
	if err != nil {
		return err
	}
	return legacySMTWriteText(out, body)
}
