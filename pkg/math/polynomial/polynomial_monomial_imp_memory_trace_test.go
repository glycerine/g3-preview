package polynomial

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestPolynomialImpOwnedMonomialManagerMemoryWrappers(t *testing.T) {
	for _, borrowedAllocator := range []bool{false, true} {
		name := "owned-allocator"
		line := 2330
		if borrowedAllocator {
			name = "borrowed-allocator"
			line = 2341
		}
		t.Run(name, func(t *testing.T) {
			q, budget := rationalManager(t, false)
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("memory")
			var allocator *umemory.SmallAllocator
			if borrowedAllocator {
				var err error
				allocator, err = umemory.NewSmallAllocator(umemory.SmallAllocatorConfig{Runtime: q.Integers().Memory, Profile: umemory.SmallAllocatorProfile{PointerBits: 64}})
				serviceMust(t, err)
			}
			full, err := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{
				Core: MultivariatePolynomialConfig{
					Integer: q.Integers(), Rationals: q,
					Checkpoint: func() bool { return true }, Cooperation: func() error { return nil },
				},
				Allocator: allocator, Trace: tags,
			})
			serviceMust(t, err)
			child := full.core.monomials
			serviceMust(t, full.CloseChecked())
			if allocator != nil {
				serviceMust(t, allocator.Close())
			}
			got := output.String()
			if !diagnostic.DebugBuild {
				if got != "" {
					t.Fatal("ordinary build emitted memory records", got)
				}
				return
			}
			address := fmt.Sprintf("%p", child)
			parent := fmt.Sprintf("%p", full)
			parentLine := 6931
			if borrowedAllocator {
				parentLine = 6935
			}
			parentAlloc := "alloc " + parent + " src/math/polynomial/polynomial.cpp:" + strconv.Itoa(parentLine) +
				" imp " + strconv.FormatUint(uint64(reflect.TypeOf(MultivariatePolynomialManager{}).Size()), 10) + "\n"
			parentDealloc := "dealloc " + parent + " src/math/polynomial/polynomial.cpp:6939\n"
			alloc := "alloc " + address + " src/math/polynomial/polynomial.cpp:" + strconv.Itoa(line) +
				" monomial_manager " + strconv.FormatUint(uint64(reflect.TypeOf(MonomialManager{}).Size()), 10) + "\n"
			dealloc := "dealloc " + address + " src/math/polynomial/polynomial.cpp:814\n"
			if strings.Count(got, alloc) != 1 || strings.Count(got, dealloc) != 1 || strings.Index(got, alloc) >= strings.Index(got, dealloc) {
				t.Fatalf("source monomial owner memory order: %q", got)
			}
			if strings.Count(got, parentAlloc) != 1 || strings.Count(got, parentDealloc) != 1 ||
				strings.Index(got, parentAlloc) >= strings.Index(got, alloc) ||
				strings.Index(got, dealloc) >= strings.Index(got, parentDealloc) {
				t.Fatalf("source polynomial imp nesting: %q", got)
			}
			if budget.Used() == 0 {
				t.Fatal("borrowed rational manager was unexpectedly closed")
			}
		})
	}
}

func TestPolynomialImpFailedPlacementMemoryRecord(t *testing.T) {
	q, budget := rationalManager(t, false)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	budget.FailAfter(1) // imp header succeeds; first constructor member fails.
	full, err := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{
		Core: MultivariatePolynomialConfig{
			Integer: q.Integers(), Rationals: q,
			Checkpoint: func() bool { return true }, Cooperation: func() error { return nil },
		}, Trace: tags,
	})
	budget.FailAfter(-1)
	if full == nil || err == nil || full.completed {
		t.Fatalf("expected failed imp after allocation: %p %v", full, err)
	}
	serviceMust(t, full.CloseChecked())
	got := output.String()
	if !diagnostic.DebugBuild {
		if got != "" {
			t.Fatal("ordinary build emitted memory records", got)
		}
		return
	}
	address := fmt.Sprintf("%p", full)
	if strings.Count(got, "alloc "+address+" src/math/polynomial/polynomial.cpp:6931 imp ") != 1 ||
		strings.Contains(got, "dealloc "+address+" src/math/polynomial/polynomial.cpp:6939") {
		t.Fatalf("failed imp placement memory record: %q", got)
	}
}

func TestPolynomialImpMonomialFailedPlacementMemoryRecord(t *testing.T) {
	q, budget := rationalManager(t, false)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("memory")
	budget.FailAfter(1) // owner header succeeds; constructor's first child fails.
	child, err := NewMonomialManager(MonomialManagerConfig{
		Budget: budget, Memory: q.Integers().Memory, Trace: tags, polynomialImpAllocationLine: 2330,
	})
	budget.FailAfter(-1)
	if child == nil || err == nil || child.completed {
		t.Fatalf("expected failed child after its allocation: %p %v", child, err)
	}
	serviceMust(t, child.CloseChecked())
	got := output.String()
	if !diagnostic.DebugBuild {
		if got != "" {
			t.Fatal("ordinary build emitted memory records", got)
		}
		return
	}
	address := fmt.Sprintf("%p", child)
	if strings.Count(got, "alloc "+address+" src/math/polynomial/polynomial.cpp:2330 monomial_manager ") != 1 ||
		strings.Contains(got, "dealloc "+address+" src/math/polynomial/polynomial.cpp:814") {
		t.Fatalf("failed monomial placement memory record: %q", got)
	}
}
