package numeral

// Source: src/util/mpf.cpp:1795-1819; src/util/mpf.h:183.

import (
	"errors"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestArbitraryFloatIsIntIndependentDyadicOracle(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	rng := rand.New(rand.NewChaCha8([32]byte{179, 5, 24}))
	for _, sb := range []uint32{2, 8, 24, 53, 120} {
		for i := 0; i < 600; i++ {
			sig := new(big.Int).SetUint64(rng.Uint64())
			sig.Lsh(sig, uint(rng.Uint32()%100))
			sig.Mod(sig, new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
			var n Int
			serviceMust(t, m.integer.SetString(&n, sig.String()))
			exp := int64(rng.Uint32()%(sb+4)) - 3
			var x ArbitraryFloat
			serviceMust(t, m.SetComponents(&x, 8, sb, i%2 != 0, exp, &n))
			got, e := m.IsInt(&x)
			serviceMust(t, e)
			full := new(big.Int).Add(sig, new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
			den := big.NewInt(1)
			if exp < int64(sb)-1 {
				den.Lsh(den, uint(int64(sb)-1-exp))
			}
			want := new(big.Rat).SetFrac(full, den).IsInt()
			if got != want {
				t.Fatal("independent dyadic integer query", sb, exp, sig, got, want)
			}
			actual, e := m.integer.String(&x.significand)
			serviceMust(t, e)
			if actual != sig.String() || x.exponent != exp || x.Sign() != (i%2 != 0) {
				t.Fatal("const input changed", x, actual)
			}
			if m.integerQuery.active || m.integerQuery.constructed {
				t.Fatal("successful query retained temporary")
			}
			serviceMust(t, m.Del(&x))
			serviceMust(t, m.integer.Del(&n))
		}
	}
}

func TestArbitraryFloatIsIntSourceSpecialsAndErasedCopy(t *testing.T) {
	m, _, b := mpfNumeralHelpersFixture(t)
	for _, sign := range []bool{false, true} {
		for _, c := range []struct {
			exp  int64
			sig  int32
			want bool
		}{
			{128, 0, false}, {128, 1, false}, {-127, 0, false}, {-127, 1, false},
			{-1, 0, false}, {0, 0, true}, {0, 1, false}, {22, 1, false}, {22, 2, true}, {23, 1, true},
		} {
			x := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, sign), exponent: c.exp, significand: NewInt(c.sig)}
			got, e := m.IsInt(&x)
			serviceMust(t, e)
			if got != c.want {
				t.Fatal("source is_int class", sign, c, got)
			}
		}
	}
	// Warm only source bounds; the actual BIG input has never been queried.
	_, e := m.topExponent(15)
	serviceMust(t, e)
	_, e = m.bottomExponent(15)
	serviceMust(t, e)
	var x ArbitraryFloat
	n := NewInt(1)
	serviceMust(t, m.integer.Mul2k(&n, 3998, &n))
	serviceMust(t, m.SetComponents(&x, 15, 4000, false, 3999, &n))
	defer func() { serviceMust(t, m.CleanupValue(&x)); serviceMust(t, m.integer.CleanupRawNumeral(&n)) }()
	before := b.Used()
	b.FailAfter(0)
	got, e := m.IsInt(&x)
	serviceMust(t, e)
	if !got || b.Used() != before || m.integerQuery.active || m.integerQuery.constructed {
		t.Fatal("integer exponent branch reached erased copy", got, b.Used(), before)
	}
	x.exponent = -1
	got, e = m.IsInt(&x)
	serviceMust(t, e)
	if got || b.Used() != before {
		t.Fatal("negative exponent copy erasure", got)
	}
	poison := ArbitraryFloat{format: arbitraryFloatFormat(15, 24, false), exponent: 23, significand: Int{Val: 1, Cell: NumericCellRef(^uint32(0))}}
	got, e = m.IsInt(&poison)
	serviceMust(t, e)
	if !got || b.Used() != before {
		t.Fatal("high exponent touched erased significand", got)
	}
	poison.exponent = -1
	got, e = m.IsInt(&poison)
	serviceMust(t, e)
	if got || b.Used() != before {
		t.Fatal("negative exponent touched erased significand", got)
	}
	b.FailAfter(-1)
}

func TestArbitraryFloatIsIntDestructorFailureClearsResult(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	s := &m.integerQuery
	*s = arbitraryFloatIntegerQueryScope{constructed: true, active: true, executing: true}
	serviceMust(t, m.integer.SetLimbs(&s.t, []uint32{1, 2, 3, 4}))
	if s.t.Cell == 0 {
		t.Fatal("fixture did not retain physical integer storage")
	}
	cell := s.t.Cell
	s.t.Cell = ^NumericCellRef(0)
	m.valueExecuting = true
	result := true // the numeric loop completed before scoped_mpz destruction
	var cause error
	m.completeIntegerQuery(&result, &cause)
	if result || !errors.Is(cause, arena.ErrInvalidSlot) || !s.active || s.sourceCause == nil || s.executing || m.valueExecuting {
		t.Fatal("failed scoped_mpz destructor published true", result, cause, *s)
	}
	if e := m.SourceFinalize(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("source finalization lost the stopped destructor", e)
	}
	s.t.Cell = cell
	serviceMust(t, m.CleanupIntegerQuery())
	if s.active {
		t.Fatal("owning cleanup retained completed integer query")
	}
}

func TestArbitraryFloatIsIntEveryColdCopyReservation(t *testing.T) {
	for _, sbits := range []uint32{4000, 40000} {
		for _, lowBit := range []int32{0, 1} {
			t.Run(big.NewInt(int64(sbits)).String()+"/"+big.NewInt(int64(lowBit)).String(), func(t *testing.T) {
				sawFailure := false
				for ordinal := int64(0); ordinal < 128; ordinal++ {
					m, r, b := mpfNumeralHelpersFixture(t)
					n := NewInt(1)
					serviceMust(t, m.integer.Mul2k(&n, sbits-2, &n))
					if lowBit != 0 {
						serviceMust(t, m.integer.Inc(&n))
					}
					var x ArbitraryFloat
					serviceMust(t, m.SetComponents(&x, 17, sbits, false, int64(sbits)-2, &n))
					serviceMust(t, m.integer.Del(&n))
					_, e := m.IsNormal(&x)
					serviceMust(t, e)
					old := x.significand // borrowed identity only
					b.FailAfter(ordinal)
					got, body := m.IsInt(&x)
					s := &m.integerQuery
					integerPhysicalAudit(t, m.integer, &x.significand, &s.t)
					if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
						t.Fatal("immediate query physical graph", ordinal, body, b.Used(), m.OwnedStorageBytes(), r.Stats())
					}
					if x.significand != old || x.exponent != int64(sbits)-2 {
						t.Fatal("query mutated input", ordinal)
					}
					if body == nil && got != (lowBit == 0) {
						t.Fatal("cold query result", lowBit, got)
					}
					if body != nil {
						sawFailure = true
						if s.copy.manager != m.integer || s.copy.Frame != &m.integer.CopyAllocation || !s.active {
							t.Fatal("failure did not capture actual SET child", ordinal, body, *s)
						}
					}
					if body != nil && !numericPhysicalReservationRefusal(body) {
						t.Fatal("unexpected failure", ordinal, body)
					}
					if s.active {
						before := *s
						if _, e := m.IsInt(&x); !errors.Is(e, ErrNumericPending) || *s != before {
							t.Fatal("failed scope was replayed", ordinal, e)
						}
					}
					b.FailAfter(-1)
					serviceMust(t, m.CleanupIntegerQuery())
					serviceMust(t, m.CleanupIntegerQuery())
					serviceMust(t, m.CleanupValue(&x))
					serviceMust(t, m.Close())
					serviceMust(t, r.Close())
					if b.Used() != 0 {
						t.Fatal("query refund", ordinal, b.Used())
					}
					if body == nil {
						if sbits == 40000 && !sawFailure {
							t.Fatal("direct SET failure coverage missing")
						}
						t.Log("complete SET reservation horizon", ordinal)
						return
					}
				}
				t.Fatal("query reservation horizon")
			})
		}
	}
}
func TestArbitraryFloatIsIntActualTransportGuardsAndAbandonedScope(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	var m *ArbitraryFloatManager
	var x ArbitraryFloat
	armed, calls := false, 0
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error {
		if armed {
			calls++
			xBefore, scopeBefore, used := x, m.integerQuery, b.Used()
			_, queryError := m.IsInt(&x)
			for _, err := range []error{queryError, m.SourceFinalize(), m.Close(), m.CleanupIntegerQuery(), m.CleanupValue(&x)} {
				if !errors.Is(err, ErrNumericPending) {
					t.Fatal("returning transport owner guard", err)
				}
			}
			if x != xBefore || m.integerQuery != scopeBefore || b.Used() != used {
				t.Fatal("transport changed retained graph")
			}
		}
		return f
	}})
	serviceMust(t, e)
	m, e = NewArbitraryFloatManager(IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	n := NewInt(1)
	serviceMust(t, m.integer.Mul2k(&n, 39998, &n))
	serviceMust(t, m.SetComponents(&x, 15, 40000, false, 0, &n))
	serviceMust(t, m.integer.Del(&n))
	// Warm the literal classification caches only, leaving the local SET cold.
	_, e = m.IsNormal(&x)
	serviceMust(t, e)
	pad, e := r.Allocate(uint64(99990 - r.State().LocalSize - 8))
	serviceMust(t, e)
	serviceMust(t, r.SetMaxSize(1))
	armed = true
	_, body := m.IsInt(&x)
	armed = false
	if body == nil || calls != 1 || !m.integerQuery.active {
		t.Fatal("actual scoped SET transport was not reached", body, calls, m.integerQuery)
	}
	integerPhysicalAuditWithRaw(t, m.integer, []umemory.Block{pad}, &x.significand, &m.integerQuery.t)
	if b.Used() != m.OwnedStorageBytes()+r.Stats().TotalBytes {
		t.Fatal("stopped actual Runtime graph", b.Used(), m.OwnedStorageBytes(), r.Stats())
	}
	serviceMust(t, r.SetMaxSize(1<<28))
	serviceMust(t, r.Deallocate(&pad))
	// Close must join the abandoned actual local and SET allocation itself.
	serviceMust(t, m.Close())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("abandoned query owner end", b.Used())
	}
}
