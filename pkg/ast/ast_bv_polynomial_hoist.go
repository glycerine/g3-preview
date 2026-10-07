// Source: src/ast/rewriter/poly_rewriter_def.h:453-487;
// src/ast/rewriter/bv_rewriter.h:35-43; src/util/buffer.h:25-59,90-96,146-151.
package ast

import (
	"reflect"

	"github.com/glycerine/g3/pkg/util"
)

// BitVectorCoefficientHoist holds one source invocation's real locals and
// exact children. It borrows the manager and caller's argument vector. Go
// wrapper storage is charged; native stack/layout/allocation ABI is not claimed.
type BitVectorCoefficientHoist struct {
	self                                 *BitVectorCoefficientHoist
	manager                              *ASTManager
	compare                              ASTBitVectorNumeralInvocation
	reads                                [2]util.RationalServiceInvocation
	values                               [2]util.OwnedRational
	placed                               [2]bool
	pps                                  ASTHandleBuffer
	mulArgs                              [2]ASTHandle
	ppsPlaced                            bool
	header                               uint64
	started, running, sourceDone, closed bool
	bodyCause, sourceCause, ownCause     error
}

func NewBitVectorCoefficientHoist(m *ASTManager) (*BitVectorCoefficientHoist, error) {
	if _, e := m.ProofsEnabled(); e != nil {
		return nil, e
	}
	header := uint64(reflect.TypeOf(BitVectorCoefficientHoist{}).Size())
	if e := m.Budget().Reserve(header); e != nil {
		return nil, e
	}
	p := &BitVectorCoefficientHoist{manager: m, header: header}
	p.self = p
	return p, nil
}

func (p *BitVectorCoefficientHoist) less(v ASTRefVector, a, b uint32) (bool, error) {
	left, e := p.manager.RefVectorGet(v, a)
	if e != nil {
		return false, e
	}
	right, e := p.manager.RefVectorGet(v, b)
	if e != nil {
		return false, e
	}
	return p.manager.BitVectorPolynomialHoistLessWithInvocation(left, right, &p.compare)
}

func (p *BitVectorCoefficientHoist) sift(v ASTRefVector, root, end uint32) error {
	for root < end/2 {
		child := root*2 + 1
		if child+1 < end {
			less, e := p.less(v, child, child+1)
			if e != nil {
				return e
			}
			if less {
				child++
			}
		}
		less, e := p.less(v, root, child)
		if e != nil {
			return e
		}
		if !less {
			return nil
		}
		if e = p.manager.RefVectorSwapElements(v, root, child); e != nil {
			return e
		}
		root = child
	}
	return nil
}

// std::sort is external. This concrete no-scratch heapsort preserves its
// comparator/output contract, without claiming C++ library comparison order.
func (p *BitVectorCoefficientHoist) sort(v ASTRefVector, n uint32) error {
	for i := n / 2; i > 0; i-- {
		if e := p.sift(v, i-1, n); e != nil {
			return e
		}
	}
	for end := n; end > 1; end-- {
		if e := p.manager.RefVectorSwapElements(v, 0, end-1); e != nil {
			return e
		}
		if e := p.sift(v, 0, end-1); e != nil {
			return e
		}
	}
	return nil
}

func (p *BitVectorCoefficientHoist) retireRead(i int) error {
	c := &p.reads[i]
	if e := c.SourceReady(); e != nil {
		return e
	}
	if e := c.Cleanup(false); e != nil {
		return e
	}
	if e := c.Cleanup(true); e != nil {
		return e
	}
	if e := c.EndContext(); e != nil {
		return e
	}
	*c = util.RationalServiceInvocation{}
	return nil
}

func (p *BitVectorCoefficientHoist) product(t ASTHandle, i int) (bool, ASTHandle, error) {
	m := p.manager
	mul, e := m.IsBitVectorOperation(t, 6)
	if e != nil || !mul {
		return false, ASTHandle{}, e
	}
	n, e := m.GetNumArgs(t)
	if e != nil || n != 2 {
		return false, ASTHandle{}, e
	}
	first, e := m.GetArg(t, 0)
	if e != nil {
		return false, ASTHandle{}, e
	}
	var width uint32
	yes, e := m.BitVectorNumeralValueWithInvocation(first, &p.values[i], &width, &p.reads[i])
	if e != nil {
		return false, ASTHandle{}, e
	}
	if e = p.retireRead(i); e != nil {
		return false, ASTHandle{}, e
	}
	if !yes {
		return false, ASTHandle{}, nil
	}
	pp, e := m.GetArg(t, 1)
	return true, pp, e
}

