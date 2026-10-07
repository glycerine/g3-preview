// Source: src/math/polynomial/polynomial.cpp:454-481,1988-2039.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"github.com/glycerine/g3/pkg/util"
)

// This token owns only the actual unsigned position write. It owns no RC of
// the borrowed monomial whose ID was read before source publication.
func (p *MonomialPositions) acceptsArithmeticMarker(r *polynomialArithmeticInvocationReceipt) bool {
	return p != nil && p.self == p && r != nil && r.manager != nil && r.manager.matchesArithmeticInvocation(r) && r.frame.executing && r.manager.callbackDepth == 0 && r.frame.buffer != nil && &r.frame.buffer.positions == p && p.manager == r.manager.monomials
}
func (p *MonomialPositions) markerMatches(r *polynomialArithmeticInvocationReceipt) bool {
	return p != nil && p.self == p && r != nil && r.manager != nil && r.manager.matchesArithmeticInvocation(r) && r.frame.buffer != nil && &r.frame.buffer.positions == p && p.manager == r.manager.monomials && p.markerFrame == r.frame && p.markerInvocation == r.invocation && r.frame.positionPublished
}
func (p *MonomialPositions) acknowledgeMarkerWrite(id uint32) {
	s := p.markerFrame
	if s != nil && s.positionID == id && s.positionPublished && s.invocation == p.markerInvocation && s.buffer != nil && &s.buffer.positions == p && s.buffer.owner != nil && s.buffer.owner.matchesArithmeticInvocation(&polynomialArithmeticInvocationReceipt{manager: s.buffer.owner, frame: s, invocation: p.markerInvocation}) {
		s.positionPublished = false
		s.positionTerm = Monomial{}
		s.positionBlock = util.Block{}
		p.markerFrame = nil
		p.markerInvocation = 0
	}
}
func (p *MonomialPositions) releaseArithmeticMarker(r *polynomialArithmeticInvocationReceipt) {
	if p.markerMatches(r) {
		p.acknowledgeMarkerWrite(r.frame.positionID)
	}
}
func (p *MonomialPositions) cleanupArithmeticMarker(r *polynomialArithmeticInvocationReceipt) error {
	if !p.markerMatches(r) {
		return ErrPolynomialOwner
	}
	if e := p.check(); e != nil {
		return e
	}
	s := r.frame
	if p.raw != s.positionBlock {
		return ErrPolynomialPending
	}
	value, e := p.read(s.positionID)
	if e != nil {
		return e
	}
	if value != s.positionValue {
		return ErrPolynomialPending
	}
	if e = p.manager.memory.WriteUint32(p.raw, 8+uint64(s.positionID)*4, ^uint32(0)); e != nil {
		return e
	}
	p.acknowledgeMarkerWrite(s.positionID)
	return nil
}

// The actual map destructor has consumed every row of this exact backing.
// A consumed-with-error release still retires its write token; a refused
// physical release leaves both block and marker intact for explicit retry.
func (p *MonomialPositions) acknowledgeReleasedMarker(block util.Block) {
	if p == nil || p.self != p || block == (util.Block{}) {
		return
	}
	s := p.markerFrame
	if s == nil || s.positionBlock != block || s.buffer == nil || s.buffer.owner == nil {
		return
	}
	r := polynomialArithmeticInvocationReceipt{manager: s.buffer.owner, frame: s, invocation: p.markerInvocation}
	if p.markerMatches(&r) {
		p.acknowledgeMarkerWrite(s.positionID)
	}
}
