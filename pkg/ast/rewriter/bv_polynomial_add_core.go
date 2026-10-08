// Source: src/ast/rewriter/poly_rewriter.h:32,62;
// src/ast/rewriter/poly_rewriter_def.h:527-660.
package rewriter

import (
	"reflect"

	"github.com/glycerine/g3/pkg/ast"
)

type BVPolynomialAdditionOptions struct{ SortSums, ASTOrder, HoistCoefficients, HoistMultiplication bool }

// BVPolynomialAdditionCore owns the selected nonflat addition invocation.
// Inputs borrow caller lifetimes through source exit, including mark destruction.
type BVPolynomialAdditionCore struct {
	pendingMarks                         [2]ast.ASTHandle
	flatArgs                             ast.ASTHandleBuffer
	currentSort                          ast.ASTHandle
	localsEnded                          bool
	family                               ast.FamilyID
	savedVector                          ast.ASTRefVector
	self                                 *BVPolynomialAdditionCore
	manager                              *ast.ASTManager
	positions                            *bvPolynomialPositions
	numeric                              *ast.BitVectorPolynomialAdditionNumerals
	visited                              *ast.ASTFastMark1
	multiple                             *ast.ASTFastMark2
	vector                               ast.ASTRefVector
	vectorPlaced                         bool
	push                                 ast.ASTRefVectorPushReceipt
	pushed                               ast.ASTHandle
	cmul                                 *ast.BitVectorCoefficientHoist
	mul                                  *BVPolynomialMultiplicationHoist
	factory                              ast.ASTBitVectorNumeralInvocation
	header                               uint64
	started, running, sourceDone, closed bool
	bodyCause, sourceCause               error
}

func NewBVPolynomialAdditionCore(m *ast.ASTManager) (*BVPolynomialAdditionCore, error) {
	if _, e := m.ProofsEnabled(); e != nil {
		return nil, e
	}
	header := uint64(reflect.TypeOf(BVPolynomialAdditionCore{}).Size())
	if e := m.Budget().Reserve(header); e != nil {
		return nil, e
	}
	table, e := newBVPolynomialPositions(m)
	if e != nil {
		m.Budget().Release(header)
		return nil, e
	}
	p := &BVPolynomialAdditionCore{manager: m, positions: table, header: header}
	p.self = p
	return p, nil
}

// A source mark sets its bit before the buffer append. Keep the reached
// candidate when append fails so owning cleanup can consume that real prefix.
func (p *BVPolynomialAdditionCore) mark(plane int, h ast.ASTHandle) error {
	p.pendingMarks[plane] = h
	var e error
	if plane == 0 {
		e = p.visited.Mark(h, true)
	} else {
		e = p.multiple.Mark(h, true)
	}
	if e == nil {
		p.pendingMarks[plane] = ast.ASTHandle{}
	}
	return e
}
func (p *BVPolynomialAdditionCore) append(h ast.ASTHandle) error {
	p.pushed = h
	e := p.manager.RefVectorPushTracked(p.vector, h, &p.push)
	if e == nil {
		p.push = ast.ASTRefVectorPushReceipt{}
	}
	return e
}
func (p *BVPolynomialAdditionCore) less(o BVPolynomialAdditionOptions, a, b uint32) (bool, error) {
	x, e := p.manager.RefVectorGet(p.vector, a)
	if e != nil {
		return false, e
	}
	y, e := p.manager.RefVectorGet(p.vector, b)
	if e != nil {
		return false, e
	}
	return bvMonomialLess(p.manager, p.family, o.ASTOrder, x, y)
}
func (p *BVPolynomialAdditionCore) sift(o BVPolynomialAdditionOptions, start, root, end uint32) error {
	for root < end/2 {
		child := root*2 + 1
		if child+1 < end {
			yes, e := p.less(o, start+child, start+child+1)
			if e != nil {
				return e
			}
			if yes {
				child++
			}
		}
		yes, e := p.less(o, start+root, start+child)
		if e != nil {
			return e
		}
		if !yes {
			return nil
		}
		if e = p.manager.RefVectorSwapElements(p.vector, start+root, start+child); e != nil {
			return e
		}
		root = child
	}
	return nil
}

