// Source: src/ast/rewriter/poly_rewriter_def.h:708-883.
package rewriter

import (
	"reflect"

	"github.com/glycerine/g3/pkg/ast"
)

// One cancellation scope borrows both inputs through source destruction. Its
// raw output buffers acquire no references; the persistent position map is
// borrowed from the polynomial owner and survives this scope.
type bvPolynomialCancellation struct {
	self                            *bvPolynomialCancellation
	manager                         *ast.ASTManager
	positions                       *bvPolynomialPositions
	numeric                         *ast.BitVectorPolynomialAdditionNumerals
	visited                         *ast.ASTFastMark1
	multiple                        *ast.ASTFastMark2
	pending                         [2]ast.ASTHandle
	sides, pair                     [2]ast.ASTHandle
	monomials                       [2]ast.AppArguments
	lhs, rhs                        ast.ASTHandleBuffer
	factory                         ast.ASTBitVectorNumeralInvocation
	header                          uint64
	bodyCause, sourceCause          error
	lhsAssigned, sourceDone, closed bool
}

func newBVPolynomialCancellation(m *ast.ASTManager, table *bvPolynomialPositions) (*bvPolynomialCancellation, error) {
	bytes := uint64(reflect.TypeOf(bvPolynomialCancellation{}).Size())
	if e := m.Budget().Reserve(bytes); e != nil {
		return nil, e
	}
	p := &bvPolynomialCancellation{manager: m, positions: table, header: bytes}
	p.self = p
	return p, nil
}
func (p *bvPolynomialCancellation) getMonomials(i int, h ast.ASTHandle) error {
	p.sides[i] = h
	yes, e := p.manager.IsBitVectorOperation(h, 4)
	if e != nil {
		return e
	}
	if yes {
		p.monomials[i], e = p.manager.AppArgumentView(h)
	} else {
		p.monomials[i] = ast.SliceArguments(p.sides[i : i+1])
	}
	return e
}
func (p *bvPolynomialCancellation) mark(i int, h ast.ASTHandle) error {
	p.pending[i] = h
	var e error
	if i == 0 {
		e = p.visited.Mark(h, true)
	} else {
		e = p.multiple.Mark(h, true)
	}
	if e == nil {
		p.pending[i] = ast.ASTHandle{}
	}
	return e
}
func (p *bvPolynomialCancellation) isNumeral(h ast.ASTHandle) (bool, error) {
	return p.manager.IsBitVectorOperation(h, 0)
}
func (p *bvPolynomialCancellation) powerProduct(h ast.ASTHandle) (ast.ASTHandle, error) {
	return bvPowerProduct(p.manager, h)
}
func (p *bvPolynomialCancellation) position(h ast.ASTHandle) (uint32, error) {
	pos, found, e := p.positions.find(h)
	if e != nil {
		return 0, e
	}
	// The original DEBUG assertion precedes the subsequent coeffs[pos] access.
	if !found {
		return 0, ast.ErrASTDomain
	}
	return pos, nil
}
func (p *bvPolynomialCancellation) body(sort ast.ASTHandle, move bool, o BVPolynomialConfigurationState, lo, ro ast.ASTRef) (RewriteStatus, error) {
	m := p.manager
	var e error
	if e = p.getMonomials(0, p.sides[0]); e != nil {
		return RewriteFailed, e
	}
	if e = p.getMonomials(1, p.sides[1]); e != nil {
		return RewriteFailed, e
	}
	p.visited, e = ast.NewASTFastMark1(m)
	if e != nil {
		return RewriteFailed, e
	}
	p.multiple, e = ast.NewASTFastMark2(m)
	if e != nil {
		return RewriteFailed, e
	}
	p.numeric, e = ast.NewBitVectorPolynomialAdditionNumerals(m)
	if e != nil {
		return RewriteFailed, e
	}
	if e = p.numeric.BeginCancellation(); e != nil {
		return RewriteFailed, e
	}
	n := p.numeric
	numCoeffs := uint32(0)
	hasMultiple := false
	for i := uint32(0); i < p.monomials[0].Count(); i++ {
		arg, e := p.monomials[0].At(i)
		if e != nil {
			return RewriteFailed, e
		}
		yes, e := n.Read(arg)
		if e != nil {
			return RewriteFailed, e
		}
		if yes {
			if e = n.AddConstant(); e != nil {
				return RewriteFailed, e
			}
			numCoeffs++
		} else {
			pp, e := p.powerProduct(arg)
			if e != nil {
				return RewriteFailed, e
			}
			if e = p.mark(0, pp); e != nil {
				return RewriteFailed, e
			}
		}
	}
	if move && numCoeffs == 0 {
		yes, e := p.isNumeral(p.sides[1])
		if e != nil {
			return RewriteFailed, e
		}
		if yes {
			return RewriteFailed, nil
		}
	}
	for i := uint32(0); i < p.monomials[1].Count(); i++ {
		arg, e := p.monomials[1].At(i)
		if e != nil {
			return RewriteFailed, e
		}
		yes, e := n.Read(arg)
		if e != nil {
			return RewriteFailed, e
		}
		if yes {
			if e = n.SubtractConstant(); e != nil {
				return RewriteFailed, e
			}
			numCoeffs++
		} else {
			pp, e := p.powerProduct(arg)
			if e != nil {
				return RewriteFailed, e
			}
			marked, e := p.visited.IsMarked(pp)
			if e != nil {
				return RewriteFailed, e
			}
			if marked {
				if e = p.mark(1, pp); e != nil {
					return RewriteFailed, e
				}
				hasMultiple = true
			}
		}
	}
	if e = n.Normalize(sort, false); e != nil {
		return RewriteFailed, e
	}
	if !hasMultiple && numCoeffs <= 1 {
		if !move && numCoeffs == 0 {
			return RewriteFailed, nil
		}
		yes, e := p.isNumeral(p.sides[1])
		if e != nil {
			return RewriteFailed, e
		}
		if yes {
			return RewriteFailed, nil
		}
	}
	if e = n.BeginCoefficients(); e != nil {
		return RewriteFailed, e
	}
	if e = p.positions.reset(); e != nil {
		return RewriteFailed, e
	}
	for side := 0; side < 2; side++ {
		for i := uint32(0); i < p.monomials[side].Count(); i++ {
			arg, e := p.monomials[side].At(i)
			if e != nil {
				return RewriteFailed, e
			}
			yes, e := p.isNumeral(arg)
			if e != nil {
				return RewriteFailed, e
			}
			if yes {
				continue
			}
			pp, e := n.PowerProduct(arg)
			if e != nil {
				return RewriteFailed, e
			}
			marked, e := p.multiple.IsMarked(pp)
			if e != nil {
				return RewriteFailed, e
			}
			if !marked {
				continue
			}
			if side == 0 {
				pos, found, e := p.positions.find(pp)
				if e != nil {
					return RewriteFailed, e
				}
				if found {
					if e = n.AccumulateCoefficient(pos); e != nil {
						return RewriteFailed, e
					}
				} else {
					count, e := n.CoefficientCount()
					if e != nil {
						return RewriteFailed, e
					}
					if e = p.positions.insert(pp, count); e != nil {
						return RewriteFailed, e
					}
					if e = n.AppendCoefficient(); e != nil {
						return RewriteFailed, e
					}
				}
			} else {
				pos, e := p.position(pp)
				if e != nil {
					return RewriteFailed, e
				}
				if e = n.SubtractCoefficient(pos); e != nil {
					return RewriteFailed, e
				}
			}
		}
	}
	if e = p.lhs.Push(m.Budget(), ast.ASTHandle{}); e != nil {
		return RewriteFailed, e
	}
	if e = p.visited.Reset(); e != nil {
		return RewriteFailed, e
	}
	for i := uint32(0); i < p.monomials[0].Count(); i++ {
		arg, e := p.monomials[0].At(i)
		if e != nil {
			return RewriteFailed, e
		}
		yes, e := p.isNumeral(arg)
		if e != nil {
			return RewriteFailed, e
		}
		if yes {
			continue
		}
		pp, e := p.powerProduct(arg)
		if e != nil {
			return RewriteFailed, e
		}
		marked, e := p.multiple.IsMarked(pp)
		if e != nil {
			return RewriteFailed, e
		}
		if !marked {
			if e = p.lhs.Push(m.Budget(), arg); e != nil {
				return RewriteFailed, e
			}
		} else {
			seen, e := p.visited.IsMarked(pp)
			if e != nil {
				return RewriteFailed, e
			}
			if seen {
				continue
			}
			if e = p.mark(0, pp); e != nil {
				return RewriteFailed, e
			}
			pos, e := p.position(pp)
			if e != nil {
				return RewriteFailed, e
			}
			if e = n.LoadCoefficient(pos); e != nil {
				return RewriteFailed, e
			}
			zero, e := n.Zero(true)
			if e != nil {
				return RewriteFailed, e
			}
			if !zero {
				node, e := n.Product(sort, pp)
				if e != nil {
					return RewriteFailed, e
				}
				if e = p.lhs.Push(m.Budget(), node); e != nil {
					return RewriteFailed, e
				}
			}
		}
	}
	if e = p.rhs.Push(m.Budget(), ast.ASTHandle{}); e != nil {
		return RewriteFailed, e
	}
	for i := uint32(0); i < p.monomials[1].Count(); i++ {
		arg, e := p.monomials[1].At(i)
		if e != nil {
			return RewriteFailed, e
		}
		yes, e := p.isNumeral(arg)
		if e != nil {
			return RewriteFailed, e
		}
		if yes {
			continue
		}
		pp, e := n.PowerProduct(arg)
		if e != nil {
			return RewriteFailed, e
		}
		marked, e := p.multiple.IsMarked(pp)
		if e != nil {
			return RewriteFailed, e
		}
		if marked {
			continue
		}
		if !move {
			if e = p.rhs.Push(m.Budget(), arg); e != nil {
				return RewriteFailed, e
			}
			continue
		}
		zero, e := n.Zero(true)
		if e != nil {
			return RewriteFailed, e
		}
		if zero {
			continue
		}
		minus, e := n.CoefficientIsMinusOne()
		if e != nil {
			return RewriteFailed, e
		}
		if minus {
			if e = p.lhs.Push(m.Budget(), pp); e != nil {
				return RewriteFailed, e
			}
			continue
		}
		if e = n.Negate(true); e != nil {
			return RewriteFailed, e
		}
		if ast.ASTSourceDebug {
			one, e := n.CoefficientIsOne()
			if e != nil {
				return RewriteFailed, e
			}
			if one {
				return RewriteFailed, ast.ErrASTDomain
			}
		}
		p.pair[0], e = n.CoefficientConstant(sort)
		if e != nil {
			return RewriteFailed, e
		}
		p.pair[1] = pp
		node, e := m.MkBitVectorPolynomialMulWithInvocation(sort, ast.SliceArguments(p.pair[:]), &p.factory)
		if e != nil {
			return RewriteFailed, e
		}
		if e = p.lhs.Push(m.Budget(), node); e != nil {
			return RewriteFailed, e
		}
	}
	atRHS := false
	if move {
		if o.SortSums {
			fid, e := m.GetFamilyID(sort)
			if e != nil {
				return RewriteFailed, e
			}
			if e = bvPolynomialSortMonomials(m, fid, o.ASTOrder, p.lhs.Slice()[1:]); e != nil {
				return RewriteFailed, e
			}
		}
		atRHS = true
	} else if len(p.rhs.Slice()) == 1 || len(p.lhs.Slice()) > 1 {
		atRHS = true
	}
	if atRHS {
		if e = n.Negate(false); e != nil {
			return RewriteFailed, e
		}
		if e = n.Normalize(sort, false); e != nil {
			return RewriteFailed, e
		}
	}
	// Preserve short-circuit checks: an empty side inserts c without is_zero.
	insertL := false
	insertR := false
	if !atRHS {
		insertL = len(p.lhs.Slice()) == 1
		if !insertL {
			zero, e := n.Zero(false)
			if e != nil {
				return RewriteFailed, e
			}
			insertL = !zero
		}
	} else {
		insertR = len(p.rhs.Slice()) == 1
		if !insertR {
			zero, e := n.Zero(false)
			if e != nil {
				return RewriteFailed, e
			}
			insertR = !zero
		}
	}
	loffset, roffset := 1, 1
	p.rhs.Slice()[0] = ast.ASTHandle{}
	if insertR {
		p.rhs.Slice()[0], e = n.Constant(sort)
		if e != nil {
			return RewriteFailed, e
		}
		roffset = 0
	}
	p.lhs.Slice()[0] = ast.ASTHandle{}
	if insertL {
		p.lhs.Slice()[0], e = n.Constant(sort)
		if e != nil {
			return RewriteFailed, e
		}
		loffset = 0
	}
	node, e := m.MkBitVectorPolynomialAddWithInvocation(sort, ast.SliceArguments(p.lhs.Slice()[loffset:]), &p.factory)
	if e != nil {
		return RewriteFailed, e
	}
	if e = m.SetRef(lo, node); e != nil {
		return RewriteFailed, e
	}
	p.lhsAssigned = true
	node, e = m.MkBitVectorPolynomialAddWithInvocation(sort, ast.SliceArguments(p.rhs.Slice()[roffset:]), &p.factory)
	if e != nil {
		return RewriteFailed, e
	}
	if e = m.SetRef(ro, node); e != nil {
		return RewriteFailed, e
	}
	return RewriteDone, nil
}
func (p *bvPolynomialCancellation) childrenSource() error {
	if p.numeric != nil {
		if e := p.numeric.SourceChildren(); e != nil {
			return e
		}
	}
	return p.manager.BitVectorNumeralInvocationSourceReady(&p.factory)
}
func (p *bvPolynomialCancellation) sourceEnd() error {
	if e := p.childrenSource(); e != nil {
		return e
	}
	p.rhs.Close(p.manager.Budget())
	p.lhs.Close(p.manager.Budget())
	if p.numeric != nil {
		if e := p.numeric.EndCoefficients(); e != nil {
			return e
		}
		if e := p.numeric.SourceEnd(); e != nil {
			return e
		}
	}
	if p.multiple != nil {
		if e := p.multiple.Close(); e != nil {
			return e
		}
		p.multiple = nil
	}
	if p.visited != nil {
		if e := p.visited.Close(); e != nil {
			return e
		}
		p.visited = nil
	}
	p.sourceDone = true
	return nil
}
func (p *bvPolynomialCancellation) rewrite(sort ast.ASTHandle, move bool, o BVPolynomialConfigurationState, lo, ro ast.ASTRef) (RewriteStatus, error) {
	st, e := p.body(sort, move, o, lo, ro)
	p.bodyCause = e
	if e != nil {
		p.sourceCause = p.childrenSource()
		return st, e
	}
	p.sourceCause = p.sourceEnd()
	return st, p.sourceCause
}
func (p *bvPolynomialCancellation) Close() error {
	if p == nil || p.self != p {
		return ast.ErrASTHandle
	}
	if p.closed {
		return nil
	}
	if e := p.manager.CleanupBitVectorNumeralInvocation(&p.factory); e != nil {
		return e
	}
	p.rhs.Close(p.manager.Budget())
	p.lhs.Close(p.manager.Budget())
	if p.numeric != nil {
		if e := p.numeric.Close(); e != nil {
			return e
		}
	}
	if !p.pending[1].IsNull() && p.multiple != nil {
		if e := p.multiple.ResetMark(p.pending[1]); e != nil {
			return e
		}
		p.pending[1] = ast.ASTHandle{}
	}
	if !p.pending[0].IsNull() && p.visited != nil {
		if e := p.visited.ResetMark(p.pending[0]); e != nil {
			return e
		}
		p.pending[0] = ast.ASTHandle{}
	}
	if p.multiple != nil {
		if e := p.multiple.Close(); e != nil {
			return e
		}
	}
	if p.visited != nil {
		if e := p.visited.Close(); e != nil {
			return e
		}
	}
	p.manager.Budget().Release(p.header)
	p.header = 0
	p.closed = true
	return nil
}
func (p *BVPolynomialRewriter) CancelMonomials(lhs, rhs ast.ASTHandle, move bool, lo, ro ast.ASTRef) (st RewriteStatus, err error) {
	if e := p.open(); e != nil {
		return RewriteFailed, e
	}
	p.running = true
	defer func() {
		p.running = false
		if err != nil {
			p.fail(err)
		}
	}()
	sort, e := p.manager.GetSort(lhs)
	if e != nil {
		return RewriteFailed, e
	}
	p.currentSort = sort
	o, e := p.config.State()
	if e != nil {
		return RewriteFailed, e
	}
	n, e := newBVPolynomialCancellation(p.manager, p.positions)
	if e != nil {
		return RewriteFailed, e
	}
	p.cancellation = n
	n.sides = [2]ast.ASTHandle{lhs, rhs}
	st, e = n.rewrite(p.currentSort, move, o, lo, ro)
	if e != nil {
		return st, e
	}
	if e = n.Close(); e != nil {
		return st, e
	}
	p.cancellation = nil
	return st, nil
}
