package numeral

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestArbitraryFloatNumeralSourceRationalString(t *testing.T) {
	core, _, _ := mpfNumeralHelpersFixture(t)
	m, err := NewArbitraryFloatNumeralManager(core, 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	var x ArbitraryFloat
	for _, c := range []struct {
		n, d int32
		want string
	}{{1, 4, "1/4"}, {-3, 2, "-3/2"}, {2, 1, "2"}} {
		if err := m.SetFractionSigned(&x, c.n, c.d); err != nil {
			t.Fatal(err)
		}
		got, err := m.ToRationalString(&x)
		if err != nil || got != c.want {
			t.Fatalf("%d/%d: %q, %v", c.n, c.d, got, err)
		}
	}
	if err := core.Del(&x); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := m.ToRationalString(&x); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed string", err)
	}
}

func TestArbitraryFloatNumeralRationalStringRetryAfterConversionRefusal(t *testing.T) {
	core, _, budget := mpfNumeralHelpersFixture(t)
	m, err := NewArbitraryFloatNumeralManager(core, 17, 120)
	if err != nil {
		t.Fatal(err)
	}
	x := ArbitraryFloat{format: arbitraryFloatFormat(17, 120, false), exponent: 160, significand: NewInt(1)}
	if err := core.integer.Mul2kInPlace(&x.significand, 119); err != nil {
		t.Fatal(err)
	}
	budget.FailAfter(0)
	_, refused := m.ToRationalString(&x)
	budget.FailAfter(-1)
	if !numericPhysicalReservationRefusal(refused) {
		t.Fatal("expected real conversion allocation refusal", refused)
	}
	got, err := m.ToRationalString(&x)
	if err != nil || got == "" {
		t.Fatal("retained source local after returned failure", got, err)
	}
	serviceMust(t, core.Del(&x))
	serviceMust(t, m.Close())
}

func TestArbitraryFloatNumeralRationalStringPreservesEarlierStoppedScope(t *testing.T) {
	core, _, budget := mpfNumeralHelpersFixture(t)
	m, err := NewArbitraryFloatNumeralManager(core, 17, 120)
	serviceMust(t, err)
	x := ArbitraryFloat{format: arbitraryFloatFormat(17, 120, false), exponent: 160, significand: NewInt(1)}
	serviceMust(t, core.integer.Mul2kInPlace(&x.significand, 119))
	var earlier utext.OwnedString
	budget.FailAfter(0)
	refused := core.ToRationalString(&x, &earlier)
	budget.FailAfter(-1)
	if !numericPhysicalReservationRefusal(refused) || !core.rationalString.active {
		t.Fatal("expected retained earlier lexical scope", refused)
	}
	if _, err := m.ToRationalString(&x); !errors.Is(err, ErrNumericPending) || !core.rationalString.active {
		t.Fatal("higher-level caller consumed an earlier invocation", err)
	}
	serviceMust(t, core.CleanupRationalString())
	earlier.OwnAbort()
	got, err := m.ToRationalString(&x)
	if err != nil || got == "" {
		t.Fatal("earlier scope cleanup did not restore caller", got, err)
	}
	serviceMust(t, core.Del(&x))
	serviceMust(t, m.Close())
}

type mpfNumeralShortWriter struct{ buf bytes.Buffer }

func (w *mpfNumeralShortWriter) String() string { return w.buf.String() }

func (w *mpfNumeralShortWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return w.buf.Write(p[:1])
}

func TestArbitraryFloatNumeralSourceStringAndDisplay(t *testing.T) {
	core, _, _ := mpfNumeralHelpersFixture(t)
	m, err := NewArbitraryFloatNumeralManager(core, 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	var x ArbitraryFloat
	if err := m.SetInt32(&x, 1); err != nil {
		t.Fatal(err)
	}
	got, err := m.ToString(&x)
	if err != nil || got != "1.0 0" {
		t.Fatal("source format", got, err)
	}
	again, err := m.ToString(&x)
	if err != nil || again != got {
		t.Fatal("repeated source format", again, err)
	}
	var out bytes.Buffer
	if err := m.DisplayTo(&out, &x); err != nil || out.String() != got {
		t.Fatal("display", out.String(), err)
	}
	var short mpfNumeralShortWriter
	if err := m.DisplayTo(&short, &x); !errors.Is(err, io.ErrShortWrite) || short.String() != got[:1] {
		t.Fatal("short writer", short.String(), err)
	}
	if err := core.Del(&x); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
}

// A formatter refusal can occur after mpz has published a result and swapped
// out its only temporary. If allocating the replacement temporary then fails,
// source leaves a scratch hole and later arithmetic cannot retry on this owner.
func TestArbitraryFloatNumeralStringColdRefusalSourceScratchStop(t *testing.T) {
	sawRefusal, sawScratchHole := false, false
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		core, _, budget := mpfNumeralHelpersFixture(t)
		m, err := NewArbitraryFloatNumeralManager(core, 8, 260)
		serviceMust(t, err)
		var x ArbitraryFloat
		one := NewInt(1)
		serviceMust(t, core.SetComponents(&x, 8, 260, false, 0, &one))
		budget.FailAfter(ordinal)
		_, body := m.ToString(&x)
		budget.FailAfter(-1)
		if body != nil {
			sawRefusal = true
			if !numericPhysicalReservationRefusal(body) {
				t.Fatal("unexpected formatter refusal", ordinal, body)
			}
		}
		if core.FormatScopes != nil {
			t.Fatal("caller retained a completed text context", ordinal)
		}
		hole := core.integer.Tmp[0] == 0 || core.integer.Tmp[1] == 0
		if hole {
			sawScratchHole = true
			if _, retry := m.ToString(&x); !errors.Is(retry, arena.ErrInvalidSlot) {
				t.Fatal("missing source scratch guarded retry", ordinal, body, retry)
			}
		} else {
			got, retry := m.ToString(&x)
			if retry != nil || got == "" {
				t.Fatal("intact formatter retry", ordinal, body, got, retry)
			}
		}
		integerPhysicalAuditWithRaw(t, core.integer, nil, &x.significand, &core.rational.NTmp)
		serviceMust(t, core.Del(&x))
		serviceMust(t, m.Close())
		if hole {
			if e := core.Close(); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("source destructor did not stop at missing scratch", ordinal, e)
			}
			serviceMust(t, core.Close())
		}
	}
	if !sawRefusal || !sawScratchHole {
		t.Fatal("cold campaign missed numeric refusal or published scratch hole")
	}
}
