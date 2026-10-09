// Source: src/tactic/model_converter.cpp:127-151,164-174;
// src/util/buffer.h:25-48,68-82,94-96; src/util/vector.h:443-447.
package tactic

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util"
)

// Model2mc copies labels rather than retaining the caller's slice; applying
// labels appends without clearing the destination.
type tacticStoredLabelBuffer struct{ values []util.Symbol }

func (b *tacticStoredLabelBuffer) push(value util.Symbol) error {
	b.values = append(b.values, value)
	return nil
}
func (b *tacticStoredLabelBuffer) close() {
	clear(b.values)
	b.values = nil
}

// Original model2mc's ownership/application portion. The stored value currently
// exposes model_core through the original derived lifetime dispatch;
// display and AST translation remain required; this wrapper
// does not build or evaluate a model.
type tacticStoredModelConverter struct {
	tacticConverterBase
	model  ModelRef
	labels tacticStoredLabelBuffer
}

func NewStoredModel(md *model.ModelCore) (ModelConverter, error) {
	return newTacticStoredModelWithLabels(md, nil)
}
func newTacticStoredModelWithLabels(md *model.ModelCore, labels []util.Symbol) (result ModelConverter, err error) {
	if md == nil {
		return nil, nil
	}
	if err = md.Check(); err != nil {
		return nil, err
	}
	c := &tacticStoredModelConverter{}
	if err = c.init(c, md.Manager().Budget(), uint64(reflect.TypeOf(*c).Size())); err != nil {
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
		if err = c.labels.push(label); err != nil {
			return nil, err
		}
	}
	return c, nil
}
func (c *tacticStoredModelConverter) ApplyModel(out *ModelRef, _ uint32) error {
	if e := c.open(); e != nil {
		return e
	}
	md, e := c.model.Get()
	if e != nil {
		return e
	}
	return out.Set(md)
}
func (c *tacticStoredModelConverter) ApplyUnindexed(out *ModelRef) error {
	return c.ApplyModel(out, 0)
}
func (c *tacticStoredModelConverter) ApplyLabels(out *ModelLabels, _ uint32) error {
	if e := c.open(); e != nil {
		return e
	}
	if out == nil {
		return ErrModelConverterOwner
	}
	for _, label := range c.labels.values {
		if e := out.Append(label); e != nil {
			return e
		}
	}
	return nil
}
func (c *tacticStoredModelConverter) Cancel() error { return c.open() }
func (c *tacticStoredModelConverter) Close() error {
	if c == nil || c.self != &c.tacticConverterBase {
		return ErrModelConverterOwner
	}
	if c.closed {
		return nil
	}
	// Reverse member destruction: label buffer, then owning model reference.
	c.labels.close()
	if e := c.model.Close(); e != nil {
		return e
	}
	c.release()
	return nil
}
