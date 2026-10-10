// Source: src/model/model_core.cpp:21-42,68-109;
// src/model/model_core.h:45-64.
package model

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

// Prepare cold provenance before source publication. Scalar table indices keep
// graph owner pointers out of hot cells; this directory is charged separately.
func (md *ModelCore) functionSlot() (uint32, error) {
	for i, fi := range md.functionDirectory {
		if fi == nil {
			return uint32(i + 1), nil
		}
	}
	md.functionDirectory = append(md.functionDirectory, nil)
	return uint32(len(md.functionDirectory)), nil
}

// RegisterFunction transfers graph ownership on success. Replacement destroys
// the old graph before publication; registering the identical pointer is a no-op.
func (md *ModelCore) RegisterFunction(d ast.ASTHandle, fi *FunctionInterpretation) error {
	hash, e := md.checkDeclaration(d)
	if e != nil {
		return e
	}
	if md.removedFunction != nil {
		return arena.ErrClosed
	}
	arity, e := md.manager.GetArity(d)
	if e != nil {
		return e
	}
	if arity == 0 || fi == nil {
		return ast.ErrASTDomain
	}
	if e = fi.open(); e != nil {
		return e
	}
	if fi.manager != md.manager {
		return ast.ErrASTHandle
	}
	if fi.arity != arity {
		return ast.ErrASTDomain
	}
	i, e := md.Functions.insertIfAbsent(md.manager, d, hash)
	if e != nil {
		return e
	}
	entry := &md.Functions.Cells[i]
	if entry.index == 0 {
		slot, e := md.functionSlot()
		if e != nil {
			return e
		}
		md.declarations = append(md.declarations, d)
		md.functionDeclarations = append(md.functionDeclarations, d)
		if e = md.manager.IncRef(d); e != nil {
			return e
		}
		entry.declarationOwned = true
		md.functionDirectory[slot-1] = fi
		entry.index = slot
	} else {
		old := md.functionDirectory[entry.index-1]
		if old != fi {
			if e = old.Close(); e != nil {
				return e
			}
		}
		md.functionDirectory[entry.index-1] = fi
	}
	return nil
}

// FunctionInterpretation returns a borrowed graph owned by the model.
func (md *ModelCore) FunctionInterpretation(d ast.ASTHandle) (*FunctionInterpretation, error) {
	hash, e := md.checkDeclaration(d)
	if e != nil {
		return nil, e
	}
	i := md.Functions.find(d, hash)
	if i < 0 || md.Functions.Cells[i].index == 0 {
		return nil, nil
	}
	return md.functionDirectory[md.Functions.Cells[i].index-1], nil
}
func (md *ModelCore) HasInterpretation(d ast.ASTHandle) (bool, error) {
	hash, e := md.checkDeclaration(d)
	if e != nil {
		return false, e
	}
	return md.Constants.find(d, hash) >= 0 || md.Functions.find(d, hash) >= 0, nil
}
func (md *ModelCore) Function(i uint32) (ast.ASTHandle, error) {
	if e := md.open(); e != nil {
		return ast.ASTHandle{}, e
	}
	if uint64(i) >= uint64(len(md.functionDeclarations)) {
		return ast.ASTHandle{}, arena.ErrBounds
	}
	return md.functionDeclarations[i], nil
}

// EvalDeclaration preserves the distinct source misses: absent functions leave
// the output unchanged, while present partial graphs assign their null result.
func (md *ModelCore) EvalDeclaration(d ast.ASTHandle, out ast.ASTRef) (bool, error) {
	if e := md.open(); e != nil {
		return false, e
	}
	arity, e := md.manager.GetArity(d)
	if e != nil {
		return false, e
	}
	if arity == 0 {
		return md.EvalConstant(d, out)
	}
	fi, e := md.FunctionInterpretation(d)
	if e != nil || fi == nil {
		return false, e
	}
	value, e := fi.Interpretation()
	if e != nil {
		return false, e
	}
	if e = md.manager.SetRef(out, value); e != nil {
		return false, e
	}
	return !value.IsNull(), nil
}
func (md *ModelCore) UnregisterDeclaration(d ast.ASTHandle) error {
	hash, e := md.checkDeclaration(d)
	if e != nil {
		return e
	}
	if i := md.Constants.find(d, hash); i >= 0 && !md.Constants.Cells[i].value.IsNull() {
		return md.UnregisterConstant(d)
	}
	if md.removedFunction != nil {
		return arena.ErrClosed
	}
	i := md.Functions.find(d, hash)
	if i < 0 || md.Functions.Cells[i].index == 0 {
		return nil
	}
	entry := md.Functions.Cells[i]
	md.removedFunction = md.functionDirectory[entry.index-1]
	md.removedDeclaration = entry.declaration
	md.removedDeclarationOwned = entry.declarationOwned
	md.functionDirectory[entry.index-1] = nil
	if e = md.Functions.remove(md.manager, d, hash); e != nil {
		return e
	}
	for i, h := range md.functionDeclarations {
		if h == d {
			copy(md.functionDeclarations[i:], md.functionDeclarations[i+1:])
			md.functionDeclarations[len(md.functionDeclarations)-1] = ast.ASTHandle{}
			md.functionDeclarations = md.functionDeclarations[:len(md.functionDeclarations)-1]
			break
		}
	}
	return md.finishFunctionRemoval()
}
func (md *ModelCore) finishFunctionRemoval() error {
	if md.removedDeclarationOwned {
		if e := md.manager.DecRef(md.removedDeclaration); e != nil {
			return e
		}
		md.removedDeclarationOwned = false
	}
	if md.removedFunction != nil {
		if e := md.removedFunction.Close(); e != nil {
			return e
		}
		md.removedFunction = nil
	}
	md.removedDeclaration = ast.ASTHandle{}
	return nil
}
