// Source: src/ast/bv_decl_plugin.cpp:767-773.
package ast

import "github.com/glycerine/g3/pkg/util"

func (m *ASTManager) BitVectorHasSignBit(n *util.OwnedRational, bits uint32) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	previous := m.formatExecuting
	m.formatExecuting = true
	defer func() { m.formatExecuting = previous }()
	return m.bitVectorHasSignBit(n, bits, nil)
}

func (m *ASTManager) bitVectorHasSignBit(n *util.OwnedRational, bits uint32, g *util.GlobalParamsService) (sign bool, result error) {
	enabled, e := astBVAssertionEnabled(g)
	if e != nil {
		return false, e
	}
	if enabled && bits == 0 {
		if e = util.NamespaceAssertionFailureOptional(g, "src/ast/bv_decl_plugin.cpp", 768, "bv_size > 0"); e != nil {
			return false, e
		}
	}
	f, e := m.beginBVNumeralFrame()
	if e != nil {
		return false, e
	}
	defer f.finishNumeral(&result, nil)
	v, e := m.normalizeBitVectorWithFrame(n, bits, false, g, &f.normalizationChild)
	if e != nil {
		return false, e
	}
	f.values[0] = v
	f.placed[0], f.complete[0] = true, true
	v, e = m.numerals.PowerOfTwoWithInvocation(bits-1, &f.children[0])
	if e != nil {
		return false, e
	}
	f.values[1] = v
	f.placed[1], f.complete[1] = true, true
	return m.numerals.GreaterEqual(&f.values[0], &f.values[1])
}
