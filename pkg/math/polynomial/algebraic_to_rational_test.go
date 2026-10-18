package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"testing"
)

func TestAlgebraicToRationalZeroBasicAndPredicateConversion(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	q := c.config.Rationals
	var zero, basic Algebraic
	value := unumeral.NewRational()
	serviceMust(t, q.SetFraction(&value, 7, 3))
	serviceMust(t, c.SetRational(&basic, &value))
	defer func() { serviceMust(t, c.Del(&basic)); serviceMust(t, q.Del(&value)) }()
	root := algebraicCompareRoot(t, c, []int32{-2, 1, 1}, 0, 0, 2, 0, false)
	defer func() { serviceMust(t, c.Del(&root)) }()
	for i, a := range []*Algebraic{&zero, &basic, &root} {
		out := unumeral.NewRational()
		serviceMust(t, q.SetInt32(&out, 99))
		serviceMust(t, c.ToRational(a, &out))
		want := []string{"0", "7/3", "1"}[i]
		if got := rationalString(t, q, &out); got != want {
			t.Fatal("source rational conversion", i, got, want)
		}
		serviceMust(t, q.Del(&out))
	}
	l, e := c.location(&root)
	serviceMust(t, e)
	if l == nil || l.kind != 0 {
		t.Fatal("VERIFY predicate did not publish basic rational root")
	}
}

func TestAlgebraicToRationalEveryCopyReservationPrefix(t *testing.T) {
	success, owned := false, false
	for prefix := int64(0); prefix < 256 && !success; prefix++ {
		t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			c, _, _ := algebraicCompareFixture(t)
			q := c.config.Rationals
			z := q.Integers()
			borrowed := unumeral.Rational{Num: z.Two64, Den: unumeral.NewInt(1)}
			var input Algebraic
			serviceMust(t, c.SetRational(&input, &borrowed))
			var out [32]unumeral.Rational
			for i := range out {
				out[i] = unumeral.NewRational()
			}
			z.Budget().FailAfter(prefix)
			var body error
			copied := 0
			for i := range out {
				body = c.ToRational(&input, &out[i])
				if body != nil {
					break
				}
				copied++
			}
			z.Budget().FailAfter(-1)
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("unexpected conversion refusal", prefix, body)
			}
			if z.MatchesCopyAllocationInvocation(&c.toRationalCopy) {
				owned = true
				before, used := z.CopyAllocation, z.Budget().Used()
				if err := c.ToRational(nil, nil); !errors.Is(err, ErrAlgebraicPending) || z.CopyAllocation != before || z.Budget().Used() != used {
					t.Fatal("pending conversion adopted replacement caller", err)
				}
			}
			integers := make([]*unumeral.Int, 0, 64)
			for i := range out {
				integers = append(integers, &out[i].Num, &out[i].Den)
			}
			algebraicCellSourcePhysical(t, c, nil, integers, nil, &input)
			serviceMust(t, c.CleanupToRational())
			if z.MatchesCopyAllocationInvocation(&c.toRationalCopy) {
				t.Fatal("exact conversion copy survived cleanup")
			}
			for i := 0; i < copied; i++ {
				same, e := q.Equal(&out[i], &borrowed)
				serviceMust(t, e)
				if !same {
					t.Fatal("conversion output changed", i)
				}
			}
			for i := range out {
				serviceMust(t, q.Del(&out[i]))
			}
			algebraicCellSourcePhysical(t, c, nil, nil, nil, &input)
			serviceMust(t, c.Del(&input))
			if body == nil {
				success = true
				t.Log("all conversion reservation prefixes before success", prefix)
			}
		})
		if t.Failed() {
			return
		}
	}
	if !success || !owned {
		t.Fatal("conversion sweep missed actual child or success", owned, success)
	}
}
func TestAlgebraicToRationalRejectsIrrationalBeforeOutput(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	defer func() { serviceMust(t, c.Del(&a)) }()
	polynomialAssertionActionsFixture(t, "T")
	out := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetInt32(&out, 99))
	body := c.ToRational(&a, &out)
	if body == nil || out.Num != unumeral.NewInt(99) || out.Den != unumeral.NewInt(1) {
		t.Fatal("irrational VERIFY changed output", body, out)
	}
	serviceMust(t, c.config.Rationals.Del(&out))
}
