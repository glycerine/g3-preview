// Source: src/tactic/model_converter.cpp:23-126,183-187;
// src/tactic/converter.h:26-66,96-124; src/util/ref.h:26-90.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package tactic

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util"
)

var ErrModelConverterOwner = errors.New("model converter: invalid, copied or retired owner")
var errTacticModelConverterGoal = errors.New("model converter: unreachable goal index or unindexed star application")

// Cold virtual-owner boundary for original model converters. Models currently
// expose model_core through the original derived lifetime dispatch. Function
// interpretation operations, evaluator, translation and display remain separate
// source dependencies. These methods
// dispatch original converter operations; they never perform a solver check.
type ModelConverter interface {
	IncRef() error
	DecRef() error
	Close() error
	ApplyModel(*ModelRef, uint32) error
	ApplyUnindexed(*ModelRef) error
	ApplyLabels(*ModelLabels, uint32) error
	Cancel() error
}

// Source owning ref assignment: retain incoming before releasing outgoing.
// A zero value is an empty reference. The borrowed AST manager outlives models.
type ModelRef struct {
	self   *ModelRef
	value  *model.ModelCore
	closed bool
}

func (r *ModelRef) open() error {
	if r == nil || r.closed || (r.self != nil && r.self != r) {
		return ErrModelConverterOwner
	}
	return nil
}
func (r *ModelRef) Set(value *model.ModelCore) error {
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
func (r *ModelRef) Get() (*model.ModelCore, error) {
	if e := r.open(); e != nil {
		return nil, e
	}
	if r.value != nil {
		if e := r.value.Check(); e != nil {
			return nil, e
		}
	}
	return r.value, nil
}
func (r *ModelRef) Close() error {
	if r == nil || (r.self != nil && r.self != r) {
		return ErrModelConverterOwner
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
type ModelLabels struct {
	Values []util.Symbol
	Budget *arena.Budget
}

func (l *ModelLabels) Append(value util.Symbol) error {
	if l == nil {
		return ErrModelConverterOwner
	}
	if len(l.Values) == cap(l.Values) {
		stride := uint64(reflect.TypeOf(util.Symbol{}).Size())
		n, e := util.SourceVectorCapacity(cap(l.Values), stride)
		if e != nil {
			return e
		}
		if e = l.Budget.Reserve(uint64(n) * stride); e != nil {
			return e
		}
		next := make([]util.Symbol, len(l.Values), n)
		copy(next, l.Values)
		l.Budget.Release(uint64(cap(l.Values)) * stride)
		l.Values = next
	}
	l.Values = append(l.Values, value)
	return nil
}
func (l *ModelLabels) Close() {
	if l == nil {
		return
	}
	l.Budget.Release(uint64(cap(l.Values)) * uint64(reflect.TypeOf(util.Symbol{}).Size()))
	l.Values = nil
}

type tacticConverterBase struct {
	self       *tacticConverterBase
	owner      ModelConverter
	budget     *arena.Budget
	ownerBytes uint64
	references uint32
	closed     bool
}

func (b *tacticConverterBase) init(owner ModelConverter, budget *arena.Budget, bytes uint64) error {
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
		return ErrModelConverterOwner
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
		return ErrModelConverterOwner
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
	first, second ModelConverter
}

func ConcatModels(b *arena.Budget, first, second ModelConverter) (result ModelConverter, err error) {
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
func (c *tacticConcatModelConverter) ApplyModel(m *ModelRef, index uint32) error {
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
func (c *tacticConcatModelConverter) ApplyUnindexed(m *ModelRef) error {
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
func (c *tacticConcatModelConverter) ApplyLabels(labels *ModelLabels, index uint32) error {
	if e := c.open(); e != nil {
		return e
	}
	if labels == nil {
		return ErrModelConverterOwner
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
		return ErrModelConverterOwner
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
	first ModelConverter
	// Cold references to source virtual converter owners; no AST or solver
	// records are boxed. Scalar subgoal sizes retain their own source vector.
	seconds    []ModelConverter
	sizes      []uint32
	pending    ModelConverter
	closeIndex uint32
}

func (c *tacticConcatStarModelConverter) pushSecond(value ModelConverter) error {
	if len(c.seconds) == cap(c.seconds) {
		stride := uint64(reflect.TypeOf((*ModelConverter)(nil)).Elem().Size())
		n, e := util.SourceVectorCapacity(cap(c.seconds), stride)
		if e != nil {
			return e
		}
		if e = c.budget.Reserve(uint64(n) * stride); e != nil {
			return e
		}
		next := make([]ModelConverter, len(c.seconds), n)
		copy(next, c.seconds)
		c.budget.Release(uint64(cap(c.seconds)) * stride)
		c.seconds = next
	}
	c.seconds = append(c.seconds, value)
	return nil
}
func (c *tacticConcatStarModelConverter) pushSize(size uint32) error {
	if len(c.sizes) == cap(c.sizes) {
		n, e := util.SourceVectorCapacity(cap(c.sizes), 4)
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
func concatTacticModelSubgoals(b *arena.Budget, first ModelConverter, seconds []ModelConverter, sizes []uint32) (result ModelConverter, err error) {
	if len(seconds) == 0 && tacticSourceDebug {
		return nil, errTacticModelConverterGoal
	}
	if len(seconds) == 1 {
		return ConcatModels(b, first, seconds[0])
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
func (c *tacticConcatStarModelConverter) ApplyModel(m *ModelRef, index uint32) error {
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
func (c *tacticConcatStarModelConverter) ApplyUnindexed(*ModelRef) error {
	if e := c.open(); e != nil {
		return e
	}
	return errTacticModelConverterGoal
}
func (c *tacticConcatStarModelConverter) ApplyLabels(labels *ModelLabels, index uint32) error {
	if labels == nil {
		return ErrModelConverterOwner
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
		return ErrModelConverterOwner
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
	c.budget.Release(uint64(cap(c.seconds)) * uint64(reflect.TypeOf((*ModelConverter)(nil)).Elem().Size()))
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
func applyTacticModelConverter(c ModelConverter, m *ModelRef, index uint32) error {
	if c == nil {
		return nil
	}
	return c.ApplyModel(m, index)
}
