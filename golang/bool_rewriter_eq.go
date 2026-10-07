// Source: src/ast/rewriter/bool_rewriter.cpp:586-722,939-997;
// src/ast/rewriter/bool_rewriter.h:128-156,167-186.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import "errors"

func (rw *boolRewriter) isBool(h ASTHandle) (bool, error) {
	s, e := rw.manager.GetSort(h)
	if e != nil {
		return false, e
	}
	b, e := rw.manager.MkBoolSort()
	return s == b, e
}
func (rw *boolRewriter) complement(a, b ASTHandle) (bool, error) {
	neg, inner, e := rw.manager.IsNot(a)
	if e != nil {
		return false, e
	}
	if neg && inner == b {
		return true, nil
	}
	neg, inner, e = rw.manager.IsNot(b)
	return neg && inner == a, e
}
func (rw *boolRewriter) binary(h ASTHandle, op int32) (bool, ASTHandle, ASTHandle, error) {
	yes, e := rw.isOperator(h, op)
	if e != nil || !yes {
		return false, ASTHandle{}, ASTHandle{}, e
	}
	a, e := rw.manager.GetArg(h, 0)
	if e != nil {
		return false, ASTHandle{}, ASTHandle{}, e
	}
	b, e := rw.manager.GetArg(h, 1)
	return true, a, b, e
}
func (rw *boolRewriter) ite(h ASTHandle) (bool, ASTHandle, ASTHandle, ASTHandle, error) {
	yes, e := rw.isOperator(h, ASTOpIte)
	if e != nil || !yes {
		return false, ASTHandle{}, ASTHandle{}, ASTHandle{}, e
	}
	c, e := rw.manager.GetArg(h, 0)
	if e != nil {
		return false, ASTHandle{}, ASTHandle{}, ASTHandle{}, e
	}
	t, e := rw.manager.GetArg(h, 1)
	if e != nil {
		return false, ASTHandle{}, ASTHandle{}, ASTHandle{}, e
	}
	f, e := rw.manager.GetArg(h, 2)
	return true, c, t, f, e
}

func (rw *boolRewriter) tryITEValue(ite, val ASTHandle, out ASTRef) (brStatus, error) {
	m := rw.manager
	yes, c, t, f, e := rw.ite(ite)
	if e != nil {
		return brFailed, e
	}
	if !yes {
		return brFailed, ErrASTDomain
	}
	tv, e := m.IsValue(t)
	if e != nil {
		return brFailed, e
	}
	fv, e := m.IsValue(f)
	if e != nil {
		return brFailed, e
	}
	if tv && fv {
		switch {
		case t != val && f != val:
			h, e := m.MkFalse()
			if e != nil {
				return brFailed, e
			}
			return brDone, m.SetRef(out, h)
		case t == val && f == val:
			h, e := m.MkTrue()
			if e != nil {
				return brFailed, e
			}
			return brDone, m.SetRef(out, h)
		case t == val:
			return brDone, m.SetRef(out, c)
		default:
			return brDone, rw.mkNot(c, out)
		}
	}
	if rw.iteExtraRules {
		if tv {
			if val == t {
				eq, e := m.MkEq(val, f)
				if e != nil {
					return brFailed, e
				}
				h, e := m.MkOr(c, eq)
				if e != nil {
					return brFailed, e
				}
				return brRewrite2, m.SetRef(out, h)
			}
			if e = rw.mkNot(c, out); e != nil {
				return brFailed, e
			}
			nc, e := m.RefGet(out)
			if e != nil {
				return brFailed, e
			}
			eq, e := m.MkEq(val, f)
			if e != nil {
				return brFailed, e
			}
			h, e := m.MkAnd(nc, eq)
			if e != nil {
				return brFailed, e
			}
			return brRewrite2, m.SetRef(out, h)
		}
		if fv {
			if val == f {
				if e = rw.mkNot(c, out); e != nil {
					return brFailed, e
				}
				nc, e := m.RefGet(out)
				if e != nil {
					return brFailed, e
				}
				eq, e := m.MkEq(val, t)
				if e != nil {
					return brFailed, e
				}
				h, e := m.MkOr(nc, eq)
				if e != nil {
					return brFailed, e
				}
				return brRewrite2, m.SetRef(out, h)
			}
			eq, e := m.MkEq(val, t)
			if e != nil {
				return brFailed, e
			}
			h, e := m.MkAnd(c, eq)
			if e != nil {
				return brFailed, e
			}
			return brRewrite2, m.SetRef(out, h)
		}
	}
	for branch := 0; branch < 2; branch++ {
		nested := t
		if branch == 1 {
			nested = f
		}
		yes, _, t2, f2, e := rw.ite(nested)
		if e != nil {
			return brFailed, e
		}
		if !yes {
			continue
		}
		tv, e := m.IsValue(t2)
		if e != nil {
			return brFailed, e
		}
		fv, e := m.IsValue(f2)
		if e != nil {
			return brFailed, e
		}
		if !tv || !fv {
			continue
		}
		if _, e = rw.tryITEValue(nested, val, out); e != nil {
			return brFailed, e
		}
		reduced, e := m.RefGet(out)
		if e != nil {
			return brFailed, e
		}
		var h ASTHandle
		if branch == 0 {
			eq, e := m.MkEq(f, val)
			if e != nil {
				return brFailed, e
			}
			h, e = m.MkIte(c, reduced, eq)
			if e != nil {
				return brFailed, e
			}
		} else {
			eq, e := m.MkEq(t, val)
			if e != nil {
				return brFailed, e
			}
			h, e = m.MkIte(c, eq, reduced)
			if e != nil {
				return brFailed, e
			}
		}
		return brRewrite2, m.SetRef(out, h)
	}
	return brFailed, nil
}

