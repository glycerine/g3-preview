// Source: src/ast/array_decl_plugin.cpp:86-113; src/util/rational.h:67,110-181.
// Source author: Leonardo de Moura (2008-01-09).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/mpnum"
	"math"
	"reflect"
)

// Concrete source lexical values, held in one real charged cold frame only
// when needed by this finite-cardinality branch. No heap rational wrapper or
// shadow pointer payload is created. Failed source constructors are owned for
// explicit abort, but are not completed values on the source DT schedule.
type astArrayCardinalityFrame struct {
	next                             *astArrayCardinalityFrame
	self                             *astArrayCardinalityFrame
	owner                            *ASTManager
	tokens                           [4]mpnum.OwnedRational // immutable latest actual returned construction/acquisition tuples
	values                           [4]mpnum.OwnedRational // domain_sz, num_elements, conversion, power result
	acquired, completed              [4]bool
	invocation, header               uint64
	sourceDone                       bool
	bodyCause, sourceCause, ownCause error
}

func (m *ASTManager) arrayCardinalityBytes() uint64 {
	var n uint64
	for f := m.arrayCardinality; f != nil; f = f.next {
		n += f.header
	}
	return n
}
func (m *ASTManager) newArrayCardinalityFrame() (*astArrayCardinalityFrame, error) {
	if m.arrayCardinalityInvocation == math.MaxUint64 {
		return nil, ErrASTPending
	}
	bytes := uint64(reflect.TypeOf(astArrayCardinalityFrame{}).Size())
	if e := m.budget.Reserve(bytes); e != nil {
		return nil, e
	}
	m.arrayCardinalityInvocation++
	f := &astArrayCardinalityFrame{owner: m, invocation: m.arrayCardinalityInvocation, header: bytes, next: m.arrayCardinality}
	f.self = f
	m.arrayCardinality = f
	return f, nil
}
func (f *astArrayCardinalityFrame) construct(i int, n uint64, set bool) error {
	if f.acquired[i] {
		return ErrASTPending
	}
	v, e := f.owner.numerals.NewValue()
	if e != nil {
		return e
	}
	f.values[i] = v
	f.tokens[i] = v
	f.acquired[i] = true
	if set {
		if e = f.owner.numerals.SetUint64(&f.values[i], n); e != nil {
			return e
		}
	}
	f.completed[i] = true
	f.tokens[i] = f.values[i]
	return nil
}
func (f *astArrayCardinalityFrame) destroy(i int, own bool) error {
	if !f.acquired[i] || !own && !f.completed[i] {
		return nil
	}
	e := f.owner.numerals.Del(&f.values[i])
	if e != nil {
		return e
	}
	f.acquired[i] = false
	f.completed[i] = false
	return nil
}
func (f *astArrayCardinalityFrame) sourceEnd() error {
	if f.sourceDone {
		return f.sourceCause
	}
	if f.sourceCause != nil {
		return f.sourceCause
	}
	for i := 3; i >= 0; i-- {
		if e := f.destroy(i, false); e != nil {
			f.sourceCause = e
			return e
		}
	}
	f.sourceDone = true
	return nil
}
func (m *ASTManager) cleanupArrayCardinality() error {
	for f := m.arrayCardinality; f != nil; f = f.next {
		if f.self != f || f.owner != m || f.invocation == 0 {
			return ErrASTHandle
		}
		for i := 3; i >= 0; i-- {
			if e := f.destroy(i, true); e != nil {
				if f.ownCause == nil {
					f.ownCause = e
				}
				return e
			}
		}
	}
	return nil
}
func (m *ASTManager) releaseArrayCardinalityHistory() {
	for f := m.arrayCardinality; f != nil; {
		next := f.next
		for _, a := range f.acquired {
			if a {
				panic(ErrASTPending)
			}
		}
		m.budget.Release(f.header)
		f.header = 0
		f = next
	}
	m.arrayCardinality = nil
}
func (m *ASTManager) finishArrayCardinality(f *astArrayCardinalityFrame, body error) error {
	if f.bodyCause == nil {
		f.bodyCause = body
	}
	source := f.sourceEnd()
	// A clean completed lexical scope has no remaining acquisition or diagnostic
	// history. Failed scopes stay named until actual enclosing owner completion.
	if body == nil && source == nil {
		for _, a := range f.acquired {
			if a {
				return ErrASTPending
			}
		}
		if m.arrayCardinality != f {
			return ErrASTPending
		}
		m.arrayCardinality = f.next
		m.budget.Release(f.header)
		f.header = 0
	}
	return errors.Join(body, source)
}
func (m *ASTManager) finiteArraySize(ps astArrayParameters) (SortSize, *astArrayCardinalityFrame, error) {
	f, e := m.newArrayCardinalityFrame()
	if e != nil {
		return SortSize{}, f, e
	}
	if e = f.construct(0, 1, true); e != nil {
		return SortSize{}, f, e
	}
	if e = f.construct(1, 0, false); e != nil {
		return SortSize{}, f, e
	}
	for i := 0; i < ps.count()-1; i++ {
		p, e := ps.at(m, i)
		if e != nil {
			return SortSize{}, f, e
		}
		info, e := m.sortInfo(m.sortAt(p.ast.slot).info)
		if e != nil {
			return SortSize{}, f, e
		}
		n, e := info.size.Size()
		if e != nil {
			return SortSize{}, f, e
		}
		if e = f.construct(2, n, true); e != nil {
			return SortSize{}, f, e
		}
		body := m.numerals.MulAssign(&f.values[0], &f.values[2])
		dt := f.destroy(2, false)
		if dt != nil && f.sourceCause == nil {
			f.sourceCause = dt
		}
		if e = errors.Join(body, dt); e != nil {
			return SortSize{}, f, e
		}
	}
	large, e := m.numerals.GreaterInt(&f.values[0], 128)
	if e != nil {
		return SortSize{}, f, e
	}
	if !large {
		p, e := ps.at(m, ps.count()-1)
		if e != nil {
			return SortSize{}, f, e
		}
		info, e := m.sortInfo(m.sortAt(p.ast.slot).info)
		if e != nil {
			return SortSize{}, f, e
		}
		n, e := info.size.Size()
		if e != nil {
			return SortSize{}, f, e
		}
		if e = m.numerals.SetUint64(&f.values[1], n); e != nil {
			return SortSize{}, f, e
		}
		exponent, e := m.numerals.Int64(&f.values[0])
		if e != nil {
			return SortSize{}, f, e
		}
		result, e := m.numerals.Power(&f.values[1], uint32(exponent))
		if e != nil {
			return SortSize{}, f, e
		}
		f.values[3] = result
		f.tokens[3] = result
		f.acquired[3] = true
		f.completed[3] = true
		body := m.numerals.Copy(&f.values[1], &f.values[3])
		dt := f.destroy(3, false)
		if dt != nil && f.sourceCause == nil {
			f.sourceCause = dt
		}
		if e = errors.Join(body, dt); e != nil {
			return SortSize{}, f, e
		}
	}
	// Source repeats the comparison after the conditional expression, then uses
	// short-circuit || before is_uint64. It does not reuse the earlier predicate.
	large, e = m.numerals.GreaterInt(&f.values[0], 128)
	if e != nil {
		return SortSize{}, f, e
	}
	if large {
		return VeryBigSortSize(), f, nil
	}
	size, e := SortSizeFromRational(m.numerals, &f.values[1])
	return size, f, e
}
