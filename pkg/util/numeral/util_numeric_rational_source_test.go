// Source: src/util/mpq.cpp:130-141; src/util/mpz.cpp:1385-1410.
package numeral

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestNumericRationalSourceInsertionPreservesActualFieldChains(t *testing.T) {
	q, e := NewRationalService(serviceConfig(arena.NewBudget(1 << 20)))
	serviceMust(t, e)
	defer func() { serviceMust(t, q.Close()) }()
	v, e := q.NewValue()
	serviceMust(t, e)
	defer q.Del(&v)
	b := arena.NewBudget(1 << 20)
	out, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: b})
	serviceMust(t, e)
	defer out.Close()
	serviceMust(t, out.SetShowPos(true))
	serviceMust(t, q.SetFractionSigned(&v, -3, 2))
	serviceMust(t, q.InsertInto(out, &v))
	s, e := out.String()
	serviceMust(t, e)
	if s != "-3/+2" {
		t.Fatal("literal signed numerator / denominator insertions", s)
	}
	serviceMust(t, out.ResetText())
	serviceMust(t, q.SetInt64(&v, 1<<40))
	serviceMust(t, q.InsertInto(out, &v))
	s, e = out.String()
	serviceMust(t, e)
	if s != "1099511627776" {
		t.Fatal("big decimal CString does not acquire numeric showpos", s)
	}
}
