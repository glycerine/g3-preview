package util

import usymbol "github.com/glycerine/g3/pkg/util/symbol"

import "github.com/glycerine/g3/pkg/arena"

// DisplayInto emits source param_descrs::display into a borrowed byte buffer.
// Error keeps the prefix emitted before it.
func (d *ParamDescrs) DisplayInto(dst *StringBuffer64, indent uint32, smt2, includeDescr bool) error {
	if dst == nil {
		return ErrStringBufferOwner
	}
	if d == nil {
		return ErrParamDescrsPrecondition
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if e := d.open(); e != nil {
		return e
	}
	var names []usymbol.Symbol
	for key := range d.entries.All() {
		names = append(names, key)
	}
	if e := d.sortSymbols(names); e != nil {
		return e
	}
	for _, key := range names {
		for j := uint32(0); j < indent; j++ {
			if e := dst.AppendByte(' '); e != nil {
				return e
			}
		}
		if smt2 {
			if e := dst.AppendByte(':'); e != nil {
				return e
			}
		}
		if e := d.writeNameInto(dst, key, smt2); e != nil {
			return e
		}
		info, found, e := d.find(key)
		if e != nil {
			return e
		}
		if !found {
			return ErrParamDescrsPrecondition
		}
		if paramDescrsStreamSourceDebug && info.descr.ref == (arena.VectorRef{}) {
			return ErrParamDescrsPrecondition
		}
		if e = dst.AppendCString(" ("); e != nil {
			return e
		}
		if e = dst.AppendCString(info.kind.String()); e != nil {
			return e
		}
		if e = dst.AppendByte(')'); e != nil {
			return e
		}
		if includeDescr {
			if e = dst.AppendByte(' '); e != nil {
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
			if e = dst.AppendCString(" (default: "); e != nil {
				return e
			}
			if e = d.writeTextInto(dst, info.def); e != nil {
				return e
			}
			if e = dst.AppendByte(')'); e != nil {
				return e
			}
		}
		if e = dst.AppendByte('\n'); e != nil {
			return e
		}
	}
	return nil
}
func (d *ParamDescrs) writeNameInto(dst *StringBuffer64, k usymbol.Symbol, smt2 bool) error {
	m := d.manager
	if e := m.Validate(k); e != nil {
		return e
	}
	if k.Value == 0 || k.IsNumerical() {
		return ErrParamDescrsPrecondition
	}
	text, _, e := m.BareString(k)
	if e != nil {
		return e
	}
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if smt2 && ch == '_' {
			ch = '-'
		} else if !smt2 && ch == '-' {
			ch = '_'
		} else if ch >= 'A' && ch <= 'Z' {
			ch = ch - 'A' + 'a'
		}
		if e = dst.AppendByte(ch); e != nil {
			return e
		}
	}
	return nil
}
func (d *ParamDescrs) writeTextInto(dst *StringBuffer64, r paramDescrText) error {
	for i := uint32(0); i < r.length; i++ {
		v, e := d.Texts.Read(r.ref, i/4)
		if e != nil {
			return e
		}
		if e = dst.AppendByte(byte(v >> (8 * (i % 4)))); e != nil {
			return e
		}
	}
	return nil
}
