// Source: src/ast/rewriter/poly_rewriter_def.h:198-367;
// src/ast/rewriter/bv_rewriter.h:35-43.
package rewriter

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/ast"
)

var ErrBVPolynomialMonomialBlowup = errors.New("sum of monomials blowup")

type BVPolynomialMultiplicationOptions struct {
	ASTOrder, HoistCoefficients, SumOfMonomials bool
	MonomialLimit                               uint32
}

// BVPolynomialMultiplicationCore names a single nonflat source invocation.
// All inputs and argument views borrow caller lifetimes through scope exit.
type BVPolynomialMultiplicationCore struct {
	flatArgs, todo                       ast.ASTHandleBuffer
	pair                                 [2]ast.ASTHandle
	currentSort                          ast.ASTHandle
	sortBinding                          *ast.ASTHandle
	localsEnded                          bool
	self                                 *BVPolynomialMultiplicationCore
	manager                              *ast.ASTManager
	numeric                              *ast.BitVectorPolynomialMultiplicationNumerals
	terms, monomial                      ast.ASTHandleBuffer
	sizes, iterator                      bvPolynomialUnsignedBuffer
	sums                                 bvPolynomialSumViews
	sum, savedSum                        ast.ASTRefVector
	sumPlaced                            bool
	push                                 ast.ASTRefVectorPushReceipt
	pushed                               ast.ASTHandle
	factory                              ast.ASTBitVectorNumeralInvocation
	header                               uint64
	started, running, sourceDone, closed bool
	bodyCause, sourceCause               error
}

