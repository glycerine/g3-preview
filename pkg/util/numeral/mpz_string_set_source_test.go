// Source: src/util/mpz.cpp:283-303; src/util/mpz.h:692.
package numeral

import (
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"
)

func TestIntegerStringSetSourceDirectPublication(t *testing.T) {
	m, _, b := mpfNumeralHelpersFixture(t)
	z := m.integer
	a := NewInt(7)
	b.FailAfter(0)
	body := z.SetString(&a, "-12345678901234567890")
	b.FailAfter(-1)
	if !numericPhysicalReservationRefusal(body) || a != NewInt(1234567890) {
		t.Fatal("caller prefix before first BIG reservation; sign not reached", body, a)
	}
	serviceMust(t, z.CleanupRawNumeral(&a))
}

// Decode only valid physical numerals. A failed Add may publish its positive
// sign and checked failed pointer; that is not an ordinary numerical value.
func integerParserSourcePrefixMatches(t *testing.T, z *IntegerManager, out *Int, text string) bool {
	t.Helper()
	s := &z.stringSet
	if out.Cell != 0 && (z.IsFailureRef(out.Cell) || ValidNumericRef(z.Cells, out.Cell) && z.Cell(out.Cell).Released) {
		return s.declared && s.stage == 2 && out.Val == 1
	}
	if e := z.check(out); e != nil {
		return false
	}
	value := roundPublicationDecode(t, z, out)
	n := new(big.Int)
	if value.Sign() == 0 {
		return true
	}
	for i := 0; i < len(text) && text[i] != 0; i++ {
		ch := text[i]
		if ch < '0' || ch > '9' {
			continue
		}
		n.Mul(n, big.NewInt(10))
		n.Add(n, big.NewInt(int64(ch-'0')))
		if value.Cmp(n) == 0 {
			return true
		}
	}
	if s.endReached && s.tmpDelDone && s.negative {
		n.Neg(n)
		return value.Cmp(n) == 0
	}
	return false
}

func TestIntegerStringSetSourceEveryColdReservation(t *testing.T) {
	retainedPublishedTMP := false
	for _, text := range []string{"-123456789012345678901234567890", "2147483648", "-2147483648", strings.Repeat("9876543210", 130)} {
		complete := false
		for ordinal := int64(0); ordinal < 400; ordinal++ {
			b := arena.NewBudget(1 << 28)
			m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
			serviceMust(t, e)
			z := m.integer
			a := NewInt(7)
			b.FailAfter(ordinal)
			body := z.SetString(&a, text)
			s := &z.stringSet
			integerPhysicalAudit(t, z, &a)
			if b.Used() != m.OwnedStorageBytes()+m.memory.Stats().TotalBytes {
				t.Fatal("parser immediate byte graph", ordinal)
			}
			if body != nil {
				if s.stage == 2 && s.tmp.Cell != 0 && !z.IsFailureRef(s.tmp.Cell) && ValidNumericRef(z.Cells, s.tmp.Cell) && !z.Cell(s.tmp.Cell).Released {
					retainedPublishedTMP = true
					prefix := new(big.Int)
					for i := 0; i < s.index; i++ {
						ch := text[i]
						if ch >= '0' && ch <= '9' {
							prefix.Mul(prefix, big.NewInt(10))
							prefix.Add(prefix, big.NewInt(int64(ch-'0')))
						}
					}
					prefix.Mul(prefix, big.NewInt(10))
					if got := roundPublicationDecode(t, z, &s.tmp); got.Cmp(prefix) != 0 {
						t.Fatal("actual Mul output before omitted RAW DEL", got, prefix)
					}
				}
				if !numericPhysicalReservationRefusal(body) || !integerParserSourcePrefixMatches(t, z, &a, text) || s.endReached || s.tmpDelDone {
					t.Fatal("literal RAW failure prefix", ordinal, body, s.index, s.stage)
				}
				before, used := s.tmp, b.Used()
				inv := s.invocation
				if e := z.SetString(&a, "1"); e != ErrNumericPending && e != arena.ErrInvalidSlot {
					t.Fatal("failed parser replay", e)
				}
				if s.tmp != before || s.invocation != inv || b.Used() != used {
					t.Fatal("replay changed actual RAW root")
				}
			}
			b.FailAfter(-1)
			if body == nil {
				want, ok := new(big.Int).SetString(text, 10)
				if !ok {
					t.Fatal(text)
				}
				assertIntegerPowerOracle(t, z, &a, want)
				if !s.endReached || !s.tmpDelDone || !s.bodySucceeded {
					t.Fatal("end DEL missing")
				}
				complete = true
			}
			end, deleted := s.endReached, s.tmpDelDone
			serviceMust(t, z.CleanupStringSet())
			if s.endReached != end || s.tmpDelDone != deleted {
				t.Fatal("owning cleanup advanced source label")
			}
			serviceMust(t, z.CleanupRawNumeral(&a))
			if e = m.Close(); e != nil {
				signedFractionScratchHole(t, m)
				serviceMust(t, m.Close())
			}
			if b.Used() != 0 {
				t.Fatal("parser refund", ordinal, b.Used())
			}
			if complete {
				break
			}
		}
		if !complete {
			t.Fatal("no complete parser prefix", len(text))
		}
	}
	if !retainedPublishedTMP {
		t.Fatal("no actual published BIG RAW temporary retained before Add failure")
	}
}

