// Source: src/ast/fpa/fpa2bv_converter.cpp:214-228,2273-2300,3072-3122,3329-3349.
package fpa

import "github.com/glycerine/g3/pkg/ast"

// MkVar expands one floating-point bound variable into the three consecutive
// bit-vector variables used by the source quantifier conversion.
func (c *Converter) MkVar(base uint32, sort ast.ASTHandle) (ast.ASTHandle, error) {
	if err := c.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	ebits, sbits, matched, err := c.manager.FloatingPointSortShape(sort)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if !matched {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	signSort, err := c.manager.BitVectorSort(1)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	sigSort, err := c.manager.BitVectorSort(sbits - 1)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	expSort, err := c.manager.BitVectorSort(ebits)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	sign, err := c.manager.MkVar(base, signSort)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	significand, err := c.manager.MkVar(base+1, sigSort)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	exponent, err := c.manager.MkVar(base+2, expSort)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return c.MkFP(sign, exponent, significand)
}

// FromIEEEBV splits a packed IEEE bit-vector according to the requested
// floating-point sort.
func (c *Converter) FromIEEEBV(sort, bits ast.ASTHandle) (ast.ASTHandle, error) {
	if err := c.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	ebits, sbits, matched, err := c.manager.FloatingPointSortShape(sort)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if !matched {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	width, err := c.manager.BitVectorSize(bits)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if width != ebits+sbits {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	sign, err := c.manager.MkBitVectorExtract(width-1, width-1, bits)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	exponent, err := c.manager.MkBitVectorExtract(width-2, sbits-1, bits)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	significand, err := c.manager.MkBitVectorExtract(sbits-2, 0, bits)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return c.MkFP(sign, exponent, significand)
}

// ToIEEEBV packs an encoded value. With the source default high-unspecified
// setting, every NaN maps to the canonical NaN bit pattern.
func (c *Converter) ToIEEEBV(value ast.ASTHandle) (ast.ASTHandle, error) {
	if err := c.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	joined, err := c.JoinFP(value)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	nan, err := c.isNaN(value)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if !c.highUnspecified {
		// The low-unspecified branch introduces a fresh converted UF plus two
		// owned side constraints; that state is ported with UF conversion.
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	sort, err := c.manager.GetSort(value)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	canonical, err := c.MkSpecial(sort, NaN)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	canonicalBits, err := c.JoinFP(canonical)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return c.boolITE(nan, canonicalBits, joined)
}