func (p *BitVectorCoefficientHoist) body(currentSort ASTHandle, args ASTRefVector) error {
	m := p.manager
	size, e := m.RefVectorSize(args)
	if e != nil {
		return e
	}
	if e = p.sort(args, size); e != nil {
		return e
	}
	// These locals are constructed after the entire sort, as in the source.
	for i := range p.values {
		p.values[i], e = m.numerals.NewValue()
		if e != nil {
			return e
		}
		p.placed[i] = true
	}
	p.ppsPlaced = true
	var i, j uint32
	for i < size {
		mon, e := m.RefVectorGet(args, i)
		if e != nil {
			return e
		}
		isMul, pp, e := p.product(mon, 0)
		if e != nil {
			return e
		}
		if isMul && i < size-1 {
			next, e := m.RefVectorGet(args, i+1)
			if e != nil {
				return e
			}
			isNext, ppNext, e := p.product(next, 1)
			if e != nil {
				return e
			}
			equal := false
			if isNext {
				equal, e = m.numerals.Equal(&p.values[0], &p.values[1])
				if e != nil {
					return e
				}
			}
			if isNext && equal {
				p.pps.Reset()
				if e = p.pps.Push(m.Budget(), pp); e != nil {
					return e
				}
				if e = p.pps.Push(m.Budget(), ppNext); e != nil {
					return e
				}
				i += 2
				for i < size {
					next, e = m.RefVectorGet(args, i)
					if e != nil {
						return e
					}
					isNext, ppNext, e = p.product(next, 1)
					if e != nil {
						return e
					}
					if !isNext {
						break
					}
					equal, e = m.numerals.Equal(&p.values[0], &p.values[1])
					if e != nil {
						return e
					}
					if !equal {
						break
					}
					if e = p.pps.Push(m.Budget(), ppNext); e != nil {
						return e
					}
					i++
				}
				if ASTSourceDebug {
					literal, e := m.GetArg(mon, 0)
					if e != nil {
						return e
					}
					var width uint32
					yes, e := m.BitVectorNumeralValueWithInvocation(literal, &p.values[1], &width, &p.reads[1])
					if e != nil {
						return e
					}
					if e = p.retireRead(1); e != nil {
						return e
					}
					equal, e = m.numerals.Equal(&p.values[0], &p.values[1])
					if e != nil {
						return e
					}
					if !yes || !equal {
						return ErrASTDomain
					}
				}
				// Preserve the debug assertion's separate read and the braced
				// initializer's left-to-right publication before the sum factory.
				literal, e := m.GetArg(mon, 0)
				if e != nil {
					return e
				}
				p.mulArgs[0] = literal
				sum, e := m.MkBitVectorPolynomialAdd(currentSort, SliceArguments(p.pps.Slice()))
				if e != nil {
					return e
				}
				p.mulArgs[1] = sum
				mul, e := m.MkBitVectorPolynomialMul(currentSort, SliceArguments(p.mulArgs[:]))
				if e != nil {
					return e
				}
				if e = m.RefVectorSet(args, j, mul); e != nil {
					return e
				}
				j++
				continue
			}
		}
		if e = m.RefVectorSet(args, j, mon); e != nil {
			return e
		}
		j++
		i++
	}
	return m.RefVectorShrink(args, j)
}

func (p *BitVectorCoefficientHoist) childrenSource() error {
	if e := p.manager.BitVectorNumeralInvocationSourceReady(&p.compare); e != nil {
		return e
	}
	for i := 1; i >= 0; i-- {
		if e := p.reads[i].SourceReady(); e != nil {
			return e
		}
	}
	return nil
}

func (p *BitVectorCoefficientHoist) sourceEnd() error {
	if e := p.childrenSource(); e != nil {
		return e
	}
	if p.ppsPlaced {
		p.pps.Close(p.manager.Budget())
		p.ppsPlaced = false
	}
	for i := 1; i >= 0; i-- {
		if p.placed[i] {
			if e := p.manager.numerals.Del(&p.values[i]); e != nil {
				return e
			}
			p.placed[i] = false
		}
	}
	p.sourceDone = true
	return nil
}

// Hoist sorts and compacts the caller-owned vector in place. This owner names
// one invocation; failed calls retain their real prefix for Close and never retry.
func (p *BitVectorCoefficientHoist) Hoist(currentSort ASTHandle, args ASTRefVector) (result error) {
	if p == nil || p.self != p || p.closed {
		return ErrASTHandle
	}
	if p.started || p.running {
		return ErrASTPending
	}
	p.started, p.running = true, true
	defer func() { p.running = false }()
	p.bodyCause = p.body(currentSort, args)
	if p.bodyCause != nil {
		p.sourceCause = p.childrenSource()
		return p.bodyCause
	}
	p.sourceCause = p.sourceEnd()
	return p.sourceCause
}

// SourceReady is historical and is unchanged by owning continuation.
func (p *BitVectorCoefficientHoist) SourceReady() error {
	if p == nil || p.self != p {
		return ErrASTHandle
	}
	if p.sourceCause != nil {
		return p.sourceCause
	}
	if p.started && !p.sourceDone {
		return ErrASTPending
	}
	return nil
}

func (p *BitVectorCoefficientHoist) Close() (result error) {
	if p == nil || p.self != p {
		return ErrASTHandle
	}
	if p.closed {
		return nil
	}
	if p.running {
		return ErrASTPending
	}
	defer func() {
		if result != nil && p.ownCause == nil {
			p.ownCause = result
		}
	}()
	if e := p.manager.CleanupBitVectorNumeralInvocation(&p.compare); e != nil {
		return e
	}
	for i := 1; i >= 0; i-- {
		if e := p.reads[i].Cleanup(false); e != nil {
			return e
		}
	}
	if p.ppsPlaced {
		p.pps.Close(p.manager.Budget())
		p.ppsPlaced = false
	}
	for i := 1; i >= 0; i-- {
		if p.placed[i] {
			if e := p.manager.numerals.CleanupValue(&p.values[i]); e != nil {
				return e
			}
			p.placed[i] = false
		}
	}
	for i := 1; i >= 0; i-- {
		if e := p.reads[i].Cleanup(true); e != nil {
			return e
		}
	}
	for i := 1; i >= 0; i-- {
		if e := p.reads[i].EndContext(); e != nil {
			return e
		}
	}
	p.manager.Budget().Release(p.header)
	p.header = 0
	p.closed = true
	return nil
}

func (p *BitVectorCoefficientHoist) Causes() (body, source, own error) {
	if p == nil || p.self != p {
		return ErrASTHandle, nil, nil
	}
	return p.bodyCause, p.SourceReady(), p.ownCause
}
