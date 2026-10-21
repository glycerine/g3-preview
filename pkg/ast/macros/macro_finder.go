// Source: src/ast/macros/macro_finder.cpp/.h.
package macros

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/ast"
)

type Formula struct {
	Expression ast.ASTHandle
	Proof      ast.ASTHandle
	Dependency ast.ASTDependency
}

// FormulaBatch owns every expression, optional proof and dependency row.
type FormulaBatch struct {
	self    *FormulaBatch
	manager *ast.ASTManager
	rows    []Formula
	header  uint64
	closed  bool
}

func NewFormulaBatch(m *ast.ASTManager) (*FormulaBatch, error) {
	if m == nil {
		return nil, ast.ErrASTHandle
	}
	n := uint64(reflect.TypeOf(FormulaBatch{}).Size())
	if err := m.Budget().Reserve(n); err != nil {
		return nil, err
	}
	b := &FormulaBatch{manager: m, header: n}
	b.self = b
	return b, nil
}

func (b *FormulaBatch) open() error {
	if b == nil || b.self != b || b.manager == nil || b.closed {
		return ast.ErrASTHandle
	}
	return nil
}
func (b *FormulaBatch) Len() int {
	if b == nil || b.self != b || b.closed {
		return 0
	}
	return len(b.rows)
}
func (b *FormulaBatch) At(i int) (Formula, error) {
	if err := b.open(); err != nil {
		return Formula{}, err
	}
	if i < 0 || i >= len(b.rows) {
		return Formula{}, ast.ErrASTDomain
	}
	return b.rows[i], nil
}
func (b *FormulaBatch) Insert(row Formula) error {
	if err := b.open(); err != nil {
		return err
	}
	if row.Expression.IsNull() {
		return ast.ErrASTHandle
	}
	if err := b.manager.IncRef(row.Expression); err != nil {
		return err
	}
	if !row.Proof.IsNull() {
		if err := b.manager.IncRef(row.Proof); err != nil {
			_ = b.manager.DecRef(row.Expression)
			return err
		}
	}
	if !row.Dependency.IsNull() {
		if err := b.manager.DependencyInc(row.Dependency); err != nil {
			if !row.Proof.IsNull() {
				_ = b.manager.DecRef(row.Proof)
			}
			_ = b.manager.DecRef(row.Expression)
			return err
		}
	}
	b.rows = append(b.rows, row)
	return nil
}
func (b *FormulaBatch) Reset() (result error) {
	if err := b.open(); err != nil {
		return err
	}
	for i := len(b.rows) - 1; i >= 0; i-- {
		row := &b.rows[i]
		if !row.Dependency.IsNull() {
			result = errors.Join(result, b.manager.DependencyDec(row.Dependency))
		}
		if !row.Proof.IsNull() {
			result = errors.Join(result, b.manager.DecRef(row.Proof))
		}
		result = errors.Join(result, b.manager.DecRef(row.Expression))
		*row = Formula{}
	}
	b.rows = b.rows[:0]
	return result
}
func (b *FormulaBatch) Close() error {
	if b == nil || b.self != b {
		return ast.ErrASTHandle
	}
	if b.closed {
		return nil
	}
	if err := b.Reset(); err != nil {
		return err
	}
	b.manager.Budget().Release(b.header)
	b.header, b.rows, b.closed = 0, nil, true
	return nil
}

// Finder discovers macros and repeatedly expands the remaining formulas.
type Finder struct {
	manager *ast.ASTManager
	macros  *Manager
	util    *Util
}

func NewFinder(m *ast.ASTManager, macros *Manager) (*Finder, error) {
	if m == nil || macros == nil || macros.manager != m {
		return nil, ast.ErrASTHandle
	}
	if err := macros.open(); err != nil {
		return nil, err
	}
	return &Finder{manager: m, macros: macros, util: macros.Util()}, nil
}

