// Licensed under the MIT license.
package g3

func (m *ASTManager) releasePages() {
	m.budget.Release(m.pageBytes + m.directoryBytes)
	m.pageBytes = 0
	m.directoryBytes = 0
	m.nodes = nil
	m.sorts = nil
	m.decls = nil
	m.apps = nil
	m.vars = nil
	m.quantifiers = nil
	m.parameters = nil
	m.infos = nil
	m.references = nil
}
