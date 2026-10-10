package sat

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/statistics"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func paramsMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
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
func statFixture(t *testing.T) *statistics.Statistics {
	t.Helper()
	return statistics.New()
}
func statKey(v string) statistics.Key { return statistics.Key(v) }
func pdnew(t *testing.T, m *usymbol.Manager, b *arena.Budget) *util.ParamDescrs {
	t.Helper()
	d, e := util.NewParamDescrs(m, util.ParamDescrsConfig{Budget: b, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2}}})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { d.Close() })
	return d
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
func pdinsert(t *testing.T, d *util.ParamDescrs, n string, k util.ParamKind, descr, def, mod *string) {
	t.Helper()
	if e := d.InsertName(n, k, descr, def, mod); e != nil {
		t.Fatal(e)
	}
}
func pdstr(s string) *string { return &s }

func pdtotal(s util.ParamDescrsStats) uint64 {
	return s.OwnerBytes + s.TableBytes + s.NamesBytes
}
func paramsBorrowedPhysicalBytes(s util.ParamsStats) uint64 {
	return s.HeaderBytes + s.BlockBytes + s.ReceiptBytes + s.SourceFrameBytes + s.SourceSetBytes + s.ReplacementBytes + s.PageBytes + s.DirectoryBytes + s.FormatBytes
}
