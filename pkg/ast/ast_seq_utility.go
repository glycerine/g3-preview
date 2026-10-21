// Source: src/ast/seq_decl_plugin.cpp:959-967,993-1018;
// src/ast/seq_decl_plugin.h:277-306,319-335.
package ast

func (m *ASTManager) sequenceBinaryParts(r astNodeRef, kind int32) (astNodeRef, astNodeRef, bool, error) {
	if m.nodeAt(r.slot).kind != ASTApp || m.sequencePlugin == nil {
		return astNodeRef{}, astNodeRef{}, false, nil
	}
	a := m.appAt(r.slot)
	family, actualKind, err := m.declIdentity(a.decl)
	if err != nil || family != m.sequencePlugin.family || actualKind != kind || a.count != 2 {
		return astNodeRef{}, astNodeRef{}, false, err
	}
	left, err := m.readInternal(a.args, 0)
	if err != nil {
		return astNodeRef{}, astNodeRef{}, false, err
	}
	right, err := m.readInternal(a.args, 1)
	return left, right, err == nil, err
}

func (m *ASTManager) SequenceConcatParts(h ASTHandle) (left, right ASTHandle, matched bool, err error) {
	if err = m.lock(); err != nil {
		return
	}
	defer m.mu.Unlock()
	r, err := m.requireExpr(h)
	if err != nil {
		return ASTHandle{}, ASTHandle{}, false, err
	}
	l, rr, matched, err := m.sequenceBinaryParts(r, ASTSequenceOpConcat)
	return m.external(l), m.external(rr), matched, err
}

func (m *ASTManager) sequenceFlattenConcat(r astNodeRef, out *[]ASTHandle) error {
	for {
		left, right, matched, err := m.sequenceBinaryParts(r, ASTSequenceOpConcat)
		if err != nil {
			return err
		}
		if !matched {
			break
		}
		if err = m.sequenceFlattenConcat(left, out); err != nil {
			return err
		}
		r = right
	}
	empty, err := m.isSequenceEmptyRef(r)
	if err != nil {
		return err
	}
	if !empty {
		*out = append(*out, m.external(r))
	}
	return nil
}

// SequenceFlattenConcat returns source-order concat leaves and omits sequence
// empty values, matching seq_util::str::get_concat.
func (m *ASTManager) SequenceFlattenConcat(h ASTHandle) ([]ASTHandle, error) {
	if err := m.lock(); err != nil {
		return nil, err
	}
	defer m.mu.Unlock()
	r, err := m.requireExpr(h)
	if err != nil {
		return nil, err
	}
	result := make([]ASTHandle, 0, 4)
	if err = m.sequenceFlattenConcat(r, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (m *ASTManager) sequenceConcatEdge(r astNodeRef, left bool) (astNodeRef, error) {
	for {
		first, second, matched, err := m.sequenceBinaryParts(r, ASTSequenceOpConcat)
		if err != nil || !matched {
			return r, err
		}
		if left {
			r = first
		} else {
			r = second
		}
	}
}

func (m *ASTManager) SequenceLeftmostConcat(h ASTHandle) (ASTHandle, error) {
	if err := m.lock(); err != nil {
		return ASTHandle{}, err
	}
	defer m.mu.Unlock()
	r, err := m.requireExpr(h)
	if err != nil {
		return ASTHandle{}, err
	}
	r, err = m.sequenceConcatEdge(r, true)
	return m.external(r), err
}

func (m *ASTManager) SequenceRightmostConcat(h ASTHandle) (ASTHandle, error) {
	if err := m.lock(); err != nil {
		return ASTHandle{}, err
	}
	defer m.mu.Unlock()
	r, err := m.requireExpr(h)
	if err != nil {
		return ASTHandle{}, err
	}
	r, err = m.sequenceConcatEdge(r, false)
	return m.external(r), err
}

// RegexLoopParameters recognizes the unary declaration-parameter forms of
// re.loop. bounded is true for the two-parameter lower/upper form.
func (m *ASTManager) RegexLoopParameters(h ASTHandle) (body ASTHandle, lower, upper int32, bounded, matched bool, err error) {
	if err = m.lock(); err != nil {
		return
	}
	defer m.mu.Unlock()
	r, err := m.requireExpr(h)
	if err != nil {
		return ASTHandle{}, 0, 0, false, false, err
	}
	if m.nodeAt(r.slot).kind != ASTApp || m.sequencePlugin == nil {
		return ASTHandle{}, 0, 0, false, false, nil
	}
	a := m.appAt(r.slot)
	family, kind, err := m.declIdentity(a.decl)
	if err != nil || family != m.sequencePlugin.family || kind != 20 || a.count != 1 {
		return ASTHandle{}, 0, 0, false, false, err
	}
	info, err := m.funcInfo(m.declAt(a.decl.slot).info)
	if err != nil || info.count < 1 || info.count > 2 {
		return ASTHandle{}, 0, 0, false, false, err
	}
	values := [2]int32{}
	for i := uint32(0); i < info.count; i++ {
		token, e := m.infoParameter(info, i)
		if e != nil {
			return ASTHandle{}, 0, 0, false, false, e
		}
		parameter, e := m.parameter(token)
		if e != nil || parameter.Kind != ASTParameterInt {
			if e == nil {
				e = ErrASTDomain
			}
			return ASTHandle{}, 0, 0, false, false, e
		}
		values[i] = parameter.integer
	}
	child, err := m.readInternal(a.args, 0)
	if err != nil {
		return ASTHandle{}, 0, 0, false, false, err
	}
	return m.external(child), values[0], values[1], info.count == 2, true, nil
}
