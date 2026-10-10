package numeral

import (
	"fmt"
	"strings"
	"testing"
)

func TestIntegerRawObservationGenerationIdentity(t *testing.T) {
	for _, kind := range []string{"division", "gcd"} {
		matched := false
		for ordinal := int64(0); ordinal < 50 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", kind, ordinal), func(t *testing.T) {
				m, budget := rationalManagerWithLimit(t, true, 8<<20)
				a, b, out := NewRational(), NewRational(), NewRational()
				serviceMust(t, m.SetString(&a, "-"+strings.Repeat("1234567890", 100)+"7"))
				serviceMust(t, m.SetString(&b, strings.Repeat("9876543210", 40)+"3"))
				t.Cleanup(func() {
					budget.FailAfter(-1)
					serviceMust(t, m.Del(&a))
					serviceMust(t, m.Del(&b))
					serviceMust(t, m.Del(&out))
				})
				if kind == "division" {
					serviceMust(t, m.Z.Div(&a.Num, &b.Num, &out.Num))
				} else {
					serviceMust(t, m.Z.GCD(&a.Num, &b.Num, &out.Num))
				}
				older := m.Z.DivisionInvocation()
				if kind == "gcd" {
					older = m.Z.GCDInvocation()
				}
				if older.Invocation == 0 || older.Matches(m.Z) {
					t.Fatal("completed primitive", older)
				}
				before, err := CaptureNumericAllocationRoots(m.Z)
				serviceMust(t, err)
				budget.FailAfter(ordinal)
				if kind == "division" {
					err = m.Z.Div(&a.Num, &b.Num, &out.Num)
				} else {
					err = m.Z.GCD(&a.Num, &b.Num, &out.Num)
				}
				budget.FailAfter(-1)
				var roots NumericAllocationRoots
				serviceMust(t, roots.After(m.Z, before))
				now := m.Z.DivisionInvocation()
				if kind == "gcd" {
					now = m.Z.GCDInvocation()
				}
				if now.Matches(m.Z) {
					if err == nil || now.Invocation == older.Invocation {
						t.Fatal("replacement", err, now)
					}
					serviceMust(t, m.Z.CleanupRawInvocation(&older, true))
					if !now.Matches(m.Z) {
						t.Fatal("older receipt consumed replacement")
					}
					matched = true
				}
				serviceMust(t, m.Z.CleanupRawInvocation(&now, true))
				serviceMust(t, roots.Cleanup(m.Z))
				qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den, &out.Num, &out.Den})
			})
		}
		if !matched {
			t.Fatal("no actual stopped primitive", kind)
		}
	}
}
