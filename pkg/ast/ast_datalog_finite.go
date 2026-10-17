// Source: src/ast/dl_decl_plugin.cpp:98-114,453-469,655-685.
package ast

import (
	"strconv"
)

const (
	astDatalogFiniteSortKind int32 = 1
	astDatalogConstantKind   int32 = 13
)

func (m *ASTManager) datalogSort(fid FamilyID, kind int32, ps []ASTParameter) (astNodeRef, error) {
	if kind != astDatalogFiniteSortKind || len(ps) != 2 {
		return astNodeRef{}, ErrASTDomain
	}
	name, err := m.parameter(ps[0])
	if err != nil {
		return astNodeRef{}, err
	}
	if name.Kind != ASTParameterSymbol {
		return astNodeRef{}, ErrASTDomain
	}
	sym := name.symbol
	value, err := m.parameter(ps[1])
	if err != nil {
		return astNodeRef{}, err
	}
	if value.Kind != ASTParameterRational {
		return astNodeRef{}, ErrASTDomain
	}
	valid, err := m.numerals.IsUint64(&value.Rational)
	if err != nil {
		return astNodeRef{}, err
	}
	if !valid {
		return astNodeRef{}, ErrASTDomain
	}
	size, err := m.numerals.Uint64(&value.Rational)
	if err != nil {
		return astNodeRef{}, err
	}
	slot, generation, err := m.newInfo(true, fid, kind, ps, FiniteSortSize(size), ASTFuncFlags{}, false)
	if err != nil {
		return astNodeRef{}, err
	}
	defer m.dropInfoStorage(slot, generation)
	return m.mkSort(sym, ASTSortInfo{m.origin, slot, generation})
}

func (m *ASTManager) datalogDecl(kind int32, ps []ASTParameter, domain []ASTHandle) (astNodeRef, error) {
	if kind != astDatalogConstantKind || len(ps) != 2 || len(domain) != 0 {
		return astNodeRef{}, ErrASTDomain
	}
	value, err := m.parameter(ps[0])
	if err != nil {
		return astNodeRef{}, err
	}
	if value.Kind != ASTParameterRational {
		return astNodeRef{}, ErrASTDomain
	}
	valid, err := m.numerals.IsUint64(&value.Rational)
	if err != nil {
		return astNodeRef{}, err
	}
	if !valid {
		return astNodeRef{}, ErrASTDomain
	}
	sp, err := m.parameter(ps[1])
	if err != nil {
		return astNodeRef{}, err
	}
	if sp.Kind != ASTParameterAST {
		return astNodeRef{}, ErrASTDomain
	}
	s := sp.ast
	if err = m.validNode(s); err != nil {
		return astNodeRef{}, err
	}
	if m.nodeAt(s.slot).kind != ASTSort || !m.isDatalogFiniteSort(s) {
		return astNodeRef{}, ErrASTDomain
	}
	p := m.auxiliaryPlugin(astDatalogPlugin)
	if p == nil {
		return astNodeRef{}, ErrASTDomain
	}
	return m.pluginDecl(p.symbols[13], p.family, kind, ps, nil, s, ASTFuncFlags{}, false)
}

func (m *ASTManager) isDatalogFiniteSort(s astNodeRef) bool {
	info := m.sortAt(s.slot).info
	if info == (ASTSortInfo{}) {
		return false
	}
	i, err := m.sortInfo(info)
	if err != nil || i.kind != astDatalogFiniteSortKind {
		return false
	}
	p := m.auxiliaryPlugin(astDatalogPlugin)
	return p != nil && i.family == p.family
}

// DatalogNumeralText is the finite-domain probe in Z3_get_numeral_rational.
// The source converts the stored uint64 to a rational before printing it.
func (m *ASTManager) DatalogNumeralText(h ASTHandle) (value string, matched bool, err error) {
	if err = m.lock(); err != nil {
		return
	}
	defer m.mu.Unlock()
	r, err := m.requireExpr(h)
	if err != nil {
		return
	}
	if m.nodeAt(r.slot).kind != ASTApp {
		return
	}
	p := m.auxiliaryPlugin(astDatalogPlugin)
	if p == nil {
		return
	}
	decl := m.appAt(r.slot).decl
	family, kind, err := m.declIdentity(decl)
	if err != nil || family != p.family || kind != astDatalogConstantKind {
		return
	}
	info, err := m.funcInfo(m.declAt(decl.slot).info)
	if err != nil {
		return
	}
	first, err := m.infoParameter(info, 0)
	if err != nil {
		return
	}
	param, err := m.parameter(first)
	if err != nil {
		return
	}
	if param.Kind != ASTParameterRational {
		err = ErrASTDomain
		return
	}
	valid, err := m.numerals.IsUint64(&param.Rational)
	if err != nil {
		return
	}
	if !valid {
		err = ErrASTDomain
		return
	}
	n, err := m.numerals.Uint64(&param.Rational)
	if err != nil {
		return
	}
	return strconv.FormatUint(n, 10), true, nil
}
