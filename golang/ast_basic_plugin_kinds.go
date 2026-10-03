// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Exact src/ast/ast.h basic_op_kind enumeration (including sentinel gap).
package g3

const (
	ASTOpTrue int32 = iota
	ASTOpFalse
	ASTOpEq
	ASTOpDistinct
	ASTOpIte
	ASTOpAnd
	ASTOpOr
	ASTOpIff
	ASTOpXor
	ASTOpNot
	ASTOpImplies
	ASTOpOEq
	ASTOpInterp
	ASTLastBasicOp
	ASTPrUndef
	ASTPrTrue
	ASTPrAsserted
	ASTPrGoal
	ASTPrModusPonens
	ASTPrReflexivity
	ASTPrSymmetry
	ASTPrTransitivity
	ASTPrTransitivityStar
	ASTPrMonotonicity
	ASTPrQuantIntro
	ASTPrDistributivity
	ASTPrAndElim
	ASTPrNotOrElim
	ASTPrRewrite
	ASTPrRewriteStar
	ASTPrPullQuant
	ASTPrPushQuant
	ASTPrElimUnusedVars
	ASTPrDER
	ASTPrQuantInst
	ASTPrHypothesis
	ASTPrLemma
	ASTPrUnitResolution
	ASTPrIffTrue
	ASTPrIffFalse
	ASTPrCommutativity
	ASTPrDefAxiom
	ASTPrDefIntro
	ASTPrApplyDef
	ASTPrIffOEq
	ASTPrNNFPos
	ASTPrNNFNeg
	ASTPrSkolemize
	ASTPrModusPonensOEq
	ASTPrTheoryLemma
	ASTPrHyperResolve
	ASTLastBasicPR
)
