// Source: src/ast/ast.cpp:917-979.
package ast

import (
	"reflect"

	"github.com/glycerine/g3/pkg/util"
)

func (m *ASTManager) finalizeBasic() error {
	var failure error
	drop := func(r *astNodeRef) {
		if m.sourceClosing && failure != nil {
			return
		}
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
			if m.sourceClosing && failure != nil {
				return
			}
		}
	}
	drop(&m.basic.boolSort)
	if m.sourceClosing && failure != nil {
		return failure
	}
	for _, k := range [...]int32{ASTOpTrue, ASTOpFalse, ASTOpAnd, ASTOpOr, ASTOpNot, ASTOpInterp, ASTOpIff, ASTOpXor, ASTOpImplies} {
		drop(&m.basic.declarations[k])
		if m.sourceClosing && failure != nil {
			return failure
		}
	}
	array(m.basic.eq)
	if m.sourceClosing && failure != nil {
		return failure
	}
	array(m.basic.ite)
	if m.sourceClosing && failure != nil {
		return failure
	}
	array(m.basic.oeq)
	if m.sourceClosing && failure != nil {
		return failure
	}
	drop(&m.basic.proofSort)
	if m.sourceClosing && failure != nil {
		return failure
	}
	for _, k := range [...]int32{ASTPrUndef, ASTPrTrue, ASTPrAsserted, ASTPrGoal, ASTPrModusPonens, ASTPrReflexivity, ASTPrSymmetry, ASTPrTransitivity, ASTPrQuantIntro, ASTPrAndElim, ASTPrNotOrElim, ASTPrRewrite, ASTPrPullQuant, ASTPrPushQuant, ASTPrElimUnusedVars, ASTPrDER, ASTPrQuantInst} {
		drop(&m.basic.declarations[k])
		if m.sourceClosing && failure != nil {
			return failure
		}
	}
	for _, k := range [...]int32{ASTPrMonotonicity, ASTPrTransitivityStar, ASTPrDistributivity, ASTPrRewriteStar} {
		array(m.basic.proofCaches[k])
		if m.sourceClosing && failure != nil {
			return failure
		}
	}
	for _, k := range [...]int32{ASTPrHypothesis, ASTPrIffTrue, ASTPrIffFalse, ASTPrCommutativity, ASTPrDefAxiom, ASTPrLemma} {
		drop(&m.basic.declarations[k])
		if m.sourceClosing && failure != nil {
			return failure
		}
	}
	array(m.basic.proofCaches[ASTPrUnitResolution])
	if m.sourceClosing && failure != nil {
		return failure
	}
	for _, k := range [...]int32{ASTPrDefIntro, ASTPrIffOEq, ASTPrSkolemize, ASTPrModusPonensOEq} {
		drop(&m.basic.declarations[k])
		if m.sourceClosing && failure != nil {
			return failure
		}
	}
	for _, k := range [...]int32{ASTPrApplyDef, ASTPrNNFPos, ASTPrNNFNeg, ASTPrTheoryLemma} {
		array(m.basic.proofCaches[k])
		if m.sourceClosing && failure != nil {
			return failure
		}
	}
	drop(&m.basic.declarations[ASTPrHyperResolve])
	if m.sourceClosing && failure != nil {
		return failure
	}
	return failure
}
func (m *ASTManager) destroyPluginBacking() {
	width := uint64(reflect.TypeOf(astNodeRef{}).Size())
	m.BudgetField.Release(uint64(cap(m.basic.eq)+cap(m.basic.ite)+cap(m.basic.oeq)) * width)
	m.basic.eq = nil
	m.basic.ite = nil
	m.basic.oeq = nil
	for i := range m.basic.proofCaches {
		m.BudgetField.Release(uint64(cap(m.basic.proofCaches[i])) * width)
		m.basic.proofCaches[i] = nil
	}
	m.BudgetField.Release(uint64(cap(m.builtin.userNames)) * uint64(reflect.TypeOf(util.Symbol{}).Size()))
	m.builtin.userNames = nil
	m.builtin.userTable.Close()
	m.plugins.close(m.BudgetField)
	m.releaseFormatPluginHeaders()
	m.releaseSequencePluginHeaders()
	m.releaseArrayPluginHeaders()
	m.releaseArithmeticPluginHeaders()
	m.releaseBitVectorPluginHeaders()
	if !m.sourceClosing {
		m.releaseFormatFrames()
	}
}
