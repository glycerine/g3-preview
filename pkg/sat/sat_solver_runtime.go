// Source: src/sat/sat_solver.cpp:34–58; src/util/memory_manager.cpp; src/util/gparams.cpp.
package sat

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

var ErrSATRuntimeLive = errors.New("sat: borrowed runtime still has live solver owners")

// The source process-wide memory/global-parameter objects become an explicit
// cold caller owner. These are actual services; Budget reservations never
// synthesize Runtime malloc events. ParamsContext and its SymbolManager remain
// borrowed and must outlive this owner and all its solver borrowers.
type RuntimeConfig struct {
	Budget        *arena.Budget
	ParamsContext *uparams.ParamsContext
}
type Runtime struct {
	self            *Runtime
	budget          *arena.Budget
	context         *uparams.ParamsContext
	memory          *umemory.Runtime
	globals         *uparams.GlobalParams
	ownerBytes      uint64
	liveSolvers     uint32
	closing, closed bool
}

func NewRuntime(c RuntimeConfig) (r *Runtime, err error) {
	if c.ParamsContext == nil || c.Budget == nil || c.ParamsContext.Budget() != c.Budget {
		return nil, ErrSATOwner
	}
	if err = c.ParamsContext.CheckOpen(); err != nil {
		return nil, err
	}
	n := uint64(reflect.TypeOf(Runtime{}).Size())
	if err = c.Budget.Reserve(n); err != nil {
		return nil, err
	}
	r = &Runtime{budget: c.Budget, context: c.ParamsContext, ownerBytes: n}
	r.self = r
	defer func() {
		if err != nil {
			if cleanup := r.Close(); cleanup != nil {
				err = errors.Join(err, cleanup)
			} else {
				r = nil
			}
		}
	}()
	if r.memory, err = umemory.NewRuntime(umemory.RuntimeConfig{Budget: c.Budget}); err != nil {
		return r, err
	}
	// GetModule does not execute descriptor registration. Optional descriptor
	// catalog production remains its actual separate binding, never a fake
	// successful callback installed by the solver constructor.
	r.globals, err = uparams.NewGlobalParams(c.ParamsContext.Symbols(), c.ParamsContext, uparams.GlobalParamsConfig{Budget: c.Budget})
	return r, err
}
func (r *Runtime) open() error {
	if r == nil || r.self != r {
		return ErrSATOwner
	}
	if r.closed || r.closing {
		return arena.ErrClosed
	}
	return r.context.CheckOpen()
}
func (r *Runtime) Close() error {
	if r == nil || r.self != r {
		return ErrSATOwner
	}
	if r.closed {
		return nil
	}
	if r.liveSolvers != 0 {
		return ErrSATRuntimeLive
	}
	r.closing = true
	// Reverse actual completed service construction; a refused close retains
	// every later owner and the reflected cold header for explicit retry.
	if r.globals != nil {
		if e := r.globals.Close(); e != nil {
			return e
		}
		r.globals = nil
	}
	if r.memory != nil {
		if e := r.memory.Close(); e != nil {
			return e
		}
		r.memory = nil
	}
	r.budget.Release(r.ownerBytes)
	r.ownerBytes = 0
	r.closed = true
	return nil
}
