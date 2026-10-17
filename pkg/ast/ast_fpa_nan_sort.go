// Source: src/ast/fpa_decl_plugin.cpp:950-957;
// src/ast/rewriter/fpa_rewriter.cpp:476-481.
package ast

// MkFPANaNOfSort is the concrete fpa_util::mk_nan(sort*) call used by the
// infinity comparison rewrites. The supplied sort selects its two widths.
func (m *ASTManager) MkFPANaNOfSort(sort ASTHandle) (ASTHandle, error) {
	if err := m.lock(); err != nil {
		return ASTHandle{}, err
	}
	defer m.mu.Unlock()
	previous := m.formatExecuting
	m.formatExecuting = true
	defer func() { m.formatExecuting = previous }()
	decl, err := m.floatingPointSpecialDecl(astFPANaN, nil, sort)
	if err != nil {
		return ASTHandle{}, err
	}
	app, err := m.mkApp(decl, nil)
	return m.external(app), err
}
