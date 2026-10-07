// Source: src/tactic/model_converter.cpp:127-151,164-174;
// src/util/buffer.h:25-48,68-82,94-96; src/util/vector.h:443-447.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/golang/arena"
)

// Concrete buffer<symbol>: sixteen inline borrowed symbol identities and a
// doubling spill buffer. Model2mc copies the labels rather than the caller's
// buffer binding; applying labels appends without clearing the destination.
type tacticStoredLabelBuffer struct {
	inline [16]Symbol
	spill  []Symbol
	size   uint32
}

func (b *tacticStoredLabelBuffer) push(budget *arena.Budget, value Symbol) error {
	capacity := len(b.inline)
	if b.spill != nil {
		capacity = len(b.spill)
	}
	if uint64(b.size) == uint64(capacity) {
		stride := uint64(reflect.TypeOf(Symbol{}).Size())
		n := uint64(capacity) * 2
		if n > uint64(^uint(0)>>1)/stride || n > uint64(^uint32(0)) {
			return arena.ErrBounds
		}
		if e := budget.Reserve(n * stride); e != nil {
			return e
		}
		next := make([]Symbol, int(n))
		if b.spill != nil {
			copy(next, b.spill[:b.size])
			budget.Release(uint64(len(b.spill)) * stride)
		} else {
			copy(next, b.inline[:b.size])
		}
		b.spill = next
	}
	if b.spill != nil {
		b.spill[b.size] = value
	} else {
		b.inline[b.size] = value
	}
	b.size++
	return nil
}
func (b *tacticStoredLabelBuffer) close(budget *arena.Budget) {
	budget.Release(uint64(len(b.spill)) * uint64(reflect.TypeOf(Symbol{}).Size()))
	b.spill = nil
	b.size = 0
	clear(b.inline[:])
}

// Original model2mc's ownership/application portion. The stored value currently
// exposes the ported model_core portion. Derived model dispatch, display and
// AST translation remain required; this wrapper does not build or evaluate a
// model and does not replace model_converter2model's source model allocation.
type tacticStoredModelConverter struct {
	tacticConverterBase
	model  tacticModelRef
	labels tacticStoredLabelBuffer
}

func newTacticStoredModel(md *modelCore) (tacticModelConverter, error) {
	return newTacticStoredModelWithLabels(md, nil)
}
func newTacticStoredModelWithLabels(md *modelCore, labels []Symbol) (result tacticModelConverter, err error) {
	if md == nil {
		return nil, nil
	}
	if err = md.open(); err != nil {
		return nil, err
	}
	c := &tacticStoredModelConverter{}
	if err = c.init(c, md.manager.budget, uint64(reflect.TypeOf(*c).Size())); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, c.Close())
		}
	}()
	if err = c.model.Set(md); err != nil {
		return nil, err
	}
	for _, label := range labels {
		if err = c.labels.push(c.budget, label); err != nil {
			return nil, err
		}
	}
	return c, nil
}
func (c *tacticStoredModelConverter) ApplyModel(out *tacticModelRef, _ uint32) error {
	if e := c.open(); e != nil {
		return e
	}
	md, e := c.model.Get()
	if e != nil {
		return e
	}
	return out.Set(md)
}
func (c *tacticStoredModelConverter) ApplyUnindexed(out *tacticModelRef) error {
	return c.ApplyModel(out, 0)
}
func (c *tacticStoredModelConverter) ApplyLabels(out *tacticModelLabels, _ uint32) error {
	if e := c.open(); e != nil {
		return e
	}
	if out == nil {
		return errTacticModelConverterOwner
	}
	for i := uint32(0); i < c.labels.size; i++ {
		var label Symbol
		if c.labels.spill != nil {
			label = c.labels.spill[i]
		} else {
			label = c.labels.inline[i]
		}
		if e := out.Append(label); e != nil {
			return e
		}
	}
	return nil
}
func (c *tacticStoredModelConverter) Cancel() error { return c.open() }
func (c *tacticStoredModelConverter) Close() error {
	if c == nil || c.self != &c.tacticConverterBase {
		return errTacticModelConverterOwner
	}
	if c.closed {
		return nil
	}
	// Reverse member destruction: label buffer, then owning model reference.
	c.labels.close(c.budget)
	if e := c.model.Close(); e != nil {
		return e
	}
	c.release()
	return nil
}
