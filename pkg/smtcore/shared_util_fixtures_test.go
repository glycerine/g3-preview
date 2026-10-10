package smtcore

import (
	"errors"
	"math"
	"os"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/polynomial"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func paramsTestOwner(t *testing.T) (*uparams.ParamsContext, *usymbol.Manager, *unumeral.RationalService) {
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
	c, e := uparams.NewParamsContext(sy, q, uparams.ParamsConfig{Budget: b, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
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
func paramsMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func catalogGraph(c *uparams.GlobalParamsCatalog) uint64 {
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
func gparamsGraph(g *uparams.GlobalParams) uint64 {
	n := g.OwnerBytes
	count := func(d *uparams.ParamDescrs) uint64 {
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
func gparamsVectorGraph(s arena.VectorStats) uint64 {
	d := s.Descriptors
	return s.OwnerBytes + s.PayloadBytes + s.BlockMetadataBytes + s.DirectoryBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
}
func catalogTestConfig(b *arena.Budget) uparams.GlobalParamsCatalogConfig {
	return uparams.GlobalParamsCatalogConfig{Budget: b, Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}}
}
func gparamsTestConfig(b *arena.Budget) uparams.GlobalParamsConfig {
	return uparams.GlobalParamsConfig{Budget: b, DescriptorPageSize: 2, DescriptorWordsPageSize: 16}
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

type paramsPhysicalRationalOwner struct {
	Service *unumeral.RationalService
	Value   *unumeral.OwnedRational
}
type paramsPhysicalValueOwner struct {
	Context *uparams.ParamsContext
	Value   *uparams.ParamsRef
}

// Cold graph uses the actual named owners and their typed backing capacities.
// Source allocation counters remain separate from this physical Go charge.
func paramsDecAuditColdGraph(t *testing.T, q *unumeral.RationalService, m *polynomial.UPolynomialManager, r *umemory.Runtime, bn *arena.Budget, c *uparams.ParamsContext, bc *arena.Budget, sy *usymbol.Manager, bs *arena.Budget) {
	t.Helper()
	qv := reflect.ValueOf(q).Elem()
	qm := testFieldBySourceName(qv, "manager").Elem()
	z := reflect.ValueOf(m.IntegerManager()).Elem()
	// Polynomial metadata and coefficient backings belong to the polynomial
	// manager; its borrowed integer storage is counted separately below.
	n := m.Stats().TotalBytes + r.Stats().TotalBytes + testFieldBySourceName(qv, "headerBytes").Uint() + testFieldBySourceName(qm, "headerBytes").Uint() + testFieldBySourceName(z, "headerBytes").Uint()
	a := testFieldBySourceName(z, "allocator").Elem()
	locations := testFieldBySourceName(a, "locations")
	n += uint64(a.Type().Size()) + uint64(locations.Cap())*uint64(locations.Type().Elem().Size())
	slabBytes := func(s reflect.Value) uint64 {
		pages := testFieldBySourceName(s, "pages")
		bytes := uint64(s.Type().Size()) + uint64(pages.Cap())*uint64(pages.Type().Elem().Size())
		for i := 0; i < pages.Len(); i++ {
			page := pages.Index(i)
			for _, f := range []string{"records", "meta"} {
				values := testFieldBySourceName(page, f)
				bytes += uint64(values.Len()) * uint64(values.Type().Elem().Size())
			}
		}
		return bytes
	}
	n += slabBytes(testFieldBySourceName(z, "cells").Elem())
	nat := testFieldBySourceName(z, "natural").Elem()
	n += testFieldBySourceName(nat, "headerBytes").Uint()
	if n != bn.Used() {
		t.Fatal("once-only named numeric/provider graph", n, bn.Used())
	}
	cs := c.Stats()
	cn := cs.HeaderBytes + cs.BlockBytes + cs.ReceiptBytes + cs.SourceFrameBytes + cs.SourceSetBytes + cs.ReplacementBytes + cs.PageBytes + cs.DirectoryBytes + cs.FormatBytes
	if cn != bc.Used() {
		t.Fatal("actual ParamsContext owner graph", cn, bc.Used())
	}
	s := sy.Stats()
	d := s.Region.Descriptors
	sn := s.OwnerBytes + s.Table.OwnerBytes + s.Table.TableBytes + s.Region.OwnerBytes + s.Region.PayloadBytes + s.Region.PageHeaderBytes + s.Region.DirectoryBytes + s.Region.LogBytes + s.Region.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes
	if sn != bs.Used() {
		t.Fatal("actual symbol provider graph", sn, bs.Used())
	}
}

// Public reallocs reach the source LP64 signed-overflow guarded domain. They
// do not simulate OOM, mutate private counters, or inject a deallocation error.
func paramsDecTLSNearMinimum(t *testing.T, r *umemory.Runtime) {
	t.Helper()
	a, e := r.Allocate(8)
	paramsMust(t, e)
	b, e := r.Allocate(8)
	paramsMust(t, e)
	delta := int64(-1000000)
	if e = r.Reallocate(&a, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	paramsMust(t, r.Deallocate(&a))
	s := r.State()
	if s.LocalSize != 0 || s.GlobalSize >= 0 {
		t.Fatal("first actual synchronization", s)
	}
	delta = math.MinInt64 + 1000 - s.GlobalSize + 16
	if e = r.Reallocate(&b, uint64(delta)+16-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	paramsMust(t, r.Deallocate(&b))
	if s = r.State(); s.LocalSize != 0 || s.GlobalSize != math.MinInt64+1000 {
		t.Fatal("actual source TLS lane", s)
	}
}
func numericFileFixture(t *testing.T) (*os.File, *utext.NumericTextStream, *arena.Budget) {
	t.Helper()
	f, e := os.CreateTemp(".", "numeric-file-*")
	if e != nil {
		t.Fatal(e)
	}
	b := arena.NewBudget(1 << 20)
	s, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b, BorrowedFile: f})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		b.FailAfter(-1)
		s.Close()
		f.Close()
		os.Remove(f.Name())
		if b.Used() != 0 {
			t.Error("file stream cold graph leak", b.Used())
		}
	})
	return f, s, b
}
func symbolManagerTest(t *testing.T, b *arena.Budget, direct bool) *usymbol.Manager {
	t.Helper()
	m, e := usymbol.NewManager(usymbol.Config{Budget: b, PageSize: 64, DescriptorPageSize: 2, DirectOnly: direct})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := m.Close(); e != nil {
			t.Fatal(e)
		}
	})
	return m
}
func internSymbol(t *testing.T, m *usymbol.Manager, text string) usymbol.Symbol {
	t.Helper()
	s, e := m.Intern(text)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func symbolText(t *testing.T, m *usymbol.Manager, s usymbol.Symbol) string {
	t.Helper()
	v, e := m.String(s)
	if e != nil {
		t.Fatal(e)
	}
	return v
}

var _ umemory.ConstructionFailure = (*unumeral.RationalService)(nil)
