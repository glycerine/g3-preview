// Source: src/util/union_find.h:25-241.
package util

import "github.com/glycerine/g3/pkg/util/backtrack"

// Compatibility aliases for consumers that have not moved to the leaf.
type UnionFindContext = backtrack.UnionFindContext
type UnionFindDefaultContext = backtrack.UnionFindDefaultContext
type UnionFind = backtrack.UnionFind
type BasicUnionFind = backtrack.BasicUnionFind

var ErrUnionFindContext = backtrack.ErrUnionFindContext

func NewUnionFindDefaultContext() *UnionFindDefaultContext {
	return backtrack.NewUnionFindDefaultContext()
}

func NewUnionFind(context UnionFindContext) (*UnionFind, error) {
	return backtrack.NewUnionFind(context)
}

func NewDefaultUnionFind() *UnionFind    { return backtrack.NewDefaultUnionFind() }
func NewBasicUnionFind() *BasicUnionFind { return backtrack.NewBasicUnionFind() }
