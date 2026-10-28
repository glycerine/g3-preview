package api

// High-level array construction over the public SMT context.

import (
	"github.com/glycerine/g3/pkg/ast"
)

// ArraySort constructs one SMT array sort with one or more index sorts. The
// range may itself be an array sort.
func (c *SMTContext) ArraySort(indexSorts []ast.ASTHandle, rangeSort ast.ASTHandle) (ast.ASTHandle, error) {
	if err := c.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	if len(indexSorts) == 0 {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	manager, err := c.command.Manager()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return manager.MkArraySort(ast.SliceArguments(indexSorts), rangeSort)
}

// NestedArraySort constructs Array I0 (Array I1 (... range)) for the supplied
// outer-to-inner index sorts.
func (c *SMTContext) NestedArraySort(indexSorts []ast.ASTHandle, rangeSort ast.ASTHandle) (ast.ASTHandle, error) {
	if err := c.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	if len(indexSorts) == 0 {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	result := rangeSort
	var err error
	for index := len(indexSorts) - 1; index >= 0; index-- {
		result, err = c.ArraySort([]ast.ASTHandle{indexSorts[index]}, result)
		if err != nil {
			return ast.ASTHandle{}, err
		}
	}
	return result, nil
}

// ConstantArray constructs a constant array of the supplied complete array
// sort. The value must have the array range sort.
func (c *SMTContext) ConstantArray(arraySort, value ast.ASTHandle) (ast.ASTHandle, error) {
	if err := c.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	manager, err := c.command.Manager()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return manager.MkConstArray(arraySort, value)
}

// NestedConstantArray constructs a right-nested constant array whose leaf is
// value. Index sorts are ordered outermost to innermost.
func (c *SMTContext) NestedConstantArray(indexSorts []ast.ASTHandle, value ast.ASTHandle) (ast.ASTHandle, error) {
	if err := c.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	if len(indexSorts) == 0 {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	manager, err := c.command.Manager()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	result := value
	for index := len(indexSorts) - 1; index >= 0; index-- {
		rangeSort, x := manager.GetSort(result)
		if x != nil {
			return ast.ASTHandle{}, x
		}
		arraySort, x := c.ArraySort([]ast.ASTHandle{indexSorts[index]}, rangeSort)
		if x != nil {
			return ast.ASTHandle{}, x
		}
		result, x = manager.MkConstArray(arraySort, result)
		if x != nil {
			return ast.ASTHandle{}, x
		}
	}
	return result, nil
}

// ArraySelect selects from a native multi-index array.
func (c *SMTContext) ArraySelect(array ast.ASTHandle, indices ...ast.ASTHandle) (ast.ASTHandle, error) {
	if err := c.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	if len(indices) == 0 {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	manager, err := c.command.Manager()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	family, err := manager.ArrayFamilyID()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	arguments := append([]ast.ASTHandle{array}, indices...)
	return manager.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, arguments)
}

// ArrayStore updates one tuple in a native multi-index array.
func (c *SMTContext) ArrayStore(array ast.ASTHandle, indices []ast.ASTHandle, value ast.ASTHandle) (ast.ASTHandle, error) {
	if err := c.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	if len(indices) == 0 {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	manager, err := c.command.Manager()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	family, err := manager.ArrayFamilyID()
	if err != nil {
		return ast.ASTHandle{}, err
	}
	arguments := make([]ast.ASTHandle, 0, len(indices)+2)
	arguments = append(arguments, array)
	arguments = append(arguments, indices...)
	arguments = append(arguments, value)
	return manager.MkFamilyApp(family, ast.ASTArrayOpStore, nil, arguments)
}

// NestedArraySelect follows a path through right-nested one-index arrays.
func (c *SMTContext) NestedArraySelect(array ast.ASTHandle, indices ...ast.ASTHandle) (ast.ASTHandle, error) {
	if err := c.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	if len(indices) == 0 {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	result := array
	var err error
	for _, index := range indices {
		result, err = c.ArraySelect(result, index)
		if err != nil {
			return ast.ASTHandle{}, err
		}
	}
	return result, nil
}

// NestedArrayStore functionally updates a leaf along a right-nested array path.
func (c *SMTContext) NestedArrayStore(array ast.ASTHandle, indices []ast.ASTHandle, value ast.ASTHandle) (ast.ASTHandle, error) {
	if err := c.open(); err != nil {
		return ast.ASTHandle{}, err
	}
	if len(indices) == 0 {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	if len(indices) == 1 {
		return c.ArrayStore(array, indices, value)
	}
	inner, err := c.ArraySelect(array, indices[0])
	if err != nil {
		return ast.ASTHandle{}, err
	}
	updated, err := c.NestedArrayStore(inner, indices[1:], value)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return c.ArrayStore(array, indices[:1], updated)
}
