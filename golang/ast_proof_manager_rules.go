// Licensed under the MIT license.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

// Lexical source ptr_buffer<expr>: inline16, doubling overflow, full old+new
// charge. Handles are scalar; the helper retains no borrowed caller backing.
type astProofArguments struct {
	inline  [16]ASTHandle
	count   int
	spill   []ASTHandle
	charged uint64
}

func (a *astProofArguments) slice() []ASTHandle {
	if a.spill == nil {
		return a.inline[:a.count]
	}
	return a.spill
}
func (a *astProofArguments) push(m *ASTManager, h ASTHandle) error {
	if a.spill == nil && a.count < len(a.inline) {
		a.inline[a.count] = h
		a.count++
		return nil
	}
	if a.spill == nil || len(a.spill) == cap(a.spill) {
		n := uint64(32)
		if a.spill != nil {
			n = uint64(cap(a.spill)) * 2
		}
		width := uint64(reflect.TypeOf(ASTHandle{}).Size())
		if n > math.MaxUint32 || n > uint64(^uint(0)>>1)/width {
			return arena.ErrBounds
		}
		bytes := n * width
		if e := m.budget.Reserve(bytes); e != nil {
			return e
		}
		next := make([]ASTHandle, a.count, int(n))
		copy(next, a.slice())
		m.budget.Release(a.charged)
		a.charged = bytes
		a.spill = next
	}
	a.spill = append(a.spill, h)
	a.count++
	return nil
}
func (a *astProofArguments) append(m *ASTManager, hs []ASTHandle) error {
	for _, h := range hs {
		if e := a.push(m, h); e != nil {
			return e
		}
	}
	return nil
}
func (a *astProofArguments) close(m *ASTManager) {
	m.budget.Release(a.charged)
	a.spill = nil
	a.count = 0
	a.charged = 0
}
func (m *ASTManager) proofParentsFact(k int32, ps []ASTHandle, f ASTHandle) (ASTHandle, error) {
	var args astProofArguments
	defer args.close(m)
	if e := args.append(m, ps); e != nil {
		return ASTHandle{}, e
	}
	if e := args.push(m, f); e != nil {
		return ASTHandle{}, e
	}
	return m.proofMake(k, args.slice()...)
}
func (m *ASTManager) MkProofCommutativity(h ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	r, e := m.requireKind(h, ASTApp)
	if e != nil {
		return ASTHandle{}, e
	}
	v := m.appAt(r.slot)
	if astSourceDebug && v.count != 2 {
		return ASTHandle{}, ErrASTDomain
	}
	a, e := m.proofArg(h, 1)
	if e != nil {
		return ASTHandle{}, e
	}
	b, e := m.proofArg(h, 0)
	if e != nil {
		return ASTHandle{}, e
	}
	args := [2]ASTHandle{a, b}
	swapped, e := m.mkApp(v.decl, args[:])
	if e != nil {
		return ASTHandle{}, e
	}
	fact, e := m.proofEq(h, m.external(swapped), false)
	if e != nil {
		return ASTHandle{}, e
	}
	return m.proofMake(ASTPrCommutativity, fact)
}
func (m *ASTManager) MkProofIffTrue(p ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	if p.IsNull() {
		return p, nil
	}
	f, e := m.proofFact(p)
	if e != nil {
		return ASTHandle{}, e
	}
	if e = m.proofCheckBool(f); e != nil {
		return ASTHandle{}, e
	}
	fact, e := m.proofMake(ASTOpIff, f, m.external(m.trueApp))
	if e != nil {
		return ASTHandle{}, e
	}
	return m.proofMake(ASTPrIffTrue, p, fact)
}
func (m *ASTManager) MkProofIffFalse(p ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	if p.IsNull() {
		return p, nil
	}
	f, e := m.proofFact(p)
	if e != nil {
		return ASTHandle{}, e
	}
	if astSourceDebug {
		yes, e := m.proofKind(f, ASTOpNot)
		if e != nil {
			return ASTHandle{}, e
		}
		if !yes {
			return ASTHandle{}, ErrASTDomain
		}
	}
	a, e := m.proofArg(f, 0)
	if e != nil {
		return ASTHandle{}, e
	}
	fact, e := m.proofMake(ASTOpIff, a, m.external(m.falseApp))
	if e != nil {
		return ASTHandle{}, e
	}
	return m.proofMake(ASTPrIffFalse, p, fact)
}
func (m *ASTManager) MkProofLemma(p, f ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	if p.IsNull() {
		return p, nil
	}
	if astSourceDebug {
		fact, e := m.proofFact(p)
		if e != nil {
			return ASTHandle{}, e
		}
		if fact != m.external(m.falseApp) {
			return ASTHandle{}, ErrASTDomain
		}
	}
	return m.proofMake(ASTPrLemma, p, f)
}
func (m *ASTManager) MkProofDistributivity(a, b ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	f, e := m.proofEq(a, b, false)
	if e != nil {
		return ASTHandle{}, e
	}
	return m.proofMake(ASTPrDistributivity, f)
}
func (m *ASTManager) proofQuantIntro(a, b, p ASTHandle, oeq bool) (ASTHandle, error) {
	if p.IsNull() {
		return ASTHandle{}, nil
	}
	if astSourceDebug {
		r, e := m.requireKind(a, ASTQuantifier)
		if e != nil {
			return ASTHandle{}, e
		}
		s, e := m.requireKind(b, ASTQuantifier)
		if e != nil {
			return ASTHandle{}, e
		}
		if m.quantifierAt(r.slot).count != m.quantifierAt(s.slot).count {
			return ASTHandle{}, ErrASTDomain
		}
		f, e := m.proofFact(p)
		if e != nil {
			return ASTHandle{}, e
		}
		k := ASTOpIff
		if oeq {
			k = ASTOpOEq
		}
		yes, e := m.proofKind(f, k)
		if e != nil {
			return ASTHandle{}, e
		}
		if !yes {
			return ASTHandle{}, ErrASTDomain
		}
	}
	k := ASTOpIff
	if oeq {
		k = ASTOpOEq
	}
	fact, e := m.proofMake(k, a, b)
	if e != nil {
		return ASTHandle{}, e
	}
	return m.proofMake(ASTPrQuantIntro, p, fact)
}
func (m *ASTManager) MkProofQuantIntro(a, b, p ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	return m.proofQuantIntro(a, b, p, false)
}
func (m *ASTManager) MkProofOEqQuantIntro(a, b, p ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	return m.proofQuantIntro(a, b, p, true)
}
func (m *ASTManager) proofGuardedRelation(k int32, a, b ASTHandle, oeq bool) (ASTHandle, error) {
	if m.proofMode == ASTProofDisabled {
		return ASTHandle{}, nil
	}
	var f ASTHandle
	var e error
	if oeq {
		f, e = m.proofEq(a, b, true)
	} else {
		f, e = m.proofMake(ASTOpIff, a, b)
	}
	if e != nil {
		return ASTHandle{}, e
	}
	return m.proofMake(k, f)
}
func (m *ASTManager) MkProofPullQuant(a, q ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	return m.proofGuardedRelation(ASTPrPullQuant, a, q, false)
}
func (m *ASTManager) MkProofPushQuant(q, a ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	return m.proofGuardedRelation(ASTPrPushQuant, q, a, false)
}
func (m *ASTManager) MkProofElimUnusedVars(q, a ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	return m.proofGuardedRelation(ASTPrElimUnusedVars, q, a, false)
}
func (m *ASTManager) MkProofDER(q, a ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	return m.proofGuardedRelation(ASTPrDER, q, a, false)
}
func (m *ASTManager) MkProofSkolemization(q, a ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	if m.proofMode == ASTProofDisabled {
		return ASTHandle{}, nil
	}
	if e := m.proofCheckBool(q); e != nil {
		return ASTHandle{}, e
	}
	if e := m.proofCheckBool(a); e != nil {
		return ASTHandle{}, e
	}
	return m.proofGuardedRelation(ASTPrSkolemize, q, a, true)
}
func (m *ASTManager) MkProofDefAxiom(a ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	return m.proofGuardedUnary(ASTPrDefAxiom, a, false)
}
func (m *ASTManager) MkProofRewriteStar(a, b ASTHandle, ps []ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	if m.proofMode == ASTProofDisabled {
		return ASTHandle{}, nil
	}
	var args astProofArguments
	defer args.close(m)
	if e := args.append(m, ps); e != nil {
		return ASTHandle{}, e
	}
	f, e := m.proofEq(a, b, false)
	if e != nil {
		return ASTHandle{}, e
	}
	if e = args.push(m, f); e != nil {
		return ASTHandle{}, e
	}
	return m.proofMake(ASTPrRewriteStar, args.slice()...)
}
func (m *ASTManager) MkProofApplyDefs(a, b ASTHandle, ps []ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	if m.proofMode == ASTProofDisabled {
		return ASTHandle{}, nil
	}
	var args astProofArguments
	defer args.close(m)
	if e := args.append(m, ps); e != nil {
		return ASTHandle{}, e
	}
	f, e := m.proofEq(a, b, true)
	if e != nil {
		return ASTHandle{}, e
	}
	if e = args.push(m, f); e != nil {
		return ASTHandle{}, e
	}
	return m.proofMake(ASTPrApplyDef, args.slice()...)
}
func (m *ASTManager) MkProofApplyDef(a, b, p ASTHandle) (ASTHandle, error) {
	ps := [1]ASTHandle{p}
	return m.MkProofApplyDefs(a, b, ps[:])
}
func (m *ASTManager) proofNNFParents(ps []ASTHandle) (bool, error) {
	for _, p := range ps {
		has, e := m.proofHasFact(p)
		if e != nil {
			return false, e
		}
		if !has {
			return false, nil
		}
		f, e := m.proofFact(p)
		if e != nil {
			return false, e
		}
		yes, e := m.proofKind(f, ASTOpOEq)
		if e != nil {
			return false, e
		}
		if !yes {
			return false, nil
		}
	}
	return true, nil
}
func (m *ASTManager) CheckNNFProofParents(ps []ASTHandle) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	return m.proofNNFParents(ps)
}
func (m *ASTManager) proofNNF(a, b ASTHandle, ps []ASTHandle, negative bool) (ASTHandle, error) {
	if m.proofMode == ASTProofDisabled {
		return ASTHandle{}, nil
	}
	if _, e := m.proofNNFParents(ps); e != nil {
		return ASTHandle{}, e
	}
	var args astProofArguments
	defer args.close(m)
	if e := args.append(m, ps); e != nil {
		return ASTHandle{}, e
	}
	k := ASTPrNNFPos
	if negative {
		var e error
		a, e = m.proofMake(ASTOpNot, a)
		if e != nil {
			return ASTHandle{}, e
		}
		k = ASTPrNNFNeg
	}
	f, e := m.proofEq(a, b, true)
	if e != nil {
		return ASTHandle{}, e
	}
	if e = args.push(m, f); e != nil {
		return ASTHandle{}, e
	}
	return m.proofMake(k, args.slice()...)
}
func (m *ASTManager) MkProofNNFPos(a, b ASTHandle, ps []ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	return m.proofNNF(a, b, ps, false)
}
func (m *ASTManager) MkProofNNFNeg(a, b ASTHandle, ps []ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	return m.proofNNF(a, b, ps, true)
}
func (m *ASTManager) MkProofIffOEq(p ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	if p.IsNull() {
		return p, nil
	}
	f, e := m.proofFact(p)
	if e != nil {
		return ASTHandle{}, e
	}
	oeq, e := m.proofKind(f, ASTOpOEq)
	if e != nil {
		return ASTHandle{}, e
	}
	if astSourceDebug {
		iff, e := m.proofKind(f, ASTOpIff)
		if e != nil {
			return ASTHandle{}, e
		}
		if !oeq && !iff {
			return ASTHandle{}, ErrASTDomain
		}
	}
	if oeq {
		return p, nil
	}
	a, e := m.proofArg(f, 0)
	if e != nil {
		return ASTHandle{}, e
	}
	b, e := m.proofArg(f, 1)
	if e != nil {
		return ASTHandle{}, e
	}
	fact, e := m.proofEq(a, b, true)
	if e != nil {
		return ASTHandle{}, e
	}
	return m.proofMake(ASTPrIffOEq, p, fact)
}
func (m *ASTManager) MkProofAndElim(p ASTHandle, i uint32) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	if m.proofMode == ASTProofDisabled {
		return ASTHandle{}, nil
	}
	f, e := m.proofFact(p)
	if e != nil {
		return ASTHandle{}, e
	}
	if astSourceDebug {
		yes, e := m.proofKind(f, ASTOpAnd)
		if e != nil {
			return ASTHandle{}, e
		}
		if !yes {
			return ASTHandle{}, ErrASTDomain
		}
	}
	a, e := m.proofArg(f, i)
	if e != nil {
		return ASTHandle{}, e
	}
	return m.proofMake(ASTPrAndElim, p, a)
}
func (m *ASTManager) MkProofNotOrElim(p ASTHandle, i uint32) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	if m.proofMode == ASTProofDisabled {
		return ASTHandle{}, nil
	}
	f, e := m.proofFact(p)
	if e != nil {
		return ASTHandle{}, e
	}
	if astSourceDebug {
		yes, e := m.proofKind(f, ASTOpNot)
		if e != nil {
			return ASTHandle{}, e
		}
		if !yes {
			return ASTHandle{}, ErrASTDomain
		}
	}
	or, e := m.proofArg(f, 0)
	if e != nil {
		return ASTHandle{}, e
	}
	if astSourceDebug {
		yes, e := m.proofKind(or, ASTOpOr)
		if e != nil {
			return ASTHandle{}, e
		}
		if !yes {
			return ASTHandle{}, ErrASTDomain
		}
	}
	a, e := m.proofArg(or, i)
	if e != nil {
		return ASTHandle{}, e
	}
	if yes, e := m.proofKind(a, ASTOpNot); e != nil {
		return ASTHandle{}, e
	} else if yes {
		a, e = m.proofArg(a, 0)
		if e != nil {
			return ASTHandle{}, e
		}
	} else {
		a, e = m.proofMake(ASTOpNot, a)
		if e != nil {
			return ASTHandle{}, e
		}
	}
	return m.proofMake(ASTPrNotOrElim, p, a)
}
