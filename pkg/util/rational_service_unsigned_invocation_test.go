// Source: src/util/rational.h:48; src/util/mpq.h:712-715.
package util

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalServiceUnsignedInvocationOriginalBoundary(t *testing.T) {
	s, b := serviceNew(t)
	for _, n := range []uint32{0, math.MaxInt32, math.MaxInt32 + 1, math.MaxUint32} {
		v := serviceValue(t, s, "123456789012345678901/99887766554433221101")
		var call RationalServiceInvocation
		if n <= math.MaxInt32 {
			b.FailAfter(0)
		}
		e := s.SetUnsignedWithInvocation(&v, n, &call)
		b.FailAfter(-1)
		serviceMust(t, e)
		if v.Raw.Num.IsSmall() != (n <= math.MaxInt32) || !v.Raw.Den.IsSmall() || v.Raw.Den.Val != 1 {
			t.Fatal("unsigned source NUM/DEN shape", n, v.Raw)
		}
		got, e := s.Uint64(&v)
		serviceMust(t, e)
		if got != uint64(n) {
			t.Fatal("unsigned value", n, got)
		}
		copied := call
		if e = s.SetUnsignedWithInvocation(&v, n, &copied); !errors.Is(e, arena.ErrInvalidSlot) {
			t.Fatal("copied invocation accepted", e)
		}
		serviceMust(t, call.SourceReady())
		serviceMust(t, call.Cleanup(false))
		serviceMust(t, call.Cleanup(true))
		serviceMust(t, call.EndContext())
		serviceMust(t, s.Del(&v))
	}
}
