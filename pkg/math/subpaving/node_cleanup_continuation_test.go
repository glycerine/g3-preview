package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestFixedNodeCloseRetainsFailedBoundSuffix(t *testing.T) {
	m, err := numeral.NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	source, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	n, err := NewFixedNode(m, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := int64(1); i <= 3; i++ {
		if err := m.SetInt64(source, i); err != nil {
			t.Fatal(err)
		}
		if err := n.PushBound(0, source, true, false); err != nil {
			t.Fatal(err)
		}
	}
	old, middle, newest := n.owned[0], n.owned[1], n.owned[2]
	original := middle.value
	invalidCopy := *original
	middle.value = &invalidCopy
	if err := n.Close(); !errors.Is(err, numeral.ErrFixedPointOwner) {
		t.Fatal("middle deletion refusal", err)
	}
	if len(n.owned) != 2 || n.trail != middle || n.children != 0 {
		t.Fatal("released prefix or retained suffix", len(n.owned), n.trail)
	}
	if err := m.SetInt64(newest.value, 4); !errors.Is(err, numeral.ErrFixedPointClosed) {
		t.Fatal("newest owner should be released", err)
	}
	if err := m.SetInt64(old.value, 4); err != nil {
		t.Fatal("older owner must survive", err)
	}
	if _, err := n.bound(0, true); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("pending cleanup must gate views", err)
	}
	if _, err := NewFixedNode(m, 1, n); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("pending cleanup must gate child construction", err)
	}
	if err := n.Close(); !errors.Is(err, numeral.ErrFixedPointOwner) || len(n.owned) != 2 || n.trail != middle {
		t.Fatal("retry must preserve the refused suffix", err, len(n.owned))
	}
	middle.value = original
	if err := n.Close(); err != nil {
		t.Fatal(err)
	}
	if err := n.Close(); err != nil {
		t.Fatal("idempotent close", err)
	}
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatal("retained numeral", err)
	}
}

func TestFastNodeCloseRetainsFailedBoundSuffix(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	cfg := numeral.DefaultFastFloatConfig(arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: budget}})
	cfg.InitialCapacity = 4
	m, err := numeral.NewFastFloatManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	source, err := numeral.NewScopedFastFloat(m)
	if err != nil {
		t.Fatal(err)
	}
	value, err := source.Get()
	if err != nil {
		t.Fatal(err)
	}
	n, err := NewFastNode(m, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := int32(1); i <= 3; i++ {
		if err := m.SetInt32(value, i); err != nil {
			t.Fatal(err)
		}
		if err := n.PushBound(0, value, true, false); err != nil {
			t.Fatal(err)
		}
	}
	old, middle, newest := n.owned[0], n.owned[1], n.owned[2]
	original := middle.owner
	invalidCopy := *original
	middle.owner = &invalidCopy
	if err := n.Close(); !errors.Is(err, numeral.ErrScopedCopy) {
		t.Fatal("middle deletion refusal", err)
	}
	if len(n.owned) != 2 || n.trail != middle || n.children != 0 {
		t.Fatal("released prefix or retained suffix", len(n.owned), n.trail)
	}
	if _, err := newest.owner.Get(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("newest owner should be released", err)
	}
	if _, err := old.owner.Get(); err != nil {
		t.Fatal("older owner must survive", err)
	}
	if _, err := n.bound(0, true); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("pending cleanup must gate views", err)
	}
	if _, err := NewFastNode(m, 1, n); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("pending cleanup must gate child construction", err)
	}
	if err := n.Close(); !errors.Is(err, numeral.ErrScopedCopy) || len(n.owned) != 2 || n.trail != middle {
		t.Fatal("retry must preserve the refused suffix", err, len(n.owned))
	}
	middle.owner = original
	if err := n.Close(); err != nil {
		t.Fatal(err)
	}
	if err := n.Close(); err != nil {
		t.Fatal("idempotent close", err)
	}
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	if err := m.CloseChecked(); err != nil {
		t.Fatal(err)
	}
	if budget.Used() != 0 {
		t.Fatal("retained fast-float storage", budget.Used())
	}
}
