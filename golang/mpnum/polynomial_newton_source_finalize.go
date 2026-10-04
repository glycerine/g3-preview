// Source: src/math/polynomial/polynomial.cpp:2923-2952; src/util/vector.h. Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// SourceFinalize executes the real vector destructor body, then the published
// pointer-vector member destructor. Pending failed children/unjoined INC roots
// and cold directory/header ownership survive for explicit Close, not source
// destructor repair. Every stopped source cause is sticky, including after a
// Runtime free physically consumed its pointer before a lane guard refused.
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
		if v.data.raw != (memory.Block{}) {
			if e := v.storage.memory.Deallocate(&v.data.raw); e != nil {
				v.sourceCause = e
				return e
			}
		}
		v.sourceStage++
	}
	v.sourceFinalized = true
	return nil
}
