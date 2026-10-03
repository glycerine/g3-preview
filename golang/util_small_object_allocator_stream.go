// Copyright (c) 2006, 2007 Microsoft Corporation. Licensed under the MIT license.
// Source allocator diagnostics through an explicit borrowed ostream adapter.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
)

// Read the actual borrowed C string at insertion time. Source strlen precedes
// field padding; the complete string consumes width once. No copied spelling,
// intermediate Go string, per-byte public stream calls or nested mutex locks.
func smallObjectCStringInto(out *NumericTextStream, region *arena.BytesRegion, text GlobalParamCString) error {
	if e := out.lock(); e != nil {
		return e
	}
	defer out.mu.Unlock()
	if !text.Present {
		return ErrParamsStreamFailure
	} // null ostream bad-state boundary
	if region == nil || text.Origin != region.OwnerID() {
		return ErrSmallObjectOwner
	}
	var length uint64
	for {
		offset := uint64(text.Offset) + length
		if offset > math.MaxUint32 {
			return arena.ErrBounds
		}
		c, e := region.Read(text.Ref, uint32(offset))
		if e != nil {
			return e
		}
		if c == 0 {
			break
		}
		length++
	}
	pad, e := out.fieldPadding(length)
	if e != nil {
		return e
	}
	if out.state.Alignment != NumericLeft {
		if e = out.padding(pad); e != nil {
			return e
		}
	}
	for i := uint64(0); i < length; i++ {
		c, e := region.Read(text.Ref, uint32(uint64(text.Offset)+i))
		if e != nil {
			return e
		}
		if e = out.emitByte(c); e != nil {
			return e
		}
	}
	if out.state.Alignment == NumericLeft {
		return out.padding(pad)
	}
	return nil
}
func (a *utilSmallObjectAllocator) diagnostic() error {
	if !a.config.Profile.Debug || a.allocSize == 0 {
		return nil
	}
	out := a.config.Diagnostics
	if e := out.AppendCString("Memory leak detected for small object allocator '"); e != nil {
		return e
	}
	if a.config.BorrowedID != nil || a.config.IDRef.Present || a.config.IDNull {
		id := a.config.IDRef
		if a.config.IDNull {
			id.Present = false
		}
		if e := smallObjectCStringInto(out, a.config.BorrowedID, id); e != nil {
			return e
		}
	} else if e := out.AppendCString(a.config.ID); e != nil {
		return e
	}
	if e := out.AppendCString("'. "); e != nil {
		return e
	}
	if e := out.AppendUint64(a.allocSize); e != nil {
		return e
	}
	if e := out.AppendCString(" bytes leaked"); e != nil {
		return e
	}
	return out.AppendByte('\n') // explicit byte sink; native stderr flush remains open
}
func (a *utilSmallObjectAllocator) verbose(end bool) error {
	if a.config.Verbosity < 20 {
		return nil
	}
	out := a.config.Verbose
	prefix := "(allocator-consolidate :wasted-size "
	if end {
		prefix = "(end-allocator-consolidate :wasted-size "
	}
	if e := out.AppendCString(prefix); e != nil {
		return e
	}
	wasted, e := a.WastedSize()
	if e != nil {
		return e
	}
	if e = out.AppendUint64(wasted); e != nil {
		return e
	}
	if e = out.AppendCString(" :memory "); e != nil {
		return e
	}
	if e = out.SetFloatMode(NumericFixedFloat); e != nil {
		return e
	}
	if e = out.SetPrecision(2); e != nil {
		return e
	}
	n, e := a.config.Runtime.AllocationSize()
	if e != nil {
		return e
	}
	if e = out.AppendFloat64(float64(n) / float64(1024*1024)); e != nil {
		return e
	}
	if e = out.AppendCString(")"); e != nil {
		return e
	}
	return out.AppendByte('\n')
}
