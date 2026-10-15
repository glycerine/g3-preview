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
	// shared_occs visits each compound node's children once, but records a
	// second incoming edge as shared. Its default profile ignores atomic nodes.
	occurrences := make(map[ASTHandle]uint32)
	var countOccurrences func(ASTHandle) error
	countOccurrences = func(h ASTHandle) error {
		kind, e := m.GetKind(h)
		if e != nil {
			return e
		}
		if kind != ASTApp {
			return nil
		}
		n, e := m.GetNumArgs(h)
		if e != nil || n == 0 {
			return e
		}
		occurrences[h]++
		if occurrences[h] > 1 {
			return nil
		}
		for i := uint32(0); i < n; i++ {
			child, e := m.GetArg(h, i)
			if e != nil {
				return e
			}
			if e = countOccurrences(child); e != nil {
				return e
			}
		}
		return nil
	}
	if err := countOccurrences(root); err != nil {
		return ASTHandle{}, err
	}
	type info struct{ weight, depth uint32 }
	metrics := make(map[ASTHandle]info)
	var measure func(ASTHandle) (info, error)
	measure = func(h ASTHandle) (info, error) {
		if value, found := metrics[h]; found {
			return value, nil
		}
		kind, e := m.GetKind(h)
		if e != nil {
			return info{}, e
		}
		value := info{weight: 1, depth: 1}
		if kind == ASTApp {
			n, e := m.GetNumArgs(h)
			if e != nil {
				return info{}, e
			}
			for i := uint32(0); i < n; i++ {
				child, e := m.GetArg(h, i)
				if e != nil {
					return info{}, e
				}
				childInfo, e := measure(child)
				if e != nil {
					return info{}, e
				}
				flat := false
				if occurrences[child] == 1 {
					childKind, e := m.GetKind(child)
					if e != nil {
						return info{}, e
					}
					if childKind == ASTApp {
						flat, e = m.smt2FlatAssocCandidate(h, child)
						if e != nil {
							return info{}, e
						}
					}
				}
				if flat {
					value.weight += childInfo.weight - 1
					if childInfo.depth > value.depth {
						value.depth = childInfo.depth
					}
				} else {
					value.weight += childInfo.weight
					if childInfo.depth+1 > value.depth {
						value.depth = childInfo.depth + 1
					}
				}
			}
		}
		metrics[h] = value
		return value, nil
	}
	if _, err := measure(root); err != nil {
		return ASTHandle{}, err
	}
	var visit func(ASTHandle, bool) ([]ASTHandle, uint32, error)
	visit = func(h ASTHandle, flatten bool) ([]ASTHandle, uint32, error) {
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
		// store_result creates an alias only for nonroot frames at the
		// configured depth or shared-weight threshold. Flat children return
		// before store_result and expose their child documents to the parent.
		if !flatten && h != root && (metrics[h].depth >= 5 || occurrences[h] > 1 && metrics[h].weight >= 10) {
			return nil, 0, ErrASTDomain // source alias registration is still open
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
				if flat && occurrences[child] == 1 {
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
