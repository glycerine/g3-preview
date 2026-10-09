// Translation of ast.h/ast.cpp::family_manager, Leonardo de Moura.
package ast

import (
	"errors"
	"math"
	"reflect"
	"sync"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"github.com/glycerine/insdict"
)

type FamilyID = int32

const NullFamilyID FamilyID = -1

var ErrFamilyOwner = errors.New("g3: invalid or copied family manager")

// FamilyManager owns the source symbol table and registration-order name vector.
// Symbols borrow their manager, which must outlive this owner. This registry does
// not instantiate AST nodes or declaration plugins. Do not shallow-copy it.
type FamilyManager struct {
	mu      sync.Mutex
	self    *FamilyManager
	symbols *util.SymbolManager
	budget  *arena.Budget
	table   insdict.Dict[util.Symbol, int32]
	names   []util.Symbol
	next    FamilyID
	closed  bool
}

type FamilyStats struct {
	NextID                 FamilyID
	Names, NamesCapacity   uint32
	TableSize              uint32
	OwnerBytes, NamesBytes uint64
}

func NewFamilyManager(symbols *util.SymbolManager, budget *arena.Budget) (*FamilyManager, error) {
	if symbols == nil {
		return nil, ErrFamilyOwner
	}
	owner := uint64(reflect.TypeOf(FamilyManager{}).Size())
	if e := budget.Reserve(owner); e != nil {
		return nil, e
	}
	table := insdict.NewDictFunc[util.Symbol, int32](func(symbol util.Symbol) uint64 {
		hash, _ := symbols.Hash(symbol)
		return uint64(hash)
	})
	f := &FamilyManager{symbols: symbols, budget: budget, table: *table}
	f.self = f
	return f, nil
}

// Called under the family mutex. Never retain the symbol lock across a public
// table operation: that table acquires its own lock before the symbol lock.
func (f *FamilyManager) open() error {
	if f == nil || f.self != f {
		return ErrFamilyOwner
	}
	if f.closed {
		return arena.ErrClosed
	}
	_, e := f.symbols.Hash(util.Symbol{})
	return e
}

func (f *FamilyManager) lock() error {
	if f == nil {
		return ErrFamilyOwner
	}
	f.mu.Lock()
	if e := f.open(); e != nil {
		f.mu.Unlock()
		return e
	}
	return nil
}

func (f *FamilyManager) pushName(s util.Symbol) {
	f.names = append(f.names, s)
}

func (f *FamilyManager) MkID(s util.Symbol) (FamilyID, error) {
	if e := f.lock(); e != nil {
		return NullFamilyID, e
	}
	defer f.mu.Unlock()
	if s == (util.Symbol{}) {
		return NullFamilyID, util.ErrInvalidSymbol
	}
	if _, e := f.symbols.Hash(s); e != nil {
		return NullFamilyID, e
	}
	id, found := f.table.Get2(s)
	if found {
		return id, nil
	}
	// Signed C++ next++ would overflow here. Reject that undefined domain before
	// mutation, while allowing already registered lookups at an exhausted counter.
	if f.next == math.MaxInt32 {
		return NullFamilyID, arena.ErrBounds
	}
	id = f.next
	f.next++
	f.table.Put(s, id)
	f.pushName(s)
	return id, nil
}

func (f *FamilyManager) GetID(s util.Symbol) (FamilyID, error) {
	if e := f.lock(); e != nil {
		return NullFamilyID, e
	}
	defer f.mu.Unlock()
	if s == (util.Symbol{}) {
		return NullFamilyID, util.ErrInvalidSymbol
	}
	if _, e := f.symbols.Hash(s); e != nil {
		return NullFamilyID, e
	}
	id, found := f.table.Get2(s)
	if !found {
		return NullFamilyID, nil
	}
	return id, nil
}

func (f *FamilyManager) HasSymbol(s util.Symbol) (bool, error) {
	if e := f.lock(); e != nil {
		return false, e
	}
	defer f.mu.Unlock()
	if s == (util.Symbol{}) {
		return false, util.ErrInvalidSymbol
	}
	if _, e := f.symbols.Hash(s); e != nil {
		return false, e
	}
	_, found := f.table.Get2(s)
	return found, nil
}

func (f *FamilyManager) GetName(id FamilyID) (util.Symbol, error) {
	if e := f.lock(); e != nil {
		return util.Symbol{}, e
	}
	defer f.mu.Unlock()
	if id < 0 || uint64(id) >= uint64(len(f.names)) {
		return util.Symbol{}, nil
	}
	return f.names[id], nil
}

func (f *FamilyManager) HasID(id FamilyID) (bool, error) {
	if e := f.lock(); e != nil {
		return false, e
	}
	defer f.mu.Unlock()
	return id >= 0 && uint64(id) < uint64(len(f.names)), nil
}

// Enumeration uses stable insertion order. Returned slices append to
// caller-owned output buffers, which are outside this owner's budget.
func (f *FamilyManager) AppendDom(dst []util.Symbol) ([]util.Symbol, error) {
	if e := f.lock(); e != nil {
		return dst, e
	}
	defer f.mu.Unlock()
	for symbol := range f.table.All() {
		dst = append(dst, symbol)
	}
	return dst, nil
}
func (f *FamilyManager) AppendRange(dst []FamilyID) ([]FamilyID, error) {
	if e := f.lock(); e != nil {
		return dst, e
	}
	defer f.mu.Unlock()
	for _, id := range f.table.All() {
		dst = append(dst, id)
	}
	return dst, nil
}

func (f *FamilyManager) Stats() FamilyStats {
	if f == nil {
		return FamilyStats{}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.self != f || f.closed {
		return FamilyStats{}
	}
	return FamilyStats{NextID: f.next, Names: uint32(len(f.names)), NamesCapacity: uint32(cap(f.names)), TableSize: uint32(f.table.Len()), OwnerBytes: uint64(reflect.TypeOf(FamilyManager{}).Size()), NamesBytes: uint64(cap(f.names)) * uint64(reflect.TypeOf(util.Symbol{}).Size())}
}

func (f *FamilyManager) Close() error {
	if f == nil {
		return ErrFamilyOwner
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.self != f {
		return ErrFamilyOwner
	}
	if f.closed {
		return nil
	}
	f.names = nil
	f.table.Clear()
	f.budget.Release(uint64(reflect.TypeOf(FamilyManager{}).Size()))
	f.closed = true
	return nil
}
