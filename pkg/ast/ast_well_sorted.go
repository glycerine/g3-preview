// Source: src/ast/well_sorted.cpp:27-83; src/ast/for_each_expr.h:27-97.
package ast

import (
	"fmt"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type wellSortedFrame struct {
	node  astNodeRef
	next  uint32
	count uint32
}

// IsWellSorted checks the retained expression graph in source child-before-
// parent order. The warning service is borrowed; nil selects the process owner.
func (m *ASTManager) IsWellSorted(h ASTHandle, warnings *diagnostic.WarningService) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	root, e := m.requireExpr(h)
	if e != nil {
		return false, e
	}
	if warnings == nil {
		warnings = diagnostic.BorrowProcessWarningService()
	}
	var inline [16]wellSortedFrame
	stack := inline[:0]
	visited := make(map[astNodeRef]bool)
	push := func(r astNodeRef) error {
		if e := m.validNode(r); e != nil {
			return e
		}
		if m.nodeAt(r.slot).rc > 1 {
			if visited[r] {
				return nil
			}
			visited[r] = true
		}
		var n uint32
		switch m.nodeAt(r.slot).kind {
		case ASTApp:
			n = m.appAt(r.slot).count
		case ASTQuantifier:
			q := m.quantifierAt(r.slot)
			n = 1 + q.patternCount + q.noPatternCount
		case ASTVar:
		default:
			return ErrASTDomain
		}
		stack = append(stack, wellSortedFrame{node: r, count: n})
		return nil
	}
	if e = push(root); e != nil {
		return false, e
	}
	valid := true
	for len(stack) != 0 {
		f := &stack[len(stack)-1]
		if f.next < f.count {
			i := f.next
			f.next++
			var child astNodeRef
			switch m.nodeAt(f.node.slot).kind {
			case ASTApp:
				child, e = m.readInternal(m.appAt(f.node.slot).args, i)
			case ASTQuantifier:
				q := m.quantifierAt(f.node.slot)
				switch {
				case i == 0:
					child = q.body
				case i <= q.patternCount:
					child, e = m.readInternal(q.patterns, i-1)
				default:
					child, e = m.readInternal(q.noPatterns, i-1-q.patternCount)
				}
			}
			if e != nil {
				return false, e
			}
			if e = push(child); e != nil {
				return false, e
			}
			continue
		}
		r := f.node
		stack = stack[:len(stack)-1]
		switch m.nodeAt(r.slot).kind {
		case ASTQuantifier:
			body, err := m.sortOf(m.quantifierAt(r.slot).body)
			if err != nil {
				return false, err
			}
			if body != m.basic.boolSort {
				_ = warnings.Warning("quantifier's body must be a boolean.")
				valid = false
			}
		case ASTApp:
			a := m.appAt(r.slot)
			d := m.declAt(a.decl.slot)
			flags, err := m.funcFlags(a.decl)
			if err != nil {
				return false, err
			}
			if a.count != d.arity && !flags.LeftAssociative && !flags.RightAssociative {
				_ = warnings.Warning("unexpected number of arguments.")
				valid = false
				continue
			}
			for i := uint32(0); i < a.count; i++ {
				child, err := m.readInternal(a.args, i)
				if err != nil {
					return false, err
				}
				actual, err := m.sortOf(child)
				if err != nil {
					return false, err
				}
				index := i
				if flags.LeftAssociative && flags.RightAssociative {
					index = 0
				}
				if index >= d.arity {
					return false, ErrASTDomain
				}
				expected, err := m.readInternal(d.domain, index)
				if err != nil {
					return false, err
				}
				if actual != expected {
					_ = warnings.Warning(fmt.Sprintf("Sort mismatch for argument %d", i+1))
					valid = false
					break
				}
			}
		}
	}
	return valid, nil
}
