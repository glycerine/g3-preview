// Licensed under the MIT license.
package g3

import "github.com/glycerine/g3/golang/arena"

func (m *ASTManager) MatchProofHyperResolve(p ASTHandle, premises ASTRefVector, conclusion ASTRef, out *ASTProofHyperOutput) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	yes, e := m.proofKind(p, ASTPrHyperResolve)
	if e != nil || !yes {
		return false, e
	}
	r, e := m.requireKind(p, ASTApp)
	if e != nil {
		return false, e
	}
	a := m.appAt(r.slot)
	if astSourceDebug && a.count == 0 {
		return false, ErrASTDomain
	}
	for i := uint32(0); i+1 < a.count; i++ {
		h, e := m.proofArg(p, i)
		if e != nil {
			return false, e
		}
		child, e := m.requireKind(h, ASTApp)
		if e != nil {
			return false, e
		} // source to_app
		if e = m.pushVector(premises, child); e != nil {
			return false, e
		}
	}
	if a.count == 0 {
		return false, arena.ErrBounds
	}
	f, e := m.proofArg(p, a.count-1)
	if e != nil {
		return false, e
	}
	child, e := m.requireExpr(f)
	if e != nil {
		return false, e
	}
	if e = m.assignRef(conclusion, child); e != nil {
		return false, e
	}
	d := m.declAt(a.decl.slot)
	if e = out.check(m); e != nil {
		return false, e
	}
	if e = out.pushSubstitution(m); e != nil {
		return false, e
	}
	if d.info.slot == 0 {
		return true, nil
	}
	info := m.infoAt(d.info.slot)
	for i := uint32(0); i < info.count; i++ {
		param, e := m.infoParameter(info, i)
		if e != nil {
			return false, e
		}
		value, e := m.parameter(param)
		if e != nil {
			return false, e
		}
		if value.kind == ASTParameterInt {
			if i+1 >= info.count {
				if astSourceDebug {
					return false, ErrASTDomain
				}
				return false, arena.ErrBounds
			}
			next, e := m.infoParameter(info, i+1)
			if e != nil {
				return false, e
			}
			second, e := m.parameter(next)
			if e != nil {
				return false, e
			}
			if second.kind != ASTParameterInt {
				return false, ErrASTDomain
			} // source inactive-union read is undefined
			if e = out.pushPosition(m, ASTProofPosition{First: uint32(value.integer), Second: uint32(second.integer)}); e != nil {
				return false, e
			}
			if e = out.pushSubstitution(m); e != nil {
				return false, e
			}
			i++
		} else {
			if value.kind != ASTParameterAST {
				return false, ErrASTDomain
			}
			r := value.ast
			if _, e = m.requireExpr(m.external(r)); e != nil {
				return false, e
			}
			if e = m.pushVector(out.substitutions[len(out.substitutions)-1], r); e != nil {
				return false, e
			}
		}
	}
	return true, nil
}
