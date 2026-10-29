package spacer

// Source: src/muz/spacer/spacer_context.h and spacer_context.cpp lemma.

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	astrewriter "github.com/glycerine/g3/pkg/ast/rewriter"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

const InfinityLevel = ^uint32(0)

type Lemma struct {
	manager  *ast.ASTManager
	context  *uparams.ParamsContext
	globals  *uparams.GlobalParamsService
	body     ast.ASTRef
	cube     []ast.ASTRef
	bindings []ast.ASTRef
	level    uint32
	closed   bool
}

func NewLemma(owner *Manager, formula ast.ASTHandle, level uint32) (_ *Lemma, resultErr error) {
	if owner == nil || owner.open() != nil || owner.contexts[0] == nil {
		return nil, ErrManagerOwner
	}
	config := owner.contexts[0].config
	body, err := normalizeFormula(owner.manager, config.Params, config.Globals, formula, true, false)
	if err != nil {
		return nil, err
	}
	return &Lemma{manager: owner.manager, context: config.Params, globals: config.Globals, body: body, level: level}, nil
}

func (l *Lemma) open() error {
	if l == nil || l.closed || l.manager == nil || l.context == nil || l.globals == nil || l.body == (ast.ASTRef{}) {
		return ErrManagerOwner
	}
	return nil
}

func (l *Lemma) Body() (ast.ASTHandle, error) {
	if err := l.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	return l.manager.RefGet(l.body)
}

func (l *Lemma) Level() (uint32, error) {
	if err := l.open(); err != nil {
		return 0, err
	}
	return l.level, nil
}

func (l *Lemma) setLevel(level uint32) error {
	if err := l.open(); err != nil {
		return err
	}
	l.level = level
	return nil
}

func (l *Lemma) makeCube() (resultErr error) {
	if len(l.cube) != 0 {
		return nil
	}
	body, err := l.Body()
	if err != nil {
		return err
	}
	kind, err := l.manager.GetKind(body)
	if err != nil {
		return err
	}
	if kind == ast.ASTQuantifier {
		return ast.ErrASTDomain
	}
	negated, err := l.manager.PushNot(body)
	if err != nil {
		return err
	}
	parts, err := l.manager.FlattenAnd([]ast.ASTHandle{negated})
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		truth, trueErr := l.manager.MkTrue()
		if trueErr != nil {
			return trueErr
		}
		parts = append(parts, truth)
	}
	l.cube, err = refsFromHandles(l.manager, parts)
	if err != nil {
		return err
	}
	if err = normalizeLess(l.manager, l.cube); err != nil {
		resultErr = errors.Join(err, releaseNormalizeRefs(l.manager, l.cube))
		l.cube = nil
		return resultErr
	}
	return nil
}

func (l *Lemma) Cube() ([]ast.ASTHandle, error) {
	if err := l.open(); err != nil {
		return nil, err
	}
	if err := l.makeCube(); err != nil {
		return nil, err
	}
	result := make([]ast.ASTHandle, len(l.cube))
	for i := range l.cube {
		handle, err := l.manager.RefGet(l.cube[i])
		if err != nil {
			return nil, err
		}
		result[i] = handle
	}
	return result, nil
}

func (l *Lemma) IsFalse() (bool, error) {
	if err := l.open(); err != nil {
		return false, err
	}
	if len(l.cube) == 1 {
		handle, err := l.manager.RefGet(l.cube[0])
		return l.manager.IsTrue(handle), err
	}
	body, err := l.Body()
	return l.manager.IsFalse(body), err
}

func (l *Lemma) AddBindings(bindings []ast.ASTHandle) error {
	if err := l.open(); err != nil {
		return err
	}
	refs, err := refsFromHandles(l.manager, bindings)
	if err != nil {
		return err
	}
	l.bindings = append(l.bindings, refs...)
	return nil
}

func (l *Lemma) bindingHandles() ([]ast.ASTHandle, error) {
	result := make([]ast.ASTHandle, len(l.bindings))
	for i := range l.bindings {
		handle, err := l.manager.RefGet(l.bindings[i])
		if err != nil {
			return nil, err
		}
		result[i] = handle
	}
	return result, nil
}

func (l *Lemma) Instances() (_ []ast.ASTRef, resultErr error) {
	body, err := l.Body()
	if err != nil {
		return nil, err
	}
	kind, err := l.manager.GetKind(body)
	if err != nil || kind != ast.ASTQuantifier || len(l.bindings) == 0 {
		return nil, err
	}
	count, err := l.manager.GetNumDeclarations(body)
	if err != nil || count == 0 || len(l.bindings)%int(count) != 0 {
		if err == nil {
			err = ast.ErrASTDomain
		}
		return nil, err
	}
	bindings, err := l.bindingHandles()
	if err != nil {
		return nil, err
	}
	var result []ast.ASTRef
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, releaseNormalizeRefs(l.manager, result))
		}
	}()
	for offset := 0; offset < len(bindings); offset += int(count) {
		instance, newErr := l.manager.NewRef(ast.ASTHandle{})
		if newErr != nil {
			return nil, newErr
		}
		if newErr = astrewriter.InstantiateQuantifierWithOrder(l.manager, body, bindings[offset:offset+int(count)], false, instance); newErr != nil {
			_ = l.manager.DelRef(&instance)
			return nil, newErr
		}
		result = append(result, instance)
	}
	return result, nil
}

func (l *Lemma) Close() (result error) {
	if l == nil || l.manager == nil {
		return ErrManagerOwner
	}
	if l.closed {
		return nil
	}
	result = errors.Join(releaseNormalizeRefs(l.manager, l.bindings), releaseNormalizeRefs(l.manager, l.cube), l.manager.DelRef(&l.body))
	if result == nil {
		l.bindings = nil
		l.cube = nil
		l.manager = nil
		l.context = nil
		l.globals = nil
		l.closed = true
	}
	return result
}
