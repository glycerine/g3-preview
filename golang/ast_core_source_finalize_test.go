// Licensed under the MIT license; see LICENSE.txt.
// Source: src/ast/ast.cpp:1471-1479; src/ast/ast.h:1454-1490.
// Source author: Leonardo de Moura.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestASTSourceFinalizeStoppedEOFProtectsAutomaticMembersAndOwnsFileUntilACK(t *testing.T) {
	sy, q, _ := astTestOwners(t)
	b := arena.NewBudget(1 << 30)
	name := literalParamsStringView(astTracePath(t))
	m, e := NewSourceASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b}, &name)
	astTestMust(t, e)
	f := m.traceFile
	fd := f.stream.file
	// End only the borrowed formatter, leaving the actual owned file open.
	astTestMust(t, m.traceStream.Close())
	e = m.SourceFinalize()
	if e == nil || m.sourceDTCause != e || !m.traceEOFReached || m.sourceDTCompleted {
		t.Fatal("actual reached EOF lifetime stop", e)
	}
	if !m.debugFreeComplete || m.debugFreeIndices.StorageBytes() == 0 || m.allocator == nil || m.families == nil {
		t.Fatal("EOF body stop destroyed older automatic members")
	}
	if m.traceFile != f || f.closed || f.stream.file != fd || !m.traceOwner {
		t.Fatal("stopped source EOF consumed owned file")
	}
	if b.Used() != m.Stats().AllocatedBytes {
		t.Fatal("immediate named source-stop graph", b.Used(), m.Stats())
	}
	if again := m.SourceFinalize(); again != e {
		t.Fatal("source destructor stop was replayed", again, e)
	}
	astTestMust(t, m.Close())
	if m.sourceDTCause != e || b.Used() != 0 {
		t.Fatal("owning ACK changed source history or lost roots", b.Used())
	}
	if _, e = fd.Write([]byte("x")); e == nil {
		t.Fatal("explicit owner did not close actual fd")
	}
}
