package subpaving

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingRationalMutableIntervalCore(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	m, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget},
		Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
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
	v, err := NewRationalInterval(m)
	must(err)
	copyOf, err := NewRationalInterval(m)
	must(err)
	lo, hi := numeral.NewRational(), numeral.NewRational()
	t.Cleanup(func() {
		for _, x := range []*RationalInterval{copyOf, v} {
			if err := x.Close(); err != nil {
				t.Error(err)
			}
		}
		for _, x := range []*numeral.Rational{&lo, &hi} {
			if err := m.Del(x); err != nil {
				t.Error(err)
			}
		}
		if err := m.CloseChecked(); err != nil {
			t.Error(err)
		}
		if budget.Used() != 0 {
			t.Error("rational interval retained budget", budget.Used())
		}
	})
	var shown bytes.Buffer
	must(v.CheckInvariant())
	must(v.Display(&shown))
	if shown.String() != "(-oo, +oo)" {
		t.Fatal("default mutable interval", shown.String())
	}
	contains, err := v.ContainsZero()
	must(err)
	if !contains {
		t.Fatal("unbounded interval excludes zero")
	}
	must(m.SetString(&lo, "-3/2"))
	must(m.SetString(&hi, "2"))
	must(v.SetFinite(&lo, &hi, true, false))
	must(copyOf.SetFrom(v))
	shown.Reset()
	must(copyOf.Display(&shown))
	if shown.String() != "(-3/2, 2]" {
		t.Fatal("finite source bounds", shown.String())
	}
	contains, err = copyOf.ContainsZero()
	must(err)
	if !contains {
		t.Fatal("crossing interval excludes zero")
	}
	if err := v.SetFinite(&hi, &lo, false, false); !errors.Is(err, interval.ErrExtendedDomain) {
		t.Fatal("reversed rational bounds", err)
	}
	if err := v.SetFinite(&lo, &lo, true, false); !errors.Is(err, interval.ErrExtendedDomain) {
		t.Fatal("open singleton rational bounds", err)
	}
	must(v.CheckInvariant())
	must(m.SetInt32(&lo, 0))
	must(m.SetInt32(&hi, 1))
	must(v.SetFinite(&lo, &hi, true, false))
	contains, err = v.ContainsZero()
	must(err)
	if contains {
		t.Fatal("open zero lower bound included")
	}
	must(v.SetFinite(&lo, &hi, false, false))
	contains, err = v.ContainsZero()
	must(err)
	if !contains {
		t.Fatal("closed zero lower bound excluded")
	}
	must(v.ResetLower())
	must(v.CheckInvariant())
	must(v.ResetUpper())
	shown.Reset()
	must(v.Display(&shown))
	if shown.String() != "(-oo, +oo)" {
		t.Fatal("reset rational interval", shown.String())
	}
	w := &shortOnceSubpavingIntervalWriter{at: 3}
	if err := v.Display(w); !errors.Is(err, io.ErrShortWrite) || w.String() != "(-oo," {
		t.Fatal("borrowed display prefix", w.String(), err)
	}
	copy := *v
	if err := copy.CheckInvariant(); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("shallow interval copy", err)
	}
	must(v.Close())
	if _, err := v.ContainsZero(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed rational interval", err)
	}
}
