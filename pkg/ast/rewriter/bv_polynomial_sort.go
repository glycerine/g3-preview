// Source: src/ast/rewriter/poly_rewriter_def.h:855-858.
package rewriter

import "github.com/glycerine/g3/pkg/ast"

// Concrete heapsort implements the std::sort output/comparator contract.
// C++ library callback/swap/failure-prefix order is not credited.
func bvPolynomialSortMonomials(m *ast.ASTManager, fid ast.FamilyID, order bool, v []ast.ASTHandle) error {
	n := len(v)
	for root := n / 2; root > 0; root-- {
		if e := bvPolynomialSiftMonomials(m, fid, order, v, root-1, n); e != nil {
			return e
		}
	}
	for end := n; end > 1; end-- {
		v[0], v[end-1] = v[end-1], v[0]
		if e := bvPolynomialSiftMonomials(m, fid, order, v, 0, end-1); e != nil {
			return e
		}
	}
	return nil
}
func bvPolynomialSiftMonomials(m *ast.ASTManager, fid ast.FamilyID, order bool, v []ast.ASTHandle, root, end int) error {
	for root < end/2 {
		child := root*2 + 1
		if child+1 < end {
			less, e := bvMonomialLess(m, fid, order, v[child], v[child+1])
			if e != nil {
				return e
			}
			if less {
				child++
			}
		}
		less, e := bvMonomialLess(m, fid, order, v[root], v[child])
		if e != nil {
			return e
		}
		if !less {
			return nil
		}
		v[root], v[child] = v[child], v[root]
		root = child
	}
	return nil
}
