// Source: src/model/model_core.h:31-78; src/model/model_core.cpp:21-66,85-109;
// src/model/model.cpp:44-50.
package model

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

// Constant interpretation portion of the original model_core, used by the
// Boolean SAT model path. The derived owner supplies universe lifetimes and
// virtual final-reference destruction. Function interpretation operations
// remain an explicit unported dependency, not an alternative evaluator.
type ModelCore struct {
	self                               *ModelCore
	manager                            *ast.ASTManager
	Constants                          modelConstantMap
	Functions                          modelFunctionMap
	functionDeclarations               modelDeclarationVector
	Derived                            *Model
	declarations, ConstantDeclarations modelDeclarationVector
	References                         uint32
	ownerBytes                         uint64
	Closed                             bool
}

func NewModelCore(m *ast.ASTManager) (md *ModelCore, err error) {
	if m == nil {
		return nil, ast.ErrASTHandle
	}
	if _, err = m.ProofsEnabled(); err != nil {
		return nil, err
	}
	amount := uint64(reflect.TypeOf(ModelCore{}).Size())
	if err = m.Budget().Reserve(amount); err != nil {
		return nil, err
	}
	md = &ModelCore{manager: m, ownerBytes: amount}
	md.self = md
	if err = md.Constants.resize(m, 8); err != nil {
		err = errors.Join(err, md.Close())
		return nil, err
	}
	if err = md.Functions.initialize(m); err != nil {
		err = errors.Join(err, md.Close())
		return nil, err
	}
	return md, nil
}
func (md *ModelCore) open() error {
	if md == nil || md.self != md {
		return ast.ErrASTHandle
	}
	if md.Closed {
		return arena.ErrClosed
	}
	_, e := md.manager.ProofsEnabled()
	return e
}
func (md *ModelCore) checkDeclaration(d ast.ASTHandle) (uint32, error) {
	if e := md.open(); e != nil {
		return 0, e
	}
	kind, e := md.manager.GetKind(d)
	if e != nil {
		return 0, e
	}
	if kind != ast.ASTFuncDecl {
		return 0, ast.ErrASTDomain
	}
	return md.manager.GetHash(d)
}
func (md *ModelCore) RegisterConstant(d, v ast.ASTHandle) error {
	hash, e := md.checkDeclaration(d)
	if e != nil {
		return e
	}
	arity, e := md.manager.GetArity(d)
	if e != nil {
		return e
	}
	if arity != 0 {
		return ast.ErrASTDomain
	}
	kind, e := md.manager.GetKind(v)
	if e != nil {
		return e
	}
	if kind != ast.ASTApp && kind != ast.ASTVar && kind != ast.ASTQuantifier {
		return ast.ErrASTDomain
	}
	i, e := md.Constants.insertIfAbsent(md.manager, d, hash)
	if e != nil {
		return e
	}
	entry := &md.Constants.Cells[i]
	m := md.manager
	if entry.value.IsNull() {
		if e = md.declarations.push(m, d); e != nil {
			return e
		}
		if e = md.ConstantDeclarations.push(m, d); e != nil {
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
func (md *ModelCore) ConstantInterpretation(d ast.ASTHandle) (ast.ASTHandle, error) {
	hash, e := md.checkDeclaration(d)
	if e != nil {
		return ast.ASTHandle{}, e
	}
	i := md.Constants.find(d, hash)
	if i < 0 {
		return ast.ASTHandle{}, nil
	}
	return md.Constants.Cells[i].value, nil
}
func (md *ModelCore) HasConstantInterpretation(d ast.ASTHandle) (bool, error) {
	hash, e := md.checkDeclaration(d)
	if e != nil {
		return false, e
	}
	return md.Constants.find(d, hash) >= 0, nil
}

// Source model_core::eval's constant branch assigns r even when the declaration
// is absent. The owning ASTRef adapter preserves both assignment and lifetime.
func (md *ModelCore) EvalConstant(d ast.ASTHandle, out ast.ASTRef) (bool, error) {
	if e := md.open(); e != nil {
		return false, e
	}
	arity, e := md.manager.GetArity(d)
	if e != nil {
		return false, e
	}
	if arity != 0 {
		return false, ast.ErrASTDomain
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
func (md *ModelCore) NumDeclarations() (uint32, error) {
	if e := md.open(); e != nil {
		return 0, e
	}
	return uint32(len(md.declarations.Slice())), nil
}
func (md *ModelCore) NumConstants() (uint32, error) {
	if e := md.open(); e != nil {
		return 0, e
	}
	return uint32(len(md.ConstantDeclarations.Slice())), nil
}
func (md *ModelCore) Declaration(i uint32) (ast.ASTHandle, error) {
	if e := md.open(); e != nil {
		return ast.ASTHandle{}, e
	}
	if uint64(i) >= uint64(len(md.declarations.Slice())) {
		return ast.ASTHandle{}, arena.ErrBounds
	}
	return md.declarations.Slice()[i], nil
}
func (md *ModelCore) Constant(i uint32) (ast.ASTHandle, error) {
	if e := md.open(); e != nil {
		return ast.ASTHandle{}, e
	}
	if uint64(i) >= uint64(len(md.ConstantDeclarations.Slice())) {
		return ast.ASTHandle{}, arena.ErrBounds
	}
	return md.ConstantDeclarations.Slice()[i], nil
}
func (md *ModelCore) UnregisterConstant(d ast.ASTHandle) error {
	hash, e := md.checkDeclaration(d)
	if e != nil {
		return e
	}
	i := md.Constants.find(d, hash)
	if i < 0 {
		return nil
	}
	entry := md.Constants.Cells[i]
	if entry.value.IsNull() {
		return nil
	}
	if e = md.Constants.remove(md.manager, d, hash); e != nil {
		return e
	}
	md.ConstantDeclarations.erase(d)
	if e = md.manager.DecRef(entry.declaration); e != nil {
		return e
	}
	return md.manager.DecRef(entry.value)
}
func (md *ModelCore) CopyConstantInterpretations(source *ModelCore) error {
	if e := md.open(); e != nil {
		return e
	}
	if e := source.open(); e != nil {
		return e
	}
	if md.manager != source.manager {
		return ast.ErrASTHandle
	}
	for _, entry := range source.Constants.Cells {
		if entry.state == modelMapUsed {
			if e := md.RegisterConstant(entry.declaration, entry.value); e != nil {
				return e
			}
		}
	}
	return nil
}
func (md *ModelCore) IncRef() error {
	if e := md.open(); e != nil {
		return e
	}
	md.References++
	return nil
}
func (md *ModelCore) DecRef() error {
	if e := md.open(); e != nil {
		return e
	}
	md.References--
	if md.References == 0 {
		return md.Close()
	}
	return nil
}
func (md *ModelCore) Close() error {
	if md == nil || md.self != md {
		return ast.ErrASTHandle
	}
	if md.Derived != nil {
		return md.Derived.Close()
	}
	return md.closeCoreMembers()
}
func (md *ModelCore) closeCoreMembers() error {
	if md == nil || md.self != md {
		return ast.ErrASTHandle
	}
	if md.Closed {
		return nil
	}
	m := md.manager
	if e := md.Functions.empty(); e != nil {
		return e
	}
	// Source map bucket iteration, key retirement followed by value retirement.
	for i := range md.Constants.Cells {
		entry := &md.Constants.Cells[i]
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
	md.functionDeclarations.close(m)
	md.ConstantDeclarations.close(m)
	md.declarations.close(m)
	md.Functions.close(m)
	md.Constants.close(m)
	m.Budget().Release(md.ownerBytes)
	md.ownerBytes = 0
	md.Closed = true
	return nil
}
