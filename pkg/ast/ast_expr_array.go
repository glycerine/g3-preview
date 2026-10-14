// Source: src/util/parray.h and the concrete ast_manager::expr_array
// specialization in src/ast/ast.h:1428-1430,1916-1930.
package ast

import "github.com/glycerine/g3/pkg/arena"

// ASTExprArray is the AST expression-array owner. Independent Go slices replace
// parray's cell trails; each slot owns one AST reference through ASTRef.
// The manager is borrowed and must outlive the array.
type ASTExprArray struct {
	self    *ASTExprArray
	manager *ASTManager
	values  []ASTRef
	closed  bool
}

func NewASTExprArray(m *ASTManager) (*ASTExprArray, error) {
	if m == nil {
		return nil, ErrASTHandle
	}
	if err := m.lock(); err != nil {
		return nil, err
	}
	m.mu.Unlock()
	a := &ASTExprArray{manager: m}
	a.self = a
	return a, nil
}

func (a *ASTExprArray) open() error {
	if a == nil || a.self != a || a.manager == nil {
		return ErrASTHandle
	}
	if a.closed {
		return arena.ErrClosed
	}
	if err := a.manager.lock(); err != nil {
		return err
	}
	a.manager.mu.Unlock()
	return nil
}

func (a *ASTExprArray) checkExpr(h ASTHandle) error {
	if h.IsNull() {
		return nil
	}
	kind, err := a.manager.GetKind(h)
	if err != nil {
		return err
	}
	if kind != ASTApp && kind != ASTVar && kind != ASTQuantifier {
		return ErrASTDomain
	}
	return nil
}

// Clone makes an independent native slice and retains each reached value.
// A partial result is returned on a failed retain so its prefix can be closed.
func (a *ASTExprArray) Clone() (*ASTExprArray, error) {
	if err := a.open(); err != nil {
		return nil, err
	}
	b := &ASTExprArray{manager: a.manager}
	b.self = b
	for _, value := range a.values {
		copyRef, err := a.manager.CopyRef(value)
		if err != nil {
			return b, err
		}
		b.values = append(b.values, copyRef)
	}
	return b, nil
}

func (a *ASTExprArray) Len() (uint32, error) {
	if err := a.open(); err != nil {
		return 0, err
	}
	return uint32(len(a.values)), nil
}

func (a *ASTExprArray) Get(i uint32) (ASTHandle, error) {
	if err := a.open(); err != nil {
		return ASTHandle{}, err
	}
	if uint64(i) >= uint64(len(a.values)) {
		return ASTHandle{}, ErrASTDomain
	}
	return a.manager.RefGet(a.values[i])
}

func (a *ASTExprArray) Set(i uint32, h ASTHandle) error {
	if err := a.open(); err != nil {
		return err
	}
	if uint64(i) >= uint64(len(a.values)) {
		return ErrASTDomain
	}
	if err := a.checkExpr(h); err != nil {
		return err
	}
	return a.manager.SetRef(a.values[i], h)
}

func (a *ASTExprArray) Push(h ASTHandle) error {
	if err := a.open(); err != nil {
		return err
	}
	if err := a.checkExpr(h); err != nil {
		return err
	}
	if uint64(len(a.values)) >= uint64(^uint32(0)) {
		return arena.ErrBounds
	}
	ref, err := a.manager.NewRef(h)
	if err != nil {
		return err
	}
	a.values = append(a.values, ref)
	return nil
}

func (a *ASTExprArray) Pop() error {
	if err := a.open(); err != nil {
		return err
	}
	if len(a.values) == 0 {
		return ErrASTDomain
	}
	i := len(a.values) - 1
	err := a.manager.DelRef(&a.values[i])
	if a.values[i] == (ASTRef{}) {
		a.values = a.values[:i]
	}
	return err
}

// Close releases slots in source root-array order. A stopped deletion leaves
// its slot and later slots available to a matching owning retry.
func (a *ASTExprArray) Close() error {
	if a == nil || a.self != a || a.manager == nil {
		return ErrASTHandle
	}
	if a.closed {
		return nil
	}
	for i := range a.values {
		if a.values[i] == (ASTRef{}) {
			continue
		}
		if err := a.manager.DelRef(&a.values[i]); err != nil {
			return err
		}
	}
	a.values = nil
	a.closed = true
	return nil
}
