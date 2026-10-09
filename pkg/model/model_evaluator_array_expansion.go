// Source: src/model/model_evaluator.cpp:207-221,422-499,565-569;
// src/util/vector.h:59-98,350-356; src/util/ref_vector.h:96-100.
package model

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
)

// Actual lexical vector<expr_ref_vector>, expr_ref and current row/argument
// owners. Embedded continuation fields keep interrupted cleanup visible.
type arrayExpansion struct {
	active             bool
	stores             []ast.ASTRefVector
	drain              int
	row, copyRow, args ast.ASTRefVector
	elseCase           ast.ASTRef
	parameter          ast.ASTParameter
	parameterLive      bool
	receipt            ast.ASTRefVectorPushReceipt
	pendingVector      ast.ASTRefVector
	pendingHandle      ast.ASTHandle
	pendingPush        bool
}

func (f *arrayExpansion) push(m *ast.ASTManager, v ast.ASTRefVector, h ast.ASTHandle) error {
	if f.pendingPush {
		return util.ErrParamsPrecondition
	}
	f.receipt = ast.ASTRefVectorPushReceipt{}
	e := m.RefVectorPushTracked(v, h, &f.receipt)
	if e != nil {
		held, stateError := f.receipt.Unpublished()
		if held {
			f.pendingVector = v
			f.pendingHandle = h
			f.pendingPush = true
		}
		return errors.Join(e, stateError)
	}
	f.receipt = ast.ASTRefVectorPushReceipt{}
	return nil
}
func (f *arrayExpansion) append(m *ast.ASTManager, v ast.ASTRefVector, args ast.AppArguments) error {
	for i := uint32(0); i < args.Count(); i++ {
		h, e := args.At(i)
		if e != nil {
			return e
		}
		if e = f.push(m, v, h); e != nil {
			return e
		}
	}
	return nil
}
func (f *arrayExpansion) publishRow(m *ast.ASTManager) error {
	if e := f.appendRowCopy(m, f.row); e != nil {
		return e
	}
	return m.DelRefVector(&f.row)
}
func (f *arrayExpansion) appendRowCopy(m *ast.ASTManager, input ast.ASTRefVector) error {
	var e error
	f.copyRow, e = m.NewRefVector()
	if e != nil {
		return e
	}
	args, e := m.RewriterVectorArguments(input)
	if e != nil {
		return e
	}
	if e = f.append(m, f.copyRow, args); e != nil {
		return e
	}
	f.stores = append(f.stores, f.copyRow)
	f.copyRow = ast.ASTRefVector{}
	return nil
}
func (f *arrayExpansion) removeRow(m *ast.ASTManager, index int) error {
	for j := index + 1; j < len(f.stores); j++ {
		if e := m.RefVectorReset(f.stores[j-1]); e != nil {
			return e
		}
		args, e := m.RewriterVectorArguments(f.stores[j])
		if e != nil {
			return e
		}
		if e = f.append(m, f.stores[j-1], args); e != nil {
			return e
		}
	}
	last := len(f.stores) - 1
	if e := m.DelRefVector(&f.stores[last]); e != nil {
		return e
	}
	f.stores = f.stores[:last]
	return nil
}
func (f *arrayExpansion) closeLexical(m *ast.ASTManager) error {
	if f.pendingPush {
		if e := m.CompleteRefVectorPush(f.pendingVector, f.pendingHandle, &f.receipt); e != nil {
			return e
		}
		f.pendingPush = false
		f.receipt = ast.ASTRefVectorPushReceipt{}
	}
	if f.parameterLive {
		if e := m.DelParameter(&f.parameter); e != nil {
			return e
		}
		f.parameterLive = false
	}
	for _, v := range []*ast.ASTRefVector{&f.args, &f.copyRow, &f.row} {
		if *v != (ast.ASTRefVector{}) {
			if e := m.DelRefVector(v); e != nil {
				return e
			}
		}
	}
	return nil
}
func (f *arrayExpansion) closeElse(m *ast.ASTManager) error {
	if f.elseCase != (ast.ASTRef{}) {
		return m.DelRef(&f.elseCase)
	}
	return nil
}
func (f *arrayExpansion) closeStores(m *ast.ASTManager) error {
	for f.drain < len(f.stores) {
		if e := m.DelRefVector(&f.stores[f.drain]); e != nil {
			return e
		}
		f.drain++
	}
	*f = arrayExpansion{}
	return nil
}
func (f *arrayExpansion) close(m *ast.ASTManager) error {
	if !f.active {
		return nil
	}
	if e := f.closeLexical(m); e != nil {
		return e
	}
	if e := f.closeElse(m); e != nil {
		return e
	}
	return f.closeStores(m)
}
func (c *evaluatorConfiguration) arrayOp(h ast.ASTHandle, op int32) (bool, error) {
	m := c.owner.model.manager
	k, e := m.GetKind(h)
	if e != nil || k != ast.ASTApp {
		return false, e
	}
	fid, e := m.GetFamilyID(h)
	if e != nil || fid != c.arrayFamily {
		return false, e
	}
	kind, e := m.GetDeclKind(h)
	return kind == op, e
}
func (c *evaluatorConfiguration) arrayArgsAreValues(v ast.ASTRefVector, unique *bool) (bool, error) {
	m := c.owner.model.manager
	size, e := m.RefVectorSize(v)
	if e != nil {
		return false, e
	}
	values := true
	for j := uint32(0); values && j+1 < size; j++ {
		h, e := m.RefVectorGet(v, j)
		if e != nil {
			return false, e
		}
		values, e = m.IsValue(h)
		if e != nil {
			return false, e
		}
		one, e := m.IsUniqueValue(h)
		if e != nil {
			return false, e
		}
		*unique = *unique && one
	}
	return values, nil
}
func (c *evaluatorConfiguration) extractArray(a ast.ASTHandle, f *arrayExpansion) (bool, bool, error) {
	m := c.owner.model.manager
	values, unique := true, true
	for {
		stored, e := c.arrayOp(a, ast.ASTArrayOpStore)
		if e != nil {
			return false, unique, e
		}
		if !stored {
			break
		}
		f.row, e = m.NewRefVector()
		if e != nil {
			return false, unique, e
		}
		args, e := m.AppArgumentView(a)
		if e != nil {
			return false, unique, e
		}
		tail, e := args.Range(1, args.Count()-1)
		if e != nil {
			return false, unique, e
		}
		if e = f.append(m, f.row, tail); e != nil {
			return false, unique, e
		}
		current, e := c.arrayArgsAreValues(f.row, &unique)
		if e != nil {
			return false, unique, e
		}
		values = values && current
		if e = f.publishRow(m); e != nil {
			return false, unique, e
		}
		a, e = args.At(0)
		if e != nil {
			return false, unique, e
		}
	}
	constant, e := c.arrayOp(a, ast.ASTArrayOpConst)
	if e != nil {
		return false, unique, e
	}
	if constant {
		value, e := m.GetArg(a, 0)
		if e != nil {
			return false, unique, e
		}
		return true, unique, m.SetRef(f.elseCase, value)
	}
	backed, e := c.arrayOp(a, ast.ASTArrayOpAsArray)
	if e != nil || !backed {
		return false, unique, e
	}
	d, e := m.AsArrayFunction(a)
	if e != nil {
		return false, unique, e
	}
	g, e := c.owner.model.FunctionInterpretation(d)
	if e != nil || g == nil {
		return false, unique, e
	}
	if e = g.open(); e != nil {
		return false, unique, e
	}
	base := len(f.stores)
	for _, entry := range g.entries {
		f.row, e = m.NewRefVector()
		if e != nil {
			return false, unique, e
		}
		if e = f.append(m, f.row, ast.SliceArguments(entry.args)); e != nil {
			return false, unique, e
		}
		value, e := entry.Result()
		if e != nil {
			return false, unique, e
		}
		if e = f.push(m, f.row, value); e != nil {
			return false, unique, e
		}
		size, e := m.RefVectorSize(f.row)
		if e != nil {
			return false, unique, e
		}
		for j := uint32(0); j < size; j++ {
			h, e := m.RefVectorGet(f.row, j)
			if e != nil {
				return false, unique, e
			}
			ground, e := m.IsGround(h)
			if e != nil || !ground {
				return false, unique, e
			}
		}
		if e = f.publishRow(m); e != nil {
			return false, unique, e
		}
	}
	fallback, e := g.Else()
	if e != nil {
		return false, unique, e
	}
	if e = m.SetRef(f.elseCase, fallback); e != nil {
		return false, unique, e
	}
	if fallback.IsNull() {
		return false, unique, nil
	}
	ground, e := m.IsGround(fallback)
	if e != nil || !ground {
		return false, unique, e
	}
	for i := len(f.stores); values && i > base; {
		i--
		size, e := m.RefVectorSize(f.stores[i])
		if e != nil {
			return false, unique, e
		}
		last, e := m.RefVectorGet(f.stores[i], size-1)
		if e != nil {
			return false, unique, e
		}
		same, e := m.AreEqual(fallback, last)
		if e != nil {
			return false, unique, e
		}
		if same {
			if e = f.removeRow(m, i); e != nil {
				return false, unique, e
			}
			continue
		}
		values, e = c.arrayArgsAreValues(f.stores[i], &unique)
		if e != nil {
			return false, unique, e
		}
	}
	return true, unique, nil
}
func (c *evaluatorConfiguration) expandValue(out ast.ASTRef) (err error) {
	m := c.owner.model.manager
	value, e := m.RefGet(out)
	if e != nil {
		return e
	}
	sort, e := m.GetSort(value)
	if e != nil {
		return e
	}
	fid, e := m.GetFamilyID(sort)
	if e != nil || fid != c.arrayFamily {
		return e
	}
	f := &c.expansion
	if f.active {
		return util.ErrParamsPrecondition
	}
	f.active = true
	defer func() { err = errors.Join(err, f.close(m)) }()
	f.elseCase, e = m.NewRef(ast.ASTHandle{})
	if e != nil {
		return e
	}
	ok, _, e := c.extractArray(value, f)
	if e != nil || !ok {
		return e
	}
	fallback, e := m.RefGet(f.elseCase)
	if e != nil {
		return e
	}
	f.parameter, e = m.NewParameterAST(sort)
	if e != nil {
		return e
	}
	f.parameterLive = true
	constant, e := m.MkFamilyApp(c.arrayFamily, ast.ASTArrayOpConst, []ast.ASTParameter{f.parameter}, []ast.ASTHandle{fallback})
	if e != nil {
		return e
	}
	if e = m.DelParameter(&f.parameter); e != nil {
		return e
	}
	f.parameterLive = false
	if e = m.SetRef(out, constant); e != nil {
		return e
	}
	for i := len(f.stores); i > 0; {
		i--
		f.args, e = m.NewRefVector()
		if e != nil {
			return e
		}
		current, e := m.RefGet(out)
		if e != nil {
			return e
		}
		if e = f.push(m, f.args, current); e != nil {
			return e
		}
		row, e := m.RewriterVectorArguments(f.stores[i])
		if e != nil {
			return e
		}
		if e = f.append(m, f.args, row); e != nil {
			return e
		}
		args, e := m.RewriterVectorArguments(f.args)
		if e != nil {
			return e
		}
		stored, e := m.MkArrayAppArguments(c.arrayFamily, ast.ASTArrayOpStore, args)
		if e != nil {
			return e
		}
		if e = m.SetRef(out, stored); e != nil {
			return e
		}
		if e = m.DelRefVector(&f.args); e != nil {
			return e
		}
	}
	return nil
}
