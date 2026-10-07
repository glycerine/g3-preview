// Source: src/ast/bv_decl_plugin.cpp:778-790; src/ast/rewriter/bv_rewriter.h:36.
package ast

import "github.com/glycerine/g3/pkg/util"

// BitVectorNumeralValueWithInvocation copies the stored numeral directly into
// the caller's receiver, naming the actual rational assignment invocation.
// Unmatched inputs leave receiver, width and invocation unread. A reached
// numerical copy requires a nonnil invocation. Width is read after that copy.
func (m *ASTManager) BitVectorNumeralValueWithInvocation(h ASTHandle, out *util.OwnedRational, width *uint32, c *util.RationalServiceInvocation) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	previous := m.formatExecuting
	m.formatExecuting = true
	defer func() { m.formatExecuting = previous }()
	r, e := m.requireExpr(h)
	if e != nil {
		return false, e
	}
	if c == nil {
		matched, e := m.isBitVectorNumeral(r)
		if e != nil || !matched {
			return false, e
		}
		return false, ErrASTHandle
	}
	return m.readBitVectorNumeralWithInvocation(r, out, width, c)
}
