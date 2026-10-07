// Source: src/ast/datatype_decl_plugin.h:237-246,314-334;
// src/ast/datatype_decl_plugin.cpp:150-162,739-750;
// src/util/obj_hashtable.h:58-109; src/util/hashtable.h:245-257.
package ast

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

type astDatatypeDefinitionEntry struct {
	hash, state uint32
	name        util.Symbol
	definition  uint32
}
type astDatatypeDefinitionMap struct {
	cells         []astDatatypeDefinitionEntry
	size, deleted uint32
}

func (d *astDatatypeDefinitionMap) construct(b *arena.Budget) error {
	bytes := 8 * uint64(reflect.TypeOf(astDatatypeDefinitionEntry{}).Size())
	if e := b.Reserve(bytes); e != nil {
		return e
	}
	d.cells = make([]astDatatypeDefinitionEntry, 8)
	// Actual entry constructors set only the FREE tag; key/payload stay unread.
	for i := range d.cells {
		d.cells[i].state = 0
	}
	return nil
}
func (d *astDatatypeDefinitionMap) destroy(b *arena.Budget) {
	b.Release(uint64(len(d.cells)) * uint64(reflect.TypeOf(astDatatypeDefinitionEntry{}).Size()))
	d.cells = nil
}

// Object-map keys are actual compressed AST pointers; zero is FREE, one is
// the source DELETED sentinel. Payload is a compressed vector/ref or bool.
// No stored hash substitutes for the source reached AST::hash() read.
type astDatatypeObjectMapEntry struct{ key, value uint64 }
type astDatatypeObjectMap struct {
	cells         []astDatatypeObjectMapEntry
	size, deleted uint32
}

func (d *astDatatypeObjectMap) construct(b *arena.Budget) error {
	if e := b.Reserve(8 * 16); e != nil {
		return e
	}
	d.cells = make([]astDatatypeObjectMapEntry, 8)
	for i := range d.cells {
		d.cells[i].key = 0
	}
	return nil
}
func (d *astDatatypeObjectMap) destroy(b *arena.Budget) {
	b.Release(uint64(len(d.cells)) * 16)
	d.cells = nil
}

// Actual inline ref/vector headers have null backing at default construction.
// They do not call an ergonomic allocating Go wrapper constructor.
type astDatatypeReferenceVector struct {
	raw     util.RuntimeVectorColumn
	manager *ASTManager
	pending ASTRefVectorPushReceipt
}
type astDatatypeUtility struct {
	self                  *astDatatypeUtility
	manager               *ASTManager
	family                FamilyID
	plugin                *astDatatypePluginOwner
	maps                  [9]astDatatypeObjectMap
	references            astDatatypeReferenceVector
	vectors               util.RuntimeUnsignedVector
	start                 uint32
	fullyInterpretedTrail RuntimeASTLiteralVector
	completedMaps         uint8
	bodyCause             error
	constructed, closed   bool
}

func (u *astDatatypeUtility) construct(m *ASTManager) error {
	if u.self != nil {
		return ErrASTHandle
	}
	u.self = u
	u.manager = m
	name, e := m.Symbols.Intern("datatype")
	if e != nil {
		u.bodyCause = e
		return e
	}
	u.family, e = m.families.MkID(name)
	if e != nil {
		u.bodyCause = e
		return e
	}
	// Nine declaration-ordered identical object-table constructor engines.
	for i := range u.maps {
		if e = u.maps[i].construct(m.BudgetField); e != nil {
			u.bodyCause = e
			for j := int(u.completedMaps) - 1; j >= 0; j-- {
				u.maps[j].destroy(m.BudgetField)
			}
			return e
		}
		u.completedMaps++
	}
	u.references.manager = m
	u.references.raw.Bind(m.numerals.RuntimeOwner())
	u.vectors.Bind(m.numerals.RuntimeOwner(), m.BudgetField)
	u.vectors.Constructed = true
	u.start = 0
	u.fullyInterpretedTrail = RuntimeASTLiteralVector{budget: m.BudgetField, constructed: true}
	u.fullyInterpretedTrail.self = &u.fullyInterpretedTrail
	u.fullyInterpretedTrail.raw.Bind(m.numerals.RuntimeOwner())
	if m.pluginAt(u.family).present && m.pluginAt(u.family).kind == astDatatypePlugin {
		u.plugin = m.datatypePlugin
	}
	if u.plugin == nil {
		u.bodyCause = ErrASTDomain
		return u.bodyCause
	} // unbound false-site remains qualified
	u.constructed = true
	return nil
}
func (u *astDatatypeUtility) destroy() error {
	if u.self != u || u.manager == nil {
		return ErrASTHandle
	}
	if u.closed {
		return nil
	}
	// Default constructor scope has no vector elements in the destructor BODY.
	if n, e := u.vectors.Column.Size(); e != nil {
		return e
	} else if n != 0 {
		return ErrASTDomain
	}
	if e := u.fullyInterpretedTrail.SourceFinalize(); e != nil {
		return e
	}
	if e := u.vectors.SourceFinalize(); e != nil {
		return e
	}
	if n, e := u.references.raw.Size(); e != nil {
		return e
	} else if n != 0 {
		return ErrASTDomain
	}
	if e := u.references.raw.Finalize(); e != nil {
		return e
	}
	for i := int(u.completedMaps) - 1; i >= 0; i-- {
		u.maps[i].destroy(u.manager.BudgetField)
	}
	u.closed = true
	return nil
}

