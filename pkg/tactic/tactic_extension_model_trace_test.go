package tactic

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: extension_model_converter.cpp:57-68. The source traverses the
// definitions backward and prints each evaluated value before publishing it.
func TestExtensionModelDefinitionSourceTraceReverseOrder(t *testing.T) {
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	m, _, out, config := extensionFixture(t, tags)
	c, err := NewExtensionModel(m, config)
	satOK(t, err)
	t.Cleanup(func() { satOK(t, c.Close()) })
	a := goalTestAtom(t, m, "extension-trace-a")
	b := goalTestAtom(t, m, "extension-trace-b")
	da, err := m.GetDecl(a)
	satOK(t, err)
	db, err := m.GetDecl(b)
	satOK(t, err)
	yes, err := m.MkTrue()
	satOK(t, err)
	satOK(t, c.Insert(da, b))
	satOK(t, c.Insert(db, yes))
	satOK(t, c.ApplyModel(out, 0))
	if output.Len() != 0 {
		t.Fatal("disabled extension trace", output.String())
	}
	tags.Enable("extension_mc")
	satOK(t, c.ApplyModel(out, 0))
	if !tags.ShouldTrace("extension_mc") {
		if output.Len() != 0 {
			t.Fatal("erased extension trace", output.String())
		}
		return
	}
	record := func(name, definition string) string {
		return "-------- [extension_mc] operator() src/tactic/extension_model_converter.cpp:59 ---------\n" +
			name + " ->\n" + definition + "\n==>\ntrue\n" +
			"------------------------------------------------\n"
	}
	want := record("extension-trace-b", "true") + record("extension-trace-a", "extension-trace-b")
	if got := output.String(); got != want {
		t.Fatalf("source extension definition records: got %q, want %q", got, want)
	}
}
