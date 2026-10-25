// Source: datatype::util::get_non_rec_constructor and its m_asts/cache owners.
package ast

import "errors"

// DatatypeUtil owns the constructor preferences of one retained source utility
// instance. Its cache pins the sort and selected declaration until Reset or Close.
type DatatypeUtil struct {
	self         *DatatypeUtil
	manager      *ASTManager
	start        uint32
	constructors map[ASTHandle]ASTHandle
	refs         []ASTRef
}

func NewDatatypeUtil(manager *ASTManager) (*DatatypeUtil, error) {
	if manager == nil {
		return nil, ErrASTHandle
	}
	if _, err := manager.ProofsEnabled(); err != nil {
		return nil, err
	}
	u := &DatatypeUtil{manager: manager, constructors: make(map[ASTHandle]ASTHandle)}
	u.self = u
	return u, nil
}

func (u *DatatypeUtil) NonRecursiveConstructor(sort ASTHandle) (ASTHandle, bool, error) {
	if u == nil || u.self != u || u.manager == nil {
		return ASTHandle{}, false, ErrASTHandle
	}
	m := u.manager
	if err := m.lock(); err != nil {
		return ASTHandle{}, false, err
	}
	if c, found := u.constructors[sort]; found {
		m.mu.Unlock()
		return c, true, nil
	}
	r, err := m.requireKind(sort, ASTSort)
	var c astNodeRef
	var found bool
	if err == nil {
		c, found, err = m.datatypeNonRecursiveConstructorCore(r, map[astNodeRef]bool{r: true}, &u.start)
	}
	constructor := m.external(c)
	m.mu.Unlock()
	if err != nil || !found {
		return ASTHandle{}, found, err
	}
	sortRef, err := m.NewRef(sort)
	if err != nil {
		return ASTHandle{}, false, err
	}
	constructorRef, err := m.NewRef(constructor)
	if err != nil {
		return ASTHandle{}, false, errors.Join(err, m.DelRef(&sortRef))
	}
	u.refs = append(u.refs, sortRef, constructorRef)
	u.constructors[sort] = constructor
	return constructor, true, nil
}

// Reset implements datatype::util::reset: release cached owners and advance
// the starting constructor preference, preserving this utility instance.
func (u *DatatypeUtil) Reset() error {
	if u == nil || u.self != u || u.manager == nil {
		return ErrASTHandle
	}
	if _, err := u.manager.ProofsEnabled(); err != nil {
		return err
	}
	if err := u.releasePreferences(); err != nil {
		return err
	}
	clear(u.constructors)
	u.start++
	return nil
}

func (u *DatatypeUtil) releasePreferences() error {
	for len(u.refs) != 0 {
		i := len(u.refs) - 1
		if err := u.manager.DelRef(&u.refs[i]); err != nil {
			return err
		}
		u.refs = u.refs[:i]
	}
	return nil
}

func (u *DatatypeUtil) Close() error {
	if u == nil || u.self != u {
		return ErrASTHandle
	}
	if u.manager == nil {
		return nil
	}
	if err := u.releasePreferences(); err != nil {
		return err
	}
	u.constructors = nil
	u.manager = nil
	return nil
}
