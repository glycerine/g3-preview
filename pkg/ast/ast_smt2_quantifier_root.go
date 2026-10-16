// Source: src/ast/ast_smt2_pp.cpp:872-902,904-917,922-989;
// src/ast/format.h:133-157.
package ast

import (
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

// smt2PPQuantifierRoot prints the reached unannotated binder profile.
// Other source annotation and automatic-name branches still need their
// declaration scopes and are refused until those paths are ported.
func (m *ASTManager) smt2PPQuantifierRoot(q ASTHandle, outer []usymbol.Symbol) (ASTHandle, error) {
	count, err := m.GetNumDeclarations(q)
	if err != nil || count == 0 {
		return ASTHandle{}, ErrASTDomain
	}
	patterns, err := m.GetNumPatterns(q)
	if err != nil {
		return ASTHandle{}, err
	}
	noPatterns, err := m.GetNumNoPatterns(q)
	if err != nil {
		return ASTHandle{}, err
	}
	weight, err := m.GetQuantifierWeight(q)
	if err != nil {
		return ASTHandle{}, err
	}
	qid, skid, err := m.GetQuantifierIDs(q)
	if err != nil {
		return ASTHandle{}, err
	}
	if patterns != 0 || noPatterns != 0 || weight > 1 || !skid.IsNull() || !qid.IsNull() && !qid.IsNumerical() {
		return ASTHandle{}, ErrASTDomain
	}
	used := make(map[string]bool, len(outer)+int(count))
	for _, name := range outer {
		spelling, e := m.smt2EnsureQuote(name)
		if e != nil {
			return ASTHandle{}, e
		}
		used[spelling] = true
	}
	binders := make([]usymbol.Symbol, 0, count)
	decls := make([]ASTHandle, 0, count)
	for i := uint32(0); i < count; i++ {
		name, e := m.GetDeclarationName(q, i)
		if e != nil {
			return ASTHandle{}, e
		}
		if name.IsNull() || name.IsNumerical() {
			return ASTHandle{}, ErrASTDomain
		}
		spelling, e := m.smt2EnsureQuote(name)
		if e != nil {
			return ASTHandle{}, e
		}
		if used[spelling] {
			return ASTHandle{}, ErrASTDomain
		}
		used[spelling] = true
		binders = append(binders, name)
		sort, e := m.GetDeclarationSort(q, i)
		if e != nil {
			return ASTHandle{}, e
		}
		printedSort, e := m.SMT2PPSort(sort)
		if e != nil {
			return ASTHandle{}, e
		}
		decl, e := m.FormatSeq1(spelling, []ASTHandle{printedSort})
		if e != nil {
			return ASTHandle{}, e
		}
		decls = append(decls, decl)
	}
	printedDecls, err := m.FormatSeq5(decls)
	if err != nil {
		return ASTHandle{}, err
	}
	body, err := m.GetQuantifierBody(q)
	if err != nil {
		return ASTHandle{}, err
	}
	printedBody, err := m.smt2PPExprTree(body, append(append([]usymbol.Symbol{}, outer...), binders...), false)
	if err != nil {
		return ASTHandle{}, err
	}
	forall, err := m.IsForall(q)
	if err != nil {
		return ASTHandle{}, err
	}
	header := "exists"
	if forall {
		header = "forall"
	}
	left, err := m.FormatString("(" + header)
	if err != nil {
		return ASTHandle{}, err
	}
	space, err := m.FormatString(" ")
	if err != nil {
		return ASTHandle{}, err
	}
	first, err := m.FormatCompose([]ASTHandle{space, printedDecls})
	if err != nil {
		return ASTHandle{}, err
	}
	first, err = m.FormatIndent(uint32(len(header)+2), first)
	if err != nil {
		return ASTHandle{}, err
	}
	first, err = m.FormatGroup(first)
	if err != nil {
		return ASTHandle{}, err
	}
	tail, err := m.smt2FormatTail([]ASTHandle{printedBody})
	if err != nil {
		return ASTHandle{}, err
	}
	tail, err = m.FormatIndent(2, tail)
	if err != nil {
		return ASTHandle{}, err
	}
	right, err := m.FormatString(")")
	if err != nil {
		return ASTHandle{}, err
	}
	whole, err := m.FormatCompose([]ASTHandle{left, first, tail, right})
	if err != nil {
		return ASTHandle{}, err
	}
	return m.FormatGroup(whole)
}
