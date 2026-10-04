// Licensed under the MIT license.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

func astRegisterUserName(dictionary *astUserDictionary, names *[]Symbol, budget *arena.Budget, name Symbol) (int32, error) {
	if k, ok, e := dictionary.find(name); e != nil || ok {
		return k, e
	}
	if uint64(len((*names))) > math.MaxInt32 {
		return 0, arena.ErrBounds
	}
	k := int32(len((*names)))
	if len((*names)) == cap((*names)) {
		width := uint64(reflect.TypeOf(Symbol{}).Size())
		n, e := astDirectoryGrowth(uint32(cap((*names))), uint32(len((*names))+1), width)
		if e != nil {
			return 0, e
		}
		if e = budget.Reserve(uint64(n) * width); e != nil {
			return 0, e
		}
		next := make([]Symbol, len((*names)), int(n))
		copy(next, (*names))
		budget.Release(uint64(cap((*names))) * width)
		(*names) = next
	}
	// Source pushes the spelling before inserting its dictionary entry. A failed
	// table allocation leaves this completed name-vector prefix intact.
	(*names) = append((*names), name)
	if e := dictionary.insert(name, k); e != nil {
		return 0, e
	}
	return k, nil
}

// An actual freshly constructed user_sort_plugin before set_manager. It owns
// its name vector and dictionary, borrows spelling identities, and has no AST
// callback or fabricated manager. Future plugin inheritance binds it normally.
type astFreshUserPlugin struct {
	dictionary astUserDictionary
	names      []Symbol
	budget     *arena.Budget
	ownerBytes uint64
}

func (m *ASTManager) freshUserPlugin() (*astFreshUserPlugin, error) {
	bytes := uint64(reflect.TypeOf(astFreshUserPlugin{}).Size())
	if e := m.budget.Reserve(bytes); e != nil {
		return nil, e
	}
	p := &astFreshUserPlugin{budget: m.budget, ownerBytes: bytes}
	if e := p.dictionary.init(m.symbols, m.budget, m.builtin.userTable.outOfMemory); e != nil {
		p.close()
		return nil, e
	}
	for _, name := range m.builtin.userNames {
		if _, e := astRegisterUserName(&p.dictionary, &p.names, p.budget, name); e != nil {
			p.close()
			return nil, e
		}
	}
	return p, nil
}
func (p *astFreshUserPlugin) close() {
	if p == nil {
		return
	}
	p.dictionary.close()
	p.budget.Release(uint64(cap(p.names))*uint64(reflect.TypeOf(Symbol{}).Size()) + p.ownerBytes)
	p.ownerBytes = 0
	p.names = nil
}
