// Source: src/util/small_object_allocator.cpp:172-242; src/util/util.cpp:22-40.
// Source authors: Nikolaj bjorner (nbjorner), 2007-08-06; Leonardo de Moura (leonardo), 2006.
// Licensed under the MIT license.
// Source allocator diagnostics through an explicit borrowed ostream adapter.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
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
	if out.fileBad {
		return nil
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
func (b *smallObjectIOBridge) diagnostic(a *memory.SmallAllocator) error {
	out := b.config.Diagnostics
	if e := out.AppendCString("Memory leak detected for small object allocator '"); e != nil {
		return e
	}
	if b.config.BorrowedID != nil || b.config.IDRef.Present || b.config.IDNull {
		id := b.config.IDRef
		if b.config.IDNull {
			id.Present = false
		}
		if e := smallObjectCStringInto(out, b.config.BorrowedID, id); e != nil {
			return e
		}
	} else if e := out.AppendCString(b.config.ID); e != nil {
		return e
	}
	if e := out.AppendCString("'. "); e != nil {
		return e
	}
	count, e := a.AllocationSize()
	if e != nil {
		return e
	}
	if e := out.AppendUint64(count); e != nil {
		return e
	}
	if e := out.AppendCString(" bytes leaked"); e != nil {
		return e
	}
	return out.AppendByte('\n') // explicit byte sink; native stderr flush remains open
}
func (b *smallObjectIOBridge) verbose(a *memory.SmallAllocator, end bool) error {
	out := b.config.Verbose
	if b.config.VerboseContext != nil {
		var e error
		out, e = b.config.VerboseContext.VerboseStream()
		if e != nil {
			return e
		}
	}
	// Source selects the stream once for this entire insertion chain.
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
	n, e := b.config.Runtime.AllocationSize()
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

func (b *smallObjectIOBridge) Emit(a *memory.SmallAllocator, event memory.AllocatorIOEvent) error {
	switch event {
	case memory.AllocatorIOLeak:
		return b.diagnostic(a)
	case memory.AllocatorIOVerboseStart:
		return b.verbose(a, false)
	case memory.AllocatorIOVerboseEnd:
		return b.verbose(a, true)
	}
	return memory.ErrAllocatorIO
}
