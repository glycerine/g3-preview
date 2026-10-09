// Source: src/util/trail.h:25-357.
package util

import "github.com/glycerine/g3/pkg/util/backtrack"

// Compatibility aliases for consumers that have not moved to the leaf.
type Trail = backtrack.Trail
type TrailFunc = backtrack.TrailFunc
type TrailStack = backtrack.TrailStack

var (
	ErrTrailNil    = backtrack.ErrTrailNil
	ErrTrailScopes = backtrack.ErrTrailScopes
)

func UndoTrailStack(trails []Trail, oldSize uint32) ([]Trail, error) {
	return backtrack.UndoTrailStack(trails, oldSize)
}

func NewTrailStack() *TrailStack { return backtrack.NewTrailStack() }
