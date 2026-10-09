// Source: src/model/model_evaluator.cpp:337-358;
// src/util/hash.h:72-122; src/util/hashtable.h:351-403,546-577.
package model

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	"testing"
)

func arrayTableMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func arrayTableManager(t *testing.T) *ast.ASTManager {
	b := arena.NewBudget(1 << 30)
	s, e := usymbol.NewManager(usymbol.Config{Budget: b})
	arrayTableMust(t, e)
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	arrayTableMust(t, e)
	t.Cleanup(func() {
		arrayTableMust(t, q.Close())
		arrayTableMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatal("tuple fixture leak", b.Used())
		}
	})
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: b})
	arrayTableMust(t, e)
	t.Cleanup(func() { arrayTableMust(t, m.Close()) })
	return m
}
func arrayTableRow(t *testing.T, m *ast.ASTManager, n, key uint32, value bool) ast.ASTRefVector {
	t.Helper()
	v, e := m.NewRefVector()
	arrayTableMust(t, e)
	t.Cleanup(func() { arrayTableMust(t, m.DelRefVector(&v)) })
	for i := uint32(0); i < n; i++ {
		h, e := m.MkBoolVal(key&(1<<i) != 0)
		arrayTableMust(t, e)
		arrayTableMust(t, m.RefVectorPush(v, h))
	}
	h, e := m.MkBoolVal(value)
	arrayTableMust(t, e)
	arrayTableMust(t, m.RefVectorPush(v, h))
	return v
}
func TestArrayTupleOriginalCompositeHashGoldenAndReadOrder(t *testing.T) {
	golden := []uint32{0x2b17922b, 0xf08c62d3, 0xdc08ad60, 0x0b669b25, 0xae9299db, 0x148468d9, 0x010876a5, 0xeea55b85, 0xc2a07325}
	for i, want := range golden {
		n := uint32(i + 1)
		var order []uint32
		got, e := arrayCompositeHash(n, func(j uint32) (uint32, error) { order = append(order, j); return j + 1, nil })
		arrayTableMust(t, e)
		if got != want {
			t.Fatal("source composite hash", n, got, want)
		}
		for j, index := range order {
			expected := uint32(j)
			if n > 3 {
				expected = n - 1 - uint32(j)
			}
			if index != expected {
				t.Fatal("source child order", n, order)
			}
		}
	}
}
func TestArrayTupleOriginalReplacementGrowthAndLastTombstone(t *testing.T) {
	m := arrayTableManager(t)
	var table arrayTupleTable
	table.arity = 8
	arrayTableMust(t, table.resize(m, 8))
	rows := make([]ast.ASTRefVector, 256)
	for i := range rows {
		rows[i] = arrayTableRow(t, m, 8, uint32(i), true)
	}
	defer table.close(m)
	for i := 0; i < 7; i++ {
		arrayTableMust(t, table.insert(m, rows[i]))
	}
	if len(table.cells) != 8 {
		t.Fatal("early expansion")
	}
	replacement := arrayTableRow(t, m, 8, 0, false)
	arrayTableMust(t, table.insert(m, replacement))
	index, e := table.find(m, rows[0])
	arrayTableMust(t, e)
	if len(table.cells) != 16 || table.size != 7 || table.cells[index].row != replacement {
		t.Fatal("expand-before-replace source order")
	}
	for i := 7; i < 48; i++ {
		arrayTableMust(t, table.insert(m, rows[i]))
	}
	if len(table.cells) != 64 {
		t.Fatal("source growth", len(table.cells))
	}
	first := -1
	mask := len(table.cells) - 1
	for i := range table.cells {
		if table.cells[i].state == modelMapUsed && table.cells[(i+1)&mask].state == modelMapUsed && table.cells[(i+2)&mask].state == modelMapUsed {
			first = i
			break
		}
	}
	if first < 0 {
		t.Fatal("fixture has no occupied triple")
	}
	a, b := table.cells[first].row, table.cells[(first+1)&mask].row
	arrayTableMust(t, table.remove(m, a))
	arrayTableMust(t, table.remove(m, b))
	candidate := -1
	for i := 48; i < len(rows); i++ {
		h, e := table.hash(m, rows[i])
		arrayTableMust(t, e)
		if int(h)&mask == first {
			candidate = i
			break
		}
	}
	if candidate < 0 {
		t.Fatal("fixture has no colliding new tuple")
	}
	arrayTableMust(t, table.insert(m, rows[candidate]))
	if table.cells[first].state != modelMapDeleted || table.cells[(first+1)&mask].row != rows[candidate] || table.deleted != 1 {
		t.Fatal("source last tombstone not reused")
	}
}
func TestArrayTupleOriginalDeletedCleanupAndResizeRefusal(t *testing.T) {
	m := arrayTableManager(t)
	rows := make([]ast.ASTRefVector, 190)
	for i := range rows {
		rows[i] = arrayTableRow(t, m, 8, uint32(i), true)
	}
	var table arrayTupleTable
	table.arity = 8
	arrayTableMust(t, table.resize(m, 256))
	defer table.close(m)
	for _, row := range rows {
		arrayTableMust(t, table.insert(m, row))
	}
	before := &table.cells[0]
	order := make([]ast.ASTRefVector, 0, len(rows))
	for _, cell := range table.cells {
		if cell.state == modelMapUsed {
			order = append(order, cell.row)
		}
	}
	cleaned := false
	for _, row := range order {
		arrayTableMust(t, table.remove(m, row))
		if before != &table.cells[0] {
			cleaned = true
		}
	}
	if !cleaned || table.size != 0 {
		t.Fatal("source deleted-entry threshold not reached")
	}
	arrayTableMust(t, table.insert(m, rows[0]))
	baseline := m.Budget().Used()
	before = &table.cells[0]
	m.Budget().FailAfter(0)
	e := table.resize(m, 512)
	m.Budget().FailAfter(-1)
	if e != arena.ErrInjectedFailure || before != &table.cells[0] || m.Budget().Used() != baseline {
		t.Fatal("failed resize published or leaked", e)
	}
	index, e := table.find(m, rows[0])
	arrayTableMust(t, e)
	if index < 0 {
		t.Fatal("resize refusal lost tuple")
	}
}
