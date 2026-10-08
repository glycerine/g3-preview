// Source: src/util/timer.h:27-36; src/util/timer.cpp:24-39.
package util

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

var ErrTimerOwner = errors.New("g3: invalid timer owner")
var ErrTimerPending = errors.New("g3: timer operation in progress")

// TimerConfig borrows the memory runtime and optional nanosecond clock.
// Nil Clock selects the stopwatch's actual process CPU clock. The selected
// source layout is the Linux 64-bit stopwatch (32 bytes, including padding).
type TimerConfig struct {
	Runtime *Runtime
	Clock   func() (uint64, error)
}

// Timer owns its source stopwatch allocation. The checked Runtime block is
// the actual backing for elapsed/running/start fields; no mirrored heap watch
// is retained. Go owner/control metadata is charged separately from that block.
// If NewTimer returns both an owner and an error, Close releases that prefix.
type Timer struct {
	self           *Timer
	config         TimerConfig
	watch          Block
	header         uint64
	cause          error
	active, closed bool
}

func NewTimer(config TimerConfig) (*Timer, error) {
	if config.Runtime == nil {
		return nil, ErrTimerOwner
	}
	if e := config.Runtime.CheckOpen(); e != nil {
		return nil, e
	}
	if config.Runtime.SizeTBits() != 64 {
		return nil, arena.ErrConfig
	}
	header := uint64(reflect.TypeOf(Timer{}).Size())
	if e := config.Runtime.Budget().Reserve(header); e != nil {
		return nil, e
	}
	p := &Timer{config: config, header: header, active: true}
	p.self = p
	var e error
	p.watch, e = config.Runtime.Allocate(32)
	if e == nil {
		// Source constructor initializes elapsed/running; start is unreached
		// until the clock succeeds. Padding and unreached timespec stay invalid.
		e = p.writeUint64(0, 0)
		if e == nil {
			e = config.Runtime.Write(p.watch, 8, 0)
		}
		if e == nil {
			w := Stopwatch{clock: config.Clock}
			e = w.Start()
			if e == nil {
				e = p.writeWatch(&w)
			}
		}
	}
	p.active = false
	p.cause = e
	return p, e
}

func (p *Timer) open() error {
	if p == nil || p.self != p {
		return ErrTimerOwner
	}
	if p.closed {
		return arena.ErrClosed
	}
	if p.active {
		return ErrTimerPending
	}
	if p.cause != nil {
		return p.cause
	}
	return p.config.Runtime.CheckOpen()
}

func (p *Timer) readUint64(offset uint64) (uint64, error) {
	var value uint64
	for i := uint64(0); i < 8; i++ {
		b, e := p.config.Runtime.Read(p.watch, offset+i)
		if e != nil {
			return 0, e
		}
		value |= uint64(b) << (8 * i)
	}
	return value, nil
}
func (p *Timer) writeUint64(offset, value uint64) error {
	for i := uint64(0); i < 8; i++ {
		if e := p.config.Runtime.Write(p.watch, offset+i, byte(value>>(8*i))); e != nil {
			return e
		}
	}
	return nil
}
func (p *Timer) readWatch() (Stopwatch, error) {
	w := Stopwatch{clock: p.config.Clock}
	var e error
	w.elapsed, e = p.readUint64(0)
	if e != nil {
		return w, e
	}
	running, e := p.config.Runtime.Read(p.watch, 8)
	if e != nil {
		return w, e
	}
	w.running = running != 0
	if w.running {
		sec, e := p.readUint64(16)
		if e != nil {
			return w, e
		}
		nsec, e := p.readUint64(24)
		if e != nil {
			return w, e
		}
		w.start = sec*1e9 + nsec
	}
	return w, nil
}
func (p *Timer) writeWatch(w *Stopwatch) error {
	if e := p.writeUint64(0, w.elapsed); e != nil {
		return e
	}
	if w.running {
		if e := p.writeUint64(16, w.start/1e9); e != nil {
			return e
		}
		if e := p.writeUint64(24, w.start%1e9); e != nil {
			return e
		}
		return p.config.Runtime.Write(p.watch, 8, 1)
	}
	return p.config.Runtime.Write(p.watch, 8, 0)
}

func (p *Timer) Start() error {
	if e := p.open(); e != nil {
		return e
	}
	p.active = true
	defer func() { p.active = false }()
	w, e := p.readWatch()
	if e != nil {
		return e
	}
	e = w.Start()
	if writeErr := p.writeWatch(&w); writeErr != nil {
		return writeErr
	}
	return e
}

func (p *Timer) GetSeconds() (float64, error) {
	if e := p.open(); e != nil {
		return 0, e
	}
	p.active = true
	defer func() { p.active = false }()
	w, e := p.readWatch()
	if e != nil {
		return 0, e
	}
	if w.running {
		e = w.Stop()
		if writeErr := p.writeWatch(&w); writeErr != nil {
			return 0, writeErr
		}
		if e != nil {
			return 0, e
		}
		// Publish the reached stop state before the restart clock read.
		e = w.Start()
		if writeErr := p.writeWatch(&w); writeErr != nil {
			return 0, writeErr
		}
		if e != nil {
			return 0, e
		}
	}
	return float64(w.elapsed) / 1e9, nil
}

func (p *Timer) Timeout(seconds uint32) (bool, error) {
	if seconds == 0 || seconds == ^uint32(0) {
		return false, nil
	}
	elapsed, e := p.GetSeconds()
	return elapsed > float64(seconds), e
}
func (p *Timer) MSTimeout(milliseconds uint32) (bool, error) {
	if milliseconds == 0 || milliseconds == ^uint32(0) {
		return false, nil
	}
	elapsed, e := p.GetSeconds()
	return elapsed*1000 > float64(milliseconds), e
}

func (p *Timer) Close() error {
	if p == nil || p.self != p {
		return ErrTimerOwner
	}
	if p.closed {
		return nil
	}
	if p.active {
		return ErrTimerPending
	}
	p.active = true
	defer func() { p.active = false }()
	if p.watch != (Block{}) {
		if e := p.config.Runtime.Deallocate(&p.watch); e != nil {
			return e
		}
	}
	p.config.Runtime.Budget().Release(p.header)
	p.header = 0
	p.closed = true
	return nil
}
