// Source: src/math/polynomial/linear_eq_solver.h.
package polynomial

import "github.com/glycerine/g3/pkg/util"

// Shared lifecycle state; each concrete solver owns native typed matrix/RHS slices.
type linearStorage struct {
	integer                          *util.IntegerManager
	n                                uint32
	header                           uint64
	sourceCause                      error
	sourceStage                      uint8
	flushStage                       uint8
	flushRow, flushColumn            uint32
	closing, sourceFinalized, closed bool
}
