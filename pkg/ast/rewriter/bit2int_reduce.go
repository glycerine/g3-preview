// Source: src/ast/rewriter/bit2int.cpp:109-183.
package rewriter

import "github.com/glycerine/g3/pkg/ast"

type bit2IntComparison uint8

const (
	bit2IntLT bit2IntComparison = iota
	bit2IntLE
	bit2IntEQ
)

func (b *Bit2Int) holdCall(c *bit2IntCall, h ast.ASTHandle, err error) (ast.ASTHandle, error) {
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return c.hold(h)
}

func (b *Bit2Int) mkBV2Int(c *bit2IntCall, value ast.ASTHandle) (ast.ASTHandle, error) {
	h, err := b.manager.MkBitVectorToInt(value)
	return b.holdCall(c, h, err)
}

func (b *Bit2Int) mkArithmetic(c *bit2IntCall, kind int32, args ...ast.ASTHandle) (ast.ASTHandle, error) {
	h, err := b.manager.MkFamilyApp(b.arithFamily, kind, nil, args)
	return b.holdCall(c, h, err)
}

func (b *Bit2Int) mkAdd(c *bit2IntCall, left, right ast.ASTHandle) (ast.ASTHandle, bool, error) {
	l, lok, err := b.extractBV(c, left)
	if err != nil || !lok || l.sign {
		return ast.ASTHandle{}, false, err
	}
	r, rok, err := b.extractBV(c, right)
	if err != nil || !rok || r.sign {
		return ast.ASTHandle{}, false, err
	}
	zero, err := b.manager.IsBitVectorZero(l.bits)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	if zero {
		return right, true, nil
	}
	zero, err = b.manager.IsBitVectorZero(r.bits)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	if zero {
		return left, true, nil
	}
	a, d, err := b.alignSizes(c, l.bits, r.bits)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	width, err := b.manager.BitVectorSize(a)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	a, err = b.alignSize(c, a, width+1)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	d, err = b.alignSize(c, d, width+1)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	sum, err := b.manager.MkBitVectorAdd(a, d)
	sum, err = b.holdCall(c, sum, err)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	result, err := b.mkBV2Int(c, sum)
	return result, err == nil, err
}

func (b *Bit2Int) mkComp(c *bit2IntCall, mode bit2IntComparison, left, right ast.ASTHandle) (ast.ASTHandle, bool, error) {
	l, lok, err := b.extractBV(c, left)
	if err != nil || !lok || l.sign {
		return ast.ASTHandle{}, false, err
	}
	r, rok, err := b.extractBV(c, right)
	if err != nil || !rok || r.sign {
		return ast.ASTHandle{}, false, err
	}
	a, d, err := b.alignSizes(c, l.bits, r.bits)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	var result ast.ASTHandle
	switch mode {
	case bit2IntLT:
		le, err := b.manager.MkBitVectorULE(d, a)
		le, err = b.holdCall(c, le, err)
		if err != nil {
			return ast.ASTHandle{}, false, err
		}
		result, err = b.manager.MkNot(le)
		result, err = b.holdCall(c, result, err)
		if err != nil {
			return ast.ASTHandle{}, false, err
		}
	case bit2IntLE:
		result, err = b.manager.MkBitVectorULE(a, d)
		result, err = b.holdCall(c, result, err)
		if err != nil {
			return ast.ASTHandle{}, false, err
		}
	case bit2IntEQ:
		result, err = b.manager.MkEq(a, d)
		result, err = b.holdCall(c, result, err)
		if err != nil {
			return ast.ASTHandle{}, false, err
		}
	default:
		return ast.ASTHandle{}, false, ast.ErrASTDomain
	}
	return result, true, nil
}

func (b *Bit2Int) mkMul(c *bit2IntCall, left, right ast.ASTHandle) (ast.ASTHandle, bool, error) {
	l, lok, err := b.extractBV(c, left)
	if err != nil || !lok {
		return ast.ASTHandle{}, false, err
	}
	r, rok, err := b.extractBV(c, right)
	if err != nil || !rok {
		return ast.ASTHandle{}, false, err
	}
	a, d, err := b.alignSizes(c, l.bits, r.bits)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	width, err := b.manager.BitVectorSize(a)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	a, err = b.alignSize(c, a, width*2)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	d, err = b.alignSize(c, d, width*2)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	product, err := b.manager.MkBitVectorMul(a, d)
	product, err = b.holdCall(c, product, err)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	result, err := b.mkBV2Int(c, product)
	if err != nil {
		return ast.ASTHandle{}, false, err
	}
	if l.sign != r.sign {
		result, err = b.mkArithmetic(c, 8, result)
		if err != nil {
			return ast.ASTHandle{}, false, err
		}
	}
	return result, true, nil
}
