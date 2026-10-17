// Source: src/ast/rewriter/fpa_rewriter.cpp:663-680.
package rewriter

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
)

// fpaBV2RM maps the source's three-bit numeral encoding to an FP rounding
// constant. Values beyond the five named modes take the source default.
func (r *FPARewriter) fpaBV2RM(arg ast.ASTHandle, out ast.ASTRef) (status RewriteStatus, result error) {
	s := r.manager.NumeralService()
	v, err := s.NewValue()
	if err != nil {
		return RewriteFailed, err
	}
	defer func() {
		if e := s.Del(&v); e != nil {
			result = errors.Join(result, e)
			status = RewriteFailed
		}
	}()
	var width uint32
	found, err := r.manager.BitVectorNumeralValue(arg, &v, &width)
	if err != nil || !found {
		return RewriteFailed, err
	}
	n, err := s.Uint64(&v)
	if err != nil {
		return RewriteFailed, err
	}
	kind := int32(4) // BV_RM_TO_ZERO, also the default.
	switch n {
	case 0, 1, 2, 3:
		kind = int32(n)
	}
	answer, err := r.manager.MkFamilyApp(r.family, kind, nil, nil)
	if err != nil {
		return RewriteFailed, err
	}
	return RewriteDone, r.manager.SetRef(out, answer)
}
