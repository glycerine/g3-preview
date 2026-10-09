package util

import (
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
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
	if m.HeaderBytes != uint64(reflect.TypeOf(IntegerManager{}).Size()) {
		t.Fatal("integer owner missing")
	}
	measured := integerActualOwnedBytes(m)
	if b.Used() != measured {
		t.Fatal("nested storage counted incorrectly", b.Used(), measured)
	}
	m.Close()
	m.Close()
	if b.Used() != 0 || m.HeaderBytes != 0 {
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
			constructionFailureOwningCleanup(t, err, b)
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

// A long decimal result uses native text outside numeric arena accounting.
func TestIntegerPhysicalBudgetGraphAfterNativeFormatting(t *testing.T) {
	m, b := mpzPublicationOwner(t)
	var a Int
	want := "1" + strings.Repeat("0", 800)
	if e := m.SetString(&a, want); e != nil {
		t.Fatal(e)
	}
	if got := integerString(t, m, &a); got != want {
		t.Fatal("format value changed")
	}
	if len(m.Natural.text) < len(want) {
		t.Fatal("fixture did not exercise native text backing")
	}
	integerPhysicalAudit(t, m, &a)
	if got, want := integerActualOwnedBytes(m), b.Used(); got != want {
		t.Fatal("real owner graph differs", got, want)
	}
}