func TestIntegerStringSetSourceIndependentPermissiveBytes(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	z := m.integer
	rng := rand.New(rand.NewChaCha8([32]byte{28, 30, 3}))
	var out Int
	for sample := 0; sample < 200; sample++ {
		data := make([]byte, rng.IntN(130))
		for i := range data {
			data[i] = byte(rng.Uint32())
		}
		if sample%3 == 0 {
			data = append([]byte("   -"), data...)
		}
		n := new(big.Int)
		signAt := 0
		for signAt < len(data) && data[signAt] == ' ' {
			signAt++
		}
		negative := signAt < len(data) && data[signAt] == '-'
		for _, ch := range data {
			if ch == 0 {
				break
			}
			if ch >= '0' && ch <= '9' {
				n.Mul(n, big.NewInt(10))
				n.Add(n, big.NewInt(int64(ch-'0')))
			}
		}
		if negative {
			n.Neg(n)
		}
		serviceMust(t, z.SetString(&out, string(data)))
		assertIntegerPowerOracle(t, z, &out, n)
	}
	serviceMust(t, z.Del(&out))
}

func TestIntegerStringSetSourceWarmAndEntryAuthority(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	z := m.integer
	a := NewInt(7)
	var frame *integerStringSetScope
	var invocation uint64
	z.stringSet.invocation = ^uint64(0)
	used := b.Used()
	if e := z.setStringWithBeginCapture(&a, "1", &frame, &invocation); e != ErrNumericPending || a != NewInt(7) || frame != nil || invocation != 0 || b.Used() != used {
		t.Fatal("entry counter/refused capture", e)
	}
	z.stringSet.invocation = 0
	copied := *z
	if e := copied.SetString(&a, "1"); e != umemory.ErrRuntimeOwner {
		t.Fatal("copied manager", e)
	}
	if e := z.SetString(nil, "1"); e != arena.ErrInvalidSlot {
		t.Fatal("nil numeral", e)
	}
	serviceMust(t, z.SetString(&a, "-12"))
	state, stats, used := r.State(), r.Stats(), b.Used()
	b.FailAfter(0)
	serviceMust(t, z.setStringWithBeginCapture(&a, "  +90a\x0033", &frame, &invocation))
	if a != NewInt(90) || frame != &z.stringSet || invocation != z.stringSet.invocation || r.State() != state || r.Stats() != stats || b.Used() != used {
		t.Fatal("warm parse/capture", a)
	}
	b.FailAfter(-1)
	serviceMust(t, z.CleanupStringSet())
	serviceMust(t, z.SetString(&a, "99"))
	later, used := z.stringSet.invocation, b.Used()
	serviceMust(t, z.CleanupStringSet())
	if z.stringSet.invocation != later || b.Used() != used || a != NewInt(99) {
		t.Fatal("ended owning cleanup changed later parse")
	}
	serviceMust(t, z.Del(&a))
}
