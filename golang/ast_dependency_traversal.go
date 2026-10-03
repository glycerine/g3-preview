// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package g3

import "github.com/glycerine/g3/golang/arena"

func (d *astDependencyManager) unmarkTodo() error {
	var first error
	for _, r := range d.todo {
		c, e := d.node(r)
		if e == nil {
			c.marked = false
			e = d.store(r, c)
		}
		if first == nil {
			first = e
		}
	}
	d.todo = d.todo[:0]
	return first
}
func (d *astDependencyManager) traverse(root arena.SmallBytesRef, target astNodeRef, out *ASTExprVector) (bool, error) {
	if dependencyNull(root) {
		return false, nil
	}
	d.todo = d.todo[:0]
	c, e := d.node(root)
	if e != nil {
		return false, e
	}
	c.marked = true
	if e = d.store(root, c); e != nil {
		return false, e
	}
	if e = d.push(root); e != nil {
		c.marked = false
		d.store(root, c)
		return false, e
	}
	// Unsafe allocation-exception adapter clears completed dependency marks.
	defer d.unmarkTodo()
	for q := 0; q < len(d.todo); q++ {
		c, e := d.node(d.todo[q])
		if e != nil {
			return false, e
		}
		if c.leaf {
			if out != nil {
				if e = d.owner.exprVectorPush(*out, c.value); e != nil {
					return false, e
				}
			} else if c.value == target {
				return true, nil
			}
		} else {
			for _, child := range [2]arena.SmallBytesRef{c.left, c.right} {
				v, e := d.node(child)
				if e != nil {
					return false, e
				}
				if !v.marked {
					if e = d.push(child); e != nil {
						return false, e
					}
					v.marked = true
					if e = d.store(child, v); e != nil {
						return false, e
					}
				}
			}
		}
	}
	return false, nil
}
func (m *ASTManager) DependencyContains(d ASTDependency, h ASTHandle) (bool, error) {
	if e := m.lock(); e != nil {
		return false, e
	}
	defer m.mu.Unlock()
	r, e := m.dependencies.valid(d)
	if e != nil {
		return false, e
	}
	var v astNodeRef
	if !h.IsNull() {
		v, e = m.requireExpr(h)
		if e != nil {
			return false, e
		}
	}
	return m.dependencies.traverse(r, v, nil)
}
func (m *ASTManager) DependencyLinearizeRaw(d ASTDependency, v ASTExprVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.dependencies.valid(d)
	if e != nil {
		return e
	}
	if _, e = m.vectorState(v.codec()); e != nil {
		return e
	}
	_, e = m.dependencies.traverse(r, astNodeRef{}, &v)
	return e
}
func (m *ASTManager) DependencyLinearize(d ASTDependency, v ASTExprVector) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.dependencies.valid(d)
	if e != nil {
		return e
	}
	if _, e = m.vectorState(v.codec()); e != nil {
		return e
	}
	if _, e = m.dependencies.traverse(r, astNodeRef{}, &v); e != nil {
		return e
	}
	// ast_util.h compacts in place after marking each previously unseen value.
	// Failed mark publication retains earlier set(j) writes and original size.
	mark, e := newASTFastMark1Locked(m)
	if e != nil {
		return e
	}
	defer mark.closeLocked(m)
	s, e := m.vectorState(v.codec())
	if e != nil {
		return e
	}
	j := uint32(0)
	for i := uint32(0); i < s.size; i++ {
		x, e := m.vectorValue(s, i)
		if e != nil {
			return e
		}
		h := m.external(x)
		seen, e := mark.nodeMarkedLocked(m, h)
		if e != nil {
			return e
		}
		if !seen {
			if e = mark.markLocked(m, h, true); e != nil {
				return e
			}
			if i != j {
				if e = m.vectorPut(s, j, x); e != nil {
					return e
				}
			}
			j++
		}
	}
	s.size = j
	if s.capacity != 0 {
		if e = m.vectorPutWord(s.payload, 4, j); e != nil {
			return e
		}
	}
	return m.storeVector(v.codec(), s)
}
