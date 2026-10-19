// Source: algebraic_numbers.cpp:235-240,337-341,440-444; mpq.h:666-669.
// Raw input views establish readable fields, never source construction/ownership.
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func basicValueRawPrefix(t *testing.T, c *AlgebraicRootCore, parts int) Algebraic {
	t.Helper()
	a := algebraicQueryRawCell(t, c, 0)
	l, e := c.location(&a)
	serviceMust(t, e)
	for i, w := range []uint32{7, 0, 0, 3, 0, 0} {
		if i >= parts*3 {
			break
		}
		serviceMust(t, c.config.Allocator.WriteUint32(l.raw, uint64(i)*4, w))
	}
	return a
}
func TestAlgebraicToRationalBasicComponentsInSourceOrder(t *testing.T) {
	for parts := 0; parts <= 2; parts++ {
		t.Run(fmt.Sprint(parts), func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			a := basicValueRawPrefix(t, c, parts)
			out := unumeral.NewRational()
			q := c.config.Rationals
			serviceMust(t, q.SetFraction(&out, 99, 5))
			body := c.ToRational(&a, &out)
			if parts < 2 && !errors.Is(body, umemory.ErrRuntimeUninitialized) {
				t.Error("missing source field guard", body)
			}
			if parts == 2 {
				serviceMust(t, body)
			}
			wantNum, wantDen := int32(99), int32(5)
			if parts > 0 {
				wantNum = 7
			}
			if parts == 2 {
				wantDen = 3
			}
			if out.Num.Val != wantNum || out.Den.Val != wantDen {
				t.Error("numerator/denominator copy order", out, wantNum, wantDen)
			}
			serviceMust(t, c.CleanupToRational())
			serviceMust(t, q.Del(&out))
		})
	}
}
func TestAlgebraicCopyBasicViewDeclaresTemporaryBeforePayload(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		for parts := 0; parts < 2; parts++ {
			for dk := 0; dk < 3; dk++ {
				t.Run(fmt.Sprintf("foreign=%v/parts=%d/dst=%d", foreign, parts, dk), func(t *testing.T) {
					target, _, _ := algebraicCompareFixtureProfile(t, true)
					source := target
					if foreign {
						source, _, _ = algebraicCompareFixtureProfile(t, true)
					}
					src := basicValueRawPrefix(t, source, parts)
					dst := algebraicForeignValue(t, target, dk)
					old := dst
					beforeInvocation := target.constQ.invocation
					body := target.CopyFromCore(&dst, source, &src)
					if !errors.Is(body, umemory.ErrRuntimeUninitialized) {
						t.Error("missing source field guard", body)
					}
					s := &target.constQ
					if s.invocation != beforeInvocation+1 || s.sourceCursor != 2 || s.active || target.localActive || !errors.Is(s.bodyCause, umemory.ErrRuntimeUninitialized) {
						t.Error("basic getter read payload before scoped TMP", s.invocation, s.sourceCursor, s.active, s.bodyCause)
					}
					if dk == 2 {
						old = Algebraic{}
					}
					if dst != old {
						t.Error("failed const copy entered mutable assignment")
					}
					serviceMust(t, target.CleanupRetained())
					serviceMust(t, target.CleanupValue(&dst))
				})
			}
		}
	}
}
