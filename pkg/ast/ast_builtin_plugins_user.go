package ast

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func astRegisterUserName(dictionary *usymbol.DictionaryInt32, names *[]usymbol.Symbol, budget *arena.Budget, name usymbol.Symbol) (int32, error) {
	if k, ok, e := dictionary.Find(name); e != nil || ok {
		return k, e
	}
	if uint64(len((*names))) > math.MaxInt32 {
		return 0, arena.ErrBounds
	}
	k := int32(len((*names)))
	if len((*names)) == cap((*names)) {
		width := uint64(reflect.TypeOf(usymbol.Symbol{}).Size())
		n, e := astDirectoryGrowth(uint32(cap((*names))), uint32(len((*names))+1), width)
		if e != nil {
			return 0, e
		}
		if e = budget.Reserve(uint64(n) * width); e != nil {
			return 0, e
		}
		next := make([]usymbol.Symbol, len((*names)), int(n))
		copy(next, (*names))
		budget.Release(uint64(cap((*names))) * width)
		(*names) = next
	}
	// Source pushes the spelling before inserting its dictionary entry. A failed
	// table allocation leaves this completed name-vector prefix intact.
	(*names) = append((*names), name)
	if e := dictionary.Insert(name, k); e != nil {
		return 0, e
	}
	return k, nil
}

// An actual freshly constructed user_sort_plugin before set_manager. It owns
// its name vector and dictionary, borrows spelling identities, and has no AST
// callback or fabricated manager. Future plugin inheritance binds it normally.
type astFreshUserPlugin struct {
	dictionary usymbol.DictionaryInt32
	names      []usymbol.Symbol
	budget     *arena.Budget
	ownerBytes uint64
}

func (m *ASTManager) freshUserPlugin() (*astFreshUserPlugin, error) {
	bytes := uint64(reflect.TypeOf(astFreshUserPlugin{}).Size())
	if e := m.BudgetField.Reserve(bytes); e != nil {
		return nil, e
	}
	p := &astFreshUserPlugin{budget: m.BudgetField, ownerBytes: bytes}
	if e := p.dictionary.Init(m.Symbols); e != nil {
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
	p.dictionary.Close()
	p.budget.Release(uint64(cap(p.names))*uint64(reflect.TypeOf(usymbol.Symbol{}).Size()) + p.ownerBytes)
	p.ownerBytes = 0
	p.names = nil
}
