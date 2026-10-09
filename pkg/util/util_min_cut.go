// Source: src/util/min_cut.h:25-59; src/util/min_cut.cpp:21-230.
package util

import "github.com/glycerine/g3/pkg/util/mincut"

// Compatibility aliases for future consumers that have not moved to the leaf.
type MinCut = mincut.MinCut

var (
	ErrMinCutNode     = mincut.ErrNode
	ErrMinCutGraph    = mincut.ErrGraph
	ErrMinCutComputed = mincut.ErrComputed
)

func NewMinCut() *MinCut { return mincut.New() }
