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
	operation    uint8 // 0 ready, 1 interrupted Reset, 2 closing
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
	if u == nil || u.self != u || u.manager == nil || u.operation != 0 {
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

// Reset clears the preference map before retiring m_asts, then advances the
// unsigned preference once. An interrupted reset cannot republish preferences.
func (u *DatatypeUtil) Reset() error {
	if u == nil || u.self != u || u.manager == nil || u.operation == 2 {
		return ErrASTHandle
	}
	if u.operation == 0 {
		clear(u.constructors)
		u.operation = 1
	}
	if err := u.releasePreferences(); err != nil {
		return err
	}
	u.start++
	u.operation = 0
	return nil
}

// Source ref_vector dec_range walks forward. Retired zero tokens identify
// actual completed decrements; owning retries never replay them.
func (u *DatatypeUtil) releasePreferences() error {
	for i := range u.refs {
		if u.refs[i] == (ASTRef{}) {
			continue
		}
		if err := u.manager.DelRef(&u.refs[i]); err != nil {
			return err
		}
	}
	u.refs = nil
	return nil
}

func (u *DatatypeUtil) Close() error {
	if u == nil || u.self != u {
		return ErrASTHandle
	}
	if u.manager == nil {
		return nil
	}
	u.operation = 2
	if err := u.releasePreferences(); err != nil {
		return err
	}
	u.constructors = nil
	u.manager = nil
	return nil
}
