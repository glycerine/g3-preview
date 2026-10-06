// Licensed under the MIT license; see LICENSE.txt.
// Source: src/util/mpq.cpp:130-141; src/util/mpz.cpp:1385-1410.
// Source author: Leonardo de Moura.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestNumericRationalSourceInsertionPreservesActualFieldChains(t *testing.T) {
	_, q, _ := astTestOwners(t)
	v, e := q.NewValue()
	astTestMust(t, e)
	defer q.Del(&v)
	b := arena.NewBudget(1 << 20)
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	astTestMust(t, e)
	defer out.Close()
	astTestMust(t, out.SetShowPos(true))
	astTestMust(t, q.SetFractionSigned(&v, -3, 2))
	astTestMust(t, q.InsertInto(out, &v))
	s, e := out.String()
	astTestMust(t, e)
	if s != "-3/+2" {
		t.Fatal("literal signed numerator / denominator insertions", s)
	}
	astTestMust(t, out.ResetText())
	astTestMust(t, q.SetInt64(&v, 1<<40))
	astTestMust(t, q.InsertInto(out, &v))
	s, e = out.String()
	astTestMust(t, e)
	if s != "1099511627776" {
		t.Fatal("big decimal CString does not acquire numeric showpos", s)
	}
}
