// Source: src/tactic/probe.h:30-61; src/tactic/probe.cpp:39-70,180-192.
package tactic

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

var ErrRegistryOwner = errors.New("tactic: invalid, copied or retired registration owner")

type ProbeKind uint8

const (
	ProbeConstant ProbeKind = iota
	ProbeDepth
	ProbeSize
	ProbeNumExprs
	ProbeMemory
)

// Concrete source probe classes share cold reference metadata. Each Eval
// branch is the original class body, borrowing the goal only for that call.
type Probe struct {
	self       *Probe
	budget     *arena.Budget
	constant   float64
	ownerBytes uint64
	References uint32
	kind       ProbeKind
	Closed     bool
}

func NewProbe(b *arena.Budget, kind ProbeKind, value float64) (*Probe, error) {
	if kind > ProbeMemory {
		return nil, ast.ErrASTDomain
	}
	n := uint64(reflect.TypeOf(Probe{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	p := &Probe{budget: b, kind: kind, constant: value, ownerBytes: n}
	p.self = p
	return p, nil
}
func (p *Probe) open() error {
	if p == nil || p.self != p || p.Closed {
		return ErrRegistryOwner
	}
	return nil
}
func (p *Probe) IncRef() error {
	if e := p.open(); e != nil {
		return e
	}
	p.References++
	return nil
}
func (p *Probe) DecRef() error {
	if e := p.open(); e != nil {
		return e
	}
	if p.References == 0 {
		return ErrRegistryOwner
	}
	p.References--
	if p.References == 0 {
		return p.Close()
	}
	return nil
}
func (p *Probe) Close() error {
	if p == nil || p.self != p {
		return ErrRegistryOwner
	}
	if p.Closed {
		return nil
	}
	p.budget.Release(p.ownerBytes)
	p.ownerBytes = 0
	p.Closed = true
	return nil
}
func (p *Probe) Eval(g *Goal) (float64, error) {
	if e := p.open(); e != nil {
		return 0, e
	}
	if e := g.open(); e != nil {
		return 0, e
	}
	switch p.kind {
	case ProbeConstant:
		return p.constant, nil
	case ProbeDepth:
		return float64(g.depth), nil
	case ProbeSize:
		n, e := g.Size()
		return float64(n), e
	case ProbeNumExprs:
		n, e := g.NumExprs()
		return float64(n), e
	case ProbeMemory:
		numerals := g.manager.NumeralService()
		if numerals == nil || numerals.RuntimeOwner() == nil {
			return 0, ast.ErrASTHandle
		}
		bytes, e := numerals.RuntimeOwner().AllocationSize()
		if e != nil {
			return 0, e
		}
		return float64(bytes) / float64(1024*1024), nil
	}
	return 0, ast.ErrASTDomain
}
func ProbeIsTrue(value float64) bool { return value != 0 }