// std::sort's comparator/output contract; C++ library callback order is uncredited.
func (p *BVPolynomialAdditionCore) sort(o BVPolynomialAdditionOptions, start uint32) error {
	n, e := p.manager.RefVectorSize(p.vector)
	if e != nil {
		return e
	}
	n -= start
	for i := n / 2; i > 0; i-- {
		if e = p.sift(o, start, i-1, n); e != nil {
			return e
		}
	}
	for end := n; end > 1; end-- {
		if e = p.manager.RefVectorSwapElements(p.vector, start, start+end-1); e != nil {
			return e
		}
		if e = p.sift(o, start, 0, end-1); e != nil {
			return e
		}
	}
	return nil
}
func (p *BVPolynomialAdditionCore) body(sort ast.ASTHandle, args ast.AppArguments, out ast.ASTRef, o BVPolynomialAdditionOptions) (RewriteStatus, error) {
	m := p.manager
	if args.Count() < 2 {
		return RewriteFailed, ast.ErrASTDomain
	}
	var e error
	p.numeric, e = ast.NewBitVectorPolynomialAdditionNumerals(m)
	if e != nil {
		return RewriteFailed, e
	}
	if e = p.numeric.Begin(); e != nil {
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
	numCoeffs := uint32(0)
	hasMultiple, ordered := false, true
	var prev ast.ASTHandle
	fid, e := m.GetFamilyID(sort)
	if e != nil {
		return RewriteFailed, e
	}
	p.family = fid
	for i := uint32(0); i < args.Count(); i++ {
		arg, e := args.At(i)
		if e != nil {
			return RewriteFailed, e
		}
		numeral, e := p.numeric.Read(arg)
		if e != nil {
			return RewriteFailed, e
		}
		if numeral {
			numCoeffs++
			if e = p.numeric.AddConstant(); e != nil {
				return RewriteFailed, e
			}
			ordered = !o.SortSums || i == 0
		} else if o.SortSums && ordered {
			if prev != (ast.ASTHandle{}) {
				less, e := bvMonomialLess(m, fid, o.ASTOrder, arg, prev)
				if e != nil {
					return RewriteFailed, e
				}
				if less {
					ordered = false
				}
			}
			prev = arg
		}
		pp, e := bvPowerProduct(m, arg)
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
		} else {
			if e = p.mark(0, pp); e != nil {
				return RewriteFailed, e
			}
		}
	}
	if e = p.numeric.Normalize(sort, false); e != nil {
		return RewriteFailed, e
	}
	if hasMultiple {
		if e = p.numeric.BeginCoefficients(); e != nil {
			return RewriteFailed, e
		}
		if e = p.positions.reset(); e != nil {
			return RewriteFailed, e
		}
		for i := uint32(0); i < args.Count(); i++ {
			arg, e := args.At(i)
			if e != nil {
				return RewriteFailed, e
			}
			numeral, e := m.IsBitVectorNumeral(arg)
			if e != nil {
				return RewriteFailed, e
			}
			if numeral {
				continue
			}
			pp, e := p.numeric.PowerProduct(arg)
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
			pos, found, e := p.positions.find(pp)
			if e != nil {
				return RewriteFailed, e
			}
			if found {
				if e = p.numeric.AccumulateCoefficient(pos); e != nil {
					return RewriteFailed, e
				}
			} else {
				count, e := p.numeric.CoefficientCount()
				if e != nil {
					return RewriteFailed, e
				}
				if e = p.positions.insert(pp, count); e != nil {
					return RewriteFailed, e
				}
				if e = p.numeric.AppendCoefficient(); e != nil {
					return RewriteFailed, e
				}
			}
		}
	} else if ordered && !o.HoistMultiplication && !o.HoistCoefficients {
		if numCoeffs == 0 {
			return RewriteFailed, nil
		}
		if numCoeffs == 1 {
			first, e := args.At(0)
			if e != nil {
				return RewriteFailed, e
			}
			numeral, e := p.numeric.Read(first)
			if e != nil {
				return RewriteFailed, e
			}
			if numeral {
				zero, e := p.numeric.Zero(true)
				if e != nil {
					return RewriteFailed, e
				}
				if !zero {
					return RewriteFailed, nil
				}
			}
		}
	}
	p.vector, e = m.NewRefVector()
	if e != nil {
		return RewriteFailed, e
	}
	p.vectorPlaced = true
	p.savedVector = p.vector
	zero, e := p.numeric.Zero(false)
	if e != nil {
		return RewriteFailed, e
	}
	if !zero {
		constant, e := p.numeric.Constant(sort)
		if e != nil {
			return RewriteFailed, e
		}
		if e = p.append(constant); e != nil {
			return RewriteFailed, e
		}
	}
	if hasMultiple {
		if e = p.visited.Reset(); e != nil {
			return RewriteFailed, e
		}
	}
	for i := uint32(0); i < args.Count(); i++ {
		arg, e := args.At(i)
		if e != nil {
			return RewriteFailed, e
		}
		numeral, e := m.IsBitVectorNumeral(arg)
		if e != nil {
			return RewriteFailed, e
		}
		if numeral {
			continue
		}
		if !hasMultiple {
			if e = p.append(arg); e != nil {
				return RewriteFailed, e
			}
			continue
		}
		pp, e := bvPowerProduct(m, arg)
		if e != nil {
			return RewriteFailed, e
		}
		marked, e := p.multiple.IsMarked(pp)
		if e != nil {
			return RewriteFailed, e
		}
		if !marked {
			if e = p.append(arg); e != nil {
				return RewriteFailed, e
			}
			continue
		}
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
		pos, found, e := p.positions.find(pp)
		if e != nil {
			return RewriteFailed, e
		}
		if !found {
			return RewriteFailed, ast.ErrASTDomain
		}
		if e = p.numeric.LoadCoefficient(pos); e != nil {
			return RewriteFailed, e
		}
		if e = p.numeric.Normalize(sort, true); e != nil {
			return RewriteFailed, e
		}
		cancelled, e := p.numeric.Zero(true)
		if e != nil {
			return RewriteFailed, e
		}
		if !cancelled {
			product, e := p.numeric.Product(sort, pp)
			if e != nil {
				return RewriteFailed, e
			}
			if e = p.append(product); e != nil {
				return RewriteFailed, e
			}
		}
	}
	if o.HoistCoefficients {
		p.cmul, e = ast.NewBitVectorCoefficientHoist(m)
		if e != nil {
			return RewriteFailed, e
		}
		if e = p.cmul.Hoist(sort, p.vector); e != nil {
			return RewriteFailed, e
		}
	} else if (hasMultiple && o.SortSums) || (!hasMultiple && !ordered) {
		start := uint32(0)
		if !zero {
			start = 1
		}
		if e = p.sort(o, start); e != nil {
			return RewriteFailed, e
		}
	}
	size, e := m.RefVectorSize(p.vector)
	if e != nil {
		return RewriteFailed, e
	}
	view, e := m.RefVectorArgumentsRange(p.vector, 0, size)
	if e != nil {
		return RewriteFailed, e
	}
	node, e := m.MkBitVectorPolynomialAddWithInvocation(sort, view, &p.factory)
	if e != nil {
		return RewriteFailed, e
	}
	if e = m.SetRef(out, node); e != nil {
		return RewriteFailed, e
	}
	if o.HoistMultiplication {
		p.mul, e = NewBVPolynomialMultiplicationHoist(m)
		if e != nil {
			return RewriteFailed, e
		}
		changed, e := p.mul.Hoist(true, out)
		if e != nil {
			return RewriteFailed, e
		}
		if changed {
			return RewriteFull, nil
		}
	}
	return RewriteDone, nil
}
func (p *BVPolynomialAdditionCore) childrenSource() error {
	if p.numeric != nil {
		if e := p.numeric.SourceChildren(); e != nil {
			return e
		}
	}
	if p.cmul != nil {
		if e := p.cmul.SourceReady(); e != nil {
			return e
		}
	}
	if p.mul != nil {
		if e := p.mul.SourceReady(); e != nil {
			return e
		}
	}
	return p.manager.BitVectorNumeralInvocationSourceReady(&p.factory)
}
func (p *BVPolynomialAdditionCore) sourceEnd() error {
	if p.localsEnded {
		return p.childrenSource()
	}
	if e := p.childrenSource(); e != nil {
		return e
	}
	if p.vectorPlaced {
		if e := p.manager.DelRefVector(&p.vector); e != nil {
			return e
		}
		p.vectorPlaced = false
	}
	if p.numeric != nil {
		if e := p.numeric.EndCoefficients(); e != nil {
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
	if p.numeric != nil {
		if e := p.numeric.SourceEnd(); e != nil {
			return e
		}
	}
	p.localsEnded = true
	return nil
}

// Rewrite is the original mk_nflat_add_core entry with the caller's current sort.
func (p *BVPolynomialAdditionCore) Rewrite(sort ast.ASTHandle, args ast.AppArguments, out ast.ASTRef, o BVPolynomialAdditionOptions) (RewriteStatus, error) {
	return p.rewrite(sort, args, out, o, false, false)
}

// RewriteFlat specializes mk_flat_add_core. Expansion is exactly depth one.
// Source: src/ast/rewriter/poly_rewriter_def.h:371-400.
func (p *BVPolynomialAdditionCore) RewriteFlat(sort ast.ASTHandle, args ast.AppArguments, out ast.ASTRef, o BVPolynomialAdditionOptions) (RewriteStatus, error) {
	return p.rewrite(sort, args, out, o, true, false)
}

// RewriteApplication is mk_add_core: a singleton assigns before sort lookup;
// otherwise the physical current sort comes from the first argument.
// Source: src/ast/rewriter/poly_rewriter.h:137-148.
func (p *BVPolynomialAdditionCore) RewriteApplication(args ast.AppArguments, out ast.ASTRef, flat bool, o BVPolynomialAdditionOptions) (RewriteStatus, error) {
	return p.rewrite(ast.ASTHandle{}, args, out, o, flat, true)
}
func (p *BVPolynomialAdditionCore) rewrite(sort ast.ASTHandle, args ast.AppArguments, out ast.ASTRef, o BVPolynomialAdditionOptions, flat, application bool) (RewriteStatus, error) {
	if p == nil || p.self != p || p.closed {
		return RewriteFailed, ast.ErrASTHandle
	}
	if p.started || p.running {
		return RewriteFailed, ast.ErrASTPending
	}
	p.started, p.running = true, true
	defer func() { p.running = false }()
	st, e := p.dispatch(sort, args, out, o, flat, application)
	p.bodyCause = e
	if e != nil {
		p.sourceCause = p.childrenSource()
		return st, e
	}
	p.sourceCause = p.sourceEnd()
	if p.sourceCause == nil {
		p.flatArgs.Close(p.manager.Budget())
		p.sourceDone = true
	}
	return st, p.sourceCause
}
func (p *BVPolynomialAdditionCore) dispatch(sort ast.ASTHandle, args ast.AppArguments, out ast.ASTRef, o BVPolynomialAdditionOptions, flat, application bool) (RewriteStatus, error) {
	if application {
		if args.Count() == 0 {
			return RewriteFailed, ast.ErrASTDomain
		}
		first, e := args.At(0)
		if e != nil {
			return RewriteFailed, e
		}
		if args.Count() == 1 {
			return RewriteDone, p.manager.SetRef(out, first)
		}
		sort, e = p.manager.GetSort(first)
		if e != nil {
			return RewriteFailed, e
		}
	}
	p.currentSort = sort
	if !flat {
		return p.body(sort, args, out, o)
	}
	return p.flatBody(sort, args, out, o)
}
func (p *BVPolynomialAdditionCore) SourceReady() error {
	if p == nil || p.self != p {
		return ast.ErrASTHandle
	}
	if p.sourceCause != nil {
		return p.sourceCause
	}
	if p.started && !p.sourceDone {
		return ast.ErrASTPending
	}
	return nil
}
func (p *BVPolynomialAdditionCore) Close() error {
	if p == nil || p.self != p {
		return ast.ErrASTHandle
	}
	if p.closed {
		return nil
	}
	if p.running {
		return ast.ErrASTPending
	}
	if p.mul != nil {
		if e := p.mul.Close(); e != nil {
			return e
		}
	}
	if p.cmul != nil {
		if e := p.cmul.Close(); e != nil {
			return e
		}
	}
	if e := p.manager.CleanupBitVectorNumeralInvocation(&p.factory); e != nil {
		return e
	}
	if p.vectorPlaced {
		if e := p.manager.DelRefVector(&p.vector); e != nil {
			return e
		}
		p.vectorPlaced = false
	}
	pending, e := p.push.Unpublished()
	if e != nil {
		return e
	}
	if pending {
		if e = p.manager.CompleteRefVectorPush(p.savedVector, p.pushed, &p.push); e != nil {
			return e
		}
	}
	if !p.pendingMarks[1].IsNull() && p.multiple != nil {
		if e = p.multiple.ResetMark(p.pendingMarks[1]); e != nil {
			return e
		}
		p.pendingMarks[1] = ast.ASTHandle{}
	}
	if !p.pendingMarks[0].IsNull() && p.visited != nil {
		if e = p.visited.ResetMark(p.pendingMarks[0]); e != nil {
			return e
		}
		p.pendingMarks[0] = ast.ASTHandle{}
	}
	if p.multiple != nil {
		if e = p.multiple.Close(); e != nil {
			return e
		}
	}
	if p.visited != nil {
		if e = p.visited.Close(); e != nil {
			return e
		}
	}
	if p.numeric != nil {
		if e = p.numeric.Close(); e != nil {
			return e
		}
	}
	p.flatArgs.Close(p.manager.Budget())
	p.positions.close()
	p.manager.Budget().Release(p.header)
	p.header = 0
	p.closed = true
	return nil
}
