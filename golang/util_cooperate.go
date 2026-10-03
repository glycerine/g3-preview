// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/golang/arena"
)

var (
	ErrCooperationOwner    = errors.New("g3: invalid cooperation owner")
	ErrCooperationEnabled  = errors.New("g3: cooperation enabled assertion")
	ErrCooperationParallel = errors.New("g3: cooperation parallel assertion")
)

// utilCooperation implements cooperate.cpp with z3_omp.h's actual _NO_OMP_
// profile. It is confined to one operation owner. OpenMP scheduling is a
// separate, unported profile. Source process globals become this cold owner;
// all header bytes, including its Go lifetime adapter, are charged once.
//
// Begin/EndSection and Begin/EndTask express the lexical source constructors
// and destructors. The source empty RAII objects have no additional payload.
// Callers execute the matching End only after a successful Begin.
type utilCooperation struct {
	self            *utilCooperation
	budget          *arena.Budget
	ownerThread     int32
	enabled, closed bool
}

func newUtilCooperation(b *arena.Budget) (*utilCooperation, error) {
	if err := b.Reserve(uint64(reflect.TypeOf(utilCooperation{}).Size())); err != nil {
		return nil, err
	}
	c := &utilCooperation{budget: b, ownerThread: -1}
	c.self = c
	// omp_set_nested and omp_init_nest_lock are source no-ops in this profile.
	return c, nil
}

func (c *utilCooperation) check() error {
	if c == nil || c.self != c {
		return ErrCooperationOwner
	}
	if c.closed {
		return arena.ErrClosed
	}
	return nil
}

func (c *utilCooperation) Enabled() (bool, error) {
	if err := c.check(); err != nil {
		return false, err
	}
	return c.enabled, nil
}

func (c *utilCooperation) Checkpoint(task string) error {
	if err := c.check(); err != nil {
		return err
	}
	if cooperationSourceDebug && !c.enabled {
		return ErrCooperationEnabled
	}
	// omp_get_thread_num is exactly zero. The source releases an owned lock
	// before entering its critical section, then acquires and publishes owner.
	// Both lock calls and the critical pragma are no-ops under _NO_OMP_.
	if c.ownerThread == 0 {
		c.ownerThread = -1
	}
	c.ownerThread = 0
	// Source m_task is never assigned. task is used only by unported TRACE;
	// there is no retained task pointer or invented string validation here.
	return nil
}

func (c *utilCooperation) Cooperate(task string) error {
	if err := c.check(); err != nil {
		return err
	}
	if c.enabled {
		return c.Checkpoint(task)
	}
	return nil
}

func (c *utilCooperation) BeginSection() error {
	if err := c.check(); err != nil {
		return err
	}
	if cooperationSourceDebug && c.enabled {
		return ErrCooperationEnabled
	}
	// The second source assertion !omp_in_parallel() always passes here.
	c.enabled = true
	return nil
}

func (c *utilCooperation) EndSection() error {
	if err := c.check(); err != nil {
		return err
	}
	if cooperationSourceDebug && !c.enabled {
		return ErrCooperationEnabled
	}
	c.enabled = false
	return nil
}

func (c *utilCooperation) BeginTask(task string) error {
	if err := c.check(); err != nil {
		return err
	}
	if cooperationSourceDebug {
		if !c.enabled {
			return ErrCooperationEnabled
		}
		// omp_in_parallel() is false: this source constructor's parallel
		// assertion cannot pass in the debug _NO_OMP_ profile.
		return ErrCooperationParallel
	}
	return c.Checkpoint(task)
}

func (c *utilCooperation) EndTask() error {
	if err := c.check(); err != nil {
		return err
	}
	if c.ownerThread == 0 {
		c.ownerThread = -1
	}
	return nil
}

func (c *utilCooperation) Close() error {
	if c == nil || c.self != c {
		return ErrCooperationOwner
	}
	if c.closed {
		return nil
	}
	// Source omp_destroy_nest_lock is a no-op; it does not execute lexical
	// section/task destructors or change the retained enable/owner scalars.
	c.closed = true
	c.budget.Release(uint64(reflect.TypeOf(utilCooperation{}).Size()))
	return nil
}
