package ast

// Source: src/ast/ast.cpp:1471-1479; src/ast/ast.h:1454-1490.

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func TestASTSourceFinalizeStoppedEOFProtectsAutomaticMembersAndOwnsFileUntilACK(t *testing.T) {
	sy, q, _ := astTestOwners(t)
	b := arena.NewBudget(1 << 30)
	name := uparams.LiteralParamsStringView(astTracePath(t))
	m, e := NewSourceASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b}, &name)
	astTestMust(t, e)
	f := m.TraceFile
	fd := f.stream.File
	// End only the borrowed formatter, leaving the actual owned file open.
	astTestMust(t, m.TraceStream.Close())
	e = m.SourceFinalize()
	if e == nil || m.sourceDTCause != e || !m.traceEOFReached || m.sourceDTCompleted {
		t.Fatal("actual reached EOF lifetime stop", e)
	}
	if !m.debugFreeComplete || m.DebugFreeIndices == nil || m.allocator == nil || m.families == nil {
		t.Fatal("EOF body stop destroyed older automatic members")
	}
	if m.TraceFile != f || f.Closed || f.stream.File != fd || !m.traceOwner {
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
