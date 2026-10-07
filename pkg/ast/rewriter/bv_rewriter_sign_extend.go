// Source: src/ast/rewriter/bv_rewriter.cpp:1525-1553.
package rewriter

import "github.com/glycerine/g3/pkg/ast"

// RewriteBitVectorSignExtend borrows the rewriter's cached extract procedure.
func RewriteBitVectorSignExtend(extract *MkExtractProc, n uint32, arg ast.ASTHandle, eliminate bool, out ast.ASTRef) (RewriteStatus, error) {
	m, e := rotationManager(extract)
	if e != nil {
		return RewriteFailed, e
	}
	if n == 0 {
		return RewriteDone, m.SetRef(out, arg)
	}
	matched, e := m.ExtendBitVectorNumeralSign(n, arg, out)
	if e != nil {
		return RewriteFailed, e
	}
	if matched {
		return RewriteDone, nil
	}
	if !eliminate {
		return RewriteFailed, nil
	}
	size, e := m.BitVectorSize(arg)
	if e != nil {
		return RewriteFailed, e
	}
	sign, e := extract.Apply(size-1, size-1, arg)
	if e != nil {
		return RewriteFailed, e
	}
	var args ast.ASTHandleBuffer
	defer args.Close(m.Budget())
	for i := uint32(0); i < n; i++ {
		if e = args.Push(m.Budget(), sign); e != nil {
			return RewriteFailed, e
		}
	}
	if e = args.Push(m.Budget(), arg); e != nil {
		return RewriteFailed, e
	}
	h, e := m.MkBitVectorConcat(args.Slice())
	if e != nil {
		return RewriteFailed, e
	}
	return Rewrite2, m.SetRef(out, h)
}
