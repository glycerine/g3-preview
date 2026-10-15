// Source: src/ast/ast_smt2_pp.cpp:630-801,1037-1060.
package ast

import usymbol "github.com/glycerine/g3/pkg/util/symbol"

func (m *ASTManager) smt2FlatAssocCandidate(parent, child ASTHandle) (bool, error) {
	if err := m.lock(); err != nil {
		return false, err
	}
	defer m.mu.Unlock()
	p, err := m.requireKind(parent, ASTApp)
	if err != nil {
		return false, err
	}
	c, err := m.requireKind(child, ASTApp)
	if err != nil {
		return false, err
	}
	pd := m.appAt(p.slot).decl
	if pd != m.appAt(c.slot).decl {
		return false, nil
	}
	flags, err := m.funcFlags(pd)
	if err != nil {
		return false, err
	}
	return flags.LeftAssociative && flags.RightAssociative, nil
}

// SMT2PPBooleanTree builds the source child-before-parent documents for
// Boolean application trees, including the default flat_assoc profile.
// Numeric, quantifier, pattern and alias paths still belong to the wider
// SMT2 printer traversal.
func (m *ASTManager) SMT2PPBooleanTree(root ASTHandle, names []usymbol.Symbol) (ASTHandle, error) {
	boolSort, err := m.MkBoolSort()
	if err != nil {
		return ASTHandle{}, err
	}
	seen := make(map[ASTHandle]bool)
	var visit func(ASTHandle, bool) ([]ASTHandle, uint32, error)
	visit = func(h ASTHandle, flatten bool) ([]ASTHandle, uint32, error) {
		if seen[h] {
			return nil, 0, ErrASTDomain // shared-subterm aliasing is still open
		}
		seen[h] = true
		kind, e := m.GetKind(h)
		if e != nil {
			return nil, 0, e
		}
		if kind != ASTVar && kind != ASTApp {
			return nil, 0, ErrASTDomain
		}
		sort, e := m.GetSort(h)
		if e != nil {
			return nil, 0, e
		}
		if sort != boolSort {
			return nil, 0, ErrASTDomain
		}
		if kind == ASTVar {
			index, e := m.GetVarIndex(h)
			if e != nil {
				return nil, 0, e
			}
			format, e := m.SMT2PPVar(index, names)
			return []ASTHandle{format}, 1, e
		}
		count, e := m.GetNumArgs(h)
		if e != nil {
			return nil, 0, e
		}
		if count == 0 {
			var labels []usymbol.Symbol
			labeled, e := m.IsLabelLit(h, &labels)
			if e != nil {
				return nil, 0, e
			}
			if labeled {
				body, e := m.FormatString("true")
				if e != nil {
					return nil, 0, e
				}
				format, e := m.SMT2PPLabels(true, labels, body)
				return []ASTHandle{format}, 1, e
			}
			decl, e := m.GetDecl(h)
			if e != nil {
				return nil, 0, e
			}
			format, _, e := m.SMT2Fdecl(decl)
			return []ASTHandle{format}, 1, e
		}
		args := make([]ASTHandle, 0, count)
		depth := uint32(1)
		for i := uint32(0); i < count; i++ {
			child, e := m.GetArg(h, i)
			if e != nil {
				return nil, 0, e
			}
			childKind, e := m.GetKind(child)
			if e != nil {
				return nil, 0, e
			}
			if childKind == ASTApp {
				flat, e := m.smt2FlatAssocCandidate(h, child)
				if e != nil {
					return nil, 0, e
				}
				if flat {
					childDocs, childDepth, e := visit(child, true)
					if e != nil {
						return nil, 0, e
					}
					if childDepth+1 > depth {
						depth = childDepth + 1
					}
					args = append(args, childDocs...)
					continue
				}
			}
			formats, childDepth, e := visit(child, false)
			if e != nil {
				return nil, 0, e
			}
			if childDepth+1 > depth {
				depth = childDepth + 1
			}
			args = append(args, formats...)
		}
		if flatten {
			return args, depth - 1, nil
		}
		if h != root && depth >= 5 {
			return nil, 0, ErrASTDomain // source depth aliasing is still open
		}
		var positive bool
		var labels []usymbol.Symbol
		labeled, e := m.IsLabel(h, &positive, &labels)
		if e != nil {
			return nil, 0, e
		}
		if labeled {
			if len(args) != 1 {
				return nil, 0, ErrASTDomain
			}
			format, e := m.SMT2PPLabels(positive, labels, args[0])
			return []ASTHandle{format}, depth, e
		}
		decl, e := m.GetDecl(h)
		if e != nil {
			return nil, 0, e
		}
		format, e := m.SMT2PPAppFormat(decl, args)
		return []ASTHandle{format}, depth, e
	}
	formats, _, err := visit(root, false)
	if err != nil {
		return ASTHandle{}, err
	}
	return formats[0], nil
}
