package diagnostic

import "testing"

func TestTraceTagsSourceControlAndFinalize(t *testing.T) {
	tags := NewTraceTags()
	if tags.IsEnabled("alpha") {
		t.Fatal("cold tags")
	}
	tags.Enable("alpha\x00ignored")
	tags.Enable("alpha")
	if !tags.IsEnabled("alpha") || tags.IsEnabled("beta") || tags.IsEnabled("alpha-x") {
		t.Fatal("C-string insertion and duplicate")
	}
	tags.EnableAll(true)
	tags.Disable("alpha")
	if !tags.IsEnabled("alpha") || !tags.IsEnabled("beta") {
		t.Fatal("all flag precedes tag lookup")
	}
	tags.EnableAll(false)
	if tags.IsEnabled("alpha") || tags.IsEnabled("beta") {
		t.Fatal("disabled tag and all flag")
	}
	tags.Enable("gamma")
	tags.EnableAll(true)
	tags.Finalize()
	if !tags.IsEnabled("gamma") || tags.tags != nil {
		t.Fatal("finalize preserves all flag but releases table")
	}
	tags.EnableAll(false)
	if tags.IsEnabled("gamma") {
		t.Fatal("finalized table retained tag")
	}
	tags.Disable("missing")
	if tags.tags == nil || len(tags.tags) != 0 {
		t.Fatal("source disable initializes empty table")
	}
}
