// Source: src/ast/rewriter/seq_rewriter.cpp:483-519.
package rewriter

import (
	"errors"
	"fmt"

	"github.com/glycerine/g3/pkg/ast"
)

// SequenceRewriter borrows its AST manager and implements the retained theory
// reductions independently of traversal.
type SequenceRewriter struct {
	manager *ast.ASTManager
}

func NewSequenceRewriter(m *ast.ASTManager) (*SequenceRewriter, error) {
	if m == nil {
		return nil, ast.ErrASTHandle
	}
	if _, err := m.ProofsEnabled(); err != nil {
		return nil, err
	}
	return &SequenceRewriter{manager: m}, nil
}

func (r *SequenceRewriter) open() error {
	if r == nil || r.manager == nil {
		return ast.ErrASTHandle
	}
	_, err := r.manager.ProofsEnabled()
	return err
}

// RewriteUnit ports the byte-valued branch of seq_rewriter::mk_seq_unit.
func (r *SequenceRewriter) RewriteUnit(element ast.ASTHandle, out ast.ASTRef) (RewriteStatus, error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	value, matched, err := r.manager.SequenceUnitByteElementValue(element)
	if err != nil || !matched {
		return RewriteFailed, err
	}
	result, err := r.manager.MkString(string([]byte{value}))
	if err != nil {
		return RewriteFailed, err
	}
	return RewriteDone, r.manager.SetRef(out, result)
}

// RewriteConcat ports seq_rewriter::mk_seq_concat. The caller owns traversal
// and applies the returned revisit depth.
func (r *SequenceRewriter) RewriteConcat(a, b ast.ASTHandle, out ast.ASTRef) (status RewriteStatus, err error) {
	if err := r.open(); err != nil {
		return RewriteFailed, err
	}
	s1, stringA, err := r.manager.SequenceStringValue(a)
	if err != nil {
		return RewriteFailed, fmt.Errorf("sequence concat left string: %w", err)
	}
	s2, stringB, err := r.manager.SequenceStringValue(b)
	if err != nil {
		return RewriteFailed, fmt.Errorf("sequence concat right string: %w", err)
	}
	if stringA && stringB {
		result, err := r.manager.MkString(s1 + s2)
		if err != nil {
			return RewriteFailed, err
		}
		return RewriteDone, r.manager.SetRef(out, result)
	}
	c, d, concatA, err := r.manager.SequenceConcatParts(a)
	if err != nil {
		return RewriteFailed, fmt.Errorf("sequence concat left parts: %w", err)
	}
	if concatA {
		right, err := r.manager.MkSequenceConcat(d, b)
		if err != nil {
			return RewriteFailed, fmt.Errorf("sequence concat suffix: %w", err)
		}
		rightRef, err := r.manager.NewRef(right)
		if err != nil {
			return RewriteFailed, fmt.Errorf("sequence concat suffix reference: %w", err)
		}
		defer func() { err = errors.Join(err, r.manager.DelRef(&rightRef)) }()
		result, err := r.manager.MkSequenceConcat(c, right)
		if err != nil {
			return RewriteFailed, fmt.Errorf("sequence concat reassociation: %w", err)
		}
		if err = r.manager.SetRef(out, result); err != nil {
			return RewriteFailed, fmt.Errorf("sequence concat result: %w", err)
		}
		return Rewrite2, nil
	}
	emptyA, err := r.manager.IsSequenceEmpty(a)
	if err != nil {
		return RewriteFailed, err
	}
	if emptyA {
		return RewriteDone, r.manager.SetRef(out, b)
	}
	emptyB, err := r.manager.IsSequenceEmpty(b)
	if err != nil {
		return RewriteFailed, err
	}
	if emptyB {
		return RewriteDone, r.manager.SetRef(out, a)
	}
	if stringA {
		c, d, concatB, err := r.manager.SequenceConcatParts(b)
		if err != nil {
			return RewriteFailed, err
		}
		if concatB {
			s2, stringC, err := r.manager.SequenceStringValue(c)
			if err != nil {
				return RewriteFailed, err
			}
			if stringC {
				prefix, err := r.manager.MkString(s1 + s2)
				if err != nil {
					return RewriteFailed, err
				}
				prefixRef, err := r.manager.NewRef(prefix)
				if err != nil {
					return RewriteFailed, err
				}
				defer func() { err = errors.Join(err, r.manager.DelRef(&prefixRef)) }()
				result, err := r.manager.MkSequenceConcat(prefix, d)
				if err != nil {
					return RewriteFailed, err
				}
				return RewriteDone, r.manager.SetRef(out, result)
			}
		}
	}
	return RewriteFailed, nil
}
