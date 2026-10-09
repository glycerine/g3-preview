// Source: src/ast/ast.cpp:1354,2213-2238; src/ast/ast.h:1811-1825.
package ast

import (
	"math"

	"github.com/glycerine/g3/pkg/util"
)

// The special token denotes only this manager's active lexical info object.
// registerNode/copyInfo creates the real owned info before node publication.
func (m *ASTManager) mkFreshFuncDecl(prefix, suffix util.Symbol, domain []ASTHandle, sort ASTHandle) (astNodeRef, error) {
	if m.freshInfoActive || m.freshInfoEpoch == math.MaxUint32 {
		return astNodeRef{}, ErrASTPending
	}
	m.freshInfoEpoch++
	m.freshInfo = astInfoRecord{live: true, family: NullFamilyID, kind: -1, flags: ASTFuncFlags{Skolem: true}}
	m.freshInfoActive = true
	defer func() { m.freshInfoActive = false; m.freshInfo = astInfoRecord{} }()
	info := ASTFuncDeclInfo{m.origin, math.MaxUint32, m.freshInfoEpoch}
	var d astNodeRef
	var e error
	if prefix == (util.Symbol{}) && suffix == (util.Symbol{}) {
		d, e = m.mkFuncDeclInfo(util.NumericalSymbol(m.freshID), domain, sort, info, false)
	} else {
		d, e = m.freshNamedDecl(prefix, suffix, domain, sort, info)
	}
	if e != nil {
		return astNodeRef{}, e
	}
	m.freshID++
	if ASTSourceDebug {
		p := m.declAt(d.slot).info
		if p == (ASTFuncDeclInfo{}) {
			return astNodeRef{}, ErrASTDomain
		}
		v, e := m.funcInfo(p)
		if e != nil {
			return astNodeRef{}, e
		}
		if !v.flags.Skolem {
			return astNodeRef{}, ErrASTDomain
		}
	}
	return d, nil
}

func (m *ASTManager) MkFreshFuncDecl(prefix, suffix util.Symbol, domain []ASTHandle, sort ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	r, e := m.mkFreshFuncDecl(prefix, suffix, domain, sort)
	return m.external(r), e
}

// MkFreshConst translates the char-prefix overload's symbol construction before
// the fresh declaration and its later zero-argument application construction.
func (m *ASTManager) MkFreshConst(prefix string, sort ASTHandle) (ASTHandle, error) {
	return m.MkFreshConstNullable(&prefix, sort)
}

// MkFreshConstNullable retains the source symbol(nullptr) constructor shortcut.
func (m *ASTManager) MkFreshConstNullable(prefix *string, sort ASTHandle) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	p, e := m.Symbols.InternNullable(prefix)
	if e != nil {
		return ASTHandle{}, e
	}
	d, e := m.mkFreshFuncDecl(p, util.Symbol{}, nil, sort)
	if e != nil {
		return ASTHandle{}, e
	}
	r, e := m.mkApp(d, nil)
	return m.external(r), e
}

// This concrete source else scope destroys buffer before fresh_id publication.
func (m *ASTManager) freshNamedDecl(prefix, suffix util.Symbol, domain []ASTHandle, sort ASTHandle, info ASTFuncDeclInfo) (astNodeRef, error) {
	var buffer util.StringBuffer64
	defer buffer.Close()
	if e := buffer.AppendSymbol(m.Symbols, prefix); e != nil {
		return astNodeRef{}, e
	}
	if prefix == (util.Symbol{}) {
		if e := buffer.AppendCString("sk"); e != nil {
			return astNodeRef{}, e
		}
	}
	if e := buffer.AppendCString("!"); e != nil {
		return astNodeRef{}, e
	}
	if suffix != (util.Symbol{}) {
		if e := buffer.AppendSymbol(m.Symbols, suffix); e != nil {
			return astNodeRef{}, e
		}
		if e := buffer.AppendCString("!"); e != nil {
			return astNodeRef{}, e
		}
	}
	if e := buffer.AppendInt32(int32(m.freshID)); e != nil {
		return astNodeRef{}, e
	}
	text, e := buffer.CStr()
	if e != nil {
		return astNodeRef{}, e
	}
	name, e := m.Symbols.InternBytes([]byte(text))
	if e != nil {
		return astNodeRef{}, e
	}
	return m.mkFuncDeclInfo(name, domain, sort, info, false)
}
