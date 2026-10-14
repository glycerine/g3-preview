package subpaving

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRationalIntervalLiveNodeBoundsAndOwnedCopy(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	m, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	root, err := NewRationalNode(m, 1, nil)
	must(err)
	view, err := NewRationalInterval(m)
	must(err)
	owned, err := NewRationalInterval(m)
	must(err)
	lo, hi := numeral.NewRational(), numeral.NewRational()
	t.Cleanup(func() {
		for _, v := range []*RationalInterval{owned, view} {
			if err := v.Close(); err != nil {
				t.Error(err)
			}
		}
		if err := root.Close(); err != nil {
			t.Error(err)
		}
		for _, v := range []*numeral.Rational{&lo, &hi} {
			if err := m.Del(v); err != nil {
				t.Error(err)
			}
		}
		if err := m.CloseChecked(); err != nil {
			t.Error(err)
		}
		if budget.Used() != 0 {
			t.Error("retained rational node budget", budget.Used())
		}
	})
	if err := view.SetConstant(root, 1); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("out-of-range variable: %v", err)
	}
	must(view.SetConstant(root, 0))
	var shown bytes.Buffer
	must(view.Display(&shown))
	if shown.String() != "(-oo, +oo)" {
		t.Fatal("initial unbounded view", shown.String())
	}
	must(m.SetString(&lo, "-3/2"))
	must(m.SetInt32(&hi, 2))
	must(root.PushBound(0, &lo, true, true))
	must(root.PushBound(0, &hi, false, false))
	loBound, hiBound, err := root.Bounds(0)
	must(err)
	resolvedLo, resolvedHi, err := view.resolved()
	must(err)
	if resolvedLo != &loBound.value || resolvedHi != &hiBound.value {
		t.Fatal("node endpoints were copied")
	}
	shown.Reset()
	must(view.Display(&shown))
	if shown.String() != "(-3/2, 2]" {
		t.Fatal("live finite view", shown.String())
	}
	must(view.CheckInvariant())
	contains, err := view.ContainsZero()
	must(err)
	if !contains {
		t.Fatal("crossing view excludes zero")
	}
	if err := view.Reset(); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("borrowed mutation: %v", err)
	}
	must(owned.SetFrom(view))
	must(m.SetInt32(&lo, 0))
	must(root.PushBound(0, &lo, true, true))
	shown.Reset()
	must(view.Display(&shown))
	if shown.String() != "(0, 2]" {
		t.Fatal("replacement bound invisible", shown.String())
	}
	contains, err = view.ContainsZero()
	must(err)
	if contains {
		t.Fatal("open zero bound includes zero")
	}
	shown.Reset()
	must(owned.Display(&shown))
	if shown.String() != "(-3/2, 2]" {
		t.Fatal("owned copy followed node", shown.String())
	}
	child, err := NewRationalNode(m, 1, root)
	must(err)
	if err := root.Close(); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("root closed before child: %v", err)
	}
	childView, err := NewRationalInterval(m)
	must(err)
	must(childView.SetConstant(child, 0))
	must(child.PushBound(0, &hi, true, false))
	shown.Reset()
	must(childView.Display(&shown))
	if shown.String() != "[2, 2]" {
		t.Fatal("child replacement view", shown.String())
	}
	must(childView.Close())
	must(child.Close())
	must(view.SetMutable())
	must(view.SetFinite(&lo, &hi, false, false))
	if err := root.Close(); err != nil {
		t.Fatal(err)
	}
	if err := view.SetConstant(root, 0); !errors.Is(err, arena.ErrClosed) {
		t.Fatalf("closed node binding: %v", err)
	}
}
