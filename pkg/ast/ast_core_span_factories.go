// Source: src/ast/ast_util.cpp:173-180; src/ast/ast.cpp:2064-2211.
// Source authors: Leonardo de Moura; Nikolaj Bjorner.
// Licensed under the MIT license; see LICENSE.txt.
package ast

// This concrete call-local argument reader selects actual borrowed storage.
// It never clones a pointer array or grants ownership of its elements.
type AppArguments struct {
	slice        []ASTHandle
	format       *astFormatRefVector
	formatOwner  *ASTManager
	vector       astRefVectorState
	vectorOffset uint32
	refVector    bool
	span         ASTLiteralVectorSpan
	count        uint32
	raw          bool
}

func (a AppArguments) at(i uint32) (ASTHandle, error) {
	if a.refVector {
		if i > ^uint32(0)-a.vectorOffset {
			return ASTHandle{}, ErrASTDomain
		}
		r, e := a.formatOwner.vectorValue(a.vector, a.vectorOffset+i)
		return a.formatOwner.external(r), e
	}
	if a.format != nil {
		r, e := a.format.at(i)
		if e != nil {
			return ASTHandle{}, e
		}
		return a.formatOwner.external(r), nil
	}
	if a.raw {
		return a.span.At(i)
	}
	return a.slice[i], nil
}

func (a AppArguments) size() uint64 {
	if a.refVector || a.raw || a.format != nil {
		return uint64(a.count)
	}
	return uint64(len(a.slice))
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
	if !m.pluginAt(0).present {
		return ASTHandle{}, nil
	}
	r, e := m.mkAppArguments(m.basic.declarations[ASTOpOr], AppArguments{span: span, count: n, raw: true})
	return m.external(r), e
}

// At reads a borrowed argument without acquiring ownership.
func (a AppArguments) At(i uint32) (ASTHandle, error) { return a.at(i) }
func (a AppArguments) Count() uint32                  { return a.count }
func (a AppArguments) Size() uint64                   { return a.size() }
func SliceArguments(values []ASTHandle) AppArguments {
	return AppArguments{slice: values, count: uint32(len(values))}
}
func (m *ASTManager) RefVectorArgumentsRange(v ASTRefVector, offset, count uint32) (AppArguments, error) {
	if e := m.lock(); e != nil {
		return AppArguments{}, e
	}
	defer m.mu.Unlock()
	state, e := m.vectorState(v)
	if e != nil {
		return AppArguments{}, e
	}
	if offset > state.size || count > state.size-offset {
		return AppArguments{}, ErrASTDomain
	}
	return AppArguments{refVector: true, vector: state, vectorOffset: offset, formatOwner: m, count: count}, nil
}
func (m *ASTManager) UpdateQuantifierArguments(q, body ASTHandle, patterns, noPatterns AppArguments) (ASTHandle, error) {
	return m.updateQuantifierArguments(q, body, false, 0, patterns, noPatterns, astUpdatePatterns|astUpdateNoPatterns)
}
