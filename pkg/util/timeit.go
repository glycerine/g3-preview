// Source: src/util/timeit.h:25-31; src/util/timeit.cpp:25-58.
package util

import (
	"errors"
	"math"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/stopwatch"
)

var ErrTimeitOwner = errors.New("g3: invalid timeit owner")

// TimeitConfig borrows all its providers. Output defaults to Streams.Stderr.
// Message is read at destruction, never copied into owned string storage.
type TimeitConfig struct {
	Runtime *Runtime
	Output  *NumericTextStream
	Streams *MemoryStandardStreams
	Message ParamsStringView
	Clock   func() (uint64, error)
}

// TimeitImp represents the source imp. Its Linux 64-bit Runtime allocation
// holds stopwatch fields at offsets 0..31 and start_memory at offset 48.
// Checked Go references represent the borrowed native pointer fields.
type TimeitImp struct {
	block   Block
	message ParamsStringView
	out     *NumericTextStream
}

// Timeit is constructed in caller-provided storage, like the source scope
// object. Do not copy it after Construct. Close executes the original report
// and deallocation; Cleanup releases a failed prefix without replaying output.
type Timeit struct {
	self                    *Timeit
	imp                     TimeitImp
	config                  TimeitConfig
	cause                   error
	active, closed, enabled bool
}

func (p *Timeit) Construct(enable bool, config TimeitConfig) (cause error) {
	if p == nil || p.self != nil {
		return ErrTimeitOwner
	}
	p.self = p
	p.enabled = enable
	if !enable {
		return nil
	}
	p.config = config
	p.active = true
	defer func() { p.active = false; p.cause = cause }()
	if config.Runtime == nil {
		return ErrTimeitOwner
	}
	if e := config.Runtime.CheckOpen(); e != nil {
		return e
	}
	if config.Runtime.SizeTBits() != 64 {
		return arena.ErrConfig
	}
	out := config.Output
	if out == nil {
		if config.Streams == nil {
			return ErrTimeitOwner
		}
		var e error
		out, e = config.Streams.Stderr()
		if e != nil {
			return e
		}
	}
	var e error
	p.imp.block, e = config.Runtime.Allocate(56)
	if e != nil {
		return e
	}
	p.imp.message = config.Message
	p.imp.out = out
	watch := p.watchAdapter()
	if e = watch.writeUint64(0, 0); e != nil {
		return e
	}
	if e = config.Runtime.Write(p.imp.block, 8, 0); e != nil {
		return e
	}
	bytes, e := config.Runtime.AllocationSize()
	if e != nil {
		return e
	}
	if e = watch.writeUint64(48, math.Float64bits(float64(bytes)/(1024*1024))); e != nil {
		return e
	}
	w := stopwatch.New(config.Clock)
	if e = w.Start(); e != nil {
		return e
	}
	return watch.writeWatch(&w)
}

func (p *Timeit) watchAdapter() Timer {
	return Timer{config: TimerConfig{Runtime: p.config.Runtime, Clock: p.config.Clock}, watch: p.imp.block}
}
func (p *Timeit) startMemory() (float64, error) {
	w := p.watchAdapter()
	bits, e := w.readUint64(48)
	return math.Float64frombits(bits), e
}

func (p *Timeit) Close() (cause error) {
	if p == nil || p.self != p {
		return ErrTimeitOwner
	}
	if p.closed {
		return nil
	}
	if p.active {
		return ErrTimerPending
	}
	if p.cause != nil {
		return p.cause
	}
	if !p.enabled {
		p.closed = true
		return nil
	}
	p.active = true
	defer func() {
		p.active = false
		if cause != nil {
			p.cause = cause
		}
	}()
	watch := p.watchAdapter()
	w, e := watch.readWatch()
	if e != nil {
		return e
	}
	e = w.Stop()
	if writeErr := watch.writeWatch(&w); writeErr != nil {
		return writeErr
	}
	if e != nil {
		return e
	}
	bytes, e := p.config.Runtime.AllocationSize()
	if e != nil {
		return e
	}
	endMemory := float64(bytes) / (1024 * 1024)
	out := p.imp.out
	if e = out.AppendCString("("); e != nil {
		return e
	}
	if e = out.AppendCStringView(p.imp.message); e != nil {
		return e
	}
	if e = out.AppendCString(" :time "); e != nil {
		return e
	}
	if e = out.SetFloatMode(NumericFixedFloat); e != nil {
		return e
	}
	if e = out.SetPrecision(2); e != nil {
		return e
	}
	seconds, e := w.Seconds()
	if e != nil {
		return e
	}
	if e = out.AppendFloat64(seconds); e != nil {
		return e
	}
	if e = out.AppendCString(" :before-memory "); e != nil {
		return e
	}
	if e = out.SetFloatMode(NumericFixedFloat); e != nil {
		return e
	}
	if e = out.SetPrecision(2); e != nil {
		return e
	}
	startMemory, e := p.startMemory()
	if e != nil {
		return e
	}
	if e = out.AppendFloat64(startMemory); e != nil {
		return e
	}
	if e = out.AppendCString(" :after-memory "); e != nil {
		return e
	}
	if e = out.SetFloatMode(NumericFixedFloat); e != nil {
		return e
	}
	if e = out.SetPrecision(2); e != nil {
		return e
	}
	if e = out.AppendFloat64(endMemory); e != nil {
		return e
	}
	if e = out.AppendCString(")"); e != nil {
		return e
	}
	if e = out.AppendByte('\n'); e != nil {
		return e
	}
	if e = out.Flush(); e != nil {
		return e
	}
	if e = p.config.Runtime.Deallocate(&p.imp.block); e != nil {
		return e
	}
	p.closed = true
	return nil
}

// Cleanup is owning cleanup, not completion of a stopped source destructor.
func (p *Timeit) Cleanup() error {
	if p == nil || p.self != p {
		return ErrTimeitOwner
	}
	if p.closed {
		return nil
	}
	if p.active {
		return ErrTimerPending
	}
	p.active = true
	defer func() { p.active = false }()
	if p.imp.block != (Block{}) {
		if e := p.config.Runtime.Deallocate(&p.imp.block); e != nil {
			return e
		}
	}
	p.closed = true
	return nil
}
