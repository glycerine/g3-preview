// Source: src/tactic/model_converter.cpp:23-126,183-187;
// src/tactic/converter.h:26-66,96-124; src/util/ref.h:26-90.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/golang/arena"
)

var errTacticModelConverterOwner = errors.New("model converter: invalid, copied or retired owner")
var errTacticModelConverterGoal = errors.New("model converter: unreachable goal index or unindexed star application")

// Cold virtual-owner boundary for original model converters. Models currently
// expose their ported model_core constant portion; derived models, evaluator,
// translation and display remain separate source dependencies. These methods
// dispatch original converter operations; they never perform a solver check.
type tacticModelConverter interface {
	IncRef() error
	DecRef() error
	Close() error
	ApplyModel(*tacticModelRef, uint32) error
	ApplyUnindexed(*tacticModelRef) error
	ApplyLabels(*tacticModelLabels, uint32) error
	Cancel() error
}

// Source owning ref assignment: retain incoming before releasing outgoing.
// A zero value is an empty reference. The borrowed AST manager outlives models.
type tacticModelRef struct {
	self   *tacticModelRef
	value  *modelCore
	closed bool
}

func (r *tacticModelRef) open() error {
	if r == nil || r.closed || (r.self != nil && r.self != r) {
		return errTacticModelConverterOwner
	}
	return nil
}
func (r *tacticModelRef) Set(value *modelCore) error {
	if e := r.open(); e != nil {
		return e
	}
	r.self = r
	if value != nil {
		if e := value.IncRef(); e != nil {
			return e
		}
	}
	if r.value != nil {
		if e := r.value.DecRef(); e != nil {
			if value != nil {
				return errors.Join(e, value.DecRef())
			}
			return e
		}
	}
	r.value = value
	return nil
}
func (r *tacticModelRef) Get() (*modelCore, error) {
	if e := r.open(); e != nil {
		return nil, e
	}
	if r.value != nil {
		if e := r.value.open(); e != nil {
			return nil, e
		}
	}
	return r.value, nil
}
func (r *tacticModelRef) Close() error {
	if r == nil || (r.self != nil && r.self != r) {
		return errTacticModelConverterOwner
	}
	if r.closed {
		return nil
	}
	if e := r.Set(nil); e != nil {
		return e
	}
	r.closed = true
	return nil
}

// Concrete labels_vec/svector<symbol>. It owns scalar backing and borrows
// symbol identities, which stay valid with the source symbol manager.
type tacticModelLabels struct {
	values []Symbol
	budget *arena.Budget
}

func (l *tacticModelLabels) Append(value Symbol) error {
	if l == nil {
		return errTacticModelConverterOwner
	}
	if len(l.values) == cap(l.values) {
		stride := uint64(reflect.TypeOf(Symbol{}).Size())
		n, e := satGoalCapacity(cap(l.values), stride)
		if e != nil {
			return e
		}
		if e = l.budget.Reserve(uint64(n) * stride); e != nil {
			return e
		}
		next := make([]Symbol, len(l.values), n)
		copy(next, l.values)
		l.budget.Release(uint64(cap(l.values)) * stride)
		l.values = next
	}
	l.values = append(l.values, value)
	return nil
}
func (l *tacticModelLabels) Close() {
	if l == nil {
		return
	}
	l.budget.Release(uint64(cap(l.values)) * uint64(reflect.TypeOf(Symbol{}).Size()))
	l.values = nil
}

type tacticConverterBase struct {
	self       *tacticConverterBase
	owner      tacticModelConverter
	budget     *arena.Budget
	ownerBytes uint64
	references uint32
	closed     bool
}

func (b *tacticConverterBase) init(owner tacticModelConverter, budget *arena.Budget, bytes uint64) error {
	if e := budget.Reserve(bytes); e != nil {
		return e
	}
	b.self = b
	b.owner = owner
	b.budget = budget
	b.ownerBytes = bytes
	return nil
}
func (b *tacticConverterBase) open() error {
	if b == nil || b.self != b || b.closed {
		return errTacticModelConverterOwner
	}
	return nil
}
func (b *tacticConverterBase) IncRef() error {
	if e := b.open(); e != nil {
		return e
	}
	b.references++
	return nil
}
func (b *tacticConverterBase) DecRef() error {
	if e := b.open(); e != nil {
		return e
	}
	if b.references == 0 {
		return errTacticModelConverterOwner
	}
	b.references--
	if b.references == 0 {
		return b.owner.Close()
	}
	return nil
}
func (b *tacticConverterBase) release() {
	b.budget.Release(b.ownerBytes)
	b.ownerBytes = 0
	b.owner = nil
	b.closed = true
}

