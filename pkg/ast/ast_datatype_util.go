// Source: datatype::util::get_non_rec_constructor and its m_asts/cache owners.
package ast

import "github.com/glycerine/g3/pkg/util/diagnostic"

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
	if err := manager.lock(); err != nil {
		return nil, err
	}
	defer manager.mu.Unlock()
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
	r, err := m.requireKind(sort, ASTSort)
	if err == nil {
		err = datatypeUtilAssert(907, "is_datatype(ty)", func() bool { return m.datatypeUtilIsDatatype(r) })
	}
	if err != nil {
		m.mu.Unlock()
		return ASTHandle{}, false, err
	}
	if c, found := u.constructors[sort]; found {
		m.mu.Unlock()
		return c, true, nil
	}
	var c astNodeRef
	var found bool
	if err == nil {
		forbidden := []astNodeRef{r}
		c, found, err = m.datatypeNonRecursiveConstructorCore(r, forbidden, &u.start)
		if err == nil {
			err = datatypeUtilAssert(916, "forbidden_set.back() == ty", func() bool { return forbidden[len(forbidden)-1] == r })
		}
		if err == nil {
			err = datatypeUtilAssert(917, "r", func() bool { return found })
		}
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
	u.refs = append(u.refs, sortRef)
	constructorRef, err := m.NewRef(constructor)
	if err != nil {
		return ASTHandle{}, false, err
	}
	u.refs = append(u.refs, constructorRef)
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
