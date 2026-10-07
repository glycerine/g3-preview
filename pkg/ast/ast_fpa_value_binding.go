// Source: src/ast/fpa_decl_plugin.h:96-108; src/util/mpf.cpp:23-59.
package ast

import (
	"github.com/glycerine/g3/pkg/util"
)

// These procedures borrow the actual receiver's sole physical value vector
// and MPF manager at each reached table call. The table stores no closures,
// mirrored numerals, or manager inferred from a significand token.
func (p *astFloatingPointPluginOwner) valueTableProcs() astFloatValueTableProcs {
	return astFloatValueTableProcs{
		hash: func(id uint32) (uint32, error) {
			if p == nil || p.self != p || p.owner == nil || p.numeric == nil {
				return 0, ErrASTHandle
			}
			value, e := p.values.at(id)
			if e != nil {
				return 0, e
			}
			return p.numeric.Hash(&value)
		},
		equal: func(a, b uint32) (bool, error) {
			if p == nil || p.self != p || p.owner == nil || p.numeric == nil {
				return false, ErrASTHandle
			}
			lhs, e := p.values.at(a)
			if e != nil {
				return false, e
			}
			rhs, e := p.values.at(b)
			if e != nil {
				return false, e
			}
			return p.numeric.EqCore(&lhs, &rhs)
		},
	}
}

// Source: src/ast/fpa_decl_plugin.cpp:52-62. No rollback is performed if
// reserve, SET, insertion, recycling, or NUM destruction throws.
func (p *astFloatingPointPluginOwner) mkValueID(value *util.ArbitraryFloat) (idResult uint32, err error) {
	defer func() {
		if err != nil && p != nil && p.self == p && p.bodyCause == nil {
			p.bodyCause = err
		}
	}()
	if p == nil || p.self != p || p.owner == nil || p.owner.Self != p.owner || p.owner.floatingPointPlugin != p || !p.published || p.sourceEnded {
		return 0, ErrASTHandle
	}
	id := p.ids.Mk()
	if id == ^uint32(0) {
		return 0, ErrASTDomain
	}
	if e := p.values.reserve(id + 1); e != nil {
		return 0, e
	}
	if e := p.values.set(id, value); e != nil {
		return 0, e
	}
	canonical, _, e := p.table.insertIfAbsent(p.owner.BudgetField, p.valueTableProcs(), id)
	if e != nil {
		return 0, e
	}
	if canonical != id {
		if e = p.ids.Recycle(id); e != nil {
			return 0, e
		}
		if e = p.numeric.DelRuntimeValue(p.values.raw.Block, 8+32*uint64(id)); e != nil {
			return 0, e
		}
	}
	return canonical, nil
}