type tacticConcatModelConverter struct {
	tacticConverterBase
	first, second tacticModelConverter
}

func concatTacticModels(b *arena.Budget, first, second tacticModelConverter) (result tacticModelConverter, err error) {
	if first == nil {
		return second, nil
	}
	if second == nil {
		return first, nil
	}
	c := &tacticConcatModelConverter{}
	if err = c.init(c, b, uint64(reflect.TypeOf(*c).Size())); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, c.Close())
		}
	}()
	if err = first.IncRef(); err != nil {
		return nil, err
	}
	c.first = first
	if err = second.IncRef(); err != nil {
		return nil, err
	}
	c.second = second
	return c, nil
}
func (c *tacticConcatModelConverter) ApplyModel(m *tacticModelRef, index uint32) error {
	if e := c.open(); e != nil {
		return e
	}
	if e := m.open(); e != nil {
		return e
	}
	if e := c.second.ApplyModel(m, index); e != nil {
		return e
	}
	return c.first.ApplyModel(m, 0)
}
func (c *tacticConcatModelConverter) ApplyUnindexed(m *tacticModelRef) error {
	if e := c.open(); e != nil {
		return e
	}
	if e := m.open(); e != nil {
		return e
	}
	if e := c.second.ApplyUnindexed(m); e != nil {
		return e
	}
	return c.first.ApplyUnindexed(m)
}
func (c *tacticConcatModelConverter) ApplyLabels(labels *tacticModelLabels, index uint32) error {
	if e := c.open(); e != nil {
		return e
	}
	if labels == nil {
		return errTacticModelConverterOwner
	}
	if e := c.second.ApplyLabels(labels, index); e != nil {
		return e
	}
	return c.first.ApplyLabels(labels, 0)
}
func (c *tacticConcatModelConverter) Cancel() error {
	if e := c.open(); e != nil {
		return e
	}
	if e := c.second.Cancel(); e != nil {
		return e
	}
	return c.first.Cancel()
}
func (c *tacticConcatModelConverter) Close() error {
	if c == nil || c.self != &c.tacticConverterBase {
		return errTacticModelConverterOwner
	}
	if c.closed {
		return nil
	}
	// Source reverse member destruction: second reference, then first.
	if c.second != nil {
		if e := c.second.DecRef(); e != nil {
			return e
		}
		c.second = nil
	}
	if c.first != nil {
		if e := c.first.DecRef(); e != nil {
			return e
		}
		c.first = nil
	}
	c.release()
	return nil
}

type tacticConcatStarModelConverter struct {
	tacticConverterBase
	first tacticModelConverter
	// Cold references to source virtual converter owners; no AST or solver
	// records are boxed. Scalar subgoal sizes retain their own source vector.
	seconds    []tacticModelConverter
	sizes      []uint32
	pending    tacticModelConverter
	closeIndex uint32
}

