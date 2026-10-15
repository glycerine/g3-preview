package api

import (
	"testing"

	"github.com/glycerine/g3/pkg/shell"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: src/api/api_context.cpp:408-419 and shell/main.cpp:251. The public
// API and shell option borrow one process tag table, including C-string names.
func TestAPIGlobalTraceControlsShareShellProcessTags(t *testing.T) {
	tags := diagnostic.BorrowProcessTraceTags()
	const name = "g3-api-trace-binding"
	DisableTrace(name)
	t.Cleanup(func() { DisableTrace(name) })
	if tags.IsEnabled(name) {
		t.Fatal("preexisting test trace tag")
	}
	EnableTrace(name + "\x00ignored")
	if !tags.IsEnabled(name) || !tags.IsEnabled(name+"\x00other") {
		t.Fatal("source C-string enable")
	}
	if shell.NewCommandLine(nil, nil, nil).Trace != tags {
		t.Fatal("shell and API use different process trace tables")
	}
	DisableTrace(name + "\x00ignored")
	if tags.IsEnabled(name) {
		t.Fatal("source C-string disable")
	}
}
