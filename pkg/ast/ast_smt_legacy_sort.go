// Source: src/ast/ast_smt_pp.cpp:207-290.
package ast

import (
	"io"
	"strconv"
	"strings"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

// LegacySMTPrinter owns the state shared by the SMT benchmark printer phases.
// The AST manager is borrowed; renaming maps are owned by the printer.
type LegacySMTPrinter struct {
	manager  *ASTManager
	renaming *SMTRenaming
}

func NewLegacySMTPrinter(manager *ASTManager) (*LegacySMTPrinter, error) {
	if manager == nil {
		return nil, ErrASTHandle
	}
	r, err := NewSMTRenaming(manager.Symbols)
	if err != nil {
		return nil, err
	}
	return &LegacySMTPrinter{manager: manager, renaming: r}, nil
}

func (p *LegacySMTPrinter) renamedText(s usymbol.Symbol) (string, error) {
	r, err := p.renaming.GetSymbol(s)
	if err != nil {
		return "", err
	}
	return p.manager.Symbols.Display(r)
}

func (p *LegacySMTPrinter) sortText(sort ASTHandle) (string, error) {
	v, err := p.manager.smt2SortView(sort)
	if err != nil {
		return "", err
	}
	var symbol string
	switch {
	case v.bvSort:
		symbol = "BitVec"
	case v.boolSort:
		symbol = "Bool"
	case v.arraySort:
		symbol = "Array"
	default:
		symbol, err = p.renamedText(v.name)
		if err != nil {
			return "", err
		}
	}
	if len(v.params) == 0 || symbol == "String" {
		return symbol, nil
	}
	indexed := symbol == "BitVec" || symbol == "FloatingPoint" || symbol == "RoundingMode"
	var out strings.Builder
	if indexed {
		out.WriteString("(_ ")
	} else {
		out.WriteByte('(')
	}
	out.WriteString(symbol)
	out.WriteByte(' ')
	for i, parameter := range v.params {
		if i != 0 {
			out.WriteByte(' ')
		}
		switch parameter.kind {
		case ASTParameterInt:
			out.WriteString(strconv.FormatInt(int64(int32(parameter.ival)), 10))
		case ASTParameterAST:
			kind, e := p.manager.GetKind(parameter.ast)
			if e != nil {
				return "", e
			}
			if kind != ASTSort {
				return "", ErrASTDomain
			}
			text, e := p.sortText(parameter.ast)
			if e != nil {
				return "", e
			}
			out.WriteString(text)
		default:
			return "", ErrASTDomain
		}
	}
	out.WriteByte(')')
	return out.String(), nil
}

func (p *LegacySMTPrinter) DisplaySort(out io.Writer, sort ASTHandle) error {
	if p == nil || p.manager == nil || p.renaming == nil || out == nil {
		return ErrASTHandle
	}
	text, err := p.sortText(sort)
	if err != nil {
		return err
	}
	n, err := io.WriteString(out, text)
	if err == nil && n != len(text) {
		return io.ErrShortWrite
	}
	return err
}