func NewBVPolynomialMultiplicationCore(m *ast.ASTManager) (*BVPolynomialMultiplicationCore, error) {
	if _, e := m.ProofsEnabled(); e != nil {
		return nil, e
	}
	bytes := uint64(reflect.TypeOf(BVPolynomialMultiplicationCore{}).Size())
	if e := m.Budget().Reserve(bytes); e != nil {
		return nil, e
	}
	p := &BVPolynomialMultiplicationCore{manager: m, header: bytes}
	p.self = p
	return p, nil
}
func (p *BVPolynomialMultiplicationCore) assign(out ast.ASTRef, node ast.ASTHandle, e error) (RewriteStatus, error) {
	if e != nil {
		return RewriteFailed, e
	}
	return RewriteDone, p.manager.SetRef(out, node)
}
func (p *BVPolynomialMultiplicationCore) less(fid ast.FamilyID, o BVPolynomialMultiplicationOptions, a, b int) (bool, error) {
	v := p.terms.Slice()
	return bvMonomialLess(p.manager, fid, o.ASTOrder, v[a], v[b])
}
func (p *BVPolynomialMultiplicationCore) sift(fid ast.FamilyID, o BVPolynomialMultiplicationOptions, root, end int) error {
	for root < end/2 {
		child := root*2 + 1
		if child+1 < end {
			less, e := p.less(fid, o, child, child+1)
			if e != nil {
				return e
			}
			if less {
				child++
			}
		}
		less, e := p.less(fid, o, root, child)
		if e != nil {
			return e
		}
		if !less {
			return nil
		}
		v := p.terms.Slice()
		v[root], v[child] = v[child], v[root]
		root = child
	}
	return nil
}
func (p *BVPolynomialMultiplicationCore) sort(fid ast.FamilyID, o BVPolynomialMultiplicationOptions) error {
	n := len(p.terms.Slice())
	for i := n / 2; i > 0; i-- {
		if e := p.sift(fid, o, i-1, n); e != nil {
			return e
		}
	}
	for end := n; end > 1; end-- {
		v := p.terms.Slice()
		v[0], v[end-1] = v[end-1], v[0]
		if e := p.sift(fid, o, 0, end-1); e != nil {
			return e
		}
	}
	return nil
}
func (p *BVPolynomialMultiplicationCore) body(sort ast.ASTHandle, args ast.AppArguments, out ast.ASTRef, o BVPolynomialMultiplicationOptions) (RewriteStatus, error) {
	m := p.manager
	if args.Count() < 2 {
		return RewriteFailed, ast.ErrASTDomain
	}
	var e error
	p.numeric, e = ast.NewBitVectorPolynomialMultiplicationNumerals(m)
	if e != nil {
		return RewriteFailed, e
	}
	n := p.numeric
	if e = n.Place(0); e != nil {
		return RewriteFailed, e
	}
	if args.Count() == 2 {
		first, e := args.At(0)
		if e != nil {
			return RewriteFailed, e
		}
		yes, e := n.Read(first, 0)
		if e != nil {
			return RewriteFailed, e
		}
		if yes {
			one, e := n.One(0)
			if e != nil {
				return RewriteFailed, e
			}
			if !one {
				zero, e := n.Zero(0)
				if e != nil {
					return RewriteFailed, e
				}
				if !zero {
					rhs, e := args.At(1)
					if e != nil {
						return RewriteFailed, e
					}
					kind, e := m.GetKind(rhs)
					if e != nil {
						return RewriteFailed, e
					}
					if kind == ast.ASTVar {
						return RewriteFailed, nil
					}
					if kind != ast.ASTApp {
						return RewriteFailed, ast.ErrASTDomain
					}
					k, e := m.GetDeclKind(rhs)
					if e != nil {
						return RewriteFailed, e
					}
					bv, e := m.IsBitVectorOperation(rhs, k)
					if e != nil {
						return RewriteFailed, e
					}
					if !bv {
						return RewriteFailed, nil
					}
				}
			}
		}
	}
	if e = n.SetOne(); e != nil {
		return RewriteFailed, e
	}
	numCoeffs, numAdd := uint32(0), uint32(0)
	var variable ast.ASTHandle
	for i := uint32(0); i < args.Count(); i++ {
		arg, e := args.At(i)
		if e != nil {
			return RewriteFailed, e
		}
		yes, e := n.Read(arg, 0)
		if e != nil {
			return RewriteFailed, e
		}
		if yes {
			numCoeffs++
			if e = n.Multiply(0); e != nil {
				return RewriteFailed, e
			}
		} else {
			variable = arg
			add, e := m.IsBitVectorOperation(arg, 4)
			if e != nil {
				return RewriteFailed, e
			}
			if add {
				numAdd++
			}
		}
	}
	if e = n.Normalize(sort); e != nil {
		return RewriteFailed, e
	}
	if numCoeffs == args.Count() {
		node, e := n.Constant(sort)
		return p.assign(out, node, e)
	}
	zero, e := n.Zero(1)
	if e != nil {
		return RewriteFailed, e
	}
	if zero {
		node, e := n.Constant(sort)
		return p.assign(out, node, e)
	}
	if numCoeffs == args.Count()-1 {
		one, e := n.One(1)
		if e != nil {
			return RewriteFailed, e
		}
		if one {
			return RewriteDone, m.SetRef(out, variable)
		}
		if e = n.Place(2); e != nil {
			return RewriteFailed, e
		}
		mul, e := m.IsBitVectorOperation(variable, 6)
		if e != nil {
			return RewriteFailed, e
		}
		if mul {
			count, e := m.GetNumArgs(variable)
			if e != nil {
				return RewriteFailed, e
			}
			if count == 2 {
				coefficient, e := m.GetArg(variable, 0)
				if e != nil {
					return RewriteFailed, e
				}
				yes, e := n.Read(coefficient, 2)
				if e != nil {
					return RewriteFailed, e
				}
				if yes {
					if e = n.Multiply(2); e != nil {
						return RewriteFailed, e
					}
					if e = n.Normalize(sort); e != nil {
						return RewriteFailed, e
					}
					pp, e := m.GetArg(variable, 1)
					if e != nil {
						return RewriteFailed, e
					}
					node, e := n.Product(sort, pp)
					if e != nil {
						return RewriteFailed, e
					}
					return Rewrite1, m.SetRef(out, node)
				}
			}
			return RewriteFailed, nil
		}
		if numAdd == 0 || o.HoistCoefficients {
			if args.Count() == 2 {
				second, e := args.At(1)
				if e != nil {
					return RewriteFailed, e
				}
				if second == variable {
					if ast.ASTSourceDebug {
						first, e := args.At(0)
						if e != nil {
							return RewriteFailed, e
						}
						if e = n.DebugAlreadySimplified(first); e != nil {
							return RewriteFailed, e
						}
					}
					return RewriteFailed, nil
				}
			}
			node, e := n.Product(sort, variable)
			return p.assign(out, node, e)
		}
		count, e := m.GetNumArgs(variable)
		if e != nil {
			return RewriteFailed, e
		}
		for i := uint32(0); i < count; i++ {
			arg, e := m.GetArg(variable, i)
			if e != nil {
				return RewriteFailed, e
			}
			node, e := n.Product(sort, arg)
			if e != nil {
				return RewriteFailed, e
			}
			if e = p.terms.Push(m.Budget(), node); e != nil {
				return RewriteFailed, e
			}
		}
		node, e := m.MkBitVectorPolynomialAddWithInvocation(sort, ast.SliceArguments(p.terms.Slice()), &p.factory)
		if e != nil {
			return RewriteFailed, e
		}
		return Rewrite2, m.SetRef(out, node)
	}
	if !o.SumOfMonomials || numAdd == 0 {
		var prev ast.ASTHandle
		ordered := true
		fid, e := m.GetFamilyID(sort)
		if e != nil {
			return RewriteFailed, e
		}
		for i := uint32(0); i < args.Count(); i++ {
			curr, e := args.At(i)
			if e != nil {
				return RewriteFailed, e
			}
			yes, e := m.IsBitVectorNumeral(curr)
			if e != nil {
				return RewriteFailed, e
			}
			if yes {
				continue
			}
			if !prev.IsNull() {
				less, e := bvMonomialLess(m, fid, o.ASTOrder, curr, prev)
				if e != nil {
					return RewriteFailed, e
				}
				if less {
					ordered = false
				}
			}
			if e = p.terms.Push(m.Budget(), curr); e != nil {
				return RewriteFailed, e
			}
			prev = curr
		}
		if ordered && numCoeffs == 0 {
			return RewriteFailed, nil
		}
		if !ordered {
			if e = p.sort(fid, o); e != nil {
				return RewriteFailed, e
			}
		}
		node, e := m.MkBitVectorPolynomialMulWithInvocation(sort, ast.SliceArguments(p.terms.Slice()), &p.factory)
		if e != nil {
			return RewriteFailed, e
		}
		if e = m.SetRef(out, node); e != nil {
			return RewriteFailed, e
		}
		node, e = n.Product(sort, node)
		return p.assign(out, node, e)
	}
	return p.expand(sort, args, out, o)
}
func (p *BVPolynomialMultiplicationCore) expand(sort ast.ASTHandle, args ast.AppArguments, out ast.ASTRef, o BVPolynomialMultiplicationOptions) (RewriteStatus, error) {
	m := p.manager
	for i := uint32(0); i < args.Count(); i++ {
		if e := p.iterator.push(m, 0); e != nil {
			return RewriteFailed, e
		}
		arg, e := args.At(i)
		if e != nil {
			return RewriteFailed, e
		}
		add, e := m.IsBitVectorOperation(arg, 4)
		if e != nil {
			return RewriteFailed, e
		}
		var view ast.AppArguments
		if add {
			view, e = m.AppArgumentView(arg)
		} else {
			view, e = args.Range(i, 1)
		}
		if e != nil {
			return RewriteFailed, e
		}
		if e = p.sums.push(m, view); e != nil {
			return RewriteFailed, e
		}
		if e = p.sizes.push(m, view.Count()); e != nil {
			return RewriteFailed, e
		}
	}
	var e error
	p.sum, e = m.NewRefVector()
	if e != nil {
		return RewriteFailed, e
	}
	p.savedSum = p.sum
	p.sumPlaced = true
	for {
		size, e := m.RefVectorSize(p.sum)
		if e != nil {
			return RewriteFailed, e
		}
		if size > o.MonomialLimit {
			return RewriteFailed, ErrBVPolynomialMonomialBlowup
		}
		p.monomial.Reset()
		for i := uint32(0); i < args.Count(); i++ {
			arg, e := p.sums.data()[i].At(p.iterator.data()[i])
			if e != nil {
				return RewriteFailed, e
			}
			if e = p.monomial.Push(m.Budget(), arg); e != nil {
				return RewriteFailed, e
			}
		}
		node, e := m.MkBitVectorPolynomialMulWithInvocation(sort, ast.SliceArguments(p.monomial.Slice()), &p.factory)
		if e != nil {
			return RewriteFailed, e
		}
		p.pushed = node
		if e = m.RefVectorPushTracked(p.sum, node, &p.push); e != nil {
			return RewriteFailed, e
		}
		p.push = ast.ASTRefVectorPushReceipt{}
		if !bvPolynomialProductNext(p.sizes.data(), p.iterator.data()) {
			break
		}
	}
	count, e := m.RefVectorSize(p.sum)
	if e != nil {
		return RewriteFailed, e
	}
	view, e := m.RefVectorArgumentsRange(p.sum, 0, count)
	if e != nil {
		return RewriteFailed, e
	}
	node, e := m.MkBitVectorPolynomialAddWithInvocation(sort, view, &p.factory)
	if e != nil {
		return RewriteFailed, e
	}
	return Rewrite2, m.SetRef(out, node)
}
func (p *BVPolynomialMultiplicationCore) childrenSource() error {
	if p.numeric != nil {
		if e := p.numeric.SourceChildren(); e != nil {
			return e
		}
	}
	return p.manager.BitVectorNumeralInvocationSourceReady(&p.factory)
}
func (p *BVPolynomialMultiplicationCore) releaseBuffers() error {
	p.monomial.Close(p.manager.Budget())
	if p.sumPlaced {
		if e := p.manager.DelRefVector(&p.sum); e != nil {
			return e
		}
		p.sumPlaced = false
	}
	p.sums.close(p.manager)
	p.iterator.close(p.manager)
	p.sizes.close(p.manager)
	p.terms.Close(p.manager.Budget())
	return nil
}

