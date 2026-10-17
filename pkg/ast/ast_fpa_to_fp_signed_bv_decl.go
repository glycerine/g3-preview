// Source: src/ast/fpa_decl_plugin.cpp:452-468 (RoundingMode, BitVec overload).
package ast

import "github.com/glycerine/g3/pkg/util/z3exception"

const FPAOpToFP int32 = astFPAToFP

func (m *ASTManager) floatingPointToFPDecl(ps []ASTParameter, domain []ASTHandle) (astNodeRef, error) {
	// The remaining source to_fp overloads are separate live branches.
	if len(domain) != 2 {
		return astNodeRef{}, ErrASTDomain
	}
	for i := range domain {
		ref, err := m.internal(domain[i])
		if err != nil {
			return astNodeRef{}, err
		}
		if ref.slot == 0 || m.nodeAt(ref.slot).kind != ASTSort {
			return astNodeRef{}, ErrASTDomain
		}
		info, err := m.sortInfo(m.sortAt(ref.slot).info)
		if err != nil {
			return astNodeRef{}, err
		}
		if i == 0 && (info.family != m.floatingPointPlugin.family || info.kind != 1) ||
			i == 1 && (info.family != m.bitVectorPlugin.family || info.kind != 0) {
			return astNodeRef{}, ErrASTDomain
		}
	}
	fail := func(message string) (astNodeRef, error) {
		return astNodeRef{}, ASTException{z3exception.NewDefaultException(message)}
	}
	if len(ps) != 2 {
		return fail("invalid number of parameters to to_fp")
	}
	eParam, eErr := m.parameterOfKind(ps[0], ASTParameterInt)
	sParam, sErr := m.parameterOfKind(ps[1], ASTParameterInt)
	if eErr != nil || sErr != nil {
		return fail("invalid parameter type to to_fp")
	}
	fpSort, err := m.mkFloatSort(uint32(eParam.integer), uint32(sParam.integer))
	if err != nil {
		return astNodeRef{}, err
	}
	name, err := m.Symbols.Intern("to_fp")
	if err != nil {
		return astNodeRef{}, err
	}
	return m.pluginDecl(name, m.floatingPointPlugin.family, astFPAToFP, ps, domain, fpSort, ASTFuncFlags{}, false)
}
