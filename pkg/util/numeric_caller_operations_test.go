// Source: src/util/mpz.cpp (caller lifetime boundary tests).
package util

import (
	"errors"
	"math/big"
	"testing"
)

func TestNumericCallerDivisionUsesCallerStorageAndRejectsExpiredOwner(t *testing.T) {
	m, budget := mpzPublicationOwner(t)
	var a, b, out Int
	serviceMust(t, m.SetString(&a, "-354224848179261915075"))
	serviceMust(t, m.SetString(&b, "5731478440138170841"))
	generation := uint64(1)
	expected := generation
	active := true
	var roots NumericAllocationRoots
	var raw IntegerScalarRawScope
	var h *NumericInvocationHooks
	var buffers [4]NaturalBufferReceipt
	bufferCalls := 0
	h = &NumericInvocationHooks{
		Roots: func(owner *IntegerManager, local *NumericAllocationRoots) *NumericAllocationRoots {
			if owner != m || generation != expected || !active {
				return nil
			}
			if local != nil {
				return local
			}
			return &roots
		},
		Bound: func(executing bool) bool { return generation == expected && (!executing || active) },
		ScalarRaw: func(owner *IntegerManager, operation string) (*IntegerScalarRawScope, error) {
			if owner != m || operation != "division" {
				t.Fatal("unexpected caller operation", operation)
			}
			return &raw, nil
		},
	}
	h.Buffer = func(owner *NaturalManager, index int) *NaturalBufferReceipt {
		if owner != m.Natural {
			t.Fatal("foreign buffer owner")
		}
		serviceMust(t, owner.BindCallerBuffer(&buffers[index], index, generation, h))
		bufferCalls++
		return &buffers[index]
	}

	serviceMust(t, m.SetCallerExecutionGuard(func() bool { return active }))
	ops := m.WithAllocationContext((NumericAllocationContext{}).WithCaller(h))
	serviceMust(t, ops.Div(&a, &b, &out))
	wantA, _ := new(big.Int).SetString("-354224848179261915075", 10)
	wantB, _ := new(big.Int).SetString("5731478440138170841", 10)
	want := new(big.Int).Div(wantA, wantB)
	if got := integerString(t, m, &out); got != want.String() {
		t.Fatal(got, want)
	}
	if e := m.CloseChecked(); !errors.Is(e, ErrNumericPending) {
		t.Fatal("caller execution must retain the borrowed owner", e)
	}
	active = false
	copiedRaw := raw
	if e := copiedRaw.Cleanup(true); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("copied caller storage accepted", e)
	}
	serviceMust(t, raw.Cleanup(true))
	for i := range buffers {
		serviceMust(t, buffers[i].Cleanup())
	}
	if bufferCalls == 0 {
		t.Fatal("caller buffers/verification bypassed", bufferCalls)
	}
	before := integerString(t, m, &out)
	generation++
	if e := ops.SetInt64(&out, 7); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("expired caller accepted", e)
	}
	if got := integerString(t, m, &out); got != before {
		t.Fatal("expired caller changed output", got, before)
	}
	serviceMust(t, m.SetCallerExecutionGuard(nil))
	for _, v := range []*Int{&a, &b, &out} {
		serviceMust(t, m.Del(v))
	}
	serviceMust(t, roots.Cleanup(m))
	serviceMust(t, m.CloseChecked())
	if budget.Used() != 0 {
		t.Fatal("caller storage leaked", budget.Used())
	}
}

func TestNumericCallerGCDAllocationFailuresRetainAndCleanReceipts(t *testing.T) {
	succeeded, failed, recorded := false, false, false
	for ordinal := int64(0); ordinal < 128 && !succeeded; ordinal++ {
		m, budget := mpzPublicationOwner(t)
		var a, b, out Int
		serviceMust(t, m.SetString(&a, "354224848179261915075"))
		serviceMust(t, m.SetString(&b, "573147844013817084101"))
		serviceMust(t, m.Mul2kInPlace(&a, 256))
		serviceMust(t, m.Mul2kInPlace(&b, 256))
		var roots NumericAllocationRoots
		var receipt IntegerRawInvocationReceipt
		h := &NumericInvocationHooks{Roots: func(owner *IntegerManager, local *NumericAllocationRoots) *NumericAllocationRoots {
			if owner != m {
				return nil
			}
			if local != nil {
				return local
			}
			return &roots
		}, Record: func(owner *IntegerManager, event string, r NumericInvocationReceipt) {
			if owner != m {
				t.Fatal("foreign receipt owner")
			}
			if r.Raw != nil {
				receipt = *r.Raw
				recorded = true
			}
		}}
		budget.FailAfter(ordinal)
		call := m.WithAllocationContext((NumericAllocationContext{}).WithCaller(h)).GCD(&a, &b, &out)
		// This independent audit rejects anonymous descriptors and allocator objects
		// immediately after each failed allocation, before disabling the injection.
		integerPhysicalAudit(t, m, &a, &b, &out)
		budget.FailAfter(-1)
		if call == nil {
			succeeded = true
			want := new(big.Int).Lsh(big.NewInt(1), 256).String()
			if got := integerString(t, m, &out); got != want {
				t.Fatal(got, want)
			}
		} else {
			failed = true
		}
		budget.FailAfter(-1)
		if receipt.Manager != nil {
			serviceMust(t, receipt.Cleanup(true))
		}
		serviceMust(t, roots.Cleanup(m))
		for _, v := range []*Int{&a, &b, &out} {
			serviceMust(t, m.Del(v))
		}
		serviceMust(t, m.CloseChecked())
		if budget.Used() != 0 {
			t.Fatal("caller receipt cleanup leaked", ordinal, budget.Used())
		}
	}
	if !succeeded || !failed || !recorded {
		t.Fatal("missing real operation/failure/receipt", succeeded, failed, recorded)
	}
}
