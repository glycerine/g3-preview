// Source: src/ast/ast_smt2_pp.cpp:872-902,904-917,922-989;
// src/ast/format.h:133-157.
package ast

import (
	"strconv"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

// smt2PPQuantifierRoot prints source binder names, children and annotations.
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
	used := make(map[string]bool, len(outer)+int(count))
	for _, name := range outer {
		spelling, e := m.smt2EnsureQuote(name)
		if e != nil {
			return ASTHandle{}, e
		}
		used[spelling] = true
	}
	binders := make([]usymbol.Symbol, 0, count)
	spellings := make([]string, 0, count)
	sorts := make([]ASTHandle, 0, count)
	for i := uint32(0); i < count; i++ {
		name, e := m.GetDeclarationName(q, i)
		if e != nil {
			return ASTHandle{}, e
		}
		if name.IsNull() {
			return ASTHandle{}, ErrASTDomain
		}
		spelling := ""
		prefix := "x"
		if !name.IsNumerical() {
			spelling, e = m.smt2EnsureQuote(name)
			if e != nil {
				return ASTHandle{}, e
			}
			prefix = spelling
		}
		if name.IsNumerical() || used[spelling] {
			for suffix := uint32(1); ; suffix++ {
				candidate := prefix + "!" + strconv.FormatUint(uint64(suffix), 10)
				if used[candidate] {
					if suffix == ^uint32(0) {
						return ASTHandle{}, ErrASTDomain
					}
					continue
				}
				name, e = m.Symbols.Intern(candidate)
				if e != nil {
					return ASTHandle{}, e
				}
				printed, quoteErr := m.smt2EnsureQuote(name)
				if quoteErr != nil {
					return ASTHandle{}, quoteErr
				}
				if printed != candidate {
					return ASTHandle{}, ErrASTDomain
				}
				spelling = candidate
				break
			}
		}
		used[spelling] = true
		binders = append(binders, name)
		sort, e := m.GetDeclarationSort(q, i)
		if e != nil {
			return ASTHandle{}, e
		}
		spellings = append(spellings, spelling)
		sorts = append(sorts, sort)
	}
	scopedNames := append(append([]usymbol.Symbol{}, outer...), binders...)
	printedPatterns := make([]ASTHandle, 0, patterns)
	for i := uint32(0); i < patterns; i++ {
		pattern, e := m.GetPattern(q, i)
		if e != nil {
			return ASTHandle{}, e
		}
		printed, e := m.smt2PPExprTree(pattern, scopedNames, false)
		if e != nil {
			return ASTHandle{}, e
		}
		printedPatterns = append(printedPatterns, printed)
	}
	printedNoPatterns := make([]ASTHandle, 0, noPatterns)
	for i := uint32(0); i < noPatterns; i++ {
		pattern, e := m.GetNoPattern(q, i)
		if e != nil {
			return ASTHandle{}, e
		}
		printed, e := m.smt2PPExprTree(pattern, scopedNames, false)
		if e != nil {
			return ASTHandle{}, e
		}
		printedNoPatterns = append(printedNoPatterns, printed)
	}
	body, err := m.GetQuantifierBody(q)
	if err != nil {
		return ASTHandle{}, err
	}
	printedBody, err := m.smt2PPExprTree(body, scopedNames, false)
	if err != nil {
		return ASTHandle{}, err
	}
	if patterns != 0 || noPatterns != 0 || weight > 1 || !skid.IsNull() || !qid.IsNull() && !qid.IsNumerical() {
		parts := []ASTHandle{printedBody}
		for _, printed := range printedPatterns {
			attr, e := m.SMT2PPAttribute(":pattern ", printed)
			if e != nil {
				return ASTHandle{}, e
			}
			parts = append(parts, attr)
		}
		for _, printed := range printedNoPatterns {
			attr, e := m.SMT2PPAttribute(":no-pattern ", printed)
			if e != nil {
				return ASTHandle{}, e
			}
			parts = append(parts, attr)
		}
		if weight > 1 {
			attr, e := m.SMT2PPSimpleAttributeInt(":weight ", weight)
			if e != nil {
				return ASTHandle{}, e
			}
			parts = append(parts, attr)
		}
		if !skid.IsNull() {
			attr, e := m.SMT2PPSimpleAttributeSymbol(":skolemid ", skid)
			if e != nil {
				return ASTHandle{}, e
			}
			parts = append(parts, attr)
		}
		if !qid.IsNull() && !qid.IsNumerical() {
			attr, e := m.SMT2PPSimpleAttributeSymbol(":qid ", qid)
			if e != nil {
				return ASTHandle{}, e
			}
			parts = append(parts, attr)
		}
		printedBody, err = m.FormatSeq1("!", parts)
		if err != nil {
			return ASTHandle{}, err
		}
	}
	decls := make([]ASTHandle, 0, count)
	for i, sort := range sorts {
		printedSort, e := m.SMT2PPSort(sort)
		if e != nil {
			return ASTHandle{}, e
		}
		decl, e := m.FormatSeq1(spellings[i], []ASTHandle{printedSort})
		if e != nil {
			return ASTHandle{}, e
		}
		decls = append(decls, decl)
	}
	printedDecls, err := m.FormatSeq5(decls)
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