type astDatatypePluginOwner struct {
	self                 *astDatatypePluginOwner
	owner                *ASTManager
	next                 *astDatatypePluginOwner
	utility              *astDatatypeUtility
	definitions          astDatatypeDefinitionMap
	definitionBlock      []util.Symbol
	classID              uint32
	family               FamilyID
	header               uint64
	bodyCause            error
	completed, published bool
	membersEnded         bool
}

func (m *ASTManager) registerDatatypePlugin() (FamilyID, error) {
	name, e := m.Symbols.Intern("datatype")
	if e != nil {
		return NullFamilyID, e
	}
	id, e := m.families.GetID(name)
	if e != nil {
		return NullFamilyID, e
	}
	if id != NullFamilyID && m.pluginAt(id).present {
		return id, nil
	}
	bytes := uint64(reflect.TypeOf(astDatatypePluginOwner{}).Size())
	if e = m.BudgetField.Reserve(bytes); e != nil {
		return NullFamilyID, e
	}
	p := &astDatatypePluginOwner{owner: m, header: bytes, next: m.datatypePluginFailed}
	p.self = p
	m.datatypePluginFailed = p
	if e = p.definitions.construct(m.BudgetField); e != nil {
		p.bodyCause = e
		return NullFamilyID, e
	}
	p.classID = 0
	p.completed = true
	id, e = m.families.MkID(name)
	if e != nil {
		p.bodyCause = e
		return NullFamilyID, e
	}
	if e = m.setPlugin(id, astDatatypePlugin); e != nil {
		p.bodyCause = e
		return NullFamilyID, e
	}
	m.datatypePluginFailed = p.next
	p.next = nil
	m.datatypePlugin = p
	p.family = id
	p.published = true
	return id, nil
}
func (m *ASTManager) RegisterDatatypePlugin() (FamilyID, error) {
	if e := m.lock(); e != nil {
		return NullFamilyID, e
	}
	defer m.mu.Unlock()
	m.formatExecuting = true
	defer func() { m.formatExecuting = false }()
	return m.registerDatatypePlugin()
}
func (m *ASTManager) datatypePluginBytes() uint64 {
	size := func(p *astDatatypePluginOwner) uint64 {
		return p.header + uint64(len(p.definitions.cells))*uint64(reflect.TypeOf(astDatatypeDefinitionEntry{}).Size()) + uint64(cap(p.definitionBlock))*uint64(reflect.TypeOf(util.Symbol{}).Size())
	}
	var n uint64
	if m.datatypePlugin != nil {
		n = size(m.datatypePlugin)
	}
	for p := m.datatypePluginFailed; p != nil; p = p.next {
		n += size(p)
	}
	return n
}
func (m *ASTManager) releaseDatatypePluginHistory() {
	release := func(p *astDatatypePluginOwner) {
		p.definitions.destroy(m.BudgetField)
		m.BudgetField.Release(uint64(cap(p.definitionBlock)) * uint64(reflect.TypeOf(util.Symbol{}).Size()))
		p.definitionBlock = nil
		m.BudgetField.Release(p.header)
		p.header = 0
	}
	if m.datatypePlugin != nil {
		release(m.datatypePlugin)
		m.datatypePlugin = nil
	}
	for p := m.datatypePluginFailed; p != nil; p = p.next {
		release(p)
	}
	m.datatypePluginFailed = nil
}

// Source ast.cpp dealloc(plugin) invokes this completed object's BODY and
// automatic members at its actual registry position. History headers survive
// separately until the enclosing Go owner ends.
func (m *ASTManager) finalizeDatatypePlugin() error {
	p := m.datatypePlugin
	if p == nil || p.membersEnded {
		return nil
	}
	for i := range p.definitions.cells {
		if p.definitions.cells[i].state == 2 {
			return ErrASTDomain
		} // populated def ownership is a separate reached dependency
	}
	// Source reset's empty branch is unread/no allocation.
	if p.definitions.size != 0 {
		return ErrASTDomain
	}
	if p.utility != nil {
		if e := p.utility.destroy(); e != nil {
			return e
		}
		p.utility = nil
	}
	// The destructor repeats the empty BODY, then reverse automatic members.
	m.BudgetField.Release(uint64(cap(p.definitionBlock)) * uint64(reflect.TypeOf(util.Symbol{}).Size()))
	p.definitionBlock = nil
	p.definitions.destroy(m.BudgetField)
	p.membersEnded = true
	return nil
}
