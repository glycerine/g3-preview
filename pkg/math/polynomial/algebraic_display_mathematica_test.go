package polynomial

import (
	"errors"
	"strings"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicMathematicaRationalsRootsAndSharedCache(t *testing.T) {
	c, _, control := algebraicControlledFixture(t)
	var a Algebraic
	q := unumeral.NewRational()
	defer func() { serviceMust(t, c.Del(&a)); serviceMust(t, c.config.Rationals.Del(&q)) }()
	for _, tc := range []struct {
		n    int64
		d    uint64
		want string
	}{
		{0, 1, "0"}, {2, 1, "2"}, {-2, 3, "-2/3"}, {-8589934593, 5, "-8589934593/5"},
		{-8589934593, 8589934592, "-8589934593/8589934592"},
	} {
		serviceMust(t, c.config.Rationals.SetFraction(&q, tc.n, tc.d))
		serviceMust(t, c.SetRational(&a, &q))
		sink := &algebraicRootTextSink{}
		serviceMust(t, c.DisplayMathematicaInto(sink, &a))
		if sink.String() != tc.want {
			t.Fatal(sink.String(), tc.want)
		}
	}
	serviceMust(t, c.Del(&a))
	a = algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	sink := &algebraicRootTextSink{}
	serviceMust(t, c.DisplayMathematicaInto(sink, &a))
	if sink.String() != "Root[#1^2 - 2 &, 2]" {
		t.Fatal(sink.String())
	}
	before := control.visits
	serviceMust(t, c.DisplayRootSMT2Into(&algebraicRootTextSink{}, &a))
	if control.visits != before {
		t.Fatal("other format recomputed root index")
	}
}

func TestAlgebraicMathematicaEveryInsertionFailure(t *testing.T) {
	marker := errors.New("Mathematica stream stopped")
	for _, basic := range []bool{true, false} {
		success := false
		for fail := 1; fail < 64 && !success; fail++ {
			c, _, _ := algebraicControlledFixture(t)
			var a Algebraic
			q := unumeral.NewRational()
			if basic {
				serviceMust(t, c.config.Rationals.SetFraction(&q, -8589934593, 5))
				serviceMust(t, c.SetRational(&a, &q))
			} else {
				a = algebraicCompareRoot(t, c, []int32{-2, 0, 1}, -2, 0, -1, 0, true)
			}
			sink := &algebraicRootTextSink{fail: fail, marker: marker}
			body := c.DisplayMathematicaInto(sink, &a)
			if body == nil {
				success = true
			} else if !errors.Is(body, marker) {
				t.Fatal(body)
			}
			serviceMust(t, c.CleanupRetained())
			retry := &algebraicRootTextSink{}
			serviceMust(t, c.DisplayMathematicaInto(retry, &a))
			if !strings.HasPrefix(retry.String(), sink.String()) {
				t.Fatal("output prefix changed", sink.String(), retry.String())
			}
			if c.rootDisplay.active {
				t.Fatal("display scope survived")
			}
			serviceMust(t, c.Del(&a))
			serviceMust(t, c.config.Rationals.Del(&q))
		}
		if !success {
			t.Fatal("no successful stream endpoint")
		}
	}
}