// Rewrite names the original nonflat multiplication body with caller sort.
func (p *BVPolynomialMultiplicationCore) Rewrite(sort ast.ASTHandle, args ast.AppArguments, out ast.ASTRef, o BVPolynomialMultiplicationOptions) (RewriteStatus, error) {
	return p.rewrite(sort, args, out, o, false, false, false)
}

// RewriteFlat follows the original coefficient-form guard and recursive stack.
func (p *BVPolynomialMultiplicationCore) RewriteFlat(sort ast.ASTHandle, args ast.AppArguments, out ast.ASTRef, o BVPolynomialMultiplicationOptions) (RewriteStatus, error) {
	return p.rewrite(sort, args, out, o, true, false, false)
}

// RewriteApplication is the mk_mul_core singleton/sort/flat dispatch caller.
// Source: src/ast/rewriter/poly_rewriter.h:125-136.
func (p *BVPolynomialMultiplicationCore) RewriteApplication(args ast.AppArguments, out ast.ASTRef, flat bool, o BVPolynomialMultiplicationOptions) (RewriteStatus, error) {
	return p.rewrite(ast.ASTHandle{}, args, out, o, flat, true, false)
}
func (p *BVPolynomialMultiplicationCore) sourceEnd() error {
	if e := p.childrenSource(); e != nil {
		return e
	}
	if p.localsEnded {
		return nil
	}
	if e := p.releaseBuffers(); e != nil {
		return e
	}
	if p.numeric != nil {
		if e := p.numeric.SourceEnd(); e != nil {
			return e
		}
	}
	p.localsEnded = true
	return nil
}
func (p *BVPolynomialMultiplicationCore) endFlat() {
	p.todo.Close(p.manager.Budget())
	p.flatArgs.Close(p.manager.Budget())
}
func (p *BVPolynomialMultiplicationCore) rewrite(sort ast.ASTHandle, args ast.AppArguments, out ast.ASTRef, o BVPolynomialMultiplicationOptions, flat, application, fallback bool) (RewriteStatus, error) {
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
	p.sourceCause = p.childrenSource()
	if e != nil {
		return st, e
	}
	if p.sourceCause != nil {
		return st, p.sourceCause
	}
	p.sourceCause = p.sourceEnd()
	if p.sourceCause != nil {
		return st, p.sourceCause
	}
	p.endFlat()
	if fallback && st == RewriteFailed {
		node, body := p.manager.MkBitVectorPolynomialMulWithInvocation(*p.sortSlot(), args, &p.factory)
		if body == nil {
			body = p.manager.SetRef(out, node)
		}
		if body != nil {
			p.bodyCause = body
			p.sourceCause = p.childrenSource()
			return st, body
		}
	}
	p.sourceCause = p.childrenSource()
	p.sourceDone = p.sourceCause == nil
	return st, p.sourceCause
}
func (p *BVPolynomialMultiplicationCore) dispatch(sort ast.ASTHandle, args ast.AppArguments, out ast.ASTRef, o BVPolynomialMultiplicationOptions, flat, application bool) (RewriteStatus, error) {
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
	*p.sortSlot() = sort
	if flat {
		return p.flatBody(sort, args, out, o)
	}
	return p.body(sort, args, out, o)
}
func (p *BVPolynomialMultiplicationCore) SourceReady() error {
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
func (p *BVPolynomialMultiplicationCore) Close() error {
	if p == nil || p.self != p {
		return ast.ErrASTHandle
	}
	if p.closed {
		return nil
	}
	if p.running {
		return ast.ErrASTPending
	}
	if e := p.manager.CleanupBitVectorNumeralInvocation(&p.factory); e != nil {
		return e
	}
	if e := p.releaseBuffers(); e != nil {
		return e
	}
	pending, e := p.push.Unpublished()
	if e != nil {
		return e
	}
	if pending {
		if e = p.manager.CompleteRefVectorPush(p.savedSum, p.pushed, &p.push); e != nil {
			return e
		}
	}
	if p.numeric != nil {
		if e = p.numeric.Close(); e != nil {
			return e
		}
	}
	p.endFlat()
	p.manager.Budget().Release(p.header)
	p.header = 0
	p.closed = true
	return nil
}

func (p *BVPolynomialMultiplicationCore) sortSlot() *ast.ASTHandle {
	if p.sortBinding != nil {
		return p.sortBinding
	}
	return &p.currentSort
}
