// Source: datatype::util::get_non_rec_constructor and its m_asts/cache owners.
package ast

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// DatatypeUtil owns the constructor preferences of one retained source utility
// instance. Its caches pin datatype sorts, all constructors and selected
// preferences until Reset or Close.
type DatatypeUtil struct {
	self         *DatatypeUtil
	manager      *ASTManager
	start        uint32
	constructors map[ASTHandle]ASTHandle
	lists        map[ASTHandle]*datatypeConstructorList
	lexical      ASTRef // interrupted constructor-local cleanup, separate from m_asts
	refs         []ASTRef
	operation    uint8 // 0 ready, 1 interrupted Reset, 2 closing
}

func NewDatatypeUtil(manager *ASTManager) (*DatatypeUtil, error) {
	if manager == nil {
		return nil, ErrASTHandle
	}
	if err := manager.lock(); err != nil {
		return nil, err
	}
	defer manager.mu.Unlock()
	return manager.newDatatypeUtil()
}

// newDatatypeUtil constructs the same concrete utility for an already locked
// manager, including the plugin's lazily allocated utility member.
func (manager *ASTManager) newDatatypeUtil() (*DatatypeUtil, error) {
	name, err := manager.Symbols.Intern("datatype")
	if err != nil {
		return nil, err
	}
	if _, err = manager.families.MkID(name); err != nil {
		return nil, err
	}
	if err = datatypeUtilAssert(745, "m_plugin", func() bool { return manager.datatypePlugin != nil && manager.datatypePlugin.published }); err != nil {
		return nil, err
	}
	u := &DatatypeUtil{manager: manager, constructors: make(map[ASTHandle]ASTHandle), lists: make(map[ASTHandle]*datatypeConstructorList)}
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
	defer m.mu.Unlock()
	return u.nonRecursiveConstructor(sort)
}

func (u *DatatypeUtil) nonRecursiveConstructor(sort ASTHandle) (ASTHandle, bool, error) {
	if u.lexical != (ASTRef{}) {
		return ASTHandle{}, false, ErrASTPending
	}
	m := u.manager
	r, err := m.requireKind(sort, ASTSort)
	if err == nil {
		err = datatypeUtilAssert(907, "is_datatype(ty)", func() bool { return m.datatypeUtilIsDatatype(r) })
	}
	if err != nil {
		return ASTHandle{}, false, err
	}
	if c, found := u.constructors[sort]; found {
		return c, !c.IsNull(), nil
	}
	var c astNodeRef
	var found bool
	if err == nil {
		forbidden := []astNodeRef{r}
		c, found, err = m.datatypeNonRecursiveConstructorCore(r, forbidden, &u.start, u)
		if err == nil {
			err = datatypeUtilAssert(916, "forbidden_set.back() == ty", func() bool { return forbidden[len(forbidden)-1] == r })
		}
		if err == nil {
			err = datatypeUtilAssert(917, "r", func() bool { return found })
		}
	}
	constructor := m.external(c)
	if err != nil {
		return ASTHandle{}, found, err
	}
	sortRef, err := m.newRef(sort)
	if err != nil {
		return ASTHandle{}, false, err
	}
	u.refs = append(u.refs, sortRef)
	constructorRef, err := m.newRef(constructor)
	if err != nil {
		return ASTHandle{}, false, err
	}
	u.refs = append(u.refs, constructorRef)
	u.constructors[sort] = constructor
	return constructor, found, nil
}

