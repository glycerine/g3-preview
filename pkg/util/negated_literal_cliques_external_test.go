package util_test

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"testing"
)

// Compile and execute the public seam without importing either solver package.
func TestNegatedLiteralCliquesUtilityOnlyConsumer(t *testing.T) {
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	b := arena.NewBudget(1 << 20)
	r, e := util.NewRuntime(util.RuntimeConfig{Budget: b, SizeTBits: 64})
	must(e)
	g, e := util.NewNegatedLiteralCliques(r, b)
	must(e)
	out, e := util.NewRuntimeUnsignedRows(r, b)
	must(e)
	var ps util.RuntimeUintColumn
	ps.Bind(r)
	// Existing output remains in place; singleton goals are discarded.
	row, e := out.BeginRow()
	must(e)
	must(row.Push(6))
	must(out.EndRow(nil))
	must(g.AddEdge(1, 3))
	must(ps.Push(0))
	must(ps.Push(2))
	must(ps.Push(4))
	must(g.Cliques(&ps, out))
	n, e := out.RowCount()
	must(e)
	if n != 2 {
		t.Fatal(n)
	}
	for i, want := range [][]uint32{{6}, {0, 2}} {
		row, e := out.Row(uint32(i))
		must(e)
		n, e := row.Size()
		must(e)
		if int(n) != len(want) {
			t.Fatal(n, want)
		}
		for j, v := range want {
			x, e := row.Read(uint32(j))
			must(e)
			if x != v {
				t.Fatal(x, v)
			}
		}
	}
	must(out.CloseChecked())
	must(ps.Finalize())
	must(g.Close())
	must(r.Close())
	if b.Used() != 0 {
		t.Fatal("retained budget", b.Used())
	}
}
