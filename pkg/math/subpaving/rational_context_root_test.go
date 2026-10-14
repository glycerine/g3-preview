package subpaving

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRationalContextRootIntegerBoundsAndChildViews(t *testing.T) {
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
	c, err := NewRationalContext(m, &control.Reslimit{Limit: 100})
	must(err)
	x, err := c.NewVariable(true)
	must(err)
	y, err := c.NewVariable(false)
	must(err)
	root, err := c.Root()
	must(err)
	if _, err := c.NewVariable(false); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("variable after root: %v", err)
	}
	view, err := NewRationalInterval(m)
	must(err)
	childView, err := NewRationalInterval(m)
	must(err)
	value := numeral.NewRational()
	t.Cleanup(func() {
		for _, v := range []*RationalInterval{childView, view} {
			if err := v.Close(); err != nil {
				t.Error(err)
			}
		}
		if err := c.Close(); err != nil {
			t.Error(err)
		}
		if err := m.Del(&value); err != nil {
			t.Error(err)
		}
		if err := m.CloseChecked(); err != nil {
			t.Error(err)
		}
		if budget.Used() != 0 {
			t.Error("rational context retained budget", budget.Used())
		}
	})
	show := func(v *RationalInterval, want string) {
		t.Helper()
		var out bytes.Buffer
		must(v.Display(&out))
		if out.String() != want {
			t.Fatalf("bounds = %q, want %q", out.String(), want)
		}
		must(v.CheckInvariant())
	}
	must(view.SetConstant(root, x))
	must(m.SetString(&value, "3/2"))
	must(c.PublishBound(root, x, &value, true, true))
	must(m.SetString(&value, "7/2"))
	must(c.PublishBound(root, x, &value, false, true))
	show(view, "[2, 3]")
	if err := root.PushBound(x, &value, true, false); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("context node bypassed normalization: %v", err)
	}
	child, err := c.NewChild(root)
	must(err)
	must(childView.SetConstant(child, x))
	show(childView, "[2, 3]")
	must(m.SetInt32(&value, 2))
	must(c.PublishBound(child, x, &value, true, true))
	show(childView, "[3, 3]")
	show(view, "[2, 3]")
	sibling, err := c.NewChild(root)
	must(err)
	must(m.SetInt32(&value, 3))
	must(c.PublishBound(sibling, x, &value, false, true))
	must(childView.SetConstant(sibling, x))
	show(childView, "[2, 2]")
	must(view.SetConstant(root, y))
	must(m.SetString(&value, "-3/2"))
	must(c.PublishBound(root, y, &value, true, true))
	must(m.SetInt32(&value, 2))
	must(c.PublishBound(root, y, &value, false, false))
	show(view, "(-3/2, 2]")
	if err := root.Close(); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatalf("root closed with live children: %v", err)
	}
	must(c.Close())
	if err := view.CheckInvariant(); !errors.Is(err, arena.ErrClosed) {
		t.Fatalf("borrowed closed root: %v", err)
	}
}
