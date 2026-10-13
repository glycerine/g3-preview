package polynomial

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicDisplayDecimalBasicRootAndExactRefinement(t *testing.T) {
	c, p := algebraicFixture(t)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, 3, 2))
	var basic Algebraic
	serviceMust(t, c.SetRational(&basic, &q))
	t.Cleanup(func() {
		serviceMust(t, c.Del(&basic))
		serviceMust(t, c.config.Rationals.Del(&q))
	})
	sqrtPoly := algebraicPolynomial(t, p, -2, 0, 1)
	sqrtView, err := p.wholeView(sqrtPoly)
	serviceMust(t, err)
	lo, hi := unumeral.NewBinaryInt(1), unumeral.NewBinaryInt(2)
	var root Algebraic
	serviceMust(t, c.SetRoot(&root, sqrtView, &lo, &hi, false))
	t.Cleanup(func() { serviceMust(t, c.Del(&root)) })
	exactPoly := algebraicPolynomial(t, p, -1, 1)
	exactView, err := p.wholeView(exactPoly)
	serviceMust(t, err)
	lo, hi = unumeral.NewBinaryInt(0), unumeral.NewBinaryInt(2)
	var exact Algebraic
	serviceMust(t, c.SetRoot(&exact, exactView, &lo, &hi, false))
	t.Cleanup(func() { serviceMust(t, c.Del(&exact)) })

	for _, tc := range []struct {
		name      string
		value     *Algebraic
		precision uint32
		want      string
	}{
		{"zero", &Algebraic{}, 4, "0"},
		{"basic", &basic, 4, "1.5"},
		{"root-approximation", &root, 4, "1.4142?"},
		{"exact-root", &exact, 4, "1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var before, after, out bytes.Buffer
			serviceMust(t, c.DisplayInterval(&before, tc.value))
			serviceMust(t, c.DisplayDecimal(&out, tc.value, tc.precision))
			serviceMust(t, c.DisplayInterval(&after, tc.value))
			if out.String() != tc.want || before.String() != after.String() {
				t.Fatal("decimal or stored interval", out.String(), before.String(), after.String())
			}
		})
	}
	refusal := errors.New("decimal write refused")
	for i, prefix := range []string{"", "1", "1."} {
		out := &algebraicIntervalRefusingWriter{stop: i + 1, err: refusal}
		if err := c.DisplayDecimal(out, &basic, 4); !errors.Is(err, refusal) || out.String() != prefix || out.writes != i+1 {
			t.Fatalf("basic decimal write %d: %q, %d writes, %v", i+1, out.String(), out.writes, err)
		}
		serviceMust(t, c.config.Rationals.CleanupDecimalStream())
	}
	basicShort := &algebraicIntervalRefusingWriter{stop: 3}
	if err := c.DisplayDecimal(basicShort, &basic, 4); !errors.Is(err, io.ErrShortWrite) || basicShort.String() != "1." {
		t.Fatal("basic decimal short write", basicShort.String(), err)
	}
	serviceMust(t, c.config.Rationals.CleanupDecimalStream())
	stop := &algebraicIntervalRefusingWriter{stop: 1, err: refusal}
	if err := c.DisplayDecimal(stop, &root, 4); !errors.Is(err, refusal) || stop.String() != "" {
		t.Fatal("decimal refusal", stop.String(), err)
	}
	serviceMust(t, c.config.Binary.CleanupDisplay())
	short := &algebraicIntervalRefusingWriter{stop: 1}
	if err := c.DisplayDecimal(short, &root, 4); !errors.Is(err, io.ErrShortWrite) || short.String() != "" {
		t.Fatal("decimal short write", short.String(), err)
	}
	serviceMust(t, c.config.Binary.CleanupDisplay())
	if err := c.DisplayDecimal(nil, &basic, 4); !errors.Is(err, arena.ErrInvalidSlot) {
		t.Fatal("nil writer", err)
	}
	if err := c.DisplayDecimal(&bytes.Buffer{}, nil, 4); !errors.Is(err, ErrAlgebraicOwner) {
		t.Fatal("nil algebraic", err)
	}
}