func (c *tacticConcatStarModelConverter) pushSecond(value tacticModelConverter) error {
	if len(c.seconds) == cap(c.seconds) {
		stride := uint64(reflect.TypeOf((*tacticModelConverter)(nil)).Elem().Size())
		n, e := satGoalCapacity(cap(c.seconds), stride)
		if e != nil {
			return e
		}
		if e = c.budget.Reserve(uint64(n) * stride); e != nil {
			return e
		}
		next := make([]tacticModelConverter, len(c.seconds), n)
		copy(next, c.seconds)
		c.budget.Release(uint64(cap(c.seconds)) * stride)
		c.seconds = next
	}
	c.seconds = append(c.seconds, value)
	return nil
}
func (c *tacticConcatStarModelConverter) pushSize(size uint32) error {
	if len(c.sizes) == cap(c.sizes) {
		n, e := satGoalCapacity(cap(c.sizes), 4)
		if e != nil {
			return e
		}
		if e = c.budget.Reserve(uint64(n) * 4); e != nil {
			return e
		}
		next := make([]uint32, len(c.sizes), n)
		copy(next, c.sizes)
		c.budget.Release(uint64(cap(c.sizes)) * 4)
		c.sizes = next
	}
	c.sizes = append(c.sizes, size)
	return nil
}
func concatTacticModelSubgoals(b *arena.Budget, first tacticModelConverter, seconds []tacticModelConverter, sizes []uint32) (result tacticModelConverter, err error) {
	if len(seconds) == 0 && astSourceDebug {
		return nil, errTacticModelConverterGoal
	}
	if len(seconds) == 1 {
		return concatTacticModels(b, first, seconds[0])
	}
	allNull := true
	for _, second := range seconds {
		if second != nil {
			allNull = false
			break
		}
	}
	if allNull {
		return first, nil
	}
	// The original shortcuts never read szs. A table is needed only when the
	// actual star constructor will copy one size for each second converter.
	if len(sizes) < len(seconds) {
		return nil, errTacticModelConverterGoal
	}
	c := &tacticConcatStarModelConverter{}
	if err = c.init(c, b, uint64(reflect.TypeOf(*c).Size())); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, c.Close())
		}
	}()
	if first != nil {
		if err = first.IncRef(); err != nil {
			return nil, err
		}
		c.first = first
	}
	for i, second := range seconds {
		if second != nil {
			if err = second.IncRef(); err != nil {
				return nil, err
			}
			c.pending = second
		}
		if err = c.pushSecond(second); err != nil {
			return nil, err
		}
		c.pending = nil
		if err = c.pushSize(sizes[i]); err != nil {
			return nil, err
		}
	}
	return c, nil
}
func (c *tacticConcatStarModelConverter) branch(index uint32) (uint32, uint32, error) {
	if e := c.open(); e != nil {
		return 0, 0, e
	}
	for i, size := range c.sizes {
		if index < size {
			return uint32(i), index, nil
		}
		index -= size
	}
	return 0, 0, errTacticModelConverterGoal
}
func (c *tacticConcatStarModelConverter) ApplyModel(m *tacticModelRef, index uint32) error {
	if e := m.open(); e != nil {
		return e
	}
	i, local, e := c.branch(index)
	if e != nil {
		return e
	}
	if c.seconds[i] != nil {
		if e = c.seconds[i].ApplyModel(m, local); e != nil {
			return e
		}
	}
	if c.first != nil {
		return c.first.ApplyModel(m, i)
	}
	return nil
}
func (c *tacticConcatStarModelConverter) ApplyUnindexed(*tacticModelRef) error {
	if e := c.open(); e != nil {
		return e
	}
	return errTacticModelConverterGoal
}
func (c *tacticConcatStarModelConverter) ApplyLabels(labels *tacticModelLabels, index uint32) error {
	if labels == nil {
		return errTacticModelConverterOwner
	}
	i, local, e := c.branch(index)
	if e != nil {
		return e
	}
	if c.seconds[i] != nil {
		if e = c.seconds[i].ApplyLabels(labels, local); e != nil {
			return e
		}
	}
	if c.first != nil {
		return c.first.ApplyLabels(labels, i)
	}
	return nil
}
func (c *tacticConcatStarModelConverter) Cancel() error {
	if e := c.open(); e != nil {
		return e
	}
	if c.first != nil {
		if e := c.first.Cancel(); e != nil {
			return e
		}
	}
	for _, second := range c.seconds {
		if second != nil {
			if e := second.Cancel(); e != nil {
				return e
			}
		}
	}
	return nil
}
func (c *tacticConcatStarModelConverter) Close() error {
	if c == nil || c.self != &c.tacticConverterBase {
		return errTacticModelConverterOwner
	}
	if c.closed {
		return nil
	}
	for uint64(c.closeIndex) < uint64(len(c.seconds)) {
		second := c.seconds[c.closeIndex]
		if second != nil {
			if e := second.DecRef(); e != nil {
				return e
			}
			c.seconds[c.closeIndex] = nil
		}
		c.closeIndex++
	}
	if c.pending != nil {
		if e := c.pending.DecRef(); e != nil {
			return e
		}
		c.pending = nil
	}
	c.budget.Release(uint64(cap(c.sizes)) * 4)
	c.sizes = nil
	c.budget.Release(uint64(cap(c.seconds)) * uint64(reflect.TypeOf((*tacticModelConverter)(nil)).Elem().Size()))
	c.seconds = nil
	if c.first != nil {
		if e := c.first.DecRef(); e != nil {
			return e
		}
		c.first = nil
	}
	c.release()
	return nil
}

// Source apply(model_converter_ref&,model_ref&,unsigned) null shortcut.
func applyTacticModelConverter(c tacticModelConverter, m *tacticModelRef, index uint32) error {
	if c == nil {
		return nil
	}
	return c.ApplyModel(m, index)
}
