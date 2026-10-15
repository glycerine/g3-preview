// Source: src/ast/ast_smt2_pp.cpp:630-844,1037-1060.
package ast

import (
	"strconv"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

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
// Boolean application trees, including flat_assoc and source-default aliases.
// Numeric, quantifier and pattern paths still belong to the wider printer.
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

	// These are the source info_stack fields. A flattened application leaves
	// its child entries on the stack, so its own weight and depth disappear.
	type result struct {
		docs          []ASTHandle
		level         uint32
		weight, depth uint32
	}
	bindings := make([]smt2LetBinding, 0)
	aliases := make(map[ASTHandle]int)
	usedNames := make(map[string]bool)
	for _, name := range names {
		s, e := m.Symbols.String(name)
		if e != nil {
			return ASTHandle{}, e
		}
		usedNames[s] = true
	}
	nextAlias := uint32(1)
	var visit func(ASTHandle, bool) (result, error)
	visit = func(h ASTHandle, flatten bool) (result, error) {
		if idx, found := aliases[h]; found {
			binding := bindings[idx]
			format, e := m.FormatString(binding.name)
			return result{docs: []ASTHandle{format}, level: binding.level + 1, weight: 1, depth: 1}, e
		}
		kind, e := m.GetKind(h)
		if e != nil {
			return result{}, e
		}
		if kind != ASTVar && kind != ASTApp {
			return result{}, ErrASTDomain
		}
		sort, e := m.GetSort(h)
		if e != nil {
			return result{}, e
		}
		if h == root && sort != boolSort {
			return result{}, ErrASTDomain
		}
		if kind == ASTVar {
			index, e := m.GetVarIndex(h)
			if e != nil {
				return result{}, e
			}
			format, e := m.SMT2PPVar(index, names)
			return result{docs: []ASTHandle{format}, weight: 1, depth: 1}, e
		}
		count, e := m.GetNumArgs(h)
		if e != nil {
			return result{}, e
		}
		if count == 0 {
			if sort != boolSort {
				format, matched, e := m.smt2PPArithLiteralDefault(h)
				if e != nil {
					return result{}, e
				}
				if matched {
					return result{docs: []ASTHandle{format}, weight: 1, depth: 1}, nil
				}
				matched, e = m.IsBitVectorNumeral(h)
				if e != nil {
					return result{}, e
				}
				if matched {
					format, e = m.ppBitVectorLiteral(h, false, true, nil)
					return result{docs: []ASTHandle{format}, weight: 1, depth: 1}, e
				}
				matched, e = m.smt2IsFloatingPointNumeral(h)
				if e != nil {
					return result{}, e
				}
				if matched {
					format, e = m.ppFloatBitVectorLiteral(h, true, nil)
					return result{docs: []ASTHandle{format}, weight: 1, depth: 1}, e
				}
				family, e := m.GetFamilyID(h)
				if e != nil {
					return result{}, e
				}
				if family != NullFamilyID {
					return result{}, ErrASTDomain
				}
			}
			var labels []usymbol.Symbol
			labeled, e := m.IsLabelLit(h, &labels)
			if e != nil {
				return result{}, e
			}
			if labeled {
				body, e := m.FormatString("true")
				if e != nil {
					return result{}, e
				}
				format, e := m.SMT2PPLabels(true, labels, body)
				return result{docs: []ASTHandle{format}, weight: 1, depth: 1}, e
			}
			decl, e := m.GetDecl(h)
			if e != nil {
				return result{}, e
			}
			format, _, e := m.SMT2Fdecl(decl)
			return result{docs: []ASTHandle{format}, weight: 1, depth: 1}, e
		}
		args := make([]ASTHandle, 0, count)
		value := result{weight: 1, depth: 1}
		for i := uint32(0); i < count; i++ {
			child, e := m.GetArg(h, i)
			if e != nil {
				return result{}, e
			}
			childKind, e := m.GetKind(child)
			if e != nil {
				return result{}, e
			}
			flat := false
			if childKind == ASTApp && occurrences[child] == 1 {
				flat, e = m.smt2FlatAssocCandidate(h, child)
				if e != nil {
					return result{}, e
				}
			}
			childResult, e := visit(child, flat)
			if e != nil {
				return result{}, e
			}
			if childResult.level > value.level {
				value.level = childResult.level
			}
			value.weight += childResult.weight
			if childResult.depth > value.depth {
				value.depth = childResult.depth
			}
			args = append(args, childResult.docs...)
		}
		if flatten {
			value.docs = args
			value.weight--
			return value, nil
		}
		value.depth++
		var positive bool
		var labels []usymbol.Symbol
		labeled, e := m.IsLabel(h, &positive, &labels)
		if e != nil {
			return result{}, e
		}
		var format ASTHandle
		if labeled {
			if len(args) != 1 {
				return result{}, ErrASTDomain
			}
			format, e = m.SMT2PPLabels(positive, labels, args[0])
		} else {
			decl, e := m.GetDecl(h)
			if e != nil {
				return result{}, e
			}
			format, e = m.SMT2PPAppFormat(decl, args)
		}
		if e != nil {
			return result{}, e
		}
		if h != root && (value.depth >= 5 || value.weight >= 10 && occurrences[h] > 1) {
			for {
				name := "a!" + strconv.FormatUint(uint64(nextAlias), 10)
				nextAlias++
				if usedNames[name] {
					continue
				}
				aliases[h] = len(bindings)
				bindings = append(bindings, smt2LetBinding{name: name, format: format, level: value.level})
				aliasFormat, e := m.FormatString(name)
				return result{docs: []ASTHandle{aliasFormat}, level: value.level + 1, weight: 1, depth: 1}, e
			}
		}
		value.docs = []ASTHandle{format}
		return value, nil
	}
	printed, err := visit(root, false)
	if err != nil {
		return ASTHandle{}, err
	}
	return m.smt2PPLet(printed.docs[0], bindings)
}
