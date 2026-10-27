// Source: src/smt/theory_fpa.h:48-65; src/ast/fpa_decl_plugin.cpp:940-948.
package ast

// MkFPAModelFactoryValue follows fpa_value_factory::get_some_value. This is
// separate from the declaration plugin's get_some_value, which supplies NaN.
func (m *ASTManager) MkFPAModelFactoryValue(sort ASTHandle) (ASTHandle, error) {
	first, _, err := m.mkFPAModelFactoryValues(sort, false)
	return first, err
}

// MkFPAModelFactoryPair follows get_some_values, reusing the same scoped MPF
// for numeric zero then numeric one. Returned handles use the model factory's
// existing borrowed-handle interface; they do not acquire factory references.
func (m *ASTManager) MkFPAModelFactoryPair(sort ASTHandle) (ASTHandle, ASTHandle, error) {
	return m.mkFPAModelFactoryValues(sort, true)
}

func (m *ASTManager) mkFPAModelFactoryValues(sort ASTHandle, pair bool) (first, second ASTHandle, result error) {
	if err := m.lock(); err != nil {
		return first, second, err
	}
	defer m.mu.Unlock()
	defer func() {
		if !pair && result != nil {
			first = ASTHandle{}
		}
	}()
	p := m.floatingPointPlugin
	if p == nil || p.self != p || !p.published || p.sourceEnded {
		return first, second, ErrASTHandle
	}
	frame, err := p.beginNumeralProbe()
	if err != nil {
		return first, second, err
	}
	var last astNodeRef
	defer p.finishNumeralConstruction(frame, &result, &last)
	ref, err := m.requireKind(sort, ASTSort)
	if err != nil {
		return first, second, err
	}
	ebits, err := m.fpaUtilExponentBits(ref)
	if err != nil {
		return first, second, err
	}
	sbits, err := m.fpaUtilSignificandBits(ref)
	if err != nil {
		return first, second, err
	}
	if result = p.numeric.SetInt32(&frame.value, ebits, sbits, 0); result != nil {
		return first, second, result
	}
	last, result = p.mkNumeral(&frame.value, nil)
	if result != nil {
		return first, second, result
	}
	first = m.external(last)
	if !pair {
		return first, second, nil
	}
	if result = p.numeric.SetInt32(&frame.value, ebits, sbits, 1); result != nil {
		return first, second, result
	}
	last, result = p.mkNumeral(&frame.value, nil)
	if result != nil {
		return first, second, result
	}
	second = m.external(last)
	return first, second, nil
}
