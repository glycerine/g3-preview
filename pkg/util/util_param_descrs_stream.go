// Licensed under the MIT license.
package util

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// DisplayInto emits source param_descrs::display into a borrowed charged byte
// destination. Lock order is destination→descriptors→symbols; private writes
// never recursively enter destination public methods. Error keeps emitted bytes.
func (d *ParamDescrs) DisplayInto(dst *StringBuffer64, indent uint32, smt2, includeDescr bool) error {
	if dst == nil {
		return ErrStringBufferOwner
	}
	dst.State.Mu.Lock()
	defer dst.State.Mu.Unlock()
	if e := dst.State.Open(dst.Self == dst); e != nil {
		return e
	}
	if d == nil {
		return ErrParamDescrsPrecondition
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return e
	}
	var names []Symbol
	defer func() { d.Budget.Release(ParamDescriptorNameBytes(cap(names))) }()
	for _, x := range d.Entries {
		if x.state != 2 {
			continue
		}
		if len(names) == cap(names) {
			capacity, e := pdStreamNameCapacity(uint32(cap(names)), d.manager.pointerWidth)
			n := int(capacity)
			if e != nil {
				return e
			}
			if uint64(n) > uint64(^uint(0)>>1)/uint64(reflect.TypeOf(Symbol{}).Size()) {
				return arena.ErrBounds
			}
			if e = d.Budget.Reserve(ParamDescriptorNameBytes(n)); e != nil {
				return e
			}
			next := make([]Symbol, len(names), n)
			copy(next, names)
			d.Budget.Release(ParamDescriptorNameBytes(cap(names)))
			names = next
		}
		names = append(names, x.key)
	}
	if e := d.sortSymbols(names); e != nil {
		return e
	}
	for _, key := range names {
		for j := uint32(0); j < indent; j++ {
			if e := dst.State.AppendByte(dst.Inline[:], ' '); e != nil {
				return e
			}
		}
		if smt2 {
			if e := dst.State.AppendByte(dst.Inline[:], ':'); e != nil {
				return e
			}
		}
		if e := d.writeNameInto(dst, key, smt2); e != nil {
			return e
		}
		idx, e := d.find(key)
		if e != nil {
			return e
		}
		if idx < 0 {
			return ErrParamDescrsPrecondition
		}
		info := d.Entries[idx].info
		if paramDescrsStreamSourceDebug && info.descr.ref == (arena.VectorRef{}) {
			return ErrParamDescrsPrecondition
		}
		if e = dst.State.AppendCString(dst.Inline[:], " ("); e != nil {
			return e
		}
		if e = dst.State.AppendCString(dst.Inline[:], info.kind.String()); e != nil {
			return e
		}
		if e = dst.State.AppendByte(dst.Inline[:], ')'); e != nil {
			return e
		}
		if includeDescr {
			if e = dst.State.AppendByte(dst.Inline[:], ' '); e != nil {
				return e
			}
			// Null CString sets source ostream failbit at this actual output action.
			// This byte-destination adapter returns a typed failure with that prefix.
			if info.descr.ref == (arena.VectorRef{}) {
				return ErrParamsStreamFailure
			}
			if e = d.writeTextInto(dst, info.descr); e != nil {
				return e
			}
		}
		if info.def.ref != (arena.VectorRef{}) {
			if e = dst.State.AppendCString(dst.Inline[:], " (default: "); e != nil {
				return e
			}
			if e = d.writeTextInto(dst, info.def); e != nil {
				return e
			}
			if e = dst.State.AppendByte(dst.Inline[:], ')'); e != nil {
				return e
			}
		}
		if e = dst.State.AppendByte(dst.Inline[:], '\n'); e != nil {
			return e
		}
	}
	return nil
}
func (d *ParamDescrs) writeNameInto(dst *StringBuffer64, k Symbol, smt2 bool) error {
	m := d.manager
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := m.checkPlain(k); e != nil {
		return e
	}
	if k.Value == 0 || k.IsNumerical() {
		return ErrParamDescrsPrecondition
	}
	r := arena.BytesRegionRef{Slot: arena.Slot(k.Value), Generation: k.Generation}
	for i := m.pointerWidth; ; i++ {
		ch, e := m.region.Read(r, i)
		if e != nil {
			return e
		}
		if ch == 0 {
			return nil
		}
		if smt2 && ch == '_' {
			ch = '-'
		} else if !smt2 && ch == '-' {
			ch = '_'
		} else if ch >= 'A' && ch <= 'Z' {
			ch = ch - 'A' + 'a'
		}
		if e = dst.State.AppendByte(dst.Inline[:], ch); e != nil {
			return e
		}
	}
}
func (d *ParamDescrs) writeTextInto(dst *StringBuffer64, r paramDescrText) error {
	for i := uint32(0); i < r.length; i++ {
		v, e := d.Texts.Read(r.ref, i/4)
		if e != nil {
			return e
		}
		if e = dst.State.AppendByte(dst.Inline[:], byte(v>>(8*(i%4)))); e != nil {
			return e
		}
	}
	return nil
}

// Source vector<symbol> SZ is uint32. Source pointer width and translated
// Go Symbol physical stride are distinct; actual host-byte bounds are checked
// separately before Reserve/make in DisplayInto.
func pdStreamNameCapacity(old, width uint32) (uint32, error) {
	if width != 4 && width != 8 {
		return 0, arena.ErrBounds
	}
	if old == 0 {
		return 2, nil
	}
	n := (3*old + 1) >> 1
	oldBytes := old*width + 8
	newBytes := n*width + 8
	if n <= old || newBytes <= oldBytes {
		return 0, arena.ErrBounds
	}
	return n, nil
}
