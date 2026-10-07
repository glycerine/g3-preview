// Source: src/ast/rewriter/bool_rewriter.cpp:268-579;
// src/ast/rewriter/bool_rewriter.h:70-84.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package rewriter

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
)

func (rw *boolRewriter) simpNestedNotOr(app ast.ASTHandle, neg *ast.ASTFastMark1, pos *ast.ASTFastMark2, out ast.ASTRef) (simplified bool, err error) {
	m := rw.manager
	var newArgs util.PointerBuffer[ast.ASTHandle]
	defer newArgs.Close(m.Budget())
	n, e := m.GetNumArgs(app)
	if e != nil {
		return false, e
	}
	simp := false
	rw.localContextCost += n
	for i := uint32(0); i < n; i++ {
		arg, e := m.GetArg(app, i)
		if e != nil {
			return false, e
		}
		marked, e := neg.IsMarked(arg)
		if e != nil {
			return false, e
		}
		if marked {
			h, e := m.MkFalse()
			if e != nil {
				return false, e
			}
			return true, m.SetRef(out, h)
		}
		marked, e = pos.IsMarked(arg)
		if e != nil {
			return false, e
		}
		if marked {
			simp = true
			continue
		}
		negative, atom, e := m.IsNot(arg)
		if e != nil {
			return false, e
		}
		if negative {
			marked, e = neg.IsMarked(atom)
			if e != nil {
				return false, e
			}
			if marked {
				simp = true
				continue
			}
			marked, e = pos.IsMarked(atom)
			if e != nil {
				return false, e
			}
			if marked {
				h, e := m.MkFalse()
				if e != nil {
					return false, e
				}
				return true, m.SetRef(out, h)
			}
		}
		if e = newArgs.Push(m.Budget(), arg); e != nil {
			return false, e
		}
	}
	if !simp {
		return false, nil
	}
	args := newArgs.Slice()
	switch len(args) {
	case 0:
		h, e := m.MkTrue()
		if e != nil {
			return false, e
		}
		return true, m.SetRef(out, h)
	case 1:
		return true, rw.mkNot(args[0], out)
	default:
		h, e := m.MkOrArgs(args)
		if e != nil {
			return false, e
		}
		h, e = m.MkNot(h)
		if e != nil {
			return false, e
		}
		return true, m.SetRef(out, h)
	}
}

func (rw *boolRewriter) simpLocalArg(arg ast.ASTHandle, neg *ast.ASTFastMark1, pos *ast.ASTFastMark2, modified *bool) (ast.ASTHandle, error) {
	m := rw.manager
	negative, atom, e := m.IsNot(arg)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	if !negative {
		atom = arg
	}
	marked, e := neg.IsMarked(atom)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	if marked {
		*modified = true
		return m.MkBoolVal(!negative)
	}
	marked, e = pos.IsMarked(atom)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	if marked {
		*modified = true
		return m.MkBoolVal(negative)
	}
	return arg, nil
}

// This source helper deliberately uses raw manager OR/NOT/ITE construction to
// avoid recursively entering local context simplification through mk_ite.
func (rw *boolRewriter) mkNestedITE(c, t, f ast.ASTHandle, out ast.ASTRef) (err error) {
	m := rw.manager
	truth, e := m.MkTrue()
	if e != nil {
		return e
	}
	falsity, e := m.MkFalse()
	if e != nil {
		return e
	}
	if c == truth {
		return m.SetRef(out, t)
	}
	if c == falsity {
		return m.SetRef(out, f)
	}
	if t == f {
		return m.SetRef(out, t)
	}
	boolean, e := rw.isBool(t)
	if e != nil {
		return e
	}
	var temps [2]ast.ASTRef
	size := 0
	newTemp := func() (ast.ASTRef, error) {
		v, e := m.NewRef(ast.ASTHandle{})
		if e != nil {
			return ast.ASTRef{}, e
		}
		temps[size] = v
		size++
		return v, nil
	}
	defer func() {
		for size > 0 {
			size--
			err = errors.Join(err, m.DelRef(&temps[size]))
		}
	}()
	rawOr := func(a, b ast.ASTHandle) error {
		h, e := m.MkOrArgs([]ast.ASTHandle{a, b})
		if e != nil {
			return e
		}
		return m.SetRef(out, h)
	}
	if boolean {
		if t == truth {
			if f == falsity {
				return m.SetRef(out, c)
			}
			return rawOr(c, f)
		}
		if t == falsity {
			if f == truth {
				return rw.mkNot(c, out)
			}
			tmp, e := newTemp()
			if e != nil {
				return e
			}
			if e = rw.mkNot(f, tmp); e != nil {
				return e
			}
			h, e := m.RefGet(tmp)
			if e != nil {
				return e
			}
			h, e = m.MkOrArgs([]ast.ASTHandle{c, h})
			if e != nil {
				return e
			}
			h, e = m.MkNot(h)
			if e != nil {
				return e
			}
			return m.SetRef(out, h)
		}
		if f == truth {
			tmp, e := newTemp()
			if e != nil {
				return e
			}
			if e = rw.mkNot(c, tmp); e != nil {
				return e
			}
			h, e := m.RefGet(tmp)
			if e != nil {
				return e
			}
			return rawOr(h, t)
		}
		if f == falsity || c == f {
			tmp1, e := newTemp()
			if e != nil {
				return e
			}
			tmp2, e := newTemp()
			if e != nil {
				return e
			}
			if e = rw.mkNot(c, tmp1); e != nil {
				return e
			}
			if e = rw.mkNot(t, tmp2); e != nil {
				return e
			}
			a, e := m.RefGet(tmp1)
			if e != nil {
				return e
			}
			b, e := m.RefGet(tmp2)
			if e != nil {
				return e
			}
			h, e := m.MkOrArgs([]ast.ASTHandle{a, b})
			if e != nil {
				return e
			}
			h, e = m.MkNot(h)
			if e != nil {
				return e
			}
			return m.SetRef(out, h)
		}
		if c == t {
			return rawOr(c, f)
		}
		negative, atom, e := m.IsNot(t)
		if e != nil {
			return e
		}
		if negative && atom == f {
			return rw.mkEq(c, t, out)
		}
		negative, atom, e = m.IsNot(f)
		if e != nil {
			return e
		}
		if negative && atom == t {
			return rw.mkEq(c, t, out)
		}
	}
	h, e := m.MkIte(c, t, f)
	if e != nil {
		return e
	}
	return m.SetRef(out, h)
}

