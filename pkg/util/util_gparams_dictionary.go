// Licensed under the MIT license.
package util

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// Each source dictionary owns a distinct backing. Source pointer values are
// scalar indices into actual cold root directories, never boxed hot records.
type GparamsDescriptorEntry struct {
	key                     Symbol
	hash, state, descriptor uint32
}
type GparamsRefEntry struct {
	key               Symbol
	hash, state, cell uint32
}
type GparamsDescriptionEntry struct {
	key         Symbol
	hash, state uint32
	text        GlobalParamCString
}

type gparamsDescriptorDictionary struct {
	Entries       []GparamsDescriptorEntry
	size, deleted uint32
}

func (d *gparamsDescriptorDictionary) allocate(b *arena.Budget, n uint32) error {
	stride := uint64(reflect.TypeOf(GparamsDescriptorEntry{}).Size())
	if n == 0 || n&(n-1) != 0 || uint64(n) > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	if e := b.Reserve(uint64(n) * stride); e != nil {
		return e
	}
	next := make([]GparamsDescriptorEntry, n)
	for _, x := range d.Entries {
		if x.state != 2 {
			continue
		}
		i := x.hash & (n - 1)
		for next[i].state != 0 {
			i = (i + 1) & (n - 1)
		}
		next[i] = x
	}
	b.Release(uint64(len(d.Entries)) * stride)
	d.Entries = next
	d.deleted = 0
	return nil
}
func (d *gparamsDescriptorDictionary) find(k Symbol, h uint32) int {
	if len(d.Entries) == 0 {
		return -1
	}
	mask := uint32(len(d.Entries) - 1)
	for n := uint32(0); n <= mask; n++ {
		i := (h + n) & mask
		x := d.Entries[i]
		if x.state == 0 {
			return -1
		}
		if x.state == 2 && x.hash == h && x.key == k {
			return int(i)
		}
	}
	return -1
}
func (d *gparamsDescriptorDictionary) insert(b *arena.Budget, x GparamsDescriptorEntry) error {
	// Source unsigned arithmetic and growth precede hash/probe overwrite.
	if (d.size+d.deleted)<<2 > uint32(len(d.Entries))*3 {
		n := uint32(len(d.Entries)) << 1
		if n <= uint32(len(d.Entries)) {
			return arena.ErrBounds
		}
		if e := d.allocate(b, n); e != nil {
			return e
		}
	}
	if len(d.Entries) == 0 {
		return arena.ErrBounds
	}
	mask := uint32(len(d.Entries) - 1)
	deleted := -1
	for n := uint32(0); n <= mask; n++ {
		i := int((x.hash + n) & mask)
		old := d.Entries[i]
		if old.state == 2 {
			if old.hash == x.hash && old.key == x.key {
				d.Entries[i].descriptor = x.descriptor
				return nil
			}
			continue
		}
		if old.state == 1 {
			deleted = i
			continue
		}
		if deleted >= 0 {
			i = deleted
			d.deleted--
		}
		x.state = 2
		d.Entries[i] = x
		d.size++
		return nil
	}
	return arena.ErrBounds
}
func (d *gparamsDescriptorDictionary) reset(b *arena.Budget) error {
	if d.size == 0 && d.deleted == 0 {
		return nil
	}
	overhead := uint32(0)
	for i := range d.Entries {
		if d.Entries[i].state == 0 {
			overhead++
		} else {
			d.Entries[i].state = 0
		}
	}
	if len(d.Entries) > 16 && overhead<<2 > uint32(len(d.Entries))*3 {
		// Actual source frees old table before the smaller allocation. A failed
		// allocation retains a guarded missing backing, not invented old storage.
		n := uint32(len(d.Entries)) >> 1
		d.close(b)
		if e := d.allocate(b, n); e != nil {
			return e
		}
	}
	d.size = 0
	d.deleted = 0
	return nil
}
func (d *gparamsDescriptorDictionary) close(b *arena.Budget) {
	b.Release(uint64(len(d.Entries)) * uint64(reflect.TypeOf(GparamsDescriptorEntry{}).Size()))
	d.Entries = nil
	d.size = 0
	d.deleted = 0
}

type gparamsRefDictionary struct {
	Entries       []GparamsRefEntry
	size, deleted uint32
}

