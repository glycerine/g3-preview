// Source: src/util/rational.h:54; src/util/mpq.h:735-738.
// Licensed under the MIT license; see LICENSE.txt.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestRationalServiceIntegerConstructorExplicitBorrowedOwner(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	s, e := NewRationalService(serviceConfig(b))
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	source, e := NewIntegerManager(cfg)
	serviceMust(t, e)
	var n Int
	serviceMust(t, source.SetString(&n, "18446744073709551617"))
	out, e := s.NewValue()
	serviceMust(t, e)
	serviceMust(t, s.SetInt64(&out, -7))
	var call RationalServiceInvocation
	serviceMust(t, s.SetIntegerFromOwnerWithInvocation(&out, source, &n, &call))
	got, e := s.String(&out)
	serviceMust(t, e)
	if got != "18446744073709551617" {
		t.Fatal("actual NUM before reset DEN", got)
	}
	serviceMust(t, call.Cleanup(false))
	serviceMust(t, s.CleanupValue(&out))
	serviceMust(t, call.Cleanup(true))
	serviceMust(t, call.EndContext())
	serviceMust(t, source.Del(&n))
	serviceMust(t, source.CloseChecked())
	serviceMust(t, s.Close())
	if b.Used() != 0 {
		t.Fatal("actual explicit borrowed owner graph", b.Used())
	}
}

func TestRationalServiceIntegerConstructorActualReturningTransportProtectsSource(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	var source *IntegerManager
	active := false
	calls := 0
	var refused error
	runtime, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, Transport: func(f memory.Failure) error {
		if active {
			calls++
			refused = source.CloseChecked()
		}
		return f
	}})
	serviceMust(t, e)
	cfg := serviceConfig(b)
	cfg.Integers.Memory = runtime
	s, e := NewRationalService(cfg)
	serviceMust(t, e)
	source, e = NewIntegerManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	var n Int
	words := make([]uint32, 2000)
	words[0], words[len(words)-1] = 17, 1
	serviceMust(t, source.SetLimbs(&n, words))
	out, e := s.NewValue()
	serviceMust(t, e)
	var call RationalServiceInvocation
	t.Cleanup(func() {
		serviceMust(t, runtime.SetMaxSize(0))
		serviceMust(t, call.Cleanup(false))
		serviceMust(t, s.CleanupValue(&out))
		serviceMust(t, call.Cleanup(true))
		serviceMust(t, call.EndContext())
		if !source.closed {
			serviceMust(t, source.Del(&n))
		}
		serviceMust(t, source.CloseChecked())
		serviceMust(t, s.Close())
		serviceMust(t, runtime.Close())
		if b.Used() != 0 {
			t.Error("actual constructor owner graph", b.Used())
		}
	})
	original := n
	before := source.memory.Stats()
	serviceMust(t, runtime.SetMaxSize(1))
	active = true
	body := s.SetIntegerFromOwnerWithInvocation(&out, source, &n, &call)
	active = false
	if calls == 0 || body == nil {
		t.Fatal("real returning Runtime allocation transport not reached", calls, body)
	}
	if refused == nil || source.closed || n != original || source.memory.Stats() != before {
		t.Fatal("returning transport consumed borrowed source before copy", refused, source.closed, n, original)
	}
	integerPhysicalAudit(t, source, &n)
	serviceAuditRationalRoots(t, s, &out.raw)
}