func (rw *boolRewriter) simpNestedEqITE(arg ast.ASTHandle, neg *ast.ASTFastMark1, pos *ast.ASTFastMark2, out ast.ASTRef) (bool, error) {
	rw.localContextCost += 3
	negative, atom, e := rw.manager.IsNot(arg)
	if e != nil {
		return false, e
	}
	if negative {
		arg = atom
	}
	eq, e := rw.isOperator(arg, ast.ASTOpIff)
	if e != nil {
		return false, e
	}
	if !eq {
		eq, e = rw.isOperator(arg, ast.ASTOpEq)
		if e != nil {
			return false, e
		}
	}
	modified := false
	if eq {
		a, e := rw.manager.GetArg(arg, 0)
		if e != nil {
			return false, e
		}
		b, e := rw.manager.GetArg(arg, 1)
		if e != nil {
			return false, e
		}
		a, e = rw.simpLocalArg(a, neg, pos, &modified)
		if e != nil {
			return false, e
		}
		b, e = rw.simpLocalArg(b, neg, pos, &modified)
		if e != nil {
			return false, e
		}
		if !modified {
			return false, nil
		}
		if e = rw.mkEq(a, b, out); e != nil {
			return false, e
		}
		if negative {
			h, e := rw.manager.RefGet(out)
			if e != nil {
				return false, e
			}
			if e = rw.mkNot(h, out); e != nil {
				return false, e
			}
		}
		return true, nil
	}
	ite, c, t, f, e := rw.ite(arg)
	if e != nil || !ite {
		return false, e
	}
	c, e = rw.simpLocalArg(c, neg, pos, &modified)
	if e != nil {
		return false, e
	}
	t, e = rw.simpLocalArg(t, neg, pos, &modified)
	if e != nil {
		return false, e
	}
	f, e = rw.simpLocalArg(f, neg, pos, &modified)
	if e != nil {
		return false, e
	}
	if !modified {
		return false, nil
	}
	if e = rw.mkNestedITE(c, t, f, out); e != nil {
		return false, e
	}
	if negative {
		h, e := rw.manager.RefGet(out)
		if e != nil {
			return false, e
		}
		if e = rw.mkNot(h, out); e != nil {
			return false, e
		}
	}
	return true, nil
}

func (rw *boolRewriter) pushLocalArg(arg ast.ASTHandle, v ast.ASTRefVector, neg *ast.ASTFastMark1, pos *ast.ASTFastMark2) error {
	negative, atom, e := rw.manager.IsNot(arg)
	if e != nil {
		return e
	}
	if negative {
		marked, e := neg.IsMarked(atom)
		if e != nil {
			return e
		}
		if marked {
			return nil
		}
		if e = neg.Mark(atom, true); e != nil {
			return e
		}
	} else {
		marked, e := pos.IsMarked(arg)
		if e != nil {
			return e
		}
		if marked {
			return nil
		}
		if e = pos.Mark(arg, true); e != nil {
			return e
		}
	}
	return rw.manager.RefVectorPush(v, arg)
}