func (d *gparamsRefDictionary) allocate(b *arena.Budget, n uint32) error {
	stride := uint64(reflect.TypeOf(GparamsRefEntry{}).Size())
	if n == 0 || n&(n-1) != 0 || uint64(n) > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	if e := b.Reserve(uint64(n) * stride); e != nil {
		return e
	}
	next := make([]GparamsRefEntry, n)
	for _, x := range d.Entries {
		if x.state != 2 {
			continue
		}
		i := x.hash & (n - 1)
		for next[i].state != 0 {
			i = (i + 1) & (n - 1)
		}
		next[i] = x
	}
	b.Release(uint64(len(d.Entries)) * stride)
	d.Entries = next
	d.deleted = 0
	return nil
}
func (d *gparamsRefDictionary) find(k Symbol, h uint32) int {
	if len(d.Entries) == 0 {
		return -1
	}
	mask := uint32(len(d.Entries) - 1)
	for n := uint32(0); n <= mask; n++ {
		i := (h + n) & mask
		x := d.Entries[i]
		if x.state == 0 {
			return -1
		}
		if x.state == 2 && x.hash == h && x.key == k {
			return int(i)
		}
	}
	return -1
}
func (d *gparamsRefDictionary) insert(b *arena.Budget, x GparamsRefEntry) error {
	// Source unsigned arithmetic and growth precede hash/probe overwrite.
	if (d.size+d.deleted)<<2 > uint32(len(d.Entries))*3 {
		n := uint32(len(d.Entries)) << 1
		if n <= uint32(len(d.Entries)) {
			return arena.ErrBounds
		}
		if e := d.allocate(b, n); e != nil {
			return e
		}
	}
	if len(d.Entries) == 0 {
		return arena.ErrBounds
	}
	mask := uint32(len(d.Entries) - 1)
	deleted := -1
	for n := uint32(0); n <= mask; n++ {
		i := int((x.hash + n) & mask)
		old := d.Entries[i]
		if old.state == 2 {
			if old.hash == x.hash && old.key == x.key {
				d.Entries[i].cell = x.cell
				return nil
			}
			continue
		}
		if old.state == 1 {
			deleted = i
			continue
		}
		if deleted >= 0 {
			i = deleted
			d.deleted--
		}
		x.state = 2
		d.Entries[i] = x
		d.size++
		return nil
	}
	return arena.ErrBounds
}
func (d *gparamsRefDictionary) reset(b *arena.Budget) error {
	if d.size == 0 && d.deleted == 0 {
		return nil
	}
	overhead := uint32(0)
	for i := range d.Entries {
		if d.Entries[i].state == 0 {
			overhead++
		} else {
			d.Entries[i].state = 0
		}
	}
	if len(d.Entries) > 16 && overhead<<2 > uint32(len(d.Entries))*3 {
		// Actual source frees old table before the smaller allocation. A failed
		// allocation retains a guarded missing backing, not invented old storage.
		n := uint32(len(d.Entries)) >> 1
		d.close(b)
		if e := d.allocate(b, n); e != nil {
			return e
		}
	}
	d.size = 0
	d.deleted = 0
	return nil
}
func (d *gparamsRefDictionary) close(b *arena.Budget) {
	b.Release(uint64(len(d.Entries)) * uint64(reflect.TypeOf(GparamsRefEntry{}).Size()))
	d.Entries = nil
	d.size = 0
	d.deleted = 0
}

type gparamsDescriptionDictionary struct {
	Entries       []GparamsDescriptionEntry
	size, deleted uint32
}

func (d *gparamsDescriptionDictionary) allocate(b *arena.Budget, n uint32) error {
	stride := uint64(reflect.TypeOf(GparamsDescriptionEntry{}).Size())
	if n == 0 || n&(n-1) != 0 || uint64(n) > uint64(^uint(0)>>1)/stride {
		return arena.ErrBounds
	}
	if e := b.Reserve(uint64(n) * stride); e != nil {
		return e
	}
	next := make([]GparamsDescriptionEntry, n)
	for _, x := range d.Entries {
		if x.state != 2 {
			continue
		}
		i := x.hash & (n - 1)
		for next[i].state != 0 {
			i = (i + 1) & (n - 1)
		}
		next[i] = x
	}
	b.Release(uint64(len(d.Entries)) * stride)
	d.Entries = next
	d.deleted = 0
	return nil
}
func (d *gparamsDescriptionDictionary) find(k Symbol, h uint32) int {
	if len(d.Entries) == 0 {
		return -1
	}
	mask := uint32(len(d.Entries) - 1)
	for n := uint32(0); n <= mask; n++ {
		i := (h + n) & mask
		x := d.Entries[i]
		if x.state == 0 {
			return -1
		}
		if x.state == 2 && x.hash == h && x.key == k {
			return int(i)
		}
	}
	return -1
}
func (d *gparamsDescriptionDictionary) insert(b *arena.Budget, x GparamsDescriptionEntry) error {
	// Source unsigned arithmetic and growth precede hash/probe overwrite.
	if (d.size+d.deleted)<<2 > uint32(len(d.Entries))*3 {
		n := uint32(len(d.Entries)) << 1
		if n <= uint32(len(d.Entries)) {
			return arena.ErrBounds
		}
		if e := d.allocate(b, n); e != nil {
			return e
		}
	}
	if len(d.Entries) == 0 {
		return arena.ErrBounds
	}
	mask := uint32(len(d.Entries) - 1)
	deleted := -1
	for n := uint32(0); n <= mask; n++ {
		i := int((x.hash + n) & mask)
		old := d.Entries[i]
		if old.state == 2 {
			if old.hash == x.hash && old.key == x.key {
				d.Entries[i].text = x.text
				return nil
			}
			continue
		}
		if old.state == 1 {
			deleted = i
			continue
		}
		if deleted >= 0 {
			i = deleted
			d.deleted--
		}
		x.state = 2
		d.Entries[i] = x
		d.size++
		return nil
	}
	return arena.ErrBounds
}
func (d *gparamsDescriptionDictionary) reset(b *arena.Budget) error {
	if d.size == 0 && d.deleted == 0 {
		return nil
	}
	overhead := uint32(0)
	for i := range d.Entries {
		if d.Entries[i].state == 0 {
			overhead++
		} else {
			d.Entries[i].state = 0
		}
	}
	if len(d.Entries) > 16 && overhead<<2 > uint32(len(d.Entries))*3 {
		// Actual source frees old table before the smaller allocation. A failed
		// allocation retains a guarded missing backing, not invented old storage.
		n := uint32(len(d.Entries)) >> 1
		d.close(b)
		if e := d.allocate(b, n); e != nil {
			return e
		}
	}
	d.size = 0
	d.deleted = 0
	return nil
}
func (d *gparamsDescriptionDictionary) close(b *arena.Budget) {
	b.Release(uint64(len(d.Entries)) * uint64(reflect.TypeOf(GparamsDescriptionEntry{}).Size()))
	d.Entries = nil
	d.size = 0
	d.deleted = 0
}
