// Source: src/util/rational.h:242-244; src/util/mpq.h:201;
// src/util/mpz.h:499-510.
package util

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalServiceNegAssignInvocationPhysicalNumerator(t *testing.T) {
	s, b := serviceNew(t)
	for _, v := range []string{"0", "7", "-3/2", "4294967297/3"} {
		a := serviceValue(t, s, v)
		den := a.Raw.Den
		cell := a.Raw.Num.Cell
		var c RationalServiceInvocation
		b.FailAfter(0)
		e := s.NegAssignWithInvocation(&a, &c)
		b.FailAfter(-1)
		serviceMust(t, e)
		if a.Raw.Den != den || a.Raw.Num.Cell != cell {
			t.Fatal("negation changed storage or denominator")
		}
		want := v
		if v != "0" {
			if v[0] == '-' {
				want = v[1:]
			} else {
				want = "-" + v
			}
		}
		serviceText(t, s, &a, want)
		serviceMust(t, c.SourceReady())
		serviceMust(t, c.Cleanup(false))
		serviceMust(t, c.Cleanup(true))
		serviceMust(t, c.EndContext())
		serviceMust(t, s.Del(&a))
	}
}

func TestRationalServiceNegAssignInvocationEveryIntMinReservation(t *testing.T) {
	success := false
	failures := 0
	for ordinal := int64(0); ordinal < 64 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			b := arena.NewBudget(1 << 26)
			rt, e := NewRuntime(RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f Failure) error { return f }})
			serviceMust(t, e)
			cfg := serviceConfig(b)
			cfg.Integers.Memory = rt
			cfg.Integers.Profile = SmallAllocatorProfile{PointerBits: 64, Debug: true}
			s, e := NewRationalService(cfg)
			serviceMust(t, e)
			a := serviceValue(t, s, "-2147483648/3")
			// Parsing the positive magnitude can retain big storage. Reach the
			// actual signed setter so this campaign starts at inline INT_MIN.
			serviceMust(t, s.Manager.Z.SetInt64(&a.Raw.Num, -2147483648))
			if !a.Raw.Num.IsSmall() {
				t.Fatal("not the inline source branch")
			}
			den := a.Raw.Den
			var c RationalServiceInvocation
			b.FailAfter(ordinal)
			body := s.NegAssignWithInvocation(&a, &c)
			b.FailAfter(-1)
			roots := []*Int{&a.Raw.Num, &a.Raw.Den}
			var extra []Block
			if block := c.transferred.Block(); block != (Block{}) {
				extra = append(extra, block)
			}
			integerPhysicalAuditWithRaw(t, s.Manager.Z, extra, roots...)
			if b.Used() != s.HeaderBytes+s.CacheBytes+s.Manager.HeaderBytes+integerActualOwnedBytes(s.Manager.Z) {
				t.Fatal("uncharged physical graph")
			}
			if a.Raw.Den != den {
				t.Fatal("denominator changed")
			}
			if body == nil {
				success = true
				serviceText(t, s, &a, "2147483648/3")
				serviceMust(t, c.SourceReady())
			} else {
				failures++
				powerPublicationComponent(t, s.Manager.Z, &a.Raw.Num, "-2147483648", "2147483648")
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal(body)
				}
				if e = s.NegAssignWithInvocation(&a, &c); !errors.Is(e, arena.ErrInvalidSlot) {
					t.Fatal("stopped receipt reused", e)
				}
			}
			copy := c
			if e = s.NegAssignWithInvocation(&a, &copy); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("copied receipt accepted", e)
			}
			serial, bodyHistory, sourceHistory := c.invocation, c.bodyCause, c.sourceCause
			serviceMust(t, c.Cleanup(false))
			serviceMust(t, s.CleanupValue(&a))
			serviceMust(t, c.Cleanup(true))
			if c.invocation != serial || c.bodyCause != bodyHistory || c.sourceCause != sourceHistory {
				t.Fatal("cleanup changed history")
			}
			serviceMust(t, c.EndContext())
			serviceMust(t, s.CloseChecked())
			serviceMust(t, rt.Close())
			if b.Used() != 0 {
				t.Fatal("retained", b.Used())
			}
		})
	}
	if !success || failures == 0 {
		t.Fatal(success, failures)
	}
}