// A concrete borrowed vector argument reader avoids cloning the source c_ptr
// when publishing the final OR. It grants no additional AST ownership.
func (rw *boolRewriter) localOrVector(v ast.ASTRefVector) (ast.ASTHandle, error) {
	return rw.manager.MkOrVector(v)
}

func (rw *boolRewriter) localContextSimplify(args []ast.ASTHandle, out ast.ASTRef) (simplified bool, err error) {
	m := rw.manager
	oldArgs, e := m.NewRefVector()
	if e != nil {
		return false, e
	}
	defer func() { err = errors.Join(err, m.DelRefVector(&oldArgs)) }()
	newArgs, e := m.NewRefVector()
	if e != nil {
		return false, e
	}
	defer func() { err = errors.Join(err, m.DelRefVector(&newArgs)) }()
	newArg, e := m.NewRef(ast.ASTHandle{})
	if e != nil {
		return false, e
	}
	defer func() { err = errors.Join(err, m.DelRef(&newArg)) }()
	neg, e := ast.NewASTFastMark1(m)
	if e != nil {
		return false, e
	}
	defer func() { err = errors.Join(err, neg.Close()) }()
	pos, e := ast.NewASTFastMark2(m)
	if e != nil {
		return false, e
	}
	defer func() { err = errors.Join(err, pos.Close()) }()
	simp, modified, forward := false, false, true
	fromVector := false
	n := uint32(len(args))
	truth, e := m.MkTrue()
	if e != nil {
		return false, e
	}
	falsity, e := m.MkFalse()
	if e != nil {
		return false, e
	}
	process := func(i uint32) (done bool, err error) {
		var arg ast.ASTHandle
		if fromVector {
			arg, e = m.RefVectorGet(oldArgs, i)
		} else {
			arg = args[i]
		}
		if e != nil {
			return false, e
		}
		negative, atom, e := m.IsNot(arg)
		if e != nil {
			return false, e
		}
		if negative {
			or, e := rw.isOperator(atom, ast.ASTOpOr)
			if e != nil {
				return false, e
			}
			if or {
				changed, e := rw.simpNestedNotOr(atom, neg, pos, newArg)
				if e != nil {
					return false, e
				}
				if changed {
					modified = true
					simp = true
					arg, e = m.RefGet(newArg)
					if e != nil {
						return false, e
					}
				}
			}
		}
		changed, e := rw.simpNestedEqITE(arg, neg, pos, newArg)
		if e != nil {
			return false, e
		}
		if changed {
			modified = true
			simp = true
			arg, e = m.RefGet(newArg)
			if e != nil {
				return false, e
			}
		}
		if arg == falsity {
			return false, nil
		}
		if arg == truth {
			return true, m.SetRef(out, arg)
		}
		or, e := rw.isOperator(arg, ast.ASTOpOr)
		if e != nil {
			return false, e
		}
		if rw.flat && or {
			size, e := m.GetNumArgs(arg)
			if e != nil {
				return false, e
			}
			for j := uint32(0); j < size; j++ {
				h, e := m.GetArg(arg, j)
				if e != nil {
					return false, e
				}
				if e = rw.pushLocalArg(h, newArgs, neg, pos); e != nil {
					return false, e
				}
			}
			return false, nil
		}
		return false, rw.pushLocalArg(arg, newArgs, neg, pos)
	}
	for {
		rw.localContextCost += 2 * n
		if forward {
			for i := uint32(0); i < n; i++ {
				done, e := process(i)
				if e != nil {
					return false, e
				}
				if done {
					return true, nil
				}
			}
			forward = false
		} else {
			for i := n; i > 0; {
				i--
				done, e := process(i)
				if e != nil {
					return false, e
				}
				if done {
					return true, nil
				}
			}
			if !modified {
				if !simp {
					return false, nil
				}
				var h ast.ASTHandle
				if fromVector {
					h, e = rw.localOrVector(oldArgs)
				} else {
					switch len(args) {
					case 0:
						h, e = m.MkFalse()
					case 1:
						h = args[0]
					default:
						h, e = m.MkOrArgs(args)
					}
				}
				if e != nil {
					return false, e
				}
				return true, m.SetRef(out, h)
			}
			if e = m.RefVectorReverse(newArgs); e != nil {
				return false, e
			}
			modified = false
			forward = true
		}
		if e = pos.Reset(); e != nil {
			return false, e
		}
		if e = neg.Reset(); e != nil {
			return false, e
		}
		if e = m.RefVectorReset(oldArgs); e != nil {
			return false, e
		}
		if e = m.SwapRefVectors(oldArgs, newArgs); e != nil {
			return false, e
		}
		fromVector = true
		n, e = m.RefVectorSize(oldArgs)
		if e != nil {
			return false, e
		}
	}
}
