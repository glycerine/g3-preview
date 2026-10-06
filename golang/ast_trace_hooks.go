// Licensed under the MIT license; see LICENSE.txt.
// Source: src/ast/ast.cpp:2114-2128,2259-2261,2368-2373;
// src/util/symbol.h:97-110.
// Source author: Leonardo de Moura.
package g3

import "github.com/glycerine/g3/golang/arena"

func (m *ASTManager) traceApp(r astNodeRef) error {
	if m.traceStream == nil {
		return nil
	}
	out := m.traceStream
	if e := out.AppendCString("[mk-app] #"); e != nil {
		return e
	}
	if e := out.AppendUint32(m.nodeAt(r.slot).id); e != nil {
		return e
	}
	if e := out.AppendCString(" "); e != nil {
		return e
	}
	a := m.appAt(r.slot)
	d := m.declAt(a.decl.slot)
	special := false
	if a.count == 0 {
		// Actual strcmp against the literal name reads borrowed spelling here,
		// before the source else-if label predicate and low-level printer.
		if d.name.tag == 0 && d.name.value != 0 {
			view := ParamsStringView{region: m.symbols.region, regionRef: arena.BytesRegionRef{Slot: arena.Slot(d.name.value), Generation: d.name.generation}, offset: m.symbols.pointerWidth, mode: 3, present: true}
			special = true
			for i, want := range []byte{'i', 'n', 't', 0} {
				ch, e := view.Byte(uint64(i))
				if e != nil {
					return e
				}
				if ch != want {
					special = false
					break
				}
			}
		}
	}
	if !special && d.info != (ASTFuncDeclInfo{}) {
		info, e := m.funcInfo(d.info)
		if e != nil {
			return e
		}
		special = info.family == 1 && info.kind == 1
	}
	if special {
		return m.lowLevelPrint(out, r, true, true)
	}
	if e := out.appendASTSymbol(m, d.name); e != nil {
		return e
	}
	for i := uint32(0); i < a.count; i++ {
		if e := out.AppendCString(" #"); e != nil {
			return e
		}
		child, e := m.readInternal(a.args, i)
		if e != nil {
			return e
		}
		if e = m.validNode(child); e != nil {
			return e
		}
		if e = out.AppendUint32(m.nodeAt(child.slot).id); e != nil {
			return e
		}
	}
	return out.AppendCString("\n")
}
func (m *ASTManager) traceVar(r astNodeRef) error {
	if m.traceStream == nil {
		return nil
	}
	if e := m.traceStream.AppendCString("[mk-var] #"); e != nil {
		return e
	}
	if e := m.traceStream.AppendUint32(m.nodeAt(r.slot).id); e != nil {
		return e
	}
	return m.traceStream.AppendCString("\n")
}
func (m *ASTManager) traceQuantifier(r astNodeRef) error {
	if m.traceStream == nil {
		return nil
	}
	out := m.traceStream
	if e := out.AppendCString("[mk-quant] #"); e != nil {
		return e
	}
	if e := out.AppendUint32(m.nodeAt(r.slot).id); e != nil {
		return e
	}
	if e := out.AppendCString(" "); e != nil {
		return e
	}
	q := m.quantifierAt(r.slot)
	if e := out.appendASTSymbol(m, q.qid); e != nil {
		return e
	}
	for i := uint32(0); i < q.patternCount; i++ {
		if e := out.AppendCString(" #"); e != nil {
			return e
		}
		child, e := m.readInternal(q.patterns, i)
		if e != nil {
			return e
		}
		if e = m.validNode(child); e != nil {
			return e
		}
		if e = out.AppendUint32(m.nodeAt(child.slot).id); e != nil {
			return e
		}
	}
	if e := out.AppendCString(" #"); e != nil {
		return e
	}
	if e := m.validNode(q.body); e != nil {
		return e
	}
	if e := out.AppendUint32(m.nodeAt(q.body.slot).id); e != nil {
		return e
	}
	return out.AppendCString("\n")
}