func (rw *boolRewriter) mkEqCore(lhs, rhs ASTHandle, out ASTRef) (brStatus, error) {
	m := rw.manager
	equal, e := m.AreEqual(lhs, rhs)
	if e != nil {
		return brFailed, e
	}
	if equal {
		h, e := m.MkTrue()
		if e != nil {
			return brFailed, e
		}
		return brDone, m.SetRef(out, h)
	}
	distinct, e := m.AreDistinct(lhs, rhs)
	if e != nil {
		return brFailed, e
	}
	if distinct {
		h, e := m.MkFalse()
		if e != nil {
			return brFailed, e
		}
		return brDone, m.SetRef(out, h)
	}
	leftITE, e := rw.isOperator(lhs, ASTOpIte)
	if e != nil {
		return brFailed, e
	}
	rightValue, e := m.IsValue(rhs)
	if e != nil {
		return brFailed, e
	}
	status := brFailed
	if leftITE && rightValue {
		status, e = rw.tryITEValue(lhs, rhs, out)
	} else {
		rightITE, e2 := rw.isOperator(rhs, ASTOpIte)
		if e2 != nil {
			return brFailed, e2
		}
		leftValue, e2 := m.IsValue(lhs)
		if e2 != nil {
			return brFailed, e2
		}
		if rightITE && leftValue {
			status, e = rw.tryITEValue(rhs, lhs, out)
		}
	}
	if e != nil || status != brFailed {
		return status, e
	}
	boolean, e := rw.isBool(lhs)
	if e != nil {
		return brFailed, e
	}
	if !boolean {
		return brFailed, nil
	}
	ln, li, e := m.IsNot(lhs)
	if e != nil {
		return brFailed, e
	}
	rn, ri, e := m.IsNot(rhs)
	if e != nil {
		return brFailed, e
	}
	unfolded := ln && rn
	if unfolded {
		lhs, rhs = li, ri
	}
	truth, e := m.MkTrue()
	if e != nil {
		return brFailed, e
	}
	falsity, e := m.MkFalse()
	if e != nil {
		return brFailed, e
	}
	switch {
	case lhs == truth:
		return brDone, m.SetRef(out, rhs)
	case lhs == falsity:
		return brDone, rw.mkNot(rhs, out)
	case rhs == truth:
		return brDone, m.SetRef(out, lhs)
	case rhs == falsity:
		return brDone, rw.mkNot(lhs, out)
	}
	comp, e := rw.complement(lhs, rhs)
	if e != nil {
		return brFailed, e
	}
	if comp {
		return brDone, m.SetRef(out, falsity)
	}
	if unfolded {
		h, e := m.MkEq(lhs, rhs)
		if e != nil {
			return brFailed, e
		}
		return brDone, m.SetRef(out, h)
	}
	liff, la, lb, e := rw.binary(lhs, ASTOpIff)
	if e != nil {
		return brFailed, e
	}
	if liff {
		riff, ra, rb, e := rw.binary(rhs, ASTOpIff)
		if e != nil {
			return brFailed, e
		}
		if riff {
			if la == ra {
				comp, e := rw.complement(rb, lb)
				if e != nil {
					return brFailed, e
				}
				if comp {
					return brDone, m.SetRef(out, falsity)
				}
			}
			if lb == rb {
				comp, e := rw.complement(ra, la)
				if e != nil {
					return brFailed, e
				}
				if comp {
					return brDone, m.SetRef(out, falsity)
				}
			}
		}
	}
	return brFailed, nil
}

