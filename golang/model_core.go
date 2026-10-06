// Source: src/model/model_core.h:37-87; src/model/model_core.cpp:21-66,85-109;
// src/model/model.cpp:44-50.
// Licensed under the MIT license. Source author: Leonardo de Moura (2011).
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// Constant interpretation portion of the original model_core, used by the
// Boolean SAT model path. Function interpretations and derived model universes
// remain explicit unported dependencies, not alternative evaluators.
type modelCore struct {
	self                               *modelCore
	manager                            *ASTManager
	constants                          modelConstantMap
	declarations, constantDeclarations modelDeclarationVector
	references                         uint32
	ownerBytes                         uint64
	closed                             bool
}

func newModelCore(m *ASTManager) (md *modelCore, err error) {
	if m == nil {
		return nil, ErrASTHandle
	}
	if _, err = m.ProofsEnabled(); err != nil {
		return nil, err
	}
	amount := uint64(reflect.TypeOf(modelCore{}).Size())
	if err = m.budget.Reserve(amount); err != nil {
		return nil, err
	}
	md = &modelCore{manager: m, ownerBytes: amount}
	md.self = md
	if err = md.constants.resize(m, 8); err != nil {
		err = errors.Join(err, md.Close())
		return nil, err
	}
	return md, nil
}
func (md *modelCore) open() error {
	if md == nil || md.self != md {
		return ErrASTHandle
	}
	if md.closed {
		return arena.ErrClosed
	}
	_, e := md.manager.ProofsEnabled()
	return e
}
func (md *modelCore) checkDeclaration(d ASTHandle) (uint32, error) {
	if e := md.open(); e != nil {
		return 0, e
	}
	kind, e := md.manager.GetKind(d)
	if e != nil {
		return 0, e
	}
	if kind != ASTFuncDecl {
		return 0, ErrASTDomain
	}
	return md.manager.GetHash(d)
}
func (md *modelCore) RegisterConstant(d, v ASTHandle) error {
	hash, e := md.checkDeclaration(d)
	if e != nil {
		return e
	}
	arity, e := md.manager.GetArity(d)
	if e != nil {
		return e
	}
	if arity != 0 {
		return ErrASTDomain
	}
	kind, e := md.manager.GetKind(v)
	if e != nil {
		return e
	}
	if kind != ASTApp && kind != ASTVar && kind != ASTQuantifier {
		return ErrASTDomain
	}
	i, e := md.constants.insertIfAbsent(md.manager, d, hash)
	if e != nil {
		return e
	}
	entry := &md.constants.cells[i]
	m := md.manager
	if entry.value.IsNull() {
		if e = md.declarations.push(m, d); e != nil {
			return e
		}
		if e = md.constantDeclarations.push(m, d); e != nil {
			return e
		}
		if e = m.IncRef(d); e != nil {
			return e
		}
		entry.declarationOwned = true
		if e = m.IncRef(v); e != nil {
			return e
		}
		entry.value = v
		entry.valueOwned = true
	} else {
		if e = m.IncRef(v); e != nil {
			return e
		}
		if e = m.DecRef(entry.value); e != nil {
			return errors.Join(e, m.DecRef(v))
		}
		entry.value = v
	}
	return nil
}
func (md *modelCore) ConstantInterpretation(d ASTHandle) (ASTHandle, error) {
	hash, e := md.checkDeclaration(d)
	if e != nil {
		return ASTHandle{}, e
	}
	i := md.constants.find(d, hash)
	if i < 0 {
		return ASTHandle{}, nil
	}
	return md.constants.cells[i].value, nil
}
func (md *modelCore) HasConstantInterpretation(d ASTHandle) (bool, error) {
	hash, e := md.checkDeclaration(d)
	if e != nil {
		return false, e
	}
	return md.constants.find(d, hash) >= 0, nil
}

