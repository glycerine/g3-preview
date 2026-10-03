// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

func (m *ASTManager) mkFamilyApp(fid FamilyID, k int32, ps []ASTParameter, args []ASTHandle) (astNodeRef, error) {
	if fid < 0 || fid >= 5 || !m.plugins[fid].present {
		return astNodeRef{}, nil
	}
	if uint64(len(args)) > math.MaxUint32 {
		return astNodeRef{}, arena.ErrBounds
	}
	// The basic expr overload bypasses decl_plugin's domain buffer for these
	// branches. This matters for overflow-reservation actions on large arities.
	if fid == 0 {
		var direct astNodeRef
		switch k {
		case ASTOpTrue, ASTOpFalse, ASTOpAnd, ASTOpOr, ASTOpNot, ASTOpInterp, ASTOpIff, ASTOpImplies, ASTOpXor:
			direct = m.basic.declarations[k]
		case ASTOpIte:
			if len(args) != 3 {
				return astNodeRef{}, nil
			}
			a, e := m.requireExpr(args[1])
			if e != nil {
				return astNodeRef{}, e
			}
			b, e := m.requireExpr(args[2])
			if e != nil {
				return astNodeRef{}, e
			}
			a, e = m.sortOf(a)
			if e != nil {
				return astNodeRef{}, e
			}
			b, e = m.sortOf(b)
			if e != nil {
				return astNodeRef{}, e
			}
			joined, e := m.joinSorts(a, b)
			if e != nil {
				return astNodeRef{}, e
			}
			direct, e = m.cachedIte(joined)
			if e != nil {
				return astNodeRef{}, e
			}
		case ASTOpEq, ASTOpOEq:
			if len(args) < 2 {
				return astNodeRef{}, nil
			}
			r, e := m.requireExpr(args[0])
			if e != nil {
				return astNodeRef{}, e
			}
			r, e = m.sortOf(r)
			if e != nil {
				return astNodeRef{}, e
			}
			for _, h := range args[1:] {
				other, e := m.requireExpr(h)
				if e != nil {
					return astNodeRef{}, e
				}
				other, e = m.sortOf(other)
				if e != nil {
					return astNodeRef{}, e
				}
				r, e = m.joinSorts(r, other)
				if e != nil {
					return astNodeRef{}, e
				}
			}
			direct, e = m.cachedEquality(k, r)
			if e != nil {
				return astNodeRef{}, e
			}
		}
		if direct.slot != 0 {
			return m.mkApp(direct, args)
		}
		if k >= ASTPrUndef && k < ASTLastBasicPR {
			if k == ASTPrUndef {
				if len(args) != 0 {
					return astNodeRef{}, ErrASTDomain
				}
			} else {
				if len(args) == 0 {
					return astNodeRef{}, ErrASTDomain
				}
				for _, h := range args[:len(args)-1] {
					r, e := m.requireExpr(h)
					if e != nil {
						return astNodeRef{}, e
					}
					r, e = m.sortOf(r)
					if e != nil {
						return astNodeRef{}, e
					}
					if r != m.basic.proofSort {
						return astNodeRef{}, ErrASTDomain
					}
				}
				last, e := m.requireExpr(args[len(args)-1])
				if e != nil {
					return astNodeRef{}, e
				}
				last, e = m.sortOf(last)
				if e != nil {
					return astNodeRef{}, e
				}
				if last != m.basic.boolSort && last != m.basic.proofSort {
					return astNodeRef{}, ErrASTDomain
				}
			}
			d, e := m.proofDecl(k, ps, uint32(len(args))-1)
			if e != nil || d.slot == 0 {
				return astNodeRef{}, e
			}
			return m.mkApp(d, args)
		}
	}
	// Source decl_plugin's expr overload builds a lexical ptr_buffer of sorts;
	// basic directly routes common operators but the same sorts have no callbacks.
	var inline [16]ASTHandle
	domain := inline[:0]
	var bytes uint64
	defer func() { m.budget.Release(bytes) }()
	for _, h := range args {
		r, e := m.requireExpr(h)
		if e != nil {
			return astNodeRef{}, e
		}
		s, e := m.sortOf(r)
		if e != nil {
			return astNodeRef{}, e
		}
		if len(domain) == cap(domain) {
			n := uint64(cap(domain)) * 2
			width := uint64(reflect.TypeOf(ASTHandle{}).Size())
			if n > math.MaxUint32 || n > uint64(^uint(0)>>1)/width {
				return astNodeRef{}, arena.ErrBounds
			}
			nextBytes := n * width
			if e = m.budget.Reserve(nextBytes); e != nil {
				return astNodeRef{}, e
			}
			next := make([]ASTHandle, len(domain), int(n))
			copy(next, domain)
			m.budget.Release(bytes)
			bytes = nextBytes
			domain = next
		}
		domain = append(domain, m.external(s))
	}
	var decl astNodeRef
	var e error
	if fid == 0 {
		decl, e = m.basicDecl(k, ps, domain)
	} else {
		decl, e = m.builtinDecl(fid, k, ps, domain)
	}
	if e != nil || decl.slot == 0 {
		return astNodeRef{}, e
	}
	return m.mkApp(decl, args)
}
func (m *ASTManager) MkFamilyApp(fid FamilyID, k int32, ps []ASTParameter, args []ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	r, e := m.mkFamilyApp(fid, k, ps, args)
	return m.external(r), e
}

// All five eager source plugins ignore the optional range in this overload.
func (m *ASTManager) MkFamilyFuncDecl(fid FamilyID, k int32, ps []ASTParameter, domain []ASTHandle, rangeSort ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	if fid < 0 || fid >= 5 || !m.plugins[fid].present {
		return ASTHandle{}, nil
	}
	if uint64(len(domain)) > math.MaxUint32 {
		return ASTHandle{}, arena.ErrBounds
	}
	var r astNodeRef
	var e error
	if fid == 0 {
		r, e = m.basicDecl(k, ps, domain)
	} else {
		r, e = m.builtinDecl(fid, k, ps, domain)
	}
	return m.external(r), e
}

func (m *ASTManager) MkFamilySort(fid FamilyID, k int32, ps []ASTParameter) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	if fid < 0 || fid >= 5 || !m.plugins[fid].present {
		return ASTHandle{}, nil
	}
	var r astNodeRef
	var e error
	switch fid {
	case 0:
		if k == 0 {
			r = m.basic.boolSort
		} else {
			if astSourceDebug && k != 1 {
				return ASTHandle{}, ErrASTDomain
			}
			r = m.basic.proofSort
		}
	case 4:
		r, e = m.userSort(k, ps)
	default:
		return ASTHandle{}, ErrASTDomain // source UNREACHABLE followed by null
	}
	return m.external(r), e
}
