package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"strings"
	"testing"
)

func TestSMT2SimpleExhaustiveBytes(t *testing.T) {
	whitelist := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ~!@$%^&*_-+=<>.?/"
	for i := 0; i < 256; i++ {
		if IsSMT2SimpleSymbolChar(byte(i)) != strings.ContainsRune(whitelist, rune(i)) {
			t.Fatalf("byte %d", i)
		}
	}
}
func TestSMT2QuotedCStringSourceBranches(t *testing.T) {
	if IsSMT2QuotedCString(nil) {
		t.Fatal("null")
	}
	cases := []struct {
		s    string
		want bool
	}{{"", false}, {"true", false}, {"a9", false}, {"9a", true}, {"a b", true}, {"|", true}, {"||", false}, {"|a b|", false}, {"|a|b|", true}, {"|a\\|b|", false}, {"|a\\\\b|", false}, {"|a\\|", true}, {"|a\\b|", true}, {"a\x00 b", false}, {"\xff", true}, {"|\n\xff|", false}}
	for _, c := range cases {
		if got := IsSMT2QuotedCString(&c.s); got != c.want {
			t.Fatalf("%q got%v", c.s, got)
		}
	}
	// Independent finite oracle: recognize an already wrapped spelling by consuming
	// valid escape pairs, otherwise use the simple-byte membership set.
	alphabet := "a9 |\\\n"
	for a := 0; a < len(alphabet); a++ {
		for b := 0; b < len(alphabet); b++ {
			for c := 0; c < len(alphabet); c++ {
				s := string([]byte{alphabet[a], alphabet[b], alphabet[c]})
				want := s[0] >= '0' && s[0] <= '9'
				if !want {
					if s[0] == '|' && s[2] == '|' {
						want = s[1] == '|' || s[1] == '\\'
					} else {
						want = strings.ContainsAny(s, " |\\\n")
					}
				}
				if IsSMT2QuotedCString(&s) != want {
					t.Fatalf("model %q", s)
				}
			}
		}
	}
	if n := testing.AllocsPerRun(100, func() { s := "|a\\|b|"; IsSMT2QuotedCString(&s) }); n != 0 {
		t.Fatal(n)
	}
}
func TestSMT2SymbolSourcePreconditionAndLifetime(t *testing.T) {
	m, e := NewSymbolManager(SymbolConfig{})
	if e != nil {
		t.Fatal(e)
	}
	foreign, _ := NewSymbolManager(SymbolConfig{})
	defer foreign.Close()
	s, _ := m.Intern("a |\\\xff")
	want := "|a \\|\\\\\xff|"
	got, e := MkSMT2QuotedSymbol(m, s, nil)
	if e != nil || got != want {
		t.Fatalf("%q %v", got, e)
	}
	for _, v := range []Symbol{{}, NumericalSymbol(5)} {
		q, e := IsSMT2QuotedSymbol(m, v)
		if e != nil || q {
			t.Fatal(q, e)
		}
		if _, e = MkSMT2QuotedSymbol(m, v, nil); !errors.Is(e, ErrSMT2NeedsQuoting) {
			t.Fatal(e)
		}
	}
	simple, _ := m.Intern("simple")
	zero := arena.NewBudget(0)
	if _, e = MkSMT2QuotedSymbol(m, simple, zero); !errors.Is(e, ErrSMT2NeedsQuoting) || zero.Used() != 0 {
		t.Fatal(e)
	}
	marked, _ := s.mark()
	for _, v := range []Symbol{marked, DummySymbol()} {
		if _, e = IsSMT2QuotedSymbol(m, v); !errors.Is(e, ErrInvalidSymbol) {
			t.Fatal(e)
		}
	}
	if _, e = IsSMT2QuotedSymbol(foreign, s); !errors.Is(e, ErrSymbolOwner) {
		t.Fatal(e)
	}
	if _, e = IsSMT2QuotedSymbol(nil, s); !errors.Is(e, ErrInvalidSymbol) {
		t.Fatal(e)
	}
	stale := s
	stale.generation++
	if _, e = IsSMT2QuotedSymbol(m, stale); e == nil {
		t.Fatal("stale")
	}
	m.Close()
	if got != want {
		t.Fatal("output lifetime")
	}
	if _, e = IsSMT2QuotedSymbol(m, s); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestSMT2QuoteFaultBudgetAndIndependentBytes(t *testing.T) {
	ownerBudget := arena.NewBudget(^uint64(0))
	m, _ := NewSymbolManager(SymbolConfig{Budget: ownerBudget})
	defer m.Close()
	for _, n := range []int{61, 62, 63, 64, 65, 200} {
		raw := strings.Repeat("x", n) + "|\\\xff"
		s, _ := m.Intern(raw)
		want := "|" + strings.ReplaceAll(strings.ReplaceAll(raw, "\\", "\\\\"), "|", "\\|") + "|"
		success := false
		for fail := int64(0); fail < 8; fail++ {
			b := arena.NewBudget(^uint64(0))
			b.FailAfter(fail)
			got, e := MkSMT2QuotedSymbol(m, s, b)
			if b.Used() != 0 {
				t.Fatal("leak", n, fail, b.Used())
			}
			if e == nil {
				if got != want {
					t.Fatalf("%q", got)
				}
				success = true
				break
			}
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(e)
			}
		}
		if !success {
			t.Fatal("no success")
		}
	}
	// 63 bytes of output require only owner storage, 64 additionally expands in CStr.
	owner, _ := NewStringBuffer64(StringBufferConfig{})
	header := owner.Stats().OwnerBytes
	owner.Close()
	for _, n := range []int{61, 62} {
		raw := " " + strings.Repeat("x", n-1)
		s, _ := m.Intern(raw)
		b := arena.NewBudget(header)
		_, e := MkSMT2QuotedSymbol(m, s, b)
		if (n == 61) != (e == nil) {
			t.Fatal(n, e)
		}
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
}

func TestSMT2QuoteExactGrowthPeakAndNoBorrowCharges(t *testing.T) {
	ownerBudget := arena.NewBudget(^uint64(0))
	m, _ := NewSymbolManager(SymbolConfig{Budget: ownerBudget})
	defer m.Close()
	s, _ := m.Intern(" " + strings.Repeat("x", 125))
	baseline := ownerBudget.Used()
	b, _ := NewStringBuffer64(StringBufferConfig{})
	header := b.Stats().OwnerBytes
	b.Close()
	for _, extra := range []uint64{383, 384} {
		budget := arena.NewBudget(header + extra)
		out, e := MkSMT2QuotedSymbol(m, s, budget)
		if extra == 383 {
			if !errors.Is(e, arena.ErrBudget) || out != "" {
				t.Fatal(e, out)
			}
		} else if e != nil || len(out) != 128 {
			t.Fatal(e, len(out))
		}
		if budget.Used() != 0 || ownerBudget.Used() != baseline {
			t.Fatal("ownership", budget.Used(), ownerBudget.Used(), baseline)
		}
	}
	if n := testing.AllocsPerRun(100, func() {
		q, e := IsSMT2QuotedSymbol(m, s)
		if !q || e != nil {
			panic("predicate")
		}
	}); n != 0 {
		t.Fatal(n)
	}
}
