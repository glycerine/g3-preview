// Source: src/smt/theory_fpa.h fpa_value_factory.
package protomodel

import "github.com/glycerine/g3/pkg/ast"

// FloatingPointFactory supplies canonical values for floating-point and
// rounding-mode sorts while the SMT model is assembled.
type FloatingPointFactory struct {
	manager *ast.ASTManager
	family  ast.FamilyID
}

func NewFloatingPointFactory(manager *ast.ASTManager, family ast.FamilyID) (*FloatingPointFactory, error) {
	if manager == nil || family == ast.NullFamilyID {
		return nil, ast.ErrASTDomain
	}
	return &FloatingPointFactory{manager: manager, family: family}, nil
}

func (f *FloatingPointFactory) FamilyID() ast.FamilyID { return f.family }

func (f *FloatingPointFactory) SomeValue(sort ast.ASTHandle) (ast.ASTHandle, error) {
	if f == nil || f.manager == nil {
		return ast.ASTHandle{}, ast.ErrASTHandle
	}
	if _, _, floating, err := f.manager.FloatingPointSortShape(sort); err != nil {
		return ast.ASTHandle{}, err
	} else if floating {
		return f.manager.MkFPAZeroOfSort(sort, false)
	}
	if rounding, err := f.manager.IsRoundingModeSort(sort); err != nil {
		return ast.ASTHandle{}, err
	} else if rounding {
		return f.manager.MkFamilyApp(f.family, ast.FPAOpRMTowardZero, nil, nil)
	}
	return ast.ASTHandle{}, ast.ErrASTDomain
}

func (f *FloatingPointFactory) SomeValues(sort ast.ASTHandle) (ast.ASTHandle, ast.ASTHandle, bool, error) {
	if _, _, floating, err := f.manager.FloatingPointSortShape(sort); err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, false, err
	} else if floating {
		first, err := f.manager.MkFPAZeroOfSort(sort, false)
		if err != nil {
			return ast.ASTHandle{}, ast.ASTHandle{}, false, err
		}
		second, err := f.manager.MkFPAZeroOfSort(sort, true)
		return first, second, err == nil, err
	}
	if rounding, err := f.manager.IsRoundingModeSort(sort); err != nil {
		return ast.ASTHandle{}, ast.ASTHandle{}, false, err
	} else if rounding {
		first, err := f.manager.MkFamilyApp(f.family, ast.FPAOpRMNearestEven, nil, nil)
		if err != nil {
			return ast.ASTHandle{}, ast.ASTHandle{}, false, err
		}
		second, err := f.manager.MkFamilyApp(f.family, ast.FPAOpRMTowardZero, nil, nil)
		return first, second, err == nil, err
	}
	return ast.ASTHandle{}, ast.ASTHandle{}, false, ast.ErrASTDomain
}

// The source factory does not synthesize fresh FPA values.
func (*FloatingPointFactory) FreshValue(ast.ASTHandle) (ast.ASTHandle, error) {
	return ast.ASTHandle{}, nil
}

func (*FloatingPointFactory) RegisterValue(ast.ASTHandle) error { return nil }
func (f *FloatingPointFactory) Close() error {
	if f != nil {
		f.manager = nil
	}
	return nil
}
