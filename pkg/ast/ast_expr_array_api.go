package ast

func (m *ASTManager) ExprArrayMk(out *ASTExprArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.exprArrays.Mk(out)
}
func (m *ASTManager) ExprArrayDel(out *ASTExprArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.exprArrays.Del(out)
}
func (m *ASTManager) ExprArrayCopy(src, out *ASTExprArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.exprArrays.Copy(src, out)
}
func (m *ASTManager) ExprArraySize(ref *ASTExprArray) (uint32, error) {
	if e := m.lock(); e != nil {
		return 0, e
	}
	defer m.mu.Unlock()
	if ref == nil {
		return 0, ErrASTHandle
	}
	return m.exprArrays.Size(*ref)
}
func (m *ASTManager) ExprArrayGet(ref *ASTExprArray, i uint32) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	r, e := m.exprArrays.Get(ref, i)
	return m.external(r), e
}
func (m *ASTManager) ExprArraySet(ref *ASTExprArray, i uint32, h ASTHandle) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.requireExpr(h)
	if e != nil {
		return e
	}
	return m.exprArrays.Set(ref, i, r)
}
func (m *ASTManager) ExprArrayPush(ref *ASTExprArray, h ASTHandle) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.requireExpr(h)
	if e != nil {
		return e
	}
	return m.exprArrays.PushBack(ref, r)
}
func (m *ASTManager) ExprArrayPop(ref *ASTExprArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.exprArrays.PopBack(ref)
}
func (m *ASTManager) ExprArraySetFrom(src *ASTExprArray, i uint32, h ASTHandle, out *ASTExprArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.requireExpr(h)
	if e != nil {
		return e
	}
	return m.exprArrays.SetFrom(src, i, r, out)
}
func (m *ASTManager) ExprArrayPushFrom(src *ASTExprArray, h ASTHandle, out *ASTExprArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.requireExpr(h)
	if e != nil {
		return e
	}
	return m.exprArrays.PushBackFrom(src, r, out)
}
func (m *ASTManager) ExprArrayPopFrom(src, out *ASTExprArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.exprArrays.PopBackFrom(src, out)
}
func (m *ASTManager) ExprArrayUnshare(ref *ASTExprArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.exprArrays.Unshare(ref)
}
func (m *ASTManager) ExprArrayUnfold(ref *ASTExprArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.exprArrays.Unfold(ref)
}
func (m *ASTManager) ExprArrayReroot(ref *ASTExprArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.exprArrays.Reroot(ref)
}
