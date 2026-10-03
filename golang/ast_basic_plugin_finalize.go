// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package g3

import "reflect"

func (m *ASTManager) finalizeBasic() error {
	var failure error
	drop := func(r *astNodeRef) {
		if r.slot == 0 {
			return
		}
		consumed, e := m.decNode(*r)
		if consumed {
			*r = astNodeRef{}
		}
		if failure == nil {
			failure = e
		}
	}
	array := func(v []astNodeRef) {
		for i := range v {
			drop(&v[i])
		}
	}
	drop(&m.basic.boolSort)
	for _, k := range [...]int32{ASTOpTrue, ASTOpFalse, ASTOpAnd, ASTOpOr, ASTOpNot, ASTOpInterp, ASTOpIff, ASTOpXor, ASTOpImplies} {
		drop(&m.basic.declarations[k])
	}
	array(m.basic.eq)
	array(m.basic.ite)
	array(m.basic.oeq)
	drop(&m.basic.proofSort)
	for _, k := range [...]int32{ASTPrUndef, ASTPrTrue, ASTPrAsserted, ASTPrGoal, ASTPrModusPonens, ASTPrReflexivity, ASTPrSymmetry, ASTPrTransitivity, ASTPrQuantIntro, ASTPrAndElim, ASTPrNotOrElim, ASTPrRewrite, ASTPrPullQuant, ASTPrPushQuant, ASTPrElimUnusedVars, ASTPrDER, ASTPrQuantInst} {
		drop(&m.basic.declarations[k])
	}
	for _, k := range [...]int32{ASTPrMonotonicity, ASTPrTransitivityStar, ASTPrDistributivity, ASTPrRewriteStar} {
		array(m.basic.proofCaches[k])
	}
	for _, k := range [...]int32{ASTPrHypothesis, ASTPrIffTrue, ASTPrIffFalse, ASTPrCommutativity, ASTPrDefAxiom, ASTPrLemma} {
		drop(&m.basic.declarations[k])
	}
	array(m.basic.proofCaches[ASTPrUnitResolution])
	for _, k := range [...]int32{ASTPrDefIntro, ASTPrIffOEq, ASTPrSkolemize, ASTPrModusPonensOEq} {
		drop(&m.basic.declarations[k])
	}
	for _, k := range [...]int32{ASTPrApplyDef, ASTPrNNFPos, ASTPrNNFNeg, ASTPrTheoryLemma} {
		array(m.basic.proofCaches[k])
	}
	drop(&m.basic.declarations[ASTPrHyperResolve])
	return failure
}
func (m *ASTManager) destroyPluginBacking() {
	width := uint64(reflect.TypeOf(astNodeRef{}).Size())
	m.budget.Release(uint64(cap(m.basic.eq)+cap(m.basic.ite)+cap(m.basic.oeq)) * width)
	m.basic.eq = nil
	m.basic.ite = nil
	m.basic.oeq = nil
	for i := range m.basic.proofCaches {
		m.budget.Release(uint64(cap(m.basic.proofCaches[i])) * width)
		m.basic.proofCaches[i] = nil
	}
	m.budget.Release(uint64(cap(m.builtin.userNames)) * uint64(reflect.TypeOf(Symbol{}).Size()))
	m.builtin.userNames = nil
	m.builtin.userTable.close()
	m.plugins = [5]astPluginBinding{}
}
