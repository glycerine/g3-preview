// Source: src/tactic/model_converter.cpp:127-151,164-174;
// src/util/buffer.h:25-48,68-82,94-96; src/util/vector.h:443-447.
package tactic

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

// Original model2mc's ownership/application portion. The stored value currently
// exposes model_core through the original derived lifetime dispatch;
// display and AST translation remain required; this wrapper
// does not build or evaluate a model.
type tacticStoredModelConverter struct {
	tacticConverterBase
	model           ModelRef
	labels          []usymbol.Symbol
	memoryManager   *ast.ASTManager
	memoryTrace     *diagnostic.TraceTags
	memoryTraceLive bool
}

func NewStoredModel(md *model.ModelCore) (ModelConverter, error) {
	return newTacticStoredModelWithLabels(md, nil)
}
func newTacticStoredModelWithLabels(md *model.ModelCore, labels []usymbol.Symbol) (result ModelConverter, err error) {
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
	c.memoryManager = md.Manager()
	c.memoryTrace = c.memoryManager.BorrowMemoryTraceTags()
	c.traceStoredModelAllocated(labels != nil)
	defer func() {
		if err != nil {
			err = errors.Join(err, c.Close())
		}
	}()
	if err = c.model.Set(md); err != nil {
		return nil, err
	}
	// Copy the source values so later changes to the caller's slice cannot
	// change the converter's labels.
	c.labels = append(c.labels, labels...)
	return c, nil
}
func (c *tacticStoredModelConverter) ApplyModel(out *ModelRef, _ uint32) error {
	if e := c.open(); e != nil {
		return e
	}
	return out.SetRef(&c.model)
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
	for _, label := range c.labels {
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
	// Clear labels before releasing the older owning model reference.
	clear(c.labels)
	c.labels = nil
	if e := c.model.Close(); e != nil {
		return e
	}
	c.traceStoredModelDeallocated()
	c.release()
	return nil
}