// Source model_core::eval's constant branch assigns r even when the declaration
// is absent. The owning ASTRef adapter preserves both assignment and lifetime.
func (md *modelCore) EvalConstant(d ASTHandle, out ASTRef) (bool, error) {
	if e := md.open(); e != nil {
		return false, e
	}
	arity, e := md.manager.GetArity(d)
	if e != nil {
		return false, e
	}
	if arity != 0 {
		return false, ErrASTDomain
	}
	value, e := md.ConstantInterpretation(d)
	if e != nil {
		return false, e
	}
	if e = md.manager.SetRef(out, value); e != nil {
		return false, e
	}
	return !value.IsNull(), nil
}
func (md *modelCore) NumDeclarations() (uint32, error) {
	if e := md.open(); e != nil {
		return 0, e
	}
	return uint32(len(md.declarations.values)), nil
}
func (md *modelCore) NumConstants() (uint32, error) {
	if e := md.open(); e != nil {
		return 0, e
	}
	return uint32(len(md.constantDeclarations.values)), nil
}
func (md *modelCore) Declaration(i uint32) (ASTHandle, error) {
	if e := md.open(); e != nil {
		return ASTHandle{}, e
	}
	if uint64(i) >= uint64(len(md.declarations.values)) {
		return ASTHandle{}, arena.ErrBounds
	}
	return md.declarations.values[i], nil
}
func (md *modelCore) Constant(i uint32) (ASTHandle, error) {
	if e := md.open(); e != nil {
		return ASTHandle{}, e
	}
	if uint64(i) >= uint64(len(md.constantDeclarations.values)) {
		return ASTHandle{}, arena.ErrBounds
	}
	return md.constantDeclarations.values[i], nil
}
func (md *modelCore) UnregisterConstant(d ASTHandle) error {
	hash, e := md.checkDeclaration(d)
	if e != nil {
		return e
	}
	i := md.constants.find(d, hash)
	if i < 0 {
		return nil
	}
	entry := md.constants.cells[i]
	if entry.value.IsNull() {
		return nil
	}
	if e = md.constants.remove(md.manager, d, hash); e != nil {
		return e
	}
	md.constantDeclarations.erase(d)
	if e = md.manager.DecRef(entry.declaration); e != nil {
		return e
	}
	return md.manager.DecRef(entry.value)
}
func (md *modelCore) CopyConstantInterpretations(source *modelCore) error {
	if e := md.open(); e != nil {
		return e
	}
	if e := source.open(); e != nil {
		return e
	}
	if md.manager != source.manager {
		return ErrASTHandle
	}
	for _, entry := range source.constants.cells {
		if entry.state == modelMapUsed {
			if e := md.RegisterConstant(entry.declaration, entry.value); e != nil {
				return e
			}
		}
	}
	return nil
}
func (md *modelCore) IncRef() error {
	if e := md.open(); e != nil {
		return e
	}
	md.references++
	return nil
}
func (md *modelCore) DecRef() error {
	if e := md.open(); e != nil {
		return e
	}
	md.references--
	if md.references == 0 {
		return md.Close()
	}
	return nil
}
func (md *modelCore) Close() error {
	if md == nil || md.self != md {
		return ErrASTHandle
	}
	if md.closed {
		return nil
	}
	m := md.manager
	// Source map bucket iteration, key retirement followed by value retirement.
	for i := range md.constants.cells {
		entry := &md.constants.cells[i]
		if entry.state != modelMapUsed {
			continue
		}
		if entry.declarationOwned {
			if e := m.DecRef(entry.declaration); e != nil {
				return e
			}
			entry.declarationOwned = false
		}
		if entry.valueOwned {
			if e := m.DecRef(entry.value); e != nil {
				return e
			}
			entry.valueOwned = false
		}
	}
	md.constants.close(m)
	md.declarations.close(m)
	md.constantDeclarations.close(m)
	m.budget.Release(md.ownerBytes)
	md.ownerBytes = 0
	md.closed = true
	return nil
}
