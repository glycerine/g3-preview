// Source: src/ast/datatype_decl_plugin.h:237-246;
// src/ast/datatype_decl_plugin.cpp:150-162.
// Datatype definitions and utility operations remain unported; registration
// needs no copied container storage for their unused members.
package ast

import "reflect"

type astDatatypePluginOwner struct {
	self                 *astDatatypePluginOwner
	owner                *ASTManager
	next                 *astDatatypePluginOwner
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
	var n uint64
	if m.datatypePlugin != nil {
		n = m.datatypePlugin.header
	}
	for p := m.datatypePluginFailed; p != nil; p = p.next {
		n += p.header
	}
	return n
}
func (m *ASTManager) releaseDatatypePluginHistory() {
	release := func(p *astDatatypePluginOwner) {
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
	p.membersEnded = true
	return nil
}
