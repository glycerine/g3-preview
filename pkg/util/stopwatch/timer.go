// Source: src/util/timer.h:27-36; src/util/timer.cpp:24-39.
package stopwatch

import "errors"

var ErrTimerOwner = errors.New("g3: invalid timer owner")
var ErrTimerPending = errors.New("g3: timer operation in progress")
var ErrTimerClosed = errors.New("g3: timer closed")

// Timer measures accumulated process CPU time. A nil clock selects the
// platform clock. It owns its stopwatch directly and must not be copied.
type Timer struct {
	self           *Timer
	watch          Stopwatch
	cause          error
	active, closed bool
}

func NewTimer(clock func() (uint64, error)) (*Timer, error) {
	p := &Timer{watch: New(clock), active: true}
	p.self = p
	p.cause = p.watch.Start()
	p.active = false
	return p, p.cause
}

func (p *Timer) open() error {
	if p == nil || p.self != p {
		return ErrTimerOwner
	}
	if p.closed {
		return ErrTimerClosed
	}
	if p.active {
		return ErrTimerPending
	}
	return p.cause
}

func (p *Timer) Start() error {
	if e := p.open(); e != nil {
		return e
	}
	p.active = true
	defer func() { p.active = false }()
	return p.watch.Start()
}

// GetSeconds stops then restarts a running watch, preserving accumulated time
// and the reached stop state if the restart clock read fails.
func (p *Timer) GetSeconds() (float64, error) {
	if e := p.open(); e != nil {
		return 0, e
	}
	p.active = true
	defer func() { p.active = false }()
	return p.watch.Seconds()
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
	if p.active {
		return ErrTimerPending
	}
	p.closed = true
	return nil
}
