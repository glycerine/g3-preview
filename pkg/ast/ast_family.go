// Translation of ast.h/ast.cpp::family_manager, Leonardo de Moura.
package ast

import (
	"errors"
	"math"
	"reflect"
	"sync"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
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
	table   *util.SymbolInt32Table
	names   []util.Symbol
	next    FamilyID
	closed  bool
}

type FamilyStats struct {
	NextID                 FamilyID
	Names, NamesCapacity   uint32
	OwnerBytes, NamesBytes uint64
	Table                  util.SymbolInt32TableStats
}

func NewFamilyManager(symbols *util.SymbolManager, budget *arena.Budget) (*FamilyManager, error) {
	if symbols == nil {
		return nil, ErrFamilyOwner
	}
	owner := uint64(reflect.TypeOf(FamilyManager{}).Size())
	if e := budget.Reserve(owner); e != nil {
		return nil, e
	}
	table, e := util.NewSymbolInt32Table(symbols, util.SymbolInt32TableConfig{Budget: budget})
	if e != nil {
		budget.Release(owner)
		return nil, e
	}
	f := &FamilyManager{symbols: symbols, budget: budget, table: table}
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

func (f *FamilyManager) pushName(s util.Symbol) error {
	if len(f.names) == cap(f.names) {
		n, e := util.NextSymbolVectorCapacity(cap(f.names))
		if e != nil {
			return e
		}
		width := uint64(reflect.TypeOf(util.Symbol{}).Size())
		if uint64(n) > uint64(^uint(0)>>1)/width {
			return arena.ErrBounds
		}
		if e = f.budget.Reserve(uint64(n) * width); e != nil {
			return e
		}
		names := make([]util.Symbol, len(f.names), n)
		copy(names, f.names)
		old := uint64(cap(f.names)) * width
		f.names = names
		f.budget.Release(old)
	}
	f.names = append(f.names, s)
	return nil
}

// Preserve source statement publication: increment next before table insertion,
// then append the name. A failure may leave a missing ID or a map-only binding;
// a later lookup does not repair the registration-order vector.
func (f *FamilyManager) MkID(s util.Symbol) (FamilyID, error) {
	if e := f.lock(); e != nil {
		return NullFamilyID, e
	}
	defer f.mu.Unlock()
	id, found, e := f.table.Find(s)
	if e != nil {
		return NullFamilyID, e
	}
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
	if e = f.table.Insert(s, id); e != nil {
		return NullFamilyID, e
	}
	if e = f.pushName(s); e != nil {
		return NullFamilyID, e
	}
	return id, nil
}

func (f *FamilyManager) GetID(s util.Symbol) (FamilyID, error) {
	if e := f.lock(); e != nil {
		return NullFamilyID, e
	}
	defer f.mu.Unlock()
	id, found, e := f.table.Find(s)
	if e != nil {
		return NullFamilyID, e
	}
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
	return f.table.Contains(s)
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

// Enumeration is source table order, not registration order. Returned slices
// append to caller-owned output buffers, which are outside this owner's budget.
func (f *FamilyManager) AppendDom(dst []util.Symbol) ([]util.Symbol, error) {
	if e := f.lock(); e != nil {
		return dst, e
	}
	defer f.mu.Unlock()
	return f.table.AppendDom(dst)
}
func (f *FamilyManager) AppendRange(dst []FamilyID) ([]FamilyID, error) {
	if e := f.lock(); e != nil {
		return dst, e
	}
	defer f.mu.Unlock()
	return f.table.AppendRange(dst)
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
	return FamilyStats{NextID: f.next, Names: uint32(len(f.names)), NamesCapacity: uint32(cap(f.names)), OwnerBytes: uint64(reflect.TypeOf(FamilyManager{}).Size()), NamesBytes: uint64(cap(f.names)) * uint64(reflect.TypeOf(util.Symbol{}).Size()), Table: f.table.Stats()}
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
	// C++ member destruction is reverse declaration order: names then table.
	f.budget.Release(uint64(cap(f.names)) * uint64(reflect.TypeOf(util.Symbol{}).Size()))
	f.names = nil
	if e := f.table.Close(); e != nil {
		return e
	}
	f.budget.Release(uint64(reflect.TypeOf(FamilyManager{}).Size()))
	f.closed = true
	return nil
}
