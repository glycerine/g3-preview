// Source: src/ast/rewriter/poly_rewriter_def.h:903-943;
// src/util/buffer.h:25-75,146-151.
package rewriter

import (
	"reflect"

	"github.com/glycerine/g3/pkg/ast"
)

type bvPolynomialValidity struct{ values []bool }

func (v *bvPolynomialValidity) data() []bool { return v.values }
func (v *bvPolynomialValidity) fill(_ *ast.ASTManager, n uint32) error {
	for uint64(len(v.values)) < uint64(n) {
		v.values = append(v.values, true)
	}
	return nil
}
func (v *bvPolynomialValidity) close(_ *ast.ASTManager) {
	clear(v.values)
	v.values = nil
}

// BVPolynomialMultiplicationHoist names one source invocation and borrows the
// manager/result reference. Its map and buffers borrow expression lifetimes.
type BVPolynomialMultiplicationHoist struct {
	self                                 *BVPolynomialMultiplicationHoist
	manager                              *ast.ASTManager
	currentSort                          ast.ASTHandle
	sortBinding                          *ast.ASTHandle
	adds, muls                           ast.ASTHandleBuffer
	valid                                bvPolynomialValidity
	table                                *bvPolynomialPositions
	merge                                *BVPolynomialMerger
	zero                                 ast.ASTBitVectorNumeralInvocation
	header                               uint64
	started, running, sourceDone, closed bool
	bodyCause, sourceCause, ownCause     error
}

func NewBVPolynomialMultiplicationHoist(m *ast.ASTManager) (*BVPolynomialMultiplicationHoist, error) {
	if _, e := m.ProofsEnabled(); e != nil {
		return nil, e
	}
	header := uint64(reflect.TypeOf(BVPolynomialMultiplicationHoist{}).Size())
	if e := m.Budget().Reserve(header); e != nil {
		return nil, e
	}
	p := &BVPolynomialMultiplicationHoist{manager: m, header: header}
	p.self = p
	return p, nil
}
func (p *BVPolynomialMultiplicationHoist) body(som ast.ASTRef) (bool, error) {
	m := p.manager
	input, e := m.RefGet(som)
	if e != nil {
		return false, e
	}
	if e = bvPolynomialFlatten(m, 4, &p.adds, input); e != nil {
		return false, e
	}
	if e = p.valid.fill(m, uint32(len(p.adds.Slice()))); e != nil {
		return false, e
	}
	p.table, e = newBVPolynomialPositions(m)
	if e != nil {
		return false, e
	}
	change := false
	for k := 0; k < len(p.adds.Slice()); k++ {
		term := p.adds.Slice()[k]
		p.muls.Reset()
		if e = bvPolynomialFlatten(m, 6, &p.muls, term); e != nil {
			return false, e
		}
		for _, factor := range p.muls.Slice() {
			num, e := m.IsBitVectorOperation(factor, 0)
			if e != nil {
				return false, e
			}
			if num {
				continue
			}
			j, found, e := p.table.find(factor)
			if e != nil {
				return false, e
			}
			if found && p.valid.data()[j] && j != uint32(k) {
				*p.sortSlot(), e = m.GetSort(p.adds.Slice()[k])
				if e != nil {
					return false, e
				}
				p.merge, e = NewBVPolynomialMerger(m, p.sortSlot())
				if e != nil {
					return false, e
				}
				merged, e := p.merge.Merge(p.adds.Slice()[j], p.adds.Slice()[k])
				if e != nil {
					return false, e
				}
				p.adds.Slice()[j] = merged
				if e = p.merge.Close(); e != nil {
					return false, e
				}
				p.merge = nil
				if e = m.AssignBitVectorPolynomialZeroWithInvocation(*p.sortSlot(), &p.adds.Slice()[k], &p.zero); e != nil {
					return false, e
				}
				p.valid.data()[j], p.valid.data()[k] = false, false
				change = true
				break
			}
			if e = p.table.insert(factor, uint32(k)); e != nil {
				return false, e
			}
		}
	}
	if !change {
		return false, nil
	}
	result, e := m.MkBitVectorPolynomialAdd(*p.sortSlot(), ast.SliceArguments(p.adds.Slice()))
	if e != nil {
		return false, e
	}
	if e = m.SetRef(som, result); e != nil {
		return false, e
	}
	return true, nil
}
func (p *BVPolynomialMultiplicationHoist) childrenSource() error {
	if p.merge != nil {
		if e := p.merge.SourceReady(); e != nil {
			return e
		}
	}
	return p.manager.BitVectorNumeralInvocationSourceReady(&p.zero)
}
func (p *BVPolynomialMultiplicationHoist) releaseLocals() {
	if p.table != nil {
		p.table.close()
		p.table = nil
	}
	p.valid.close(p.manager)
	p.muls.Close(p.manager.Budget())
	p.adds.Close(p.manager.Budget())
}
func (p *BVPolynomialMultiplicationHoist) Hoist(enabled bool, som ast.ASTRef) (bool, error) {
	if !enabled {
		return false, nil
	}
	if p == nil || p.self != p || p.closed {
		return false, ast.ErrASTHandle
	}
	if p.started || p.running {
		return false, ast.ErrASTPending
	}
	p.started, p.running = true, true
	defer func() { p.running = false }()
	changed, body := p.body(som)
	p.bodyCause = body
	p.sourceCause = p.childrenSource()
	if body != nil {
		return false, body
	}
	if p.sourceCause != nil {
		return false, p.sourceCause
	}
	p.releaseLocals()
	p.sourceDone = true
	return changed, nil
}
func (p *BVPolynomialMultiplicationHoist) SourceReady() error {
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
func (p *BVPolynomialMultiplicationHoist) Close() (result error) {
	if p == nil || p.self != p {
		return ast.ErrASTHandle
	}
	if p.closed {
		return nil
	}
	if p.running {
		return ast.ErrASTPending
	}
	defer func() {
		if result != nil && p.ownCause == nil {
			p.ownCause = result
		}
	}()
	if p.merge != nil {
		if e := p.merge.Close(); e != nil {
			return e
		}
	}
	if e := p.manager.CleanupBitVectorNumeralInvocation(&p.zero); e != nil {
		return e
	}
	p.releaseLocals()
	p.manager.Budget().Release(p.header)
	p.header = 0
	p.closed = true
	return nil
}

func (p *BVPolynomialMultiplicationHoist) sortSlot() *ast.ASTHandle {
	if p.sortBinding != nil {
		return p.sortBinding
	}
	return &p.currentSort
}
