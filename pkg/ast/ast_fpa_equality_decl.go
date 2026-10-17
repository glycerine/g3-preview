// Source: src/ast/fpa_decl_plugin.cpp:292-311,724-730.
package ast

import "github.com/glycerine/g3/pkg/util/z3exception"

const FPAOpEq int32 = astFPAEq

// The source binary-relation constructor accepts arity >= 2, checks only the
// first two equal FloatingPoint domains, and creates a two-argument chainable
// declaration. Parameters and the requested range are ignored.
func (m *ASTManager) floatingPointEqualityDecl(domain []ASTHandle) (astNodeRef, error) {
	if len(domain) < 2 {
		return astNodeRef{}, ASTException{z3exception.NewDefaultException("invalid number of arguments to floating point relation")}
	}
	first, err := m.internal(domain[0])
	if err != nil {
		return astNodeRef{}, err
	}
	second, err := m.internal(domain[1])
	if err != nil {
		return astNodeRef{}, err
	}
	valid := first == second && first.slot != 0 && m.nodeAt(first.slot).kind == ASTSort
	if valid {
		var info *astInfoRecord
		info, err = m.sortInfo(m.sortAt(first.slot).info)
		valid = err == nil && info.family == m.floatingPointPlugin.family && info.kind == 0
	}
	if err != nil {
		return astNodeRef{}, err
	}
	if !valid {
		return astNodeRef{}, ASTException{z3exception.NewDefaultException("sort mismatch, expected equal FloatingPoint sorts as arguments")}
	}
	name, err := m.Symbols.Intern("fp.eq")
	if err != nil {
		return astNodeRef{}, err
	}
	return m.pluginDecl(name, m.floatingPointPlugin.family, astFPAEq, nil, domain[:2], m.boolSort, ASTFuncFlags{Chainable: true}, false)
}
