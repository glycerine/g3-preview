// Source: src/ast/rewriter/poly_rewriter_def.h:371-400.
package rewriter

import "github.com/glycerine/g3/pkg/ast"

func (p *BVPolynomialAdditionCore) flatBody(sort ast.ASTHandle, args ast.AppArguments, out ast.ASTRef, o BVPolynomialAdditionOptions) (RewriteStatus, error) {
	m := p.manager
	i := uint32(0)
	for ; i < args.Count(); i++ {
		arg, e := args.At(i)
		if e != nil {
			return RewriteFailed, e
		}
		add, e := m.IsBitVectorOperation(arg, 4)
		if e != nil {
			return RewriteFailed, e
		}
		if add {
			break
		}
	}
	if i == args.Count() {
		return p.body(sort, args, out, o)
	}
	for j := uint32(0); j < i; j++ {
		arg, e := args.At(j)
		if e != nil {
			return RewriteFailed, e
		}
		if e = p.flatArgs.Push(m.Budget(), arg); e != nil {
			return RewriteFailed, e
		}
	}
	for ; i < args.Count(); i++ {
		arg, e := args.At(i)
		if e != nil {
			return RewriteFailed, e
		}
		add, e := m.IsBitVectorOperation(arg, 4)
		if e != nil {
			return RewriteFailed, e
		}
		if add {
			n, e := m.GetNumArgs(arg)
			if e != nil {
				return RewriteFailed, e
			}
			for j := uint32(0); j < n; j++ {
				h, e := m.GetArg(arg, j)
				if e != nil {
					return RewriteFailed, e
				}
				if e = p.flatArgs.Push(m.Budget(), h); e != nil {
					return RewriteFailed, e
				}
			}
		} else if e = p.flatArgs.Push(m.Budget(), arg); e != nil {
			return RewriteFailed, e
		}
	}
	flat := ast.SliceArguments(p.flatArgs.Slice())
	st, e := p.body(sort, flat, out, o)
	if e != nil {
		return st, e
	}
	// The nested core's locals end at its return, before the FAILED fallback.
	if e = p.sourceEnd(); e != nil {
		return st, e
	}
	if st != RewriteFailed {
		return st, nil
	}
	node, e := m.MkBitVectorPolynomialAddWithInvocation(sort, flat, &p.factory)
	if e != nil {
		return RewriteFailed, e
	}
	return RewriteDone, m.SetRef(out, node)
}
