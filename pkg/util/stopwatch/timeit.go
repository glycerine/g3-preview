// Source: src/util/timeit.h:25-31; src/util/timeit.cpp:25-58.
package stopwatch

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

var ErrTimeitOwner = errors.New("g3: invalid timeit owner")
var ErrTimeitPending = errors.New("g3: timeit operation in progress")

// TimeitConfig supplies optional report providers. Output defaults to stderr;
// Memory reports bytes and defaults to zero. A nil Clock uses process CPU time.
type TimeitConfig struct {
	Output  io.Writer
	Message string
	Memory  func() (uint64, error)
	Clock   func() (uint64, error)
}

// Timeit reports elapsed time and memory snapshots when closed. Construct it
// in its final location and do not copy it. Disabled scopes read no providers.
type Timeit struct {
	self                    *Timeit
	watch                   Stopwatch
	message                 string
	output                  io.Writer
	memory                  func() (uint64, error)
	beforeMemory            float64
	cause                   error
	active, closed, enabled bool
}

func (p *Timeit) Construct(enabled bool, config TimeitConfig) (cause error) {
	if p == nil || p.self != nil {
		return ErrTimeitOwner
	}
	p.self, p.enabled = p, enabled
	if !enabled {
		return nil
	}
	p.active = true
	defer func() { p.active = false; p.cause = cause }()
	p.output = config.Output
	if p.output == nil {
		p.output = os.Stderr
	}
	p.message, _, _ = strings.Cut(config.Message, "\x00")
	p.memory = config.Memory
	var e error
	p.beforeMemory, e = p.memoryMB()
	if e != nil {
		return e
	}
	p.watch = New(config.Clock)
	return p.watch.Start()
}

func (p *Timeit) memoryMB() (float64, error) {
	if p.memory == nil {
		return 0, nil
	}
	bytes, e := p.memory()
	return float64(bytes) / (1024 * 1024), e
}

func (p *Timeit) Close() (cause error) {
	if p == nil || p.self != p {
		return ErrTimeitOwner
	}
	if p.closed {
		return nil
	}
	if p.active {
		return ErrTimeitPending
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
	if e := p.watch.Stop(); e != nil {
		return e
	}
	afterMemory, e := p.memoryMB()
	if e != nil {
		return e
	}
	seconds, e := p.watch.Seconds()
	if e != nil {
		return e
	}
	if _, e = fmt.Fprintf(p.output, "(%s :time %.2f :before-memory %.2f :after-memory %.2f)\n", p.message, seconds, p.beforeMemory, afterMemory); e != nil {
		return e
	}
	if out, ok := p.output.(interface{ Flush() error }); ok {
		if e = out.Flush(); e != nil {
			return e
		}
	}
	p.closed = true
	return nil
}

// Cleanup ends a failed scope without repeating clock reads or report output.
func (p *Timeit) Cleanup() error {
	if p == nil || p.self != p {
		return ErrTimeitOwner
	}
	if p.active {
		return ErrTimeitPending
	}
	p.closed = true
	p.watch = Stopwatch{}
	p.message, p.output, p.memory = "", nil, nil
	return nil
}
