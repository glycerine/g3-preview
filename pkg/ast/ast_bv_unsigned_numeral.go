// Source: src/ast/rewriter/bv_rewriter.h:177; src/util/rational.h:48.
package ast

func (m *ASTManager) BitVectorNumeralUnsigned(value uint32, bits uint32) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	previous := m.formatExecuting
	m.formatExecuting = true
	defer func() { m.formatExecuting = previous }()
	r, e := m.bitVectorNumeralUnsigned(value, bits)
	return m.external(r), e
}

func (m *ASTManager) bitVectorNumeralUnsigned(value uint32, bits uint32) (node astNodeRef, result error) {
	f, e := m.beginBVNumeralFrame()
	if e != nil {
		return node, e
	}
	f.unsignedNumeral = true
	defer f.finishNumeral(&result, nil)
	// The outer rational(unsigned) temporary ends after the numeral call's
	// parameter temporary. Constructor failure retains an incomplete owner.
	f.values[5], e = m.numerals.NewValue()
	if e != nil {
		return node, e
	}
	f.placed[5] = true
	if e = m.numerals.SetUnsignedWithInvocation(&f.values[5], value, &f.children[9]); e != nil {
		return node, e
	}
	f.complete[5] = true
	if e = f.copyConstruct(0, &f.values[5]); e != nil {
		return node, e
	}
	return m.bitVectorNumeralAppCore(f, bits)
}
