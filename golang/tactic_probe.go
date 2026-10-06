// Source: src/tactic/probe.h:30-61; src/tactic/probe.cpp:39-70,180-192.
// Source author: Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

type tacticProbeKind uint8

const (
	tacticProbeConstant tacticProbeKind = iota
	tacticProbeDepth
	tacticProbeSize
	tacticProbeNumExprs
)

// Four concrete source probe classes share cold reference metadata. Each Eval
// branch is the original class body, borrowing the goal only for that call.
type tacticProbe struct {
	self       *tacticProbe
	budget     *arena.Budget
	constant   float64
	ownerBytes uint64
	references uint32
	kind       tacticProbeKind
	closed     bool
}

func newTacticProbe(b *arena.Budget, kind tacticProbeKind, value float64) (*tacticProbe, error) {
	if kind > tacticProbeNumExprs {
		return nil, ErrASTDomain
	}
	n := uint64(reflect.TypeOf(tacticProbe{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	p := &tacticProbe{budget: b, kind: kind, constant: value, ownerBytes: n}
	p.self = p
	return p, nil
}
func (p *tacticProbe) open() error {
	if p == nil || p.self != p || p.closed {
		return errTacticRegistryOwner
	}
	return nil
}
func (p *tacticProbe) IncRef() error {
	if e := p.open(); e != nil {
		return e
	}
	p.references++
	return nil
}
func (p *tacticProbe) DecRef() error {
	if e := p.open(); e != nil {
		return e
	}
	if p.references == 0 {
		return errTacticRegistryOwner
	}
	p.references--
	if p.references == 0 {
		return p.Close()
	}
	return nil
}
func (p *tacticProbe) Close() error {
	if p == nil || p.self != p {
		return errTacticRegistryOwner
	}
	if p.closed {
		return nil
	}
	p.budget.Release(p.ownerBytes)
	p.ownerBytes = 0
	p.closed = true
	return nil
}
func (p *tacticProbe) Eval(g *tacticGoal) (float64, error) {
	if e := p.open(); e != nil {
		return 0, e
	}
	if e := g.open(); e != nil {
		return 0, e
	}
	switch p.kind {
	case tacticProbeConstant:
		return p.constant, nil
	case tacticProbeDepth:
		return float64(g.depth), nil
	case tacticProbeSize:
		n, e := g.Size()
		return float64(n), e
	case tacticProbeNumExprs:
		n, e := g.NumExprs()
		return float64(n), e
	}
	return 0, ErrASTDomain
}
func tacticProbeIsTrue(value float64) bool { return value != 0 }
