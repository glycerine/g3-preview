package ast

func (m *ASTManager) DependencyEmpty() (ASTDependency, error) {
	if e := m.lock(); e != nil {
		return ASTDependency{}, e
	}
	defer m.mu.Unlock()
	return ASTDependency{}, nil
}
func (m *ASTManager) DependencyLeaf(h ASTHandle) (ASTDependency, error) {
	if e := m.lock(); e != nil {
		return ASTDependency{}, e
	}
	defer m.mu.Unlock()
	if h.IsNull() {
		return ASTDependency{}, nil
	}
	v, e := m.requireExpr(h)
	if e != nil {
		return ASTDependency{}, e
	}
	r, e := m.dependencies.leaf(v)
	return m.dependencies.external(r), e
}
func (m *ASTManager) DependencyJoin(a, b ASTDependency) (ASTDependency, error) {
	if e := m.lock(); e != nil {
		return ASTDependency{}, e
	}
	defer m.mu.Unlock()
	x, e := m.dependencies.valid(a)
	if e != nil {
		return ASTDependency{}, e
	}
	y, e := m.dependencies.valid(b)
	if e != nil {
		return ASTDependency{}, e
	}
	r, e := m.dependencies.join(x, y)
	return m.dependencies.external(r), e
}
func (m *ASTManager) DependencyInc(h ASTDependency) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.dependencies.valid(h)
	if e != nil {
		return e
	}
	return m.dependencies.inc(r)
}
func (m *ASTManager) DependencyDec(h ASTDependency) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.dependencies.valid(h)
	if e != nil {
		return e
	}
	return m.dependencies.dec(r)
}
func (m *ASTManager) DependencyRefCount(h ASTDependency) (uint32, error) {
	if e := m.lock(); e != nil {
		return 0, e
	}
	defer m.mu.Unlock()
	r, e := m.dependencies.valid(h)
	if e != nil {
		return 0, e
	}
	if dependencyNull(r) {
		return 0, ErrASTDomain
	}
	c, e := m.dependencies.node(r)
	return c.rc, e
}
func (m *ASTManager) DependencyIsLeaf(h ASTDependency) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	r, e := m.dependencies.valid(h)
	if e != nil {
		return false, e
	}
	if dependencyNull(r) {
		return false, ErrASTDomain
	}
	c, e := m.dependencies.node(r)
	return c.leaf, e
}

// Both scalars publish the source statement prefix when a join allocation fails.
// Caller must retain and later release each returned nonnull RC0 root explicitly.
func (m *ASTManager) DependencyJoinValues(values []ASTHandle) (ASTDependencyJoinProgress, error) {
	if e := m.lock(); e != nil {
		return ASTDependencyJoinProgress{}, e
	}
	defer m.mu.Unlock()
	var p ASTDependencyJoinProgress
	for _, h := range values {
		var v astNodeRef
		if !h.IsNull() {
			var e error
			v, e = m.requireExpr(h)
			if e != nil {
				return p, e
			}
		}
		leaf, e := m.dependencies.leaf(v)
		if e != nil {
			return p, e
		}
		p.PendingLeaf = m.dependencies.external(leaf)
		root, e := m.dependencies.join(p.Completed.record, leaf)
		if e != nil {
			return p, e
		}
		p.Completed = m.dependencies.external(root)
		p.PendingLeaf = ASTDependency{}
	}
	return p, nil
}