func (f *Finder) simpleMacro(term ast.ASTHandle) (declaration ast.ASTHandle, matched bool, err error) {
	kind, err := f.manager.GetKind(term)
	if err != nil || kind != ast.ASTQuantifier {
		return declaration, false, err
	}
	forall, err := f.manager.IsForall(term)
	if err != nil || !forall {
		return declaration, false, err
	}
	body, err := f.manager.GetQuantifierBody(term)
	if err != nil {
		return declaration, false, err
	}
	n, err := f.manager.GetNumDeclarations(term)
	if err != nil {
		return declaration, false, err
	}
	head, _, yes, err := f.util.IsSimpleMacro(body, n)
	if err != nil || !yes {
		return declaration, false, err
	}
	declaration, err = f.manager.GetDecl(head)
	return declaration, err == nil, err
}

func (f *Finder) pass(input *FormulaBatch) (output *FormulaBatch, found bool, result error) {
	output, err := NewFormulaBatch(f.manager)
	if err != nil {
		return nil, false, err
	}
	fail := func(e error) (*FormulaBatch, bool, error) { return nil, false, errors.Join(e, output.Close()) }
	proofs, err := f.manager.ProofsEnabled()
	if err != nil {
		return fail(err)
	}
	for _, row := range input.rows {
		out, e := f.manager.NewRef(ast.ASTHandle{})
		if e != nil {
			return fail(e)
		}
		outProof, e := f.manager.NewRef(ast.ASTHandle{})
		if e != nil {
			_ = f.manager.DelRef(&out)
			return fail(e)
		}
		proof := row.Proof
		if !proofs {
			proof = ast.ASTHandle{}
		}
		dependency, e := f.macros.ExpandMacros(row.Expression, proof, row.Dependency, out, outProof)
		if e != nil {
			e = errors.Join(e, f.manager.DelRef(&outProof), f.manager.DelRef(&out))
			return fail(e)
		}
		expression, e := f.manager.RefGet(out)
		if e != nil {
			if !dependency.IsNull() {
				e = errors.Join(e, f.manager.DependencyDec(dependency))
			}
			e = errors.Join(e, f.manager.DelRef(&outProof), f.manager.DelRef(&out))
			return fail(e)
		}
		newProof, e := f.manager.RefGet(outProof)
		if e != nil {
			if !dependency.IsNull() {
				e = errors.Join(e, f.manager.DependencyDec(dependency))
			}
			e = errors.Join(e, f.manager.DelRef(&outProof), f.manager.DelRef(&out))
			return fail(e)
		}
		declaration, simple, e := f.simpleMacro(expression)
		if e == nil && simple {
			var inserted bool
			inserted, e = f.macros.Insert(declaration, expression, newProof, dependency)
			found = found || inserted
			if inserted {
				e = errors.Join(e, f.manager.DependencyDec(dependency), f.manager.DelRef(&outProof), f.manager.DelRef(&out))
				if e != nil {
					return fail(e)
				}
				continue
			}
		}
		if e == nil {
			e = output.Insert(Formula{Expression: expression, Proof: newProof, Dependency: dependency})
		}
		if !dependency.IsNull() {
			e = errors.Join(e, f.manager.DependencyDec(dependency))
		}
		e = errors.Join(e, f.manager.DelRef(&outProof), f.manager.DelRef(&out))
		if e != nil {
			return fail(e)
		}
	}
	return output, found, nil
}

// Apply borrows inputs and returns an owning batch. It repeats until a pass
// installs no new simple macro, matching the source fixed-point driver.
func (f *Finder) Apply(inputs []Formula) (current *FormulaBatch, result error) {
	if f == nil || f.manager == nil || f.macros == nil {
		return nil, ast.ErrASTHandle
	}
	current, err := NewFormulaBatch(f.manager)
	if err != nil {
		return nil, err
	}
	for _, input := range inputs {
		if err = current.Insert(input); err != nil {
			return nil, errors.Join(err, current.Close())
		}
	}
	for {
		next, found, err := f.pass(current)
		if err != nil {
			return nil, errors.Join(err, current.Close())
		}
		if err = current.Close(); err != nil {
			return nil, errors.Join(err, next.Close())
		}
		current = next
		if !found {
			return current, nil
		}
	}
}
