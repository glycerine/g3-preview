// Source: src/util/warning.cpp:63-89,109-169; src/util/warning.h:24-57.
// Source author: Leonardo de Moura (2006-12-01).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

var ErrWarningOwner = errors.New("g3: invalid warning context")
var ErrWarningExecuting = errors.New("g3: warning action is executing")

type WarningContextConfig struct {
	Budget  *arena.Budget
	Runtime *memory.Runtime
	Streams *MemoryStandardStreams
}

// Caller-confined representation of actual warning globals. Streams/Runtime are
// borrowed. The supported printf forms are the literal and single %s calls of
// src/ast/well_sorted.cpp:39,50,70, not arbitrary C variadic arguments.
type WarningContext struct {
	self                                                   *WarningContext
	budget                                                 *arena.Budget
	runtime                                                *memory.Runtime
	streams                                                *MemoryStandardStreams
	warningStream, errorStream                             *NumericTextStream
	frames                                                 *warningFormatFrame
	header, serial                                         uint64
	enabled, useStdout, showErrorPrefix, executing, closed bool
	bodyCause                                              error
}
type WarningStats struct{ OwnerBytes, FrameBytes uint64 }

func NewWarningContext(c WarningContextConfig) (*WarningContext, error) {
	if e := c.Runtime.CheckOpen(); e != nil {
		return nil, e
	}
	if e := c.Streams.open(); e != nil {
		return nil, e
	}
	if c.Runtime.Budget() != c.Budget || c.Streams.budget != c.Budget {
		return nil, arena.ErrConfig
	}
	n := uint64(reflect.TypeOf(WarningContext{}).Size())
	if e := c.Budget.Reserve(n); e != nil {
		return nil, e
	}
	w := &WarningContext{budget: c.Budget, runtime: c.Runtime, streams: c.Streams, header: n, enabled: true, showErrorPrefix: true}
	w.self = w
	return w, nil
}
func (w *WarningContext) open() error {
	if w == nil || w.self != w {
		return ErrWarningOwner
	}
	if w.closed {
		return arena.ErrClosed
	}
	if w.executing {
		return ErrWarningExecuting
	}
	return nil
}
func (w *WarningContext) EnableWarningMessages(v bool) error {
	if e := w.open(); e != nil {
		return e
	}
	w.enabled = v
	return nil
}
func (w *WarningContext) SendWarningsToStdout(v bool) error {
	if e := w.open(); e != nil {
		return e
	}
	w.useStdout = v
	return nil
}
func (w *WarningContext) SetWarningStream(out *NumericTextStream) error {
	if e := w.open(); e != nil {
		return e
	}
	if out != nil && (out.self != out || out.budget != w.budget) {
		return arena.ErrConfig
	}
	w.warningStream = out
	return nil
}
func (w *WarningContext) SetErrorStream(out *NumericTextStream) error {
	if e := w.open(); e != nil {
		return e
	}
	if out != nil && (out.self != out || out.budget != w.budget) {
		return arena.ErrConfig
	}
	w.errorStream = out
	return nil
}
func (w *WarningContext) DisableErrorMessagePrefix() error {
	if e := w.open(); e != nil {
		return e
	}
	w.showErrorPrefix = false
	return nil
}
func (w *WarningContext) Stats() WarningStats {
	if w == nil || w.self != w || w.closed {
		return WarningStats{}
	}
	s := WarningStats{OwnerBytes: w.header}
	for f := w.frames; f != nil; f = f.next {
		s.FrameBytes += f.header
	}
	return s
}
func (w *WarningContext) WarnLiteral(msg string) error { return w.warn(warningText{text: msg}) }
func (w *WarningContext) WarnCString(region *arena.BytesRegion, p GlobalParamCString) error {
	return w.warn(warningText{region: region, pointer: p, borrowed: true})
}
func (w *WarningContext) warn(text warningText) (result error) {
	if e := w.open(); e != nil {
		return e
	}
	if !w.enabled {
		return nil
	}
	w.executing = true
	defer func() {
		w.executing = false
		if result != nil && w.bodyCause == nil {
			w.bodyCause = result
		}
	}()
	out := w.warningStream
	if out == nil {
		return w.printStdio(text)
	}
	if e := out.AppendCString("WARNING: "); e != nil {
		return e
	}
	f, e := w.newFormatFrame()
	if e != nil {
		return e
	}
	result = w.formatInto(f, out, text)
	if result != nil && f.bodyCause == nil {
		f.bodyCause = result
	}
	// format2ostream's automatic buffer destructor precedes print_msg's
	// separate newline insertion and flush, on success and body unwind.
	if e = f.sourceDestroy(); e != nil {
		result = errors.Join(result, e)
	}
	if result == nil {
		result = out.AppendCString("\n")
		if result == nil {
			result = out.Flush()
		}
	}
	if result == nil {
		w.frames = f.next
		w.budget.Release(f.header)
	}
	return result
}
func (w *WarningContext) Close() error {
	if w == nil || w.self != w {
		return ErrWarningOwner
	}
	if w.closed {
		return nil
	}
	if w.executing {
		return ErrWarningExecuting
	}
	for f := w.frames; f != nil; f = f.next {
		if f.self != f || f.owner != w || f.invocation == 0 || f.invocation > w.serial || f.column.runtime != w.runtime {
			return ErrWarningOwner
		}
		if e := f.column.finalize(); e != nil {
			if f.ownCause == nil {
				f.ownCause = e
			}
			return e
		}
	}
	for w.frames != nil {
		f := w.frames
		w.frames = f.next
		w.budget.Release(f.header)
	}
	w.budget.Release(w.header)
	w.closed = true
	return nil
}
