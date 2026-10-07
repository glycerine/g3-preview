// Source: src/util/params.cpp:639-648,902-930; src/util/symbol.h:96;
// scripts/mk_genfile_common.py:895-939.
// Source author: Leonardo de Moura (2011-05-09).
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"math"

	"github.com/glycerine/g3/pkg/arena"
)

// Cold borrowed source pointer. Captured storage/provider/ref comes from the
// actual returned entry; no context workspace, copied string or RC is acquired.
type ParamsStringView struct {
	words        *arena.Words
	ref          arena.VectorRef
	size         uint32
	region       *arena.BytesRegion
	regionRef    arena.BytesRegionRef
	offset       uint32
	literal      string
	Mode         uint8
	PresentField bool
}

func LiteralParamsStringView(s string) ParamsStringView {
	return ParamsStringView{literal: s, Mode: 1, PresentField: true}
}
func (v ParamsStringView) Present() bool { return v.PresentField }
func (v ParamsStringView) Byte(i uint64) (byte, error) {
	if !v.PresentField {
		return 0, ErrParamsStreamFailure
	}
	switch v.Mode {
	case 1:
		if i >= uint64(len(v.literal)) {
			if i == uint64(len(v.literal)) {
				return 0, nil
			}
			return 0, arena.ErrBounds
		}
		return v.literal[i], nil
	case 2:
		if v.words == nil {
			return 0, ErrParamsOwner
		}
		if i > uint64(v.size) {
			return 0, arena.ErrBounds
		}
		// Existing params text codec omits a separate NUL word for multiples of
		// four. Its virtual terminator retains the actual descriptor lifetime check.
		if i == uint64(v.size) {
			_, e := v.words.Len(v.ref)
			return 0, e
		}
		n, e := v.words.Read(v.ref, uint32(i/4))
		return byte(n >> (8 * (i % 4))), e
	case 3:
		if v.region == nil {
			return 0, ErrParamsOwner
		}
		if i > math.MaxUint32-uint64(v.offset) {
			return 0, arena.ErrBounds
		}
		return v.region.Read(v.regionRef, v.offset+uint32(i))
	}
	return 0, ErrParamsOwner
}
func (v ParamsStringView) Length() (uint64, error) {
	for i := uint64(0); ; i++ {
		b, e := v.Byte(i)
		if e != nil {
			return 0, e
		}
		if b == 0 {
			return i, nil
		}
		if i == math.MaxUint64 {
			return 0, arena.ErrBounds
		}
	}
}
func (c *ParamsContext) GetStringViewFallbackName(v *ParamsRef, k *string, f *ParamsRef, d *string) (ParamsStringView, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	x, ok, e := c.get(v, Symbol{}, k, true, ParamString, f)
	if e != nil {
		return ParamsStringView{}, e
	}
	if !ok {
		if d == nil {
			return ParamsStringView{}, nil
		}
		return LiteralParamsStringView(*d), nil
	}
	if !x.text.present {
		return ParamsStringView{}, nil
	}
	if x.text.borrowed {
		return ParamsStringView{region: c.SymbolsField.region, regionRef: arena.BytesRegionRef{Slot: arena.Slot(x.symbol.Value), Generation: x.symbol.Generation}, offset: c.SymbolsField.pointerWidth, Mode: 3, PresentField: true}, nil
	}
	return ParamsStringView{words: c.Texts, ref: x.text.ref, size: x.text.size, Mode: 2, PresentField: true}, nil
}
