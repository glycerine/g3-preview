// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Source parameter constructors, copy/move, destruction, equality and hash.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"math"
)

func (m *ASTManager) parameter(p ASTParameter) (*astParameterRecord, error) {
	if p.origin != m.origin || p.slot == 0 || p.slot >= m.nextParameter {
		return nil, ErrASTHandle
	}
	r := m.parameterAt(p.slot)
	if !r.live || r.generation != p.generation {
		return nil, ErrASTHandle
	}
	return r, nil
}
func (m *ASTManager) allocParameter() (ASTParameter, error) {
	for m.freeParameter != 0 {
		slot := m.freeParameter
		r := m.parameterAt(slot)
		m.freeParameter = r.nextFree
		if r.generation == math.MaxUint32 {
			continue
		}
		gen := r.generation + 1
		*r = astParameterRecord{generation: gen, live: true}
		m.liveParameters++
		return ASTParameter{m.origin, slot, gen}, nil
	}
	if m.nextParameter == math.MaxUint32 {
		return ASTParameter{}, arena.ErrBounds
	}
	slot := m.nextParameter
	if e := m.ensureParameterPage(slot); e != nil {
		return ASTParameter{}, e
	}
	m.nextParameter++
	*m.parameterAt(slot) = astParameterRecord{generation: 1, live: true}
	m.liveParameters++
	return ASTParameter{m.origin, slot, 1}, nil
}
func (m *ASTManager) dropParameter(p ASTParameter) error {
	r, e := m.parameter(p)
	if e != nil {
		return e
	}
	if r.kind == ASTParameterRational && r.rational != (mpnum.OwnedRational{}) {
		if e = m.numerals.Del(&r.rational); e != nil {
			return e
		}
	}
	r.live = false
	r.nextFree = m.freeParameter
	m.freeParameter = p.slot
	m.liveParameters--
	return nil
}
func (m *ASTManager) newParameterInt(value int32) (ASTParameter, error) {
	p, e := m.allocParameter()
	if e == nil {
		m.parameterAt(p.slot).integer = value
	}
	return p, e
}
func (m *ASTManager) NewParameterInt(value int32) (ASTParameter, error) {
	if e := m.lock(); e != nil {
		return ASTParameter{}, e
	}
	defer m.mu.Unlock()
	return m.newParameterInt(value)
}
func (m *ASTManager) NewParameterUint(value uint32) (ASTParameter, error) {
	return m.NewParameterInt(int32(value))
}
func (m *ASTManager) newParameterAST(value ASTHandle) (ASTParameter, error) {
	var r astNodeRef
	var e error
	if !value.IsNull() {
		r, e = m.internal(value)
		if e != nil {
			return ASTParameter{}, e
		}
	}
	p, e := m.allocParameter()
	if e == nil {
		v := m.parameterAt(p.slot)
		v.kind = ASTParameterAST
		v.ast = r
	}
	return p, e
}
func (m *ASTManager) NewParameterAST(value ASTHandle) (ASTParameter, error) {
	if e := m.lock(); e != nil {
		return ASTParameter{}, e
	}
	defer m.mu.Unlock()
	return m.newParameterAST(value)
}
func (m *ASTManager) newParameterSymbol(value Symbol) (ASTParameter, error) {
	if _, e := m.symbols.Hash(value); e != nil {
		return ASTParameter{}, e
	}
	p, e := m.allocParameter()
	if e == nil {
		r := m.parameterAt(p.slot)
		r.kind = ASTParameterSymbol
		r.symbol = value
	}
	return p, e
}
func (m *ASTManager) NewParameterSymbol(value Symbol) (ASTParameter, error) {
	if e := m.lock(); e != nil {
		return ASTParameter{}, e
	}
	defer m.mu.Unlock()
	return m.newParameterSymbol(value)
}
func (m *ASTManager) NewParameterString(value string) (ASTParameter, error) {
	if e := m.lock(); e != nil {
		return ASTParameter{}, e
	}
	defer m.mu.Unlock()
	s, e := m.symbols.Intern(value)
	if e != nil {
		return ASTParameter{}, e
	}
	return m.newParameterSymbol(s)
}
func (m *ASTManager) newParameterRational(value *mpnum.OwnedRational) (ASTParameter, error) {
	p, e := m.allocParameter()
	if e != nil {
		return p, e
	}
	r := m.parameterAt(p.slot)
	r.kind = ASTParameterRational
	r.rational, e = m.numerals.NewValue()
	if e == nil {
		e = m.numerals.Copy(&r.rational, value)
	}
	if e != nil {
		m.dropParameter(p)
		return ASTParameter{}, e
	}
	return p, nil
}
func (m *ASTManager) NewParameterRational(value *mpnum.OwnedRational) (ASTParameter, error) {
	if e := m.lock(); e != nil {
		return ASTParameter{}, e
	}
	defer m.mu.Unlock()
	return m.newParameterRational(value)
}
func (m *ASTManager) NewParameterDouble(value float64) (ASTParameter, error) {
	if e := m.lock(); e != nil {
		return ASTParameter{}, e
	}
	defer m.mu.Unlock()
	p, e := m.allocParameter()
	if e == nil {
		r := m.parameterAt(p.slot)
		r.kind = ASTParameterDouble
		r.double = value
	}
	return p, e
}
func (m *ASTManager) NewParameterExternal(value uint32) (ASTParameter, error) {
	if e := m.lock(); e != nil {
		return ASTParameter{}, e
	}
	defer m.mu.Unlock()
	p, e := m.allocParameter()
	if e == nil {
		r := m.parameterAt(p.slot)
		r.kind = ASTParameterExternal
		r.external = value
	}
	return p, e
}
func (m *ASTManager) assignParameter(dst, src ASTParameter) error {
	a, e := m.parameter(dst)
	if e != nil {
		return e
	}
	b, e := m.parameter(src)
	if e != nil {
		return e
	}
	if dst == src {
		return nil
	}
	if a.kind == ASTParameterRational && a.rational != (mpnum.OwnedRational{}) {
		if e = m.numerals.Del(&a.rational); e != nil {
			return e
		}
	}
	// Source deletes old owned rational before publishing the new kind/copy.
	a.kind = b.kind
	switch b.kind {
	case ASTParameterInt:
		a.integer = b.integer
	case ASTParameterAST:
		a.ast = b.ast
	case ASTParameterSymbol:
		a.symbol = b.symbol
	case ASTParameterRational:
		a.rational, e = m.numerals.NewValue()
		if e == nil {
			e = m.numerals.Copy(&a.rational, &b.rational)
		}
		// A failed new object/copy leaves a destructible zero rational adapter,
		// preserving destruction of the old value rather than restoring it.
	case ASTParameterDouble:
		a.double = b.double
	case ASTParameterExternal:
		a.external = b.external
	default:
		return ErrASTDomain
	}
	return e
}
func (m *ASTManager) copyParameter(src ASTParameter) (ASTParameter, error) {
	if _, e := m.parameter(src); e != nil {
		return ASTParameter{}, e
	}
	p, e := m.allocParameter()
	if e != nil {
		return p, e
	}
	if e = m.assignParameter(p, src); e != nil {
		m.dropParameter(p)
		return ASTParameter{}, e
	}
	return p, nil
}
func (m *ASTManager) CopyParameter(src ASTParameter) (ASTParameter, error) {
	if e := m.lock(); e != nil {
		return ASTParameter{}, e
	}
	defer m.mu.Unlock()
	return m.copyParameter(src)
}
func (m *ASTManager) AssignParameter(dst, src ASTParameter) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.assignParameter(dst, src)
}
func (m *ASTManager) MoveParameter(src ASTParameter) (ASTParameter, error) {
	if e := m.lock(); e != nil {
		return ASTParameter{}, e
	}
	defer m.mu.Unlock()
	r, e := m.parameter(src)
	if e != nil {
		return ASTParameter{}, e
	}
	saved := *r
	p, e := m.allocParameter()
	if e != nil {
		return p, e
	}
	dst := m.parameterAt(p.slot)
	gen := dst.generation
	*dst = saved
	dst.generation = gen
	dst.nextFree = 0
	if saved.kind == ASTParameterRational {
		m.parameterAt(src.slot).rational = mpnum.OwnedRational{}
	}
	return p, nil
}
func (m *ASTManager) DelParameter(p *ASTParameter) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	if p == nil {
		return ErrASTHandle
	}
	if e := m.dropParameter(*p); e != nil {
		return e
	}
	*p = ASTParameter{}
	return nil
}
func (m *ASTManager) ParameterKind(p ASTParameter) (ASTParameterKind, error) {
	if e := m.lock(); e != nil {
		return 0, e
	}
	defer m.mu.Unlock()
	r, e := m.parameter(p)
	if e != nil {
		return 0, e
	}
	return r.kind, nil
}
func (m *ASTManager) parameterHash(p ASTParameter) (uint32, error) {
	r, e := m.parameter(p)
	if e != nil {
		return 0, e
	}
	var v uint32
	switch r.kind {
	case ASTParameterInt:
		v = uint32(r.integer)
	case ASTParameterAST:
		if e = m.validNode(r.ast); e != nil {
			return 0, e
		}
		v = m.nodeAt(r.ast.slot).hash
	case ASTParameterSymbol:
		v, e = m.symbols.Hash(r.symbol)
	case ASTParameterRational:
		v, e = m.numerals.Hash(&r.rational)
	case ASTParameterDouble:
		if math.IsNaN(r.double) || math.IsInf(r.double, 0) || r.double <= -1 || r.double >= 4294967296 {
			return 0, ErrASTDomain
		}
		v = uint32(r.double)
	case ASTParameterExternal:
		v = r.external
	default:
		return 0, ErrASTDomain
	}
	return (v << 2) | uint32(r.kind), e
}
func (m *ASTManager) ParameterHash(p ASTParameter) (uint32, error) {
	if e := m.lock(); e != nil {
		return 0, e
	}
	defer m.mu.Unlock()
	return m.parameterHash(p)
}
func (m *ASTManager) parameterEqual(a, b ASTParameter) (bool, error) {
	x, e := m.parameter(a)
	if e != nil {
		return false, e
	}
	y, e := m.parameter(b)
	if e != nil {
		return false, e
	}
	if x.kind != y.kind {
		return false, nil
	}
	switch x.kind {
	case ASTParameterInt:
		return x.integer == y.integer, nil
	case ASTParameterAST:
		return x.ast == y.ast, nil
	case ASTParameterSymbol:
		return x.symbol == y.symbol, nil
	case ASTParameterRational:
		return m.numerals.Equal(&x.rational, &y.rational)
	case ASTParameterDouble:
		return x.double == y.double, nil
	case ASTParameterExternal:
		return x.external == y.external, nil
	}
	return false, ErrASTDomain
}
func (m *ASTManager) ParameterEqual(a, b ASTParameter) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	return m.parameterEqual(a, b)
}
