// Source: src/util/symbol.h:97-110; src/smt/params/smt_params.cpp:138.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

func (out *NumericTextStream) AppendSMTSymbol(g *GlobalParamsService, s Symbol) error {
	enabled, e := g.NamespaceAssertionsEnabled()
	if e != nil {
		return e
	}
	if enabled && s.IsMarked() {
		if e = g.NamespaceAssertionFailure("src/util/symbol.h", 98, "!s.is_marked()"); e != nil {
			return e
		}
	}
	if s.Tag == 0 {
		if s.Value == 0 {
			return out.AppendCString("null")
		}
		// This call acquires scalar pointer identity only. Actual strlen/bytes are
		// reached by the existing C-string engine, including after stream badbit.
		if s.owner != g.Symbols.owner {
			return ErrSymbolOwner
		}
		return out.AppendCStringView(ParamsStringView{region: g.Symbols.region, regionRef: arena.BytesRegionRef{Slot: arena.Slot(s.Value), Generation: s.Generation}, offset: g.Symbols.pointerWidth, Mode: 3, PresentField: true})
	}
	if e = out.AppendCString("k!"); e != nil {
		return e
	}
	return out.AppendUint32(s.Value)
}
