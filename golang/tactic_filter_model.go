// Source: src/tactic/filter_model_converter.cpp:23-57;
// src/tactic/filter_model_converter.h:26-49; src/util/ref_vector.h:96-100.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"errors"
	"reflect"
)

// Original filter converter's constant/universe path. The function-copy loop,
// AST translation and display remain unported. Function maps cannot currently
// be populated through modelCore; a nonempty map reports that missing operation
// rather than silently producing a model with missing function interpretations.
type tacticFilterModelConverter struct {
	tacticConverterBase
	manager      *ASTManager
	declarations modelDeclarationVector // func_decl_ref_vector: owning entries
}

func newTacticFilterModel(m *ASTManager) (*tacticFilterModelConverter, error) {
	if m == nil {
		return nil, ErrASTHandle
	}
	if _, e := m.ProofsEnabled(); e != nil {
		return nil, e
	}
	c := &tacticFilterModelConverter{manager: m}
	if e := c.init(c, m.budget, uint64(reflect.TypeOf(*c).Size())); e != nil {
		return nil, e
	}
	return c, nil
}

func (c *tacticFilterModelConverter) Insert(d ASTHandle) error {
	if e := c.open(); e != nil {
		return e
	}
	kind, e := c.manager.GetKind(d)
	if e != nil {
		return e
	}
	if kind != ASTFuncDecl {
		return ErrASTDomain
	}
	// ref_vector's push_back increments before the pointer vector grows.
	if e = c.manager.IncRef(d); e != nil {
		return e
	}
	if e = c.declarations.push(c.manager, d); e != nil {
		return errors.Join(e, c.manager.DecRef(d))
	}
	return nil
}

func (c *tacticFilterModelConverter) ApplyModel(out *tacticModelRef, _ uint32) (err error) {
	if err = c.open(); err != nil {
		return err
	}
	old, e := out.Get()
	if e != nil {
		return e
	}
	if old == nil || old.manager != c.manager || old.derived == nil {
		return ErrASTHandle
	}
	m := c.manager
	marks, e := NewASTFastMark1(m)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, marks.Close()) }()
	for _, d := range c.declarations.values {
		if e = marks.Mark(d, true); e != nil {
			return e
		}
	}
	next, e := newSolverModel(m)
	if e != nil {
		return e
	}
	// The source raw new_model is published only after copying. Explicitly
	// retire the reached new owner on a Go error before that publication.
	published := false
	defer func() {
		if !published {
			err = errors.Join(err, next.Close())
		}
	}()
	n, e := old.NumConstants()
	if e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		d, e := old.Constant(i)
		if e != nil {
			return e
		}
		marked, e := marks.IsMarked(d)
		if e != nil {
			return e
		}
		if marked {
			continue
		}
		v, e := old.ConstantInterpretation(d)
		if e != nil {
			return e
		}
		if e = next.core.RegisterConstant(d, v); e != nil {
			return e
		}
	}
	if e = old.functions.empty(); e != nil {
		return e
	}
	if len(old.functionDeclarations.values) != 0 {
		return errors.New("filter model converter: function interpretation copying remains unported")
	}
	if e = next.CopyUniverseInterpretations(old.derived); e != nil {
		return e
	}
	if e = out.Set(&next.core); e != nil {
		return e
	}
	published = true
	return nil
}

func (c *tacticFilterModelConverter) ApplyUnindexed(out *tacticModelRef) error {
	return c.ApplyModel(out, 0)
}
func (c *tacticFilterModelConverter) ApplyLabels(_ *tacticModelLabels, _ uint32) error {
	return c.open()
}
func (c *tacticFilterModelConverter) Cancel() error { return c.open() }
func (c *tacticFilterModelConverter) Close() error {
	if c == nil || c.self != &c.tacticConverterBase {
		return errTacticModelConverterOwner
	}
	if c.closed {
		return nil
	}
	// func_decl_ref_vector releases its entries in forward order.
	for i, d := range c.declarations.values {
		if !d.IsNull() {
			if e := c.manager.DecRef(d); e != nil {
				return e
			}
			c.declarations.values[i] = ASTHandle{}
		}
	}
	c.declarations.close(c.manager)
	c.release()
	return nil
}
