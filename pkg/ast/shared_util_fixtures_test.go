package ast

import (
	"testing"

	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	"os"
)

func paramsMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func catalogGraph(c *util.GlobalParamsCatalog) uint64 {
	n := c.OwnerBytes
	if c.Region != nil {
		s := c.Region.Stats()
		d := s.Descriptors
		n += s.OwnerBytes + s.PayloadBytes + s.PageHeaderBytes + s.DirectoryBytes + s.LogBytes + s.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
	}
	if c.Pending != nil {
		s := c.Pending.Stats()
		n += s.OwnerBytes + s.TableBytes + s.NamesBytes
	}
	return n
}
func gparamsGraph(g *util.GlobalParams) uint64 {
	n := g.OwnerBytes
	count := func(d *util.ParamDescrs) uint64 {
		if d == nil || d.Budget != g.Budget {
			return 0
		}
		s := d.Stats()
		return s.OwnerBytes + s.TableBytes + s.NamesBytes
	}
	n += count(g.GlobalDescriptors)
	for _, d := range g.Descriptors {
		n += count(d)
	}
	return n
}

type paramsPhysicalRationalOwner struct {
	Service *unumeral.RationalService
	Value   *unumeral.OwnedRational
}

func paramsDecFinish(t *testing.T, fn func() error) {
	t.Helper()
	for i := 0; i < 32; i++ {
		e := fn()
		if e == nil {
			return
		}
		if !errors.Is(e, arena.ErrBounds) {
			t.Fatal("actual owning cleanup", e)
		}
	}
	t.Fatal("actual owning cleanup failed to advance")
}

// Cold Go metadata reservations retain their arena category. An actual source
// malloc refusal goes through memory_manager.cpp's out_of_memory_error(101).
func numericStorageReservationRefusal(err error) bool {
	if errors.Is(err, arena.ErrInjectedFailure) {
		return true
	}
	var physical umemory.Failure
	return errors.As(err, &physical) && physical.Kind == umemory.FailureMemout && physical.ErrorCode() == 101
}
func paramsTestOwner(t *testing.T) (*util.ParamsContext, *usymbol.Manager, *unumeral.RationalService) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	sy, e := usymbol.NewManager(usymbol.Config{Budget: b, PageSize: 64, DescriptorPageSize: 2})
	if e != nil {
		t.Fatal(e)
	}
	q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	if e != nil {
		t.Fatal(e)
	}
	c, e := util.NewParamsContext(sy, q, util.ParamsConfig{Budget: b, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := c.Close(); e != nil {
			t.Error(e)
		}
		if e := q.Close(); e != nil {
			t.Error(e)
		}
		if e := sy.Close(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Errorf("retained %d", b.Used())
		}
	})
	return c, sy, q
}
func catalogTestConfig(b *arena.Budget) util.GlobalParamsCatalogConfig {
	return util.GlobalParamsCatalogConfig{Budget: b, Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}}
}
func gparamsTestConfig(b *arena.Budget) util.GlobalParamsConfig {
	return util.GlobalParamsConfig{Budget: b, DescriptorPageSize: 2, DescriptorWordsPageSize: 16}
}

func debugActionFixture(t *testing.T, text string) (*umemory.Runtime, *util.DebugContext, *util.DebugActionContext, *util.NumericCharInput, *util.MemoryStandardStreams) {
	t.Helper()
	d, r, b := debugDiagnosticFixture(t)
	inputFile, e := os.CreateTemp(".", "debug-action-input-*")
	paramsMust(t, e)
	_, e = inputFile.WriteString(text)
	paramsMust(t, e)
	_, e = inputFile.Seek(0, 0)
	paramsMust(t, e)
	out, e := os.CreateTemp(".", "debug-action-output-*")
	paramsMust(t, e)
	streams, e := util.NewMemoryStandardStreams(util.MemoryStandardStreamsConfig{Budget: b, Stdout: out, Stderr: out})
	paramsMust(t, e)
	input, e := util.NewDebuggerCharInput(streams, inputFile)
	paramsMust(t, e)
	action, e := util.NewDebugActionContext(util.DebugActionConfig{Budget: b, Streams: streams, Input: input})
	paramsMust(t, e)
	paramsMust(t, d.BindAssertionStreams(streams))
	paramsMust(t, d.BindAssertionActions(action))
	t.Cleanup(func() {
		b.FailAfter(-1)
		paramsMust(t, action.Close())
		paramsMust(t, input.Close())
		paramsMust(t, streams.Close())
		inputFile.Close()
		out.Close()
		os.Remove(inputFile.Name())
		os.Remove(out.Name())
	})
	return r, d, action, input, streams
}
func gparamsVectorGraph(s arena.VectorStats) uint64 {
	d := s.Descriptors
	return s.OwnerBytes + s.PayloadBytes + s.BlockMetadataBytes + s.DirectoryBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
}

var _ umemory.ConstructionFailure = (*unumeral.RationalService)(nil)

func debugDiagnosticFixture(t *testing.T) (*util.DebugContext, *umemory.Runtime, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b})
	paramsMust(t, e)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	paramsMust(t, e)
	d, e := util.NewDebugContext(region, b)
	paramsMust(t, e)
	t.Cleanup(func() {
		b.FailAfter(-1)
		paramsMust(t, d.Close())
		paramsMust(t, region.Close())
		paramsMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("unnamed diagnostic owner", b.Used())
		}
	})
	return d, r, b
}

const DebugSourcePrompt = "(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