func (rw *boolRewriter) mkNotCore(arg ASTHandle, out ASTRef) (status brStatus, err error) {
	m := rw.manager
	negative, inner, e := m.IsNot(arg)
	if e != nil {
		return brFailed, e
	}
	if negative {
		return brDone, m.SetRef(out, inner)
	}
	truth, e := m.MkTrue()
	if e != nil {
		return brFailed, e
	}
	falsity, e := m.MkFalse()
	if e != nil {
		return brFailed, e
	}
	if arg == truth {
		return brDone, m.SetRef(out, falsity)
	}
	if arg == falsity {
		return brDone, m.SetRef(out, truth)
	}
	eq, e := rw.isOperator(arg, ASTOpEq)
	if e != nil {
		return brFailed, e
	}
	if !eq {
		eq, e = rw.isOperator(arg, ASTOpIff)
		if e != nil {
			return brFailed, e
		}
	}
	if eq {
		a, e := m.GetArg(arg, 0)
		if e != nil {
			return brFailed, e
		}
		boolean, e := rw.isBool(a)
		if e != nil {
			return brFailed, e
		}
		if !boolean {
			return brFailed, nil
		}
		b, e := m.GetArg(arg, 1)
		if e != nil {
			return brFailed, e
		}
		tmp, e := m.NewRef(ASTHandle{})
		if e != nil {
			return brFailed, e
		}
		defer func() { err = errors.Join(err, m.DelRef(&tmp)) }()
		if e = rw.mkNot(a, tmp); e != nil {
			return brFailed, e
		}
		neg, e := m.RefGet(tmp)
		if e != nil {
			return brFailed, e
		}
		return brDone, rw.mkEq(neg, b, out)
	}
	return brFailed, nil
}

func (rw *boolRewriter) mkEq(a, b ASTHandle, out ASTRef) error {
	status, e := rw.mkEqCore(a, b, out)
	if e != nil || status != brFailed {
		return e
	}
	h, e := rw.manager.MkEq(a, b)
	if e != nil {
		return e
	}
	return rw.manager.SetRef(out, h)
}
func (rw *boolRewriter) mkNot(a ASTHandle, out ASTRef) error {
	status, e := rw.mkNotCore(a, out)
	if e != nil || status != brFailed {
		return e
	}
	h, e := rw.manager.MkNot(a)
	if e != nil {
		return e
	}
	return rw.manager.SetRef(out, h)
}
func (rw *boolRewriter) mkAnd(args []ASTHandle, out ASTRef) error {
	status, e := rw.mkAndCore(args, out)
	if e != nil || status != brFailed {
		return e
	}
	h, e := rw.manager.MkAndArgs(args)
	if e != nil {
		return e
	}
	return rw.manager.SetRef(out, h)
}
func (rw *boolRewriter) mkOr(args []ASTHandle, out ASTRef) error {
	var status brStatus
	var e error
	if rw.flat {
		status, e = rw.mkFlatOrCore(args, out)
	} else {
		status, e = rw.mkNFlatOrCore(args, out)
	}
	if e != nil || status != brFailed {
		return e
	}
	h, e := rw.manager.MkOrArgs(args)
	if e != nil {
		return e
	}
	return rw.manager.SetRef(out, h)
}
func (rw *boolRewriter) mkDerived(op uint8, args []ASTHandle, out ASTRef) (err error) {
	m := rw.manager
	tmp, e := m.NewRef(ASTHandle{})
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, m.DelRef(&tmp)) }()
	switch op {
	case 0, 1:
		if e = rw.mkNot(args[0], tmp); e != nil {
			return e
		}
		neg, e := m.RefGet(tmp)
		if e != nil {
			return e
		}
		if op == 0 {
			return rw.mkEq(neg, args[1], out)
		}
		return rw.mkOr([]ASTHandle{neg, args[1]}, out)
	case 2:
		if e = rw.mkAnd(args, tmp); e != nil {
			return e
		}
	case 3:
		if e = rw.mkOr(args, tmp); e != nil {
			return e
		}
	}
	h, e := m.RefGet(tmp)
	if e != nil {
		return e
	}
	return rw.mkNot(h, out)
}
func (rw *boolRewriter) mkXor(a, b ASTHandle, out ASTRef) error {
	return rw.mkDerived(0, []ASTHandle{a, b}, out)
}
func (rw *boolRewriter) mkImplies(a, b ASTHandle, out ASTRef) error {
	return rw.mkDerived(1, []ASTHandle{a, b}, out)
}
func (rw *boolRewriter) mkNand(args []ASTHandle, out ASTRef) error { return rw.mkDerived(2, args, out) }
func (rw *boolRewriter) mkNor(args []ASTHandle, out ASTRef) error  { return rw.mkDerived(3, args, out) }
