package ast

// Source: src/ast/datatype_decl_plugin.h:237-246; src/ast/datatype_decl_plugin.cpp:150-162.

import "testing"

func TestASTDatatypeRegistrationLifecycle(t *testing.T) {
	m := astTestManager(t)
	fid, e := m.RegisterDatatypePlugin()
	astTestMust(t, e)
	p := m.datatypePlugin
	if p == nil || p.self != p || p.owner != m || p.family != fid || !p.published || !p.completed {
		t.Fatal("datatype registration", p)
	}
	before := m.BudgetField.Used()
	m.BudgetField.FailAfter(0)
	again, e := m.RegisterDatatypePlugin()
	m.BudgetField.FailAfter(-1)
	astTestMust(t, e)
	if again != fid || m.datatypePlugin != p || m.BudgetField.Used() != before {
		t.Fatal("repeated registration changed owner or family")
	}
	if m.datatypePluginBytes() != p.header || p.header == 0 {
		t.Fatal("actual plugin header accounting")
	}
	astTestMust(t, m.SourceFinalize())
	if !p.membersEnded || p.header == 0 {
		t.Fatal("source finalization must end members before owning history")
	}
	astTestMust(t, m.Close())
	if p.header != 0 || m.datatypePlugin != nil {
		t.Fatal("owning cleanup retained datatype history")
	}
}
