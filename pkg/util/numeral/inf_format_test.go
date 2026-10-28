package numeral

// Owner-accounted formatting adaptation of inf_rational / inf_int_rational.

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestInfFormatOneFinalAllocation(t *testing.T) {
	s, _ := serviceNew(t)
	v := infValue(t, s, 7, 5, "-123456789012345678901234567890/11")
	r := serviceValue(t, s, "314159265358979323846/7")
	iv, e := s.InfIntNewValue()
	serviceMust(t, e)
	serviceMust(t, s.InfIntSetParts(&iv, &r, -12345))
	want := "(7/5 -e*123456789012345678901234567890/11)"
	iwant := "(314159265358979323846/7 -e*12345)"
	for j := 0; j < 10; j++ {
		infText(t, s, &v, want)
		text, e := s.InfIntString(&iv)
		serviceMust(t, e)
		if text != iwant {
			t.Fatal(text)
		}
	}
	var text string
	var err error
	before := numericWarmProfileSnapshot(t, s.Manager.Z, s.Manager.Z.BudgetField)
	// Direct DEBUG allocation owns the lexical absolute-value copy: data and
	// initialized-byte metadata, in addition to the final output string.
	wantAllocs := float64(1)
	if SourceExtendedGCDDebug {
		wantAllocs = 3
	}
	n := testing.AllocsPerRun(50, func() { text, err = s.InfString(&v) })
	if n != wantAllocs || err != nil || text != want {
		t.Error("rational epsilon final output allocations", n, text, err)
	}
	numericWarmProfileAudit(t, s.Manager.Z, s.Manager.Z.BudgetField, before, 0, SourceExtendedGCDDebug)
	if n := testing.AllocsPerRun(50, func() { text, err = s.InfIntString(&iv) }); n != 1 || err != nil || text != iwant {
		t.Fatal("integer epsilon final output allocations", n, text, err)
	}
	serviceMust(t, s.InfDel(&v))
	serviceMust(t, s.InfIntDel(&iv))
	serviceMust(t, s.Del(&r))
}
func TestInfFormatSignedZeroAndDomain(t *testing.T) {
	s, _ := serviceNew(t)
	for _, tc := range []struct{ eps, want string }{{"0", "7/5"}, {"1", "(7/5 +e*1)"}, {"-1", "(7/5 -e*1)"}, {"2/3", "(7/5 +e*2/3)"}, {"-2/3", "(7/5 -e*2/3)"}} {
		v := infValue(t, s, 7, 5, tc.eps)
		infText(t, s, &v, tc.want)
		serviceMust(t, s.InfDel(&v))
	}
	r := serviceValue(t, s, "-7/5")
	v, e := s.InfIntNewValue()
	serviceMust(t, e)
	for _, tc := range []struct {
		eps  int32
		want string
	}{{0, "-7/5"}, {1, "(-7/5 +e*1)"}, {-1, "(-7/5 -e*1)"}, {math.MaxInt32, "(-7/5 +e*2147483647)"}, {-math.MaxInt32, "(-7/5 -e*2147483647)"}} {
		serviceMust(t, s.InfIntSetParts(&v, &r, tc.eps))
		text, e := s.InfIntString(&v)
		if e != nil || text != tc.want {
			t.Fatal(tc, text, e)
		}
	}
	serviceMust(t, s.InfIntSetParts(&v, &r, math.MinInt32))
	if _, e := s.InfIntString(&v); !errors.Is(e, ErrInfIntOverflow) {
		t.Fatal(e)
	}
	foreign, _ := serviceNew(t)
	fv, e := foreign.InfIntNewValue()
	serviceMust(t, e)
	if _, e = s.InfIntString(&fv); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal(e)
	}
	serviceMust(t, foreign.InfIntDel(&fv))
	serviceMust(t, s.InfIntDel(&v))
	serviceMust(t, s.Del(&r))
	serviceMust(t, s.Close())
	if _, e = s.InfIntString(&v); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

// Every failure is inspected before further formatting or cleanup can mask a
// leaked absolute-value temporary. Operands exceed the numerical inline
// workspace. InfString still allocates its absolute-value numeral temporary;
// InfIntString formats borrowed numerals using only native scratch.
func TestInfFormatFailurePrefixesImmediateRoots(t *testing.T) {
	for kind := 0; kind < 2; kind++ {
		failures := 0
		success := false
		for prefix := int64(0); prefix < 256; prefix++ {
			budget := arena.NewBudget(1 << 25)
			s, e := NewRationalService(serviceConfig(budget))
			serviceMust(t, e)
			first := serviceValue(t, s, strings.Repeat("1234567890", 30)+"/11")
			eps := serviceValue(t, s, "-"+strings.Repeat("9876543210", 25)+"/13")
			v, e := s.NewInfValue()
			serviceMust(t, e)
			serviceMust(t, s.InfSetPair(&v, &first, &eps))
			iv, e := s.InfIntNewValue()
			serviceMust(t, e)
			serviceMust(t, s.InfIntSetParts(&iv, &first, -12345))
			// Borrowed scalar copy is used only to enumerate existing caller roots.
			borrowed := OwnedRational{Raw: iv.First, ownerID: s.ownerID}
			before := budget.Used()
			budget.FailAfter(prefix)
			var text string
			if kind == 0 {
				text, e = s.InfString(&v)
			} else {
				text, e = s.InfIntString(&iv)
			}
			infAuditRoots(t, s, []*OwnedInfRational{&v}, []*OwnedRational{&first, &eps, &borrowed})
			if budget.Used() < before {
				t.Fatal("format released caller storage")
			}
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(kind, prefix, e)
				}
				failures++
			} else {
				success = true
				if text == "" {
					t.Fatal("empty result")
				}
			}
			budget.FailAfter(-1)
			serviceMust(t, s.InfDel(&v))
			serviceMust(t, s.InfIntDel(&iv))
			serviceMust(t, s.Del(&first))
			serviceMust(t, s.Del(&eps))
			serviceMust(t, s.Close())
			if budget.Used() != 0 {
				t.Fatal("workspace not refunded", budget.Used())
			}
			if success {
				break
			}
		}
		if !success || (kind == 0 && failures == 0) {
			t.Fatal(kind, success, failures)
		}
	}
}

func TestInfFormatNativeWorkspaceGrowth(t *testing.T) {
	s, b := serviceNew(t)
	v := infValue(t, s, 7, 5, "-2/3")
	infText(t, s, &v, "(7/5 -e*2/3)")
	n := s.Manager.Z.Natural
	before := b.Used()
	old := append([]byte(nil), n.text...)
	b.FailAfter(0)
	serviceMust(t, n.reserveText(uint32(len(old)+8192)))
	if !bytes.Equal(old, n.text[:len(old)]) || b.Used() != before {
		t.Fatal("native growth changed assembled prefix/arena budget")
	}
	b.FailAfter(-1)
	infAuditRoots(t, s, []*OwnedInfRational{&v}, nil)
	infText(t, s, &v, "(7/5 -e*2/3)")
	serviceMust(t, s.InfDel(&v))
	serviceMust(t, s.Close())
	if b.Used() != 0 || n.text != nil {
		t.Fatal("workspace graph survived Close", b.Used())
	}
}
