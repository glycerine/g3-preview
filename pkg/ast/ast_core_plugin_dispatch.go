// Source: src/ast/ast.cpp:1568-1638; src/ast/ast.h:1323-1402.
package ast

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

func (m *ASTManager) mkFamilyApp(fid FamilyID, k int32, ps []ASTParameter, args []ASTHandle) (astNodeRef, error) {
	if fid < 0 || !m.pluginAt(fid).present {
		return astNodeRef{}, nil
	}
	if uint64(len(args)) > math.MaxUint32 {
		return astNodeRef{}, arena.ErrBounds
	}
	if m.pluginAt(fid).kind == astArithmeticPlugin {
		previous := m.formatExecuting
		m.formatExecuting = true
		defer func() { m.formatExecuting = previous }()
		d, e := m.arithmeticExprDecl(k, ps, args)
		if e != nil || d.slot == 0 {
			return astNodeRef{}, e
		}
		return m.mkApp(d, args)
	}
	if m.pluginAt(fid).kind == astBitVectorPlugin {
		previous := m.formatExecuting
		m.formatExecuting = true
		defer func() { m.formatExecuting = previous }()
		d, handled, e := m.bitVectorExprDecl(k, ps, args)
		if e != nil {
			return astNodeRef{}, e
		}
		if handled {
			if d.slot == 0 {
				return astNodeRef{}, nil
			}
			return m.mkApp(d, args)
		}
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
	return m.mkFamilyAppDomainArguments(fid, k, ps, SliceArguments(args))
}

func (m *ASTManager) mkFamilyAppDomainArguments(fid FamilyID, k int32, ps []ASTParameter, args AppArguments) (astNodeRef, error) {
	// Source decl_plugin's expr overload builds a lexical ptr_buffer of sorts;
	// basic directly routes common operators but the same sorts have no callbacks.
	var inline [16]ASTHandle
	domain := inline[:0]
	var bytes uint64
	defer func() { m.BudgetField.Release(bytes) }()
	for i := uint32(0); i < args.Count(); i++ {
		h, e := args.At(i)
		if e != nil {
			return astNodeRef{}, e
		}
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
			if e = m.BudgetField.Reserve(nextBytes); e != nil {
				return astNodeRef{}, e
			}
			next := make([]ASTHandle, len(domain), int(n))
			copy(next, domain)
			m.BudgetField.Release(bytes)
			bytes = nextBytes
			domain = next
		}
		domain = append(domain, m.external(s))
	}
	var decl astNodeRef
	var e error
	if fid == 0 {
		decl, e = m.basicDecl(k, ps, domain)
	} else if m.pluginAt(fid).kind == astArrayPlugin {
		decl, e = m.arrayDecl(k, ps, domain, ASTHandle{})
	} else if m.pluginAt(fid).kind == astFormatPlugin {
		decl, e = m.formatDecl(k, ps, domain)
	} else {
		decl, e = m.builtinDecl(fid, k, ps, domain)
	}
	if e != nil || decl.slot == 0 {
		return astNodeRef{}, e
	}
	return m.mkAppArguments(decl, args)
}
func (m *ASTManager) MkFamilyApp(fid FamilyID, k int32, ps []ASTParameter, args []ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	r, e := m.mkFamilyApp(fid, k, ps, args)
	return m.external(r), e
}

// The eager base plugins ignore optional range; array const declarations use it.
func (m *ASTManager) MkFamilyFuncDecl(fid FamilyID, k int32, ps []ASTParameter, domain []ASTHandle, rangeSort ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	return m.mkFamilyFuncDecl(fid, k, ps, domain, rangeSort)
}

func (m *ASTManager) mkFamilyFuncDecl(fid FamilyID, k int32, ps []ASTParameter, domain []ASTHandle, rangeSort ASTHandle) (ASTHandle, error) {
	if fid < 0 || !m.pluginAt(fid).present {
		return ASTHandle{}, nil
	}
	if uint64(len(domain)) > math.MaxUint32 {
		return ASTHandle{}, arena.ErrBounds
	}
	var r astNodeRef
	var e error
	if m.pluginAt(fid).kind == astArithmeticPlugin {
		previous := m.formatExecuting
		m.formatExecuting = true
		defer func() { m.formatExecuting = previous }()
		r, e = m.arithmeticDecl(k, ps, domain)
	} else if fid == 0 {
		r, e = m.basicDecl(k, ps, domain)
	} else if m.pluginAt(fid).kind == astArrayPlugin {
		r, e = m.arrayDecl(k, ps, domain, rangeSort)
	} else if m.pluginAt(fid).kind == astFormatPlugin {
		r, e = m.formatDecl(k, ps, domain)
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
	return m.mkFamilySort(fid, k, ps)
}

func (m *ASTManager) mkFamilySort(fid FamilyID, k int32, ps []ASTParameter) (ASTHandle, error) {
	if fid < 0 || !m.pluginAt(fid).present {
		return ASTHandle{}, nil
	}
	var r astNodeRef
	var e error
	if m.pluginAt(fid).kind == astBitVectorPlugin {
		r, e = m.bitVectorSort(ps)
		return m.external(r), e
	}
	if m.pluginAt(fid).kind == astFormatPlugin {
		if ASTSourceDebug && k != 0 {
			return ASTHandle{}, ErrASTDomain
		}
		return m.external(m.formatPlugin.sort), nil
	}

	if m.pluginAt(fid).kind == astFloatingPointPlugin {
		m.formatExecuting = true
		defer func() { m.formatExecuting = false }()
		r, e = m.floatingPointSort(k, ps)
		return m.external(r), e
	}
	if m.pluginAt(fid).kind == astSequencePlugin {
		m.formatExecuting = true
		defer func() { m.formatExecuting = false }()
		if e = m.initializeSequence(m.sequencePlugin); e != nil {
			return ASTHandle{}, e
		}
		var param *astParameterRecord
		if k != 2 {
			if len(ps) != 1 {
				return ASTHandle{}, ErrASTDomain
			}
			param, e = m.parameter(ps[0])
			if e != nil {
				return ASTHandle{}, e
			}
		}
		r, e = m.sequenceSort(m.sequencePlugin, k, param)
		return m.external(r), e
	}
	if m.pluginAt(fid).kind == astArrayPlugin {
		m.formatExecuting = true
		defer func() { m.formatExecuting = false }()
		r, e = m.arraySort(k, ps)
		return m.external(r), e
	}
	if m.pluginAt(fid).kind == astArithmeticPlugin {
		if k == 0 || k == 1 {
			return m.external(m.arithmeticPlugin.fields[k]), nil
		}
		return ASTHandle{}, nil
	}
	switch fid {
	case 0:
		if k == 0 {
			r = m.basic.boolSort
		} else {
			if ASTSourceDebug && k != 1 {
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
