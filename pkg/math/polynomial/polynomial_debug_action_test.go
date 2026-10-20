package polynomial

import (
	"os"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Bind a source T action for fixtures that deliberately reach a failed SASSERT.
func polynomialThrowAssertionFixture(t *testing.T) {
	t.Helper()
	polynomialAssertionActionsFixture(t, "T")
}

func polynomialAssertionActionsFixture(t *testing.T, choices string) *os.File {
	t.Helper()
	if !polynomialSourceDebug {
		return nil
	}
	budget := arena.NewBudget(1 << 20)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	serviceMust(t, e)
	debug, e := diagnostic.NewDebugContext(region, budget)
	serviceMust(t, e)
	inputFile, e := os.CreateTemp(t.TempDir(), "polynomial-assert-input-*")
	serviceMust(t, e)
	_, e = inputFile.WriteString(choices)
	serviceMust(t, e)
	_, e = inputFile.Seek(0, 0)
	serviceMust(t, e)
	outputFile, e := os.CreateTemp(t.TempDir(), "polynomial-assert-output-*")
	serviceMust(t, e)
	streams, e := diagnostic.NewMemoryStandardStreams(diagnostic.MemoryStandardStreamsConfig{Budget: budget, Stdout: outputFile, Stderr: outputFile})
	serviceMust(t, e)
	input, e := diagnostic.NewDebuggerCharInput(streams, inputFile)
	serviceMust(t, e)
	actions, e := diagnostic.NewDebugActionContext(diagnostic.DebugActionConfig{Budget: budget, Streams: streams, Input: input})
	serviceMust(t, e)
	serviceMust(t, debug.BindAssertionStreams(streams))
	serviceMust(t, debug.BindAssertionActions(actions))
	previous, e := diagnostic.BindProcessDebugContext(debug)
	serviceMust(t, e)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(previous)
		serviceMust(t, e)
		serviceMust(t, actions.Close())
		serviceMust(t, input.Close())
		serviceMust(t, streams.Close())
		serviceMust(t, inputFile.Close())
		serviceMust(t, outputFile.Close())
		serviceMust(t, debug.Close())
		serviceMust(t, region.Close())
	})
	return outputFile
}
