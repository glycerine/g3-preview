// Source: src/util/symbol.h:97-110; src/smt/params/smt_params.cpp:138.
// Source author: Leonardo de Moura (2006-09-11; 2008-02-20).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import "github.com/glycerine/g3/golang/arena"

func (out *NumericTextStream) appendSMTSymbol(g *GlobalParamsService, s Symbol) error {
	enabled, e := g.namespaceAssertionsEnabled()
	if e != nil {
		return e
	}
	if enabled && s.IsMarked() {
		if e = g.namespaceAssertionFailure("src/util/symbol.h", 98, "!s.is_marked()"); e != nil {
			return e
		}
	}
	if s.tag == 0 {
		if s.value == 0 {
			return out.AppendCString("null")
		}
		// This call acquires scalar pointer identity only. Actual strlen/bytes are
		// reached by the existing C-string engine, including after stream badbit.
		if s.owner != g.symbols.owner {
			return ErrSymbolOwner
		}
		return out.AppendCStringView(ParamsStringView{region: g.symbols.region, regionRef: arena.BytesRegionRef{Slot: arena.Slot(s.value), Generation: s.generation}, offset: g.symbols.pointerWidth, mode: 3, present: true})
	}
	if e = out.AppendCString("k!"); e != nil {
		return e
	}
	return out.AppendUint32(s.value)
}
