// Source: src/sat/tactic/atom2bool_var.cpp:45-126;
// src/ast/for_each_expr.h:26-103,134-136.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import "errors"

func (tr *satGoalTranslator) collectInterface(formulas []ASTHandle) (err error) {
	fvisited, err := NewASTFastMark2(tr.manager)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, fvisited.Close()) }()
	tvisited, err := NewASTFastMark1(tr.manager)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, tvisited.Close()) }()
	// The source todo and traversal stacks reuse the concrete frame backing at
	// disjoint times. Keep the outer todo in its own charged vector because a
	// general expression walk may start while other formulas are still pending.
	todo := satGoalTranslator{solver: tr.solver}
	defer func() { tr.solver.budget.Release(uint64(cap(todo.frames)) * satGoalFrameBytes()) }()
	for _, formula := range formulas {
		marked, e := fvisited.IsMarked(formula)
		if e != nil {
			return e
		}
		if marked {
			continue
		}
		if e = fvisited.Mark(formula, true); e != nil {
			return e
		}
		if e = todo.pushFrame(satGoalFrame{term: formula}); e != nil {
			return e
		}
		for len(todo.frames) != 0 {
			i := len(todo.frames) - 1
			h := todo.frames[i].term
			todo.frames = todo.frames[:i]
			uninterpreted, e := tr.isUninterpretedConstant(h)
			if e != nil {
				return e
			}
			if uninterpreted {
				continue
			}
			kind, e := tr.manager.GetKind(h)
			if e != nil {
				return e
			}
			basic := false
			var count uint32
			if kind == ASTApp {
				family, e := tr.manager.GetFamilyID(h)
				if e != nil {
					return e
				}
				count, e = tr.manager.GetNumArgs(h)
				if e != nil {
					return e
				}
				basic = family == 0 && count > 0
			}
			if basic {
				op, e := tr.manager.GetDeclKind(h)
				if e != nil {
					return e
				}
				descend := op == ASTOpOr || op == ASTOpNot || op == ASTOpIff
				if op == ASTOpEq || op == ASTOpIte {
					arg, e := tr.manager.GetArg(h, 1)
					if e != nil {
						return e
					}
					descend, e = tr.isBool(arg)
					if e != nil {
						return e
					}
				}
				// Source deliberately omits AND from this branch; preserve it.
				if descend {
					for j := uint32(0); j < count; j++ {
						arg, e := tr.manager.GetArg(h, j)
						if e != nil {
							return e
						}
						marked, e := fvisited.IsMarked(arg)
						if e != nil {
							return e
						}
						if !marked {
							if e = fvisited.Mark(arg, true); e != nil {
								return e
							}
							if e = todo.pushFrame(satGoalFrame{term: arg}); e != nil {
								return e
							}
						}
					}
				}
			} else {
				if e = tr.collectGeneralInterface(h, tvisited); e != nil {
					return e
				}
			}
		}
	}
	return nil
}

func (tr *satGoalTranslator) markGeneral(h ASTHandle, visited *ASTFastMark1) (bool, error) {
	rc, e := tr.manager.GetRefCount(h)
	if e != nil {
		return false, e
	}
	if rc <= 1 {
		return false, nil
	}
	marked, e := visited.IsMarked(h)
	if e != nil || marked {
		return marked, e
	}
	return false, visited.Mark(h, true)
}
func (tr *satGoalTranslator) generalChildren(h ASTHandle, kind ASTKind) (uint32, error) {
	switch kind {
	case ASTApp:
		return tr.manager.GetNumArgs(h)
	case ASTVar:
		return 0, nil
	case ASTQuantifier:
		n, e := tr.manager.GetNumPatterns(h)
		if e != nil {
			return 0, e
		}
		nn, e := tr.manager.GetNumNoPatterns(h)
		return 1 + n + nn, e
	}
	return 0, ErrASTDomain
}
func (tr *satGoalTranslator) generalChild(h ASTHandle, kind ASTKind, index uint32) (ASTHandle, error) {
	if kind == ASTApp {
		return tr.manager.GetArg(h, index)
	}
	if kind != ASTQuantifier {
		return ASTHandle{}, ErrASTDomain
	}
	if index == 0 {
		return tr.manager.GetQuantifierBody(h)
	}
	n, e := tr.manager.GetNumPatterns(h)
	if e != nil {
		return ASTHandle{}, e
	}
	if index <= n {
		return tr.manager.GetPattern(h, index-1)
	}
	return tr.manager.GetNoPattern(h, index-n-1)
}
func (tr *satGoalTranslator) collectGeneralInterface(h ASTHandle, visited *ASTFastMark1) error {
	marked, e := tr.markGeneral(h, visited)
	if e != nil || marked {
		return e
	}
	if e = tr.pushFrame(satGoalFrame{term: h}); e != nil {
		return e
	}
	for len(tr.frames) != 0 {
		i := len(tr.frames) - 1
		fr := tr.frames[i]
		kind, e := tr.manager.GetKind(fr.term)
		if e != nil {
			return e
		}
		count, e := tr.generalChildren(fr.term, kind)
		if e != nil {
			return e
		}
		if fr.index < count {
			child, e := tr.generalChild(fr.term, kind, fr.index)
			if e != nil {
				return e
			}
			tr.frames[i].index++
			marked, e := tr.markGeneral(child, visited)
			if e != nil {
				return e
			}
			if marked {
				continue
			}
			if e = tr.pushFrame(satGoalFrame{term: child}); e != nil {
				return e
			}
			continue
		}
		uninterpreted, e := tr.isUninterpretedConstant(fr.term)
		if e != nil {
			return e
		}
		if uninterpreted {
			hash, e := tr.manager.GetHash(fr.term)
			if e != nil {
				return e
			}
			if e = tr.interfaces.prepare(tr.solver.budget); e != nil {
				return e
			}
			tr.interfaces.insertPrepared(fr.term, hash, 0)
		}
		tr.frames = tr.frames[:i]
	}
	return nil
}
