// Source: src/smt/theory_fpa.h:48-65; src/ast/fpa_decl_plugin.cpp:940-948.
package ast

// MkFPAModelFactoryValue follows fpa_value_factory::get_some_value. This is
// separate from the declaration plugin's get_some_value, which supplies NaN.
func (m *ASTManager) MkFPAModelFactoryValue(sort ASTHandle) (ASTHandle, error) {
	first, err := m.mkFPAModelFactoryValues(sort, false, ASTRef{}, ASTRef{})
	return first, err
}

// MkFPAModelFactoryPair writes caller-owned outputs before the next setter or
// the scoped MPF destructor. It never retains a separate factory reference.
func (m *ASTManager) MkFPAModelFactoryPair(sort ASTHandle, first, second ASTRef) error {
	_, err := m.mkFPAModelFactoryValues(sort, true, first, second)
	return err
}

func (m *ASTManager) mkFPAModelFactoryValues(sort ASTHandle, pair bool, first, second ASTRef) (out ASTHandle, result error) {
	if err := m.lock(); err != nil {
		return out, err
	}
	defer m.mu.Unlock()
	defer func() {
		if !pair && result != nil {
			out = ASTHandle{}
		}
	}()
	p := m.floatingPointPlugin
	if p == nil || p.self != p || !p.published || p.sourceEnded {
		return out, ErrASTHandle
	}
	frame, err := p.beginNumeralProbe()
	if err != nil {
		return out, err
	}
	var last astNodeRef
	defer p.finishNumeralConstruction(frame, &result, &last)
	ref, err := m.requireKind(sort, ASTSort)
	if err != nil {
		return out, err
	}
	ebits, err := m.fpaUtilExponentBits(ref)
	if err != nil {
		return out, err
	}
	sbits, err := m.fpaUtilSignificandBits(ref)
	if err != nil {
		return out, err
	}
	if result = p.numeric.SetInt32(&frame.value, ebits, sbits, 0); result != nil {
		return out, result
	}
	last, result = p.mkNumeral(&frame.value, nil)
	if result != nil {
		return out, result
	}
	if !pair {
		return m.external(last), nil
	}
	if result = m.assignRef(first, last); result != nil {
		return out, result
	}
	if result = p.numeric.SetInt32(&frame.value, ebits, sbits, 1); result != nil {
		return out, result
	}
	last, result = p.mkNumeral(&frame.value, nil)
	if result != nil {
		return out, result
	}
	result = m.assignRef(second, last)
	return out, result
}
