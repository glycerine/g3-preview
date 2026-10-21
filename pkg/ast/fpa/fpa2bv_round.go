// Source: src/ast/fpa/fpa2bv_converter.cpp:3744-3791.
package fpa

import "github.com/glycerine/g3/pkg/ast"

func (c *Converter) roundingModeIs(bits ast.ASTHandle, code uint32) (ast.ASTHandle, error) {
	value, err := c.manager.BitVectorNumeralUnsigned(code, 3)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return c.boolEq(bits, value)
}

func (c *Converter) roundingDecisionBits(rm, sign, last, round, sticky ast.ASTHandle) (ast.ASTHandle, error) {
	for _, field := range []struct {
		value ast.ASTHandle
		width uint32
	}{{rm, 3}, {sign, 1}, {last, 1}, {round, 1}, {sticky, 1}} {
		width, err := c.manager.BitVectorSize(field.value)
		if err != nil || width != field.width {
			return ast.ASTHandle{}, ast.ErrASTDomain
		}
	}
	lastOrSticky, err := c.manager.MkBitVectorOr([]ast.ASTHandle{last, sticky})
	if err != nil {
		return ast.ASTHandle{}, err
	}
	roundOrSticky, err := c.manager.MkBitVectorOr([]ast.ASTHandle{round, sticky})
	if err != nil {
		return ast.ASTHandle{}, err
	}
	notRound, err := c.manager.MkBitVectorNot(round)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	notLastOrSticky, err := c.manager.MkBitVectorNot(lastOrSticky)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	notRoundOrSticky, err := c.manager.MkBitVectorNot(roundOrSticky)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	notSign, err := c.manager.MkBitVectorNot(sign)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	evenRefusal, err := c.manager.MkBitVectorOr([]ast.ASTHandle{notRound, notLastOrSticky})
	if err != nil {
		return ast.ASTHandle{}, err
	}
	incrementEven, err := c.manager.MkBitVectorNot(evenRefusal)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	positiveRefusal, err := c.manager.MkBitVectorOr([]ast.ASTHandle{sign, notRoundOrSticky})
	if err != nil {
		return ast.ASTHandle{}, err
	}
	incrementPositive, err := c.manager.MkBitVectorNot(positiveRefusal)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	negativeRefusal, err := c.manager.MkBitVectorOr([]ast.ASTHandle{notSign, notRoundOrSticky})
	if err != nil {
		return ast.ASTHandle{}, err
	}
	incrementNegative, err := c.manager.MkBitVectorNot(negativeRefusal)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	zero, err := c.zero(1)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	isNegative, err := c.roundingModeIs(rm, 3)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	result, err := c.boolITE(isNegative, incrementNegative, zero)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	isPositive, err := c.roundingModeIs(rm, 2)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	result, err = c.boolITE(isPositive, incrementPositive, result)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	isAway, err := c.roundingModeIs(rm, 1)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	result, err = c.boolITE(isAway, round, result)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	isEven, err := c.roundingModeIs(rm, 0)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return c.boolITE(isEven, incrementEven, result)
}

// RoundingDecision returns the source rounder's one-bit increment decision.
// The rounding mode is the FPA BV2RM wrapper used by converter expressions.
func (c *Converter) RoundingDecision(roundingMode, sign, last, round, sticky ast.ASTHandle) (ast.ASTHandle, error) {
	if err := c.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	rm, err := c.splitRoundingMode(roundingMode)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return c.roundingDecisionBits(rm, sign, last, round, sticky)
}
