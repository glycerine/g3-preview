// Source: src/util/rational.h:52; src/util/mpq.h set(mpq const&).
package numeral

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestRationalServiceRawRationalConstructorEveryReservation(t *testing.T) {
	failed, denominatorPrefix := false, false
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		b := arena.NewBudget(1 << 27)
		s, e := NewRationalService(serviceConfig(b))
		serviceMust(t, e)
		source, e := NewRationalManager(RationalConfig{Integers: serviceConfig(b).Integers})
		serviceMust(t, e)
		value := NewRational()
		// Deliberately unreduced, with a negative denominator. Both components
		// need real cells; constructor fidelity does not normalize the record.
		serviceMust(t, source.Z.SetString(&value.Num, "18446744073709551618"))
		serviceMust(t, source.Z.SetString(&value.Den, "-18446744073709551618"))
		original := value
		var call RationalServiceInvocation
		b.FailAfter(ordinal)
		out, body := s.NewFromRationalOwnerWithInvocation(source, &value, &call)
		if value != original {
			t.Fatal("constructor changed borrowed record", ordinal)
		}
		integerPhysicalAudit(t, source.Z, &value.Num, &value.Den)
		roots := []*Int{&out.Raw.Num, &out.Raw.Den}
		for i := range s.Cache {
			roots = append(roots, &s.Cache[i].Num, &s.Cache[i].Den)
		}
		for i := uint8(0); i < call.scope.Count; i++ {
			r := &call.scope.Roots[i]
			roots = append(roots, &r.Num, &r.Den)
			if !r.Num.IsSmall() && r.Num.Cell != s.Manager.Z.FailedNullRef && r.Den.Cell == s.Manager.Z.FailedNullRef {
				denominatorPrefix = true
			}
		}
		integerPhysicalAudit(t, s.Manager.Z, roots...)
		if body != nil {
			failed = true
			if !numericPhysicalReservationRefusal(body) || out.ownerID != 0 {
				t.Fatal("failed constructor published value or unexpected error", ordinal, out, body)
			}
		} else {
			if out.ownerID == 0 || out.Raw.Num.Val != value.Num.Val || out.Raw.Den.Val != value.Den.Val {
				t.Fatal("constructor changed record tags", ordinal, out)
			}
			n, err := s.Manager.Z.String(&out.Raw.Num)
			serviceMust(t, err)
			d, err := s.Manager.Z.String(&out.Raw.Den)
			serviceMust(t, err)
			if n != "18446744073709551618" || d != "-18446744073709551618" {
				t.Fatal("constructor normalized or changed components", n, d)
			}
		}
		b.FailAfter(-1)
		serviceMust(t, call.Cleanup(false))
		if out.ownerID != 0 {
			serviceMust(t, s.CleanupValue(&out))
		}
		serviceMust(t, call.Cleanup(true))
		serviceMust(t, call.EndContext())
		serviceMust(t, source.Del(&value))
		serviceMust(t, source.CloseChecked())
		serviceMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatal("constructor owner graph leaked", ordinal, b.Used())
		}
		if body == nil {
			if !failed || !denominatorPrefix {
				t.Fatal("did not reach allocation failure in both-component construction", failed, denominatorPrefix)
			}
			return
		}
	}
	t.Fatal("constructor never reached success")
}

func TestRationalServiceRawRationalConstructorReturningTransportProtectsSource(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	var source *RationalManager
	var sourceRefusal, integerRefusal, invocationRefusal error
	var call RationalServiceInvocation
	active, calls := false, 0
	runtime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if active {
			calls++
			sourceRefusal = source.CloseChecked()
			integerRefusal = source.Z.CloseChecked()
			invocationRefusal = call.Cleanup(false)
		}
		return f
	}})
	serviceMust(t, e)
	cfg := serviceConfig(b)
	cfg.Integers.Memory = runtime
	s, e := NewRationalService(cfg)
	serviceMust(t, e)
	source, e = NewRationalManager(RationalConfig{Integers: serviceConfig(b).Integers})
	serviceMust(t, e)
	value := NewRational()
	words := make([]uint32, 2000)
	words[0], words[len(words)-1] = 17, 1
	serviceMust(t, source.Z.SetLimbs(&value.Num, words))
	original, before := value, source.Z.Memory.Stats()
	serviceMust(t, runtime.SetMaxSize(1))
	active = true
	out, body := s.NewFromRationalOwnerWithInvocation(source, &value, &call)
	active = false
	if calls == 0 || body == nil || out.ownerID != 0 {
		t.Fatal("actual returning allocation transport not reached", calls, body, out)
	}
	if sourceRefusal == nil || integerRefusal == nil || invocationRefusal == nil || source.closed || source.Z.closed || value != original || source.Z.Memory.Stats() != before {
		t.Fatal("transport consumed executing constructor or borrowed source", sourceRefusal, integerRefusal, invocationRefusal)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, call.Cleanup(false))
	serviceMust(t, call.Cleanup(true))
	serviceMust(t, call.EndContext())
	serviceMust(t, source.Del(&value))
	serviceMust(t, source.CloseChecked())
	serviceMust(t, s.Close())
	serviceMust(t, runtime.Close())
	if b.Used() != 0 {
		t.Fatal("constructor owner graph leaked", b.Used())
	}
}

func TestRationalServiceRawRationalConstructorSameOwner(t *testing.T) {
	s, _ := serviceNew(t)
	input := serviceValue(t, s, "18446744073709551617/19")
	var call RationalServiceInvocation
	out, e := s.NewFromRationalOwnerWithInvocation(s.Manager, &input.Raw, &call)
	serviceMust(t, e)
	serviceMust(t, call.Cleanup(false))
	serviceMust(t, call.Cleanup(true))
	serviceMust(t, call.EndContext())
	serviceMust(t, s.Del(&input))
	got, e := s.String(&out)
	serviceMust(t, e)
	if got != "18446744073709551617/19" {
		t.Fatal("returned value borrowed source storage", got)
	}
	serviceMust(t, s.Del(&out))
}

func TestRationalServiceSourceDelValueAssignmentFailurePrefix(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	runtime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(b)
	cfg.Integers.Memory = runtime
	s, e := NewRationalService(cfg)
	serviceMust(t, e)
	input, e := s.NewValue()
	serviceMust(t, e)
	words := make([]uint32, 2000)
	words[0], words[len(words)-1] = 17, 1
	serviceMust(t, s.Manager.Z.SetLimbs(&input.Raw.Num, words))
	out, e := s.NewValue()
	serviceMust(t, e)
	serviceMust(t, s.SetInt64(&out, 99))
	var call RationalServiceInvocation
	serviceMust(t, runtime.SetMaxSize(1))
	body := s.CopyWithInvocation(&out, &input, &call)
	if body == nil || out.Raw.Num.Cell != s.Manager.Z.FailedNullRef {
		t.Fatal("actual assignment failure prefix not reached", body, out)
	}
	// The actual failed allocation records a source NULL whose deletion is
	// defined. Do not invent a dangling-header stop merely from the Go tag.
	serviceMust(t, s.SourceDelValue(&out))
	if out != (OwnedRational{}) {
		t.Fatal("source NULL destructor retained token", out)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, call.Cleanup(false))
	serviceMust(t, call.Cleanup(true))
	serviceMust(t, call.EndContext())
	serviceMust(t, s.SourceDelValue(&input))
	if input != (OwnedRational{}) {
		t.Fatal("completed source destructor retained token")
	}
	serviceMust(t, s.Close())
	serviceMust(t, runtime.Close())
	if b.Used() != 0 {
		t.Fatal("source/owning destructor graph leaked", b.Used())
	}
}
