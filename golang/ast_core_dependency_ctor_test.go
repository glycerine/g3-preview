// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
package g3

import "testing"

func TestASTEagerDependencyConstructorBorrowedFieldsAreReal(t *testing.T) {
	m := astTestManager(t)
	for _, owner := range []*ASTManager{m, m.format} {
		if owner.dependencies.owner != owner || owner.dependencies.allocator != owner.allocator || owner.dependencyArrays.owner != owner || owner.dependencyArrays.allocator != owner.allocator {
			t.Fatal("source empty dependency owner was not bound to actual managers/allocator")
		}
		if len(owner.dependencies.todo) != 0 || len(owner.dependencyArrays.getTmp) != 0 || len(owner.dependencyArrays.rerootTmp) != 0 {
			t.Fatal("source constructor must not invent dependency operations")
		}
	}
}
