// Licensed under the MIT license.
package g3

func (m *ASTManager) DependencyArrayMk(out *ASTDependencyArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.dependencyArrays.Mk(out)
}
func (m *ASTManager) DependencyArrayDel(out *ASTDependencyArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.dependencyArrays.Del(out)
}
func (m *ASTManager) DependencyArrayCopy(src, out *ASTDependencyArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.dependencyArrays.Copy(src, out)
}
func (m *ASTManager) DependencyArraySize(ref *ASTDependencyArray) (uint32, error) {
	if e := m.lock(); e != nil {
		return 0, e
	}
	defer m.mu.Unlock()
	if ref == nil {
		return 0, ErrASTHandle
	}
	return m.dependencyArrays.Size(*ref)
}
func (m *ASTManager) DependencyArrayGet(ref *ASTDependencyArray, i uint32) (ASTDependency, error) {
	if e := m.lock(); e != nil {
		return ASTDependency{}, e
	}
	defer m.mu.Unlock()
	r, e := m.dependencyArrays.Get(ref, i)
	if e != nil {
		return ASTDependency{}, e
	}
	return m.dependencies.external(r), nil
}
func (m *ASTManager) DependencyArraySet(ref *ASTDependencyArray, i uint32, h ASTDependency) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.dependencies.valid(h)
	if e != nil {
		return e
	}
	return m.dependencyArrays.Set(ref, i, r)
}
func (m *ASTManager) DependencyArrayPush(ref *ASTDependencyArray, h ASTDependency) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.dependencies.valid(h)
	if e != nil {
		return e
	}
	return m.dependencyArrays.PushBack(ref, r)
}
func (m *ASTManager) DependencyArrayPop(ref *ASTDependencyArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.dependencyArrays.PopBack(ref)
}
func (m *ASTManager) DependencyArraySetFrom(src *ASTDependencyArray, i uint32, h ASTDependency, out *ASTDependencyArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.dependencies.valid(h)
	if e != nil {
		return e
	}
	return m.dependencyArrays.SetFrom(src, i, r, out)
}
func (m *ASTManager) DependencyArrayPushFrom(src *ASTDependencyArray, h ASTDependency, out *ASTDependencyArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.dependencies.valid(h)
	if e != nil {
		return e
	}
	return m.dependencyArrays.PushBackFrom(src, r, out)
}
func (m *ASTManager) DependencyArrayPopFrom(src, out *ASTDependencyArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.dependencyArrays.PopBackFrom(src, out)
}
func (m *ASTManager) DependencyArrayUnshare(ref *ASTDependencyArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.dependencyArrays.Unshare(ref)
}
func (m *ASTManager) DependencyArrayUnfold(ref *ASTDependencyArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.dependencyArrays.Unfold(ref)
}
func (m *ASTManager) DependencyArrayReroot(ref *ASTDependencyArray) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	return m.dependencyArrays.Reroot(ref)
}

func (m *ASTManager) DependencyArrayEmpty(ref *ASTDependencyArray) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	if ref == nil {
		return false, ErrASTHandle
	}
	return m.dependencyArrays.Empty(*ref)
}
