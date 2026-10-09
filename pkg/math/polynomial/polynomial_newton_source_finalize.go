// Source: src/math/polynomial/polynomial.cpp:2923-2952; src/util/vector.h.
package polynomial

import (
	"github.com/glycerine/g3/pkg/arena"
)

// SourceFinalize destroys published children before dropping native active
// storage. Failed children and genuine unjoined roots await explicit Close.
func (v *PolynomialNewtonVector) SourceFinalize() error {
	if v == nil || v.self != v {
		return ErrPolynomialOwner
	}
	if v.closed || (v.closing && !v.sourceFinalizing) {
		return arena.ErrClosed
	}
	if v.sourceCause != nil {
		return v.sourceCause
	}
	if v.sourceFinalized {
		return nil
	}
	v.sourceFinalizing = true
	if v.sourceStage == 0 {
		if e := v.flushMode(false); e != nil {
			v.sourceCause = e
			return e
		}
		v.sourceStage++
	}
	if v.sourceStage == 1 {
		v.children = nil
		v.sourceStage++
	}
	v.sourceFinalized = true
	return nil
}
