// Source: src/ast/format.cpp:25-117; src/ast/ast.cpp:1578-1585,1635-1638.
// Source author: Leonardo de Moura (2008-01-20).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

// Concrete pointer-free source ptr_vector specialization. The actual backing
// contains its capacity/size words followed by encoded plugin bindings. There
// is no second hot registry or accounting-only Runtime allocation.
type astPluginVector struct{ words []uint32 }

func (v *astPluginVector) bytes() uint64         { return uint64(len(v.words)) * 4 }
func (v *astPluginVector) close(b *arena.Budget) { b.Release(v.bytes()); v.words = nil }
func (m *ASTManager) pluginAt(id FamilyID) astPluginBinding {
	if id < 0 || len(m.plugins.words) == 0 || uint32(id) >= m.plugins.words[1] {
		return astPluginBinding{}
	}
	x := m.plugins.words[2+uint32(id)]
	if x == 0 {
		return astPluginBinding{}
	}
	return astPluginBinding{astPluginKind(x - 1), true}
}
func (m *ASTManager) setPlugin(id FamilyID, kind astPluginKind) error {
	if id < 0 || uint64(id) >= math.MaxUint32 {
		return arena.ErrBounds
	}
	need := uint32(id) + 1
	v := &m.plugins
	var size, capacity uint32
	if len(v.words) != 0 {
		capacity = v.words[0]
		size = v.words[1]
	}
	// Literal setx resize(default-null) runs one push at a time, retaining every
	// completed size/growth prefix if a later source allocation refuses.
	for size < need {
		if size == capacity {
			next := (uint64(capacity)*3 + 1) / 2
			if next < 2 {
				next = 2
			}
			if next > math.MaxUint32 || next+2 > uint64(^uint(0)>>1)/4 {
				return arena.ErrBounds
			}
			bytes := (next + 2) * 4
			if e := m.budget.Reserve(bytes); e != nil {
				return e
			}
			a := make([]uint32, int(next)+2)
			a[0] = uint32(next)
			a[1] = size
			for i := uint32(0); i < size; i++ {
				a[2+i] = v.words[2+i]
			}
			m.budget.Release(v.bytes())
			v.words = a
			capacity = uint32(next)
		}
		v.words[2+size] = 0
		size++
		v.words[1] = size
	}
	v.words[2+uint32(id)] = uint32(kind) + 1
	return nil
}

type astFormatPluginOwner struct {
	next                       *astFormatPluginOwner
	self                       *astFormatPluginOwner
	owner                      *ASTManager
	names                      [7]Symbol
	family                     FamilyID
	sort                       astNodeRef
	header                     uint64
	completed, published       bool
	bodyCause, destructorCause error
}

func (m *ASTManager) formatPluginBytes() uint64 {
	var n uint64
	if m.formatPlugin != nil {
		n += m.formatPlugin.header
	}
	for p := m.formatPluginFailed; p != nil; p = p.next {
		n += p.header
	}
	return n
}
func (m *ASTManager) releaseFormatPluginHeaders() {
	if p := m.formatPlugin; p != nil {
		m.budget.Release(p.header)
		p.header = 0
		m.formatPlugin = nil
	}
	if m.sourceClosing {
		return
	}
	for p := m.formatPluginFailed; p != nil; {
		next := p.next
		m.budget.Release(p.header)
		p.header = 0
		p = next
	}
	m.formatPluginFailed = nil
}
func (m *ASTManager) finalizeFormatPlugin() error {
	p := m.formatPlugin
	if p == nil || p.sort.slot == 0 {
		return nil
	}
	consumed, e := m.decNode(p.sort)
	if consumed {
		p.sort = astNodeRef{}
	}
	if e != nil && p.destructorCause == nil {
		p.destructorCause = e
	}
	return e
}
func (m *ASTManager) formatFamilyID() (FamilyID, error) {
	// Source constructs symbol before reading has_plugin. Its source lookup is
	// non-owning; the cold wrapper is acquired only on the actual new-expression.
	name, e := m.symbols.Intern("format")
	if e != nil {
		return NullFamilyID, e
	}
	id, e := m.families.GetID(name)
	if e != nil {
		return NullFamilyID, e
	}
	found := id != NullFamilyID
	if found && m.pluginAt(id).present {
		return m.families.MkID(name)
	}
	bytes := uint64(reflect.TypeOf(astFormatPluginOwner{}).Size())
	if e = m.budget.Reserve(bytes); e != nil {
		return NullFamilyID, e
	}
	p := &astFormatPluginOwner{owner: m, header: bytes}
	p.self = p
	// Unfinished wrapper is named immediately, before actual symbol constructors.
	p.next = m.formatPluginFailed
	m.formatPluginFailed = p
	for i, s := range [...]string{"nil", "string", "indent", "compose", "choice", "cr", "cr++"} {
		p.names[i], e = m.symbols.Intern(s)
		if e != nil {
			p.bodyCause = e
			return NullFamilyID, e
		}
	}
	p.completed = true
	id, e = m.families.MkID(name)
	if e != nil {
		p.bodyCause = e
		return NullFamilyID, e
	}
	if e = m.setPlugin(id, astFormatPlugin); e != nil {
		p.bodyCause = e
		return NullFamilyID, e
	}
	// register_plugin publishes actual pointer before set_manager creates sort.
	m.formatPluginFailed = p.next
	p.next = nil
	m.formatPlugin = p
	p.published = true
	p.family = id
	if !m.IsFormatManager() {
		p.bodyCause = ErrASTDomain
		return NullFamilyID, ErrASTDomain
	}
	r, e := m.formatSortCreate(name, id)
	p.sort = r
	if e == nil {
		e = m.incNode(r)
	}
	if e != nil {
		p.bodyCause = e
		return NullFamilyID, e
	}
	return m.families.MkID(name)
}

func (m *ASTManager) GetFormatFamilyID() (FamilyID, error) {
	f, e := m.GetFormatManager()
	if e != nil {
		return NullFamilyID, e
	}
	if e = f.lock(); e != nil {
		return NullFamilyID, e
	}
	defer f.mu.Unlock()
	f.formatExecuting = true
	defer func() { f.formatExecuting = false }()
	return f.formatFamilyID()
}
