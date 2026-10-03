package mpnum

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func TestIntegerOwnerHeaderBudgets(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	s, err := newNumericCellSlab(arena.Config{PageSize: 2, Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := b.Used(), uint64(reflect.TypeOf(numericCellSlab{}).Size()); got != want || s.Stats().OwnerBytes != want {
		t.Fatal("cell owner missing", got, want)
	}
	s.Reset()
	if b.Used() != s.Stats().OwnerBytes {
		t.Fatal("reset released live owner")
	}
	s.Close()
	s.Close()
	if b.Used() != 0 || s.Stats().OwnerBytes != 0 {
		t.Fatal("cell close")
	}
	cfg := IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}
	m, err := NewIntegerManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if m.headerBytes != uint64(reflect.TypeOf(IntegerManager{}).Size()) {
		t.Fatal("integer owner missing")
	}
	cs, ws := m.cells.Stats(), m.words.Stats()
	measured := m.headerBytes + cs.OwnerBytes + cs.RecordBytes + cs.MetadataBytes + cs.DirectoryBytes + ws.OwnerBytes + ws.PayloadBytes + ws.BlockMetadataBytes + ws.DirectoryBytes + ws.Descriptors.OwnerBytes + ws.Descriptors.RecordBytes + ws.Descriptors.MetadataBytes + ws.Descriptors.DirectoryBytes
	if b.Used() != measured {
		t.Fatal("nested storage counted incorrectly", b.Used(), measured)
	}
	m.Close()
	m.Close()
	if b.Used() != 0 || m.headerBytes != 0 {
		t.Fatal("integer close")
	}
}

func TestIntegerOwnerConstructorFailureUnwind(t *testing.T) {
	completed := false
	for fault := int64(0); fault < 64; fault++ {
		b := arena.NewBudget(1 << 20)
		b.FailAfter(fault)
		cfg := IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}
		m, err := NewIntegerManager(cfg)
		if err != nil {
			if m != nil || b.Used() != 0 {
				t.Fatal(fault, "constructor leak", b.Used())
			}
			continue
		}
		m.Close()
		if b.Used() != 0 {
			t.Fatal(fault, "close leak")
		}
		completed = true
		break
	}
	if !completed {
		t.Fatal("constructor sweep never completed")
	}
	b := arena.NewBudget(1 << 20)
	_, err := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 3, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	if err != arena.ErrConfig || b.Used() != 0 {
		t.Fatal("invalid nested config unwind", err, b.Used())
	}
}
