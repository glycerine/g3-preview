// Source: src/ast/ast_util.cpp:173-180; src/ast/ast.cpp:2064-2211.
// Source authors: Leonardo de Moura; Nikolaj Bjorner.
// Licensed under the MIT license; see LICENSE.txt.
package g3

// This concrete call-local argument reader selects actual borrowed storage.
// It never clones a pointer array or grants ownership of its elements.
type astAppArguments struct {
	slice []ASTHandle
	span  ASTLiteralVectorSpan
	count uint32
	raw   bool
}

func (a astAppArguments) at(i uint32) (ASTHandle, error) {
	if a.raw {
		return a.span.At(i)
	}
	return a.slice[i], nil
}
func (m *ASTManager) MkOrSpan(n uint32, span ASTLiteralVectorSpan) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	switch n {
	case 0:
		return m.external(m.falseApp), nil
	case 1:
		return span.At(0)
	}
	// Source basic-plugin expr overload uses this same OR declaration.
	if !m.plugins[0].present {
		return ASTHandle{}, nil
	}
	r, e := m.mkAppArguments(m.basic.declarations[ASTOpOr], astAppArguments{span: span, count: n, raw: true})
	return m.external(r), e
}