// Reset clears the preference map before retiring m_asts, then advances the
// unsigned preference once. An interrupted reset cannot republish preferences.
func (u *DatatypeUtil) Reset() error {
	if u == nil || u.self != u || u.manager == nil || u.operation == 2 {
		return ErrASTHandle
	}
	if u.operation == 0 {
		clear(u.constructors)
		clear(u.lists)
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
	m := u.manager
	if err := m.lock(); err != nil {
		return err
	}
	defer m.mu.Unlock()
	return u.releasePreferencesLocked()
}

func (u *DatatypeUtil) releasePreferencesLocked() error {
	if u.lexical != (ASTRef{}) {
		if err := u.manager.delRef(&u.lexical); err != nil {
			return err
		}
	}
	for i := range u.refs {
		if u.refs[i] == (ASTRef{}) {
			continue
		}
		if err := u.manager.delRef(&u.refs[i]); err != nil {
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
	u.beginClose()
	m := u.manager
	if err := m.lock(); err != nil {
		return err
	}
	defer m.mu.Unlock()
	return u.closeLocked()
}

func (u *DatatypeUtil) beginClose() {
	u.operation = 2
	// Source destructor body deletes vectors before m_asts retires pins.
	for _, list := range u.lists {
		list.values = nil
	}
}

func (u *DatatypeUtil) closeLocked() error {
	if u.manager == nil {
		return nil
	}
	u.beginClose()
	if err := u.releasePreferencesLocked(); err != nil {
		return err
	}
	u.constructors = nil
	u.lists = nil
	u.manager = nil
	return nil
}

// Callers hold the manager lock. Source datatype recognition inspects the
// plugin family/kind, rather than attempting definition instantiation.
func (m *ASTManager) datatypeUtilIsDatatype(r astNodeRef) bool {
	if m.datatypePlugin == nil {
		return false
	}
	info, err := m.sortInfo(m.sortAt(r.slot).info)
	return err == nil && info.family == m.datatypePlugin.family && info.kind == ASTDatatypeSort
}

func datatypeForbiddenContains(forbidden []astNodeRef, sort astNodeRef) bool {
	for _, r := range forbidden {
		if r == sort {
			return true
		}
	}
	return false
}

func datatypeUtilAssert(line int32, condition string, evaluate func() bool) error {
	if !diagnostic.DebugBuild {
		return nil
	}
	d := diagnostic.BorrowProcessDebugContext()
	enabled, err := diagnostic.SourceAssertionsEnabled(d)
	if err != nil || !enabled {
		return err
	}
	if evaluate() {
		return nil
	}
	return diagnostic.DebugAssertionFailure(d, "src/ast/datatype_decl_plugin.cpp", line, condition)
}

// Source datatype::util::get_datatype_constructors (753-767). The map owns
// a vector object whose prefix is visible even after instantiation stops.
type datatypeConstructorList struct{ values []ASTHandle }

// Constructors returns a snapshot of the borrowed declarations. The utility
// retains the source pins until Reset or Close; the snapshot owns no AST refs.
func (u *DatatypeUtil) Constructors(sort ASTHandle) ([]ASTHandle, error) {
	if u == nil || u.self != u || u.manager == nil || u.operation != 0 {
		return nil, ErrASTHandle
	}
	m := u.manager
	if err := m.lock(); err != nil {
		return nil, err
	}
	defer m.mu.Unlock()
	r, err := m.requireKind(sort, ASTSort)
	if err != nil {
		return nil, err
	}
	values, err := u.constructorList(r)
	if err != nil {
		return nil, err
	}
	return append([]ASTHandle(nil), values...), nil
}

// Caller holds the manager lock. No callbacks or universal owner adapters.
func (u *DatatypeUtil) constructorList(sort astNodeRef) ([]ASTHandle, error) {
	m := u.manager
	if err := datatypeUtilAssert(753, "is_datatype(ty)", func() bool { return m.datatypeUtilIsDatatype(sort) }); err != nil {
		return nil, err
	}
	if u.lexical != (ASTRef{}) {
		return nil, ErrASTPending
	}
	h := m.external(sort)
	if list := u.lists[h]; list != nil {
		return list.values, nil
	}
	pin, err := m.newRef(h)
	if err != nil {
		return nil, err
	}
	u.refs = append(u.refs, pin)
	list := &datatypeConstructorList{}
	u.lists[h] = list
	definition, err := m.datatypeDefinitionForSort(sort)
	if err != nil {
		return nil, err
	}
	for i := range definition.constructors {
		decl, err := m.datatypeConstructorDecl(sort, definition, &definition.constructors[i])
		if err != nil {
			return nil, err
		}
		// instantiate returns an owning func_decl_ref before m_asts acquires
		// its separate reference and the borrowed declaration is appended.
		u.lexical, err = m.newRef(m.external(decl))
		if err != nil {
			return nil, err
		}
		pin, err = m.newRef(m.external(decl))
		if err == nil {
			u.refs = append(u.refs, pin)
			list.values = append(list.values, m.external(decl))
		}
		closeErr := m.delRef(&u.lexical)
		if err = errors.Join(err, closeErr); err != nil {
			return nil, err
		}
	}
	return list.values, nil
}
