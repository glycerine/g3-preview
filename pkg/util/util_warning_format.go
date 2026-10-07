// Source: src/util/warning.cpp:109-159; src/util/vector.h:59-100,422-435,182-184.
// Source author: Leonardo de Moura (2006-12-01).
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

type warningText struct {
	text     string
	region   *arena.BytesRegion
	pointer  GlobalParamCString
	borrowed bool
}

// Actual lexical svector<char> uses the existing byte column engine. Its frame
// is cold Go owner metadata, prepared after the source prefix insertion. No
// shadow byte array or Runtime accounting ticket is used.
type warningFormatFrame struct {
	self                             *warningFormatFrame
	owner                            *WarningContext
	invocation                       uint64
	next                             *warningFormatFrame
	column                           RuntimeVectorColumn
	header                           uint64
	bodyCause, sourceCause, ownCause error
	sourceDone                       bool
}

func (w *WarningContext) newFormatFrame() (*warningFormatFrame, error) {
	if w.serial == math.MaxUint64 {
		return nil, arena.ErrSlotLimit
	}
	n := uint64(reflect.TypeOf(warningFormatFrame{}).Size())
	if e := w.budget.Reserve(n); e != nil {
		return nil, e
	}
	w.serial++
	f := &warningFormatFrame{next: w.frames, header: n, owner: w, invocation: w.serial}
	f.self = f
	f.column.Bind(w.runtime)
	w.frames = f
	return f, nil
}
func (f *warningFormatFrame) sourceDestroy() error {
	if f.sourceCause != nil {
		return f.sourceCause
	}
	if f.sourceDone {
		return nil
	}
	// Source destroys only published m_data, never the unjoined allocation.
	if f.column.Block != (Block{}) {
		if e := f.column.Runtime.Deallocate(&f.column.Block); e != nil {
			f.sourceCause = e
			return e
		}
	}
	f.sourceDone = true
	return nil
}
func (w *WarningContext) formatInto(f *warningFormatFrame, out *NumericTextStream, t warningText) error {
	c := &f.column
	// Literal empty svector.resize(1024): repeated geometric expansions, publish
	// size before ascending value-initialization. Exact native unsigned headers.
	for {
		cap, e := c.Capacity()
		if e != nil {
			return e
		}
		if cap >= 1024 {
			break
		}
		if e = c.Expand(); e != nil {
			return e
		}
	}
	if e := WriteRuntimeWord(c.Runtime, c.Block, 4, 1024); e != nil {
		return e
	}
	for i := uint64(0); i < 1024; i++ {
		if e := c.Runtime.Write(c.Block, 8+i, 0); e != nil {
			return e
		}
	}
	format := t.text
	var arguments [1]PrintfArg
	args := arguments[:0]
	if t.borrowed {
		format = "%s"
		arguments[0] = PrintfCStringView(ParamsStringView{region: t.region, regionRef: t.pointer.Ref, offset: t.pointer.Offset, Mode: 3, PresentField: t.pointer.Present})
		args = arguments[:]
		// Check actual source pointer/provider identity at its reached printf leaf.
		if t.region == nil || t.pointer.Origin != t.region.OwnerID() {
			return ErrSmallObjectOwner
		}
	} else {
		format = format[:cSymbolLength(format)]
	}
	count, failure, e := pfRenderSource(pfSink{runtime: c.Runtime, block: c.Block, offset: 8, extent: 1024}, PrintfNonWindowsAMD64, format, args)
	if failure == PrintfArgumentFailure || failure == PrintfUnsupportedFailure {
		return e
	}
	if count < 0 {
		return out.AppendCString(format)
	}

	return warningRawCStringInto(out, c.Runtime, c.Block)
}
func warningRawCStringInto(out *NumericTextStream, r *Runtime, block Block) error {
	if e := out.lock(); e != nil {
		return e
	}
	defer out.mu.Unlock()
	var n uint64
	for {
		v, e := r.Read(block, 8+n)
		if e != nil {
			return e
		}
		if v == 0 {
			break
		}
		n++
	}
	if out.FileBad || out.fileFail {
		return nil
	}
	pad, e := out.fieldPadding(n)
	if e != nil {
		return e
	}
	if out.state.Alignment != NumericLeft {
		if e = out.padding(pad); e != nil {
			return e
		}
	}
	for i := uint64(0); i < n; i++ {
		v, e := r.Read(block, 8+i)
		if e != nil {
			return e
		}
		if e = out.emitByte(v); e != nil {
			return e
		}
	}
	if out.state.Alignment == NumericLeft {
		return out.padding(pad)
	}
	return nil
}
func (w *WarningContext) printStdio(t warningText) error {
	var out *NumericTextStream
	var e error
	if w.useStdout {
		out, e = w.streams.Stdout()
	} else {
		out, e = w.streams.Stderr()
	}
	if e != nil {
		return e
	}
	if e = out.lock(); e != nil {
		return e
	}
	defer out.mu.Unlock()
	if out.File == nil {
		return arena.ErrConfig
	}
	// Direct unbuffered Go-file adapter for fprintf/fflush. Caller file is borrowed;
	// failures are ignored as C return values are. Ostream flags/state unchanged.
	one := [1]byte{}
	for _, v := range []byte("WARNING: ") {
		one[0] = v
		_, _ = out.File.Write(one[:])
	}
	format := t.text
	var arguments [1]PrintfArg
	args := arguments[:0]
	if t.borrowed {
		format = "%s"
		arguments[0] = PrintfCStringView(ParamsStringView{region: t.region, regionRef: t.pointer.Ref, offset: t.pointer.Offset, Mode: 3, PresentField: t.pointer.Present})
		args = arguments[:]
		if t.region == nil || t.pointer.Origin != t.region.OwnerID() {
			return ErrSmallObjectOwner
		}
	} else {
		format = format[:cSymbolLength(format)]
	}
	_, failure, e := pfRenderSource(pfSink{file: out.File}, PrintfNonWindowsAMD64, format, args)
	if failure == PrintfArgumentFailure || failure == PrintfUnsupportedFailure {
		return e
	}

	one[0] = '\n'
	_, _ = out.File.Write(one[:])
	return nil
}
