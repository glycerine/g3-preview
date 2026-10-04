// Licensed under the MIT license.
// Concrete dictionary<int>: dictionary.h -> map.h -> default_hash_entry/core_hashtable.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

// The Symbol and value remain present in FREE and DELETED entries. Unlike
// symbol_table<int>, this source specialization has a separately cached hash.
type astUserDictionaryEntry struct {
	hash, state uint32
	key         Symbol
	value       int32
}
type astUserDictionary struct {
	symbols       *SymbolManager
	budget        *arena.Budget
	outOfMemory   func() bool
	entries       []astUserDictionaryEntry
	size, deleted uint32
	bytes         uint64
}

// Embedded owner metadata is charged by the enclosing ASTManager header.
func (d *astUserDictionary) init(s *SymbolManager, b *arena.Budget, oom func() bool) error {
	d.symbols = s
	d.budget = b
	d.outOfMemory = oom
	return d.replace(8, false)
}
func (d *astUserDictionary) replace(n uint32, move bool) error {
	width := uint64(reflect.TypeOf(astUserDictionaryEntry{}).Size())
	if n == 0 || n&(n-1) != 0 || uint64(n) > uint64(^uint(0)>>1)/width {
		return arena.ErrBounds
	}
	bytes := uint64(n) * width
	if e := d.budget.Reserve(bytes); e != nil {
		return e
	}
	next := make([]astUserDictionaryEntry, int(n))
	if move {
		for i := range d.entries {
			old := d.entries[i]
			if old.state != 2 {
				continue
			}
			j := old.hash & (n - 1)
			for next[j].state != 0 {
				j = (j + 1) & (n - 1)
			}
			next[j] = old
		}
	}
	d.budget.Release(d.bytes)
	d.entries = next
	d.bytes = bytes
	d.deleted = 0
	return nil
}
func (d *astUserDictionary) findSlot(k Symbol) (uint32, bool, error) {
	h, e := d.symbols.Hash(k)
	if e != nil {
		return 0, false, e
	}
	n := uint32(len(d.entries))
	if n == 0 {
		return 0, false, arena.ErrClosed
	}
	j := h & (n - 1)
	for walked := uint32(0); walked < n; walked++ {
		v := &d.entries[j]
		if v.state == 0 {
			return 0, false, nil
		}
		if v.state == 2 && v.hash == h && v.key == k {
			return j, true, nil
		}
		j = (j + 1) & (n - 1)
	}
	return 0, false, nil
}
func (d *astUserDictionary) find(k Symbol) (int32, bool, error) {
	j, ok, e := d.findSlot(k)
	if e != nil || !ok {
		return 0, ok, e
	}
	return d.entries[j].value, true, nil
}
func (d *astUserDictionary) insert(k Symbol, value int32) error {
	n := uint32(len(d.entries))
	if n == 0 {
		return arena.ErrClosed
	}
	// This unsigned expression and its order are source-visible, including on
	// duplicate keys. Checked physical growth adapts source allocation failure.
	if (d.size+d.deleted)<<2 > n*3 {
		if n > math.MaxUint32/2 {
			return arena.ErrBounds
		}
		if e := d.replace(n<<1, true); e != nil {
			return e
		}
		n = uint32(len(d.entries))
	}
	h, e := d.symbols.Hash(k)
	if e != nil {
		return e
	}
	j := h & (n - 1)
	lastDeleted := uint32(math.MaxUint32)
	for walked := uint32(0); walked < n; walked++ {
		v := &d.entries[j]
		if v.state == 2 {
			if v.hash == h && v.key == k {
				v.key = k
				v.value = value
				return nil
			}
		} else if v.state == 1 {
			lastDeleted = j
		} else {
			if lastDeleted != math.MaxUint32 {
				j = lastDeleted
				d.deleted--
			}
			d.entries[j] = astUserDictionaryEntry{hash: h, state: 2, key: k, value: value}
			d.size++
			return nil
		}
		j = (j + 1) & (n - 1)
	}
	return arena.ErrBounds // source UNREACHABLE: no free slot after skipped/overflow growth
}
func (d *astUserDictionary) remove(k Symbol) error {
	j, ok, e := d.findSlot(k)
	if e != nil || !ok {
		return e
	}
	next := (j + 1) & (uint32(len(d.entries)) - 1)
	d.size--
	if d.entries[next].state == 0 {
		d.entries[j].state = 0
		return nil
	}
	d.entries[j].state = 1
	d.deleted++
	if d.deleted > d.size && d.deleted > 64 {
		if d.outOfMemory != nil && d.outOfMemory() {
			return nil
		}
		return d.replace(uint32(len(d.entries)), true)
	}
	return nil
}
func (d *astUserDictionary) reset() error {
	if d.size == 0 && d.deleted == 0 {
		return nil
	}
	var overhead uint32
	for i := range d.entries {
		if d.entries[i].state == 0 {
			overhead++
		} else {
			d.entries[i].state = 0
		}
	}
	n := uint32(len(d.entries))
	if n > 16 && overhead<<2 > n*3 {
		if e := d.replace(n>>1, false); e != nil {
			// Source has already deleted the old allocation before throwing
			// here. Keep its completed FREE marking in live borrowed Go
			// storage, with coherent empty counters, instead of reproducing
			// that dangling-table continuation.
			d.size = 0
			d.deleted = 0
			return e
		}
	}
	d.size = 0
	d.deleted = 0
	return nil
}
func (d *astUserDictionary) finalize() error {
	if len(d.entries) > 64 {
		if e := d.replace(64, false); e != nil {
			return e
		}
		d.size = 0
		return nil
	}
	return d.reset()
}
func (d *astUserDictionary) close() {
	d.budget.Release(d.bytes)
	d.entries = nil
	d.bytes = 0
	d.size = 0
	d.deleted = 0
}
