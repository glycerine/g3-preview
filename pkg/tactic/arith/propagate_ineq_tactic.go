// Source: src/tactic/arith/propagate_ineqs_tactic.cpp:39-103,515-549.
package arith

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	goaltactic "github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
)

type inequalityImplementation struct {
	store   *BoundEquationStore
	builder *PolynomialBuilder
}

// PropagateIneqTactic owns one concrete implementation and a parameter copy.
// The AST manager, rational manager, parameter context and goals are borrowed.
type PropagateIneqTactic struct {
	self    *PropagateIneqTactic
	manager *ast.ASTManager
	numbers *numeral.RationalManager
	context *params.ParamsContext
	params  params.ParamsRef
	current inequalityImplementation
	retired []inequalityImplementation
	closed  bool
}

func NewPropagateIneqTactic(m *ast.ASTManager, numbers *numeral.RationalManager, context *params.ParamsContext, value *params.ParamsRef) (*PropagateIneqTactic, error) {
	if m == nil || numbers == nil {
		return nil, arena.ErrInvalidSlot
	}
	if context == nil || value == nil {
		return nil, params.ErrParamsOwner
	}
	if _, err := m.ProofsEnabled(); err != nil {
		return nil, err
	}
	owned, err := context.NewValue()
	if err != nil {
		return nil, err
	}
	if err = context.Assign(&owned, value); err != nil {
		return nil, errors.Join(err, context.Del(&owned))
	}
	t := &PropagateIneqTactic{manager: m, numbers: numbers, context: context, params: owned}
	t.self = t
	if t.current, err = t.build(); err != nil {
		return nil, errors.Join(err, t.Close())
	}
	return t, nil
}

func (t *PropagateIneqTactic) build() (inequalityImplementation, error) {
	store, err := NewBoundEquationStore(t.numbers)
	if err != nil {
		return inequalityImplementation{}, err
	}
	imp := inequalityImplementation{store: store}
	if err = store.UpdateParams(t.context, &t.params); err != nil {
		return imp, err
	}
	imp.builder, err = NewPolynomialBuilder(t.manager, store)
	if err != nil {
		return imp, err
	}
	return imp, nil
}

func (t *PropagateIneqTactic) open() error {
	if t == nil || t.self != t {
		return arena.ErrInvalidSlot
	}
	if t.closed {
		return arena.ErrClosed
	}
	if t.current.store == nil || t.current.builder == nil {
		return arena.ErrInvalidSlot
	}
	return t.current.builder.open()
}

// Apply returns a borrowed input on the source no-bound branch or transfers a
// new goal to the caller. A stopped new goal remains caller-owned for Close.
func (t *PropagateIneqTactic) Apply(input *goaltactic.Goal) (*goaltactic.Goal, error) {
	if err := t.open(); err != nil {
		return nil, err
	}
	return t.current.builder.Apply(input)
}

func (t *PropagateIneqTactic) UpdateParams(value *params.ParamsRef) error {
	if err := t.open(); err != nil {
		return err
	}
	if value == nil {
		return params.ErrParamsOwner
	}
	if err := t.context.Assign(&t.params, value); err != nil {
		return err
	}
	return t.current.store.UpdateParams(t.context, &t.params)
}

func closeInequalityImplementation(imp *inequalityImplementation) error {
	if imp.builder != nil {
		if err := imp.builder.Close(); err != nil {
			return err
		}
		imp.builder = nil
	}
	if imp.store != nil {
		if err := imp.store.Close(); err != nil {
			return err
		}
		imp.store = nil
	}
	return nil
}

// Cleanup constructs the replacement before retiring the old implementation,
// matching the source swap-then-delete order. A refused old close stays owned
// in retired until Close retries it.
func (t *PropagateIneqTactic) Cleanup() error {
	if err := t.open(); err != nil {
		return err
	}
	fresh, err := t.build()
	if err != nil {
		if closeErr := closeInequalityImplementation(&fresh); closeErr != nil {
			t.retired = append(t.retired, fresh)
			return errors.Join(err, closeErr)
		}
		return err
	}
	old := t.current
	t.current = fresh
	if err = closeInequalityImplementation(&old); err != nil {
		t.retired = append(t.retired, old)
		return err
	}
	return nil
}

func (t *PropagateIneqTactic) Close() error {
	if t == nil || t.self != t {
		return arena.ErrInvalidSlot
	}
	if t.closed {
		return nil
	}
	for i := range t.retired {
		if err := closeInequalityImplementation(&t.retired[i]); err != nil {
			return err
		}
	}
	t.retired = nil
	if err := closeInequalityImplementation(&t.current); err != nil {
		return err
	}
	if err := t.context.Del(&t.params); err != nil {
		return err
	}
	t.closed = true
	return nil
}
