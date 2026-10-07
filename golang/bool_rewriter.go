// Source: src/ast/rewriter/bool_rewriter.cpp:89-280;
// src/ast/rewriter/bool_rewriter.h:60-69; src/util/buffer.h:25-48.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import "errors"

var errBoolRewriterLocalContext = errors.New("Boolean rewriter: local-context simplification remains unported")

// Original Boolean rewriter's AND/OR core fragment. This inline helper borrows
// its AST manager. Full parameter construction, local-context simplification,
// other operators and the recursive rewriter engine remain dependencies.
type boolRewriter struct {
	manager                             *ASTManager
	flat, iteExtraRules                 bool
	localContext                        bool
	localContextCost, localContextLimit uint32
}

func (rw *boolRewriter) isOperator(h ASTHandle, op int32) (bool, error) {
	k, e := rw.manager.GetKind(h)
	if e != nil || k != ASTApp {
		return false, e
	}
	fid, e := rw.manager.GetFamilyID(h)
	if e != nil || fid != 0 {
		return false, e
	}
	kind, e := rw.manager.GetDeclKind(h)
	return kind == op, e
}

func (rw *boolRewriter) mkNFlatAndCore(args []ASTHandle, out ASTRef) (brStatus, error) {
	return rw.mkNFlatCore(args, out, false)
}
func (rw *boolRewriter) mkNFlatOrCore(args []ASTHandle, out ASTRef) (brStatus, error) {
	return rw.mkNFlatCore(args, out, true)
}

// The two original loops are dual except for OR's optional local-context call.
// Keep that source branch after mark reset and before the changed-result test.
func (rw *boolRewriter) mkNFlatCore(args []ASTHandle, out ASTRef, disjunction bool) (status brStatus, err error) {
	m := rw.manager
	var buffer astProofArguments
	defer buffer.close(m)
	neg, e := NewASTFastMark1(m)
	if e != nil {
		return brFailed, e
	}
	defer func() { err = errors.Join(err, neg.Close()) }()
	pos, e := NewASTFastMark2(m)
	if e != nil {
		return brFailed, e
	}
	defer func() { err = errors.Join(err, pos.Close()) }()
	unit, e := m.MkBoolVal(!disjunction)
	if e != nil {
		return brFailed, e
	}
	absorb, e := m.MkBoolVal(disjunction)
	if e != nil {
		return brFailed, e
	}
	changed := false
	for _, arg := range args {
		if arg == unit {
			changed = true
			continue
		}
		if arg == absorb {
			return brDone, m.SetRef(out, absorb)
		}
		negative, atom, e := m.IsNot(arg)
		if e != nil {
			return brFailed, e
		}
		if negative {
			marked, e := neg.IsMarked(atom)
			if e != nil {
				return brFailed, e
			}
			if marked {
				changed = true
				continue
			}
			marked, e = pos.IsMarked(atom)
			if e != nil {
				return brFailed, e
			}
			if marked {
				return brDone, m.SetRef(out, absorb)
			}
			if e = neg.Mark(atom, true); e != nil {
				return brFailed, e
			}
		} else {
			marked, e := pos.IsMarked(arg)
			if e != nil {
				return brFailed, e
			}
			if marked {
				changed = true
				continue
			}
			marked, e = neg.IsMarked(arg)
			if e != nil {
				return brFailed, e
			}
			if marked {
				return brDone, m.SetRef(out, absorb)
			}
			if e = pos.Mark(arg, true); e != nil {
				return brFailed, e
			}
		}
		if e = buffer.push(m, arg); e != nil {
			return brFailed, e
		}
	}
	remaining := buffer.slice()
	switch len(remaining) {
	case 0:
		return brDone, m.SetRef(out, unit)
	case 1:
		return brDone, m.SetRef(out, remaining[0])
	}
	if disjunction && rw.localContext && rw.localContextCost <= rw.localContextLimit {
		if e = neg.Reset(); e != nil {
			return brFailed, e
		}
		if e = pos.Reset(); e != nil {
			return brFailed, e
		}
		return brFailed, errBoolRewriterLocalContext
	}
	if !changed {
		return brFailed, nil
	}
	var result ASTHandle
	if disjunction {
		result, e = m.MkOrArgs(remaining)
	} else {
		result, e = m.MkAndArgs(remaining)
	}
	if e != nil {
		return brFailed, e
	}
	return brDone, m.SetRef(out, result)
}

func (rw *boolRewriter) mkFlatAndCore(args []ASTHandle, out ASTRef) (brStatus, error) {
	return rw.mkFlatCore(args, out, false)
}
func (rw *boolRewriter) mkFlatOrCore(args []ASTHandle, out ASTRef) (brStatus, error) {
	return rw.mkFlatCore(args, out, true)
}
func (rw *boolRewriter) mkFlatCore(args []ASTHandle, out ASTRef, disjunction bool) (brStatus, error) {
	op := int32(ASTOpAnd)
	if disjunction {
		op = ASTOpOr
	}
	i := 0
	for ; i < len(args); i++ {
		nested, e := rw.isOperator(args[i], op)
		if e != nil {
			return brFailed, e
		}
		if nested {
			break
		}
	}
	if i == len(args) {
		return rw.mkNFlatCore(args, out, disjunction)
	}
	m := rw.manager
	var flat astProofArguments
	defer flat.close(m)
	if e := flat.append(m, args[:i]); e != nil {
		return brFailed, e
	}
	for ; i < len(args); i++ {
		arg := args[i]
		nested, e := rw.isOperator(arg, op)
		if e != nil {
			return brFailed, e
		}
		if !nested {
			if e = flat.push(m, arg); e != nil {
				return brFailed, e
			}
			continue
		}
		n, e := m.GetNumArgs(arg)
		if e != nil {
			return brFailed, e
		}
		for j := uint32(0); j < n; j++ {
			h, e := m.GetArg(arg, j)
			if e != nil {
				return brFailed, e
			}
			if e = flat.push(m, h); e != nil {
				return brFailed, e
			}
		}
	}
	status, e := rw.mkNFlatCore(flat.slice(), out, disjunction)
	if e != nil {
		return brFailed, e
	}
	if status == brFailed {
		var result ASTHandle
		if disjunction {
			result, e = m.MkOrArgs(flat.slice())
		} else {
			result, e = m.MkAndArgs(flat.slice())
		}
		if e != nil {
			return brFailed, e
		}
		if e = m.SetRef(out, result); e != nil {
			return brFailed, e
		}
	}
	return brDone, nil
}
