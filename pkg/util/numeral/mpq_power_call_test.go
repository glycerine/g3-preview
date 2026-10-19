package numeral

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"math/big"
	"strconv"
	"testing"
)

func TestRationalPowerInvocationEveryReservation(t *testing.T) {
	for _, input := range []string{"18446744073709551619/18446744073709551629", "-18446744073709551619/18446744073709551629"} {
		t.Run(input, func(t *testing.T) {
			succeeded, refused, raw, normalize, gcd := false, false, false, false, false
			for prefix := int64(0); prefix < 8192 && !succeeded; prefix++ {
				if !t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
					m, b, _ := rawArithmeticGuardOwner(t)
					a, out := NewRational(), NewRational()
					serviceMust(t, m.SetString(&a, input))
					var call RationalPowerInvocation
					b.FailAfter(prefix)
					body := m.PowerWithInvocation(&a, 3, &out, &call)
					b.FailAfter(-1)
					if body == nil {
						succeeded = true
					} else {
						refused = true
						var failure umemory.Failure
						if !errors.Is(body, arena.ErrInjectedFailure) && !(errors.As(body, &failure) && failure.Kind == umemory.FailureMemout && failure.ErrorCode() == 101) {
							t.Fatal("unexpected power refusal", body)
						}
					}
					normalize = normalize || call.normalize.manager == m && call.normalize.generation != 0
					gcd = gcd || m.Z.MatchesRawInvocation(&call.gcd)
					if m.MatchesRawPowerInvocation(&call.power) && call.SourceCause() == nil {
						raw = true
						before := m.RawArithmetic.powerCursor
						serviceMust(t, call.Cleanup(false))
						if !m.RawArithmetic.PowerActive || m.RawArithmetic.powerCursor != before {
							t.Fatal("source cleanup consumed RAW power")
						}
					} else {
						serviceMust(t, call.Cleanup(false))
					}
					serviceMust(t, call.Cleanup(true))
					if call.Pending() || m.RawArithmetic.Pending() || m.Z.GcdRaw.Active || m.Z.DivisionRaw.Active {
						t.Fatal("actual power children survived owning cleanup")
					}
					if body == nil {
						want, _ := new(big.Rat).SetString(input)
						want.Mul(new(big.Rat).Mul(want, want), want)
						got, e := m.String(&out)
						serviceMust(t, e)
						if got != want.RatString() {
							t.Fatal("independent cube", got, want)
						}
					}
					serviceMust(t, m.Del(&out))
					serviceMust(t, m.Del(&a))
				}) {
					return
				}
			}
			if !succeeded || !refused || !raw || !normalize || !gcd {
				t.Fatal("missing success, refusal or accepted RAW children", succeeded, refused, raw, normalize, gcd)
			}
		})
	}
}

func TestRationalPowerInvocationRejectsReplacementAndCopiedOwner(t *testing.T) {
	m, _, _ := rawArithmeticGuardOwner(t)
	a, out := NewRational(), NewRational()
	serviceMust(t, m.SetString(&a, "18446744073709551617"))
	serviceMust(t, m.SetString(&out, "18446744073709551619"))
	rawArithmeticCorrupt(t, m.Z, &out.Num)
	var first, second RationalPowerInvocation
	if e := m.PowerWithInvocation(&a, 0, &out, &first); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	copied := first
	if e := copied.Cleanup(true); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("copied power owner accepted", e)
	}
	if e := m.PowerWithInvocation(nil, 0, nil, &first); !errors.Is(e, ErrNumericPending) {
		t.Fatal("pending call read operands", e)
	}
	serviceMust(t, m.CleanupRawPowerInvocation(&first.power))
	if e := m.PowerWithInvocation(&a, 0, &out, &second); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	state := m.RawInvocationState()
	serviceMust(t, first.Cleanup(false))
	serviceMust(t, first.Cleanup(true))
	if state != m.RawInvocationState() || !second.Pending() {
		t.Fatal("old power acquired replacement")
	}
	serviceMust(t, second.Cleanup(false))
	serviceMust(t, second.Cleanup(true))
}

func TestRationalPowerInvocationExecutingTransportGuards(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	var m *RationalManager
	var runtime *umemory.Runtime
	var call RationalPowerInvocation
	armed, calls := false, 0
	var e error
	runtime, e = umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, SizeTBits: 32, Transport: func(f umemory.Failure) error {
		if armed {
			calls++
			state, used, physical := m.RawInvocationState(), budget.Used(), runtime.Stats()
			for _, abort := range []bool{false, true} {
				if e := call.Cleanup(abort); !errors.Is(e, ErrNumericPending) {
					t.Error("executing power cleanup", e)
				}
			}
			if e := m.PowerWithInvocation(nil, 0, nil, &call); !errors.Is(e, ErrNumericPending) {
				t.Error("executing power reentry", e)
			}
			if state != m.RawInvocationState() || used != budget.Used() || physical != runtime.Stats() {
				t.Error("callback mutated accepted power")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewRationalManager(RationalConfig{Synchronized: true, Integers: IntegerConfig{Memory: runtime, Profile: umemory.SmallAllocatorProfile{PointerBits: 32, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: budget}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}}})
	serviceMust(t, e)
	a, out := NewRational(), NewRational()
	serviceMust(t, m.SetString(&a, "18446744073709551619/7"))
	armed = true
	budget.FailAfter(0)
	body := m.PowerWithInvocation(&a, 3, &out, &call)
	budget.FailAfter(-1)
	armed = false
	if body == nil || calls != 1 {
		t.Fatal("actual returning transport not reached", body, calls)
	}
	serviceMust(t, call.Cleanup(false))
	serviceMust(t, call.Cleanup(true))
	serviceMust(t, m.Del(&out))
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.CloseChecked())
	serviceMust(t, runtime.Close())
	if budget.Used() != 0 {
		t.Fatal("transport cleanup leaked", budget.Used())
	}
}
