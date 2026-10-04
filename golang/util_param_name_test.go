// Licensed under the MIT license.
// Source: src/util/params.cpp norm_param_name; independent finite fixtures.
package g3

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func TestParamNameNormalizationSourceBranches(t *testing.T) {
	if got := NormParamName(nil); got != "_" {
		t.Fatal(got)
	}
	for _, pair := range [][2]string{
		{"", "_"}, {":", "_"}, {"::", "_"}, {"::ABC", "_abc"},
		{":MAX-MEMORY", "max_memory"}, {"a:b-c", "a_b_c"}, {" A.B ", " a.b "},
		{"ÄÉ-Z", "ÄÉ_z"}, {"_", "_"}, {"\x00:ABC", "_"},
		{":A\x00TAIL", "a"}, {":\x00tail", "_"},
	} {
		input := pair[0]
		if got := NormParamName(&input); got != pair[1] {
			t.Fatalf("%q -> %q, want %q", input, got, pair[1])
		}
	}
}

func TestParamNameNormalizationByteDomain(t *testing.T) {
	for value := 1; value < 256; value++ {
		b := byte(value)
		want := b
		if b >= 'A' && b <= 'Z' {
			want = b | 0x20
		} else if b == ':' || b == '-' {
			want = '_'
		}
		input := string([]byte{'a', b, 'z'})
		if got := NormParamName(&input); got != string([]byte{'a', want, 'z'}) {
			t.Fatalf("byte %02x: %q", b, got)
		}
	}
	for _, text := range []string{"Name", ":A-B:C", "ÄÉ", ""} {
		for suffix := 0; suffix < 256; suffix++ {
			input := text + "\x00" + string(byte(suffix))
			if got := NormParamName(&input); got != NormParamName(&text) {
				t.Fatalf("C-string suffix affected %q", input)
			}
		}
	}
}

func TestParamNameSymbolOwnershipAndSourceRoutes(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	m := symbolManagerTest(t, budget, false)
	other := symbolManagerTest(t, nil, false)
	for _, pair := range [][2]string{{":MAX-MEMORY", "max_memory"}, {"", "_"}, {"ÄÉ-A:B", "ÄÉ_a_b"}} {
		symbol := internSymbol(t, m, pair[0])
		before := m.Stats()
		used := budget.Used()
		budget.FailAfter(0)
		got, err := NormParamSymbolName(m, symbol)
		budget.FailAfter(-1)
		if err != nil || got != pair[1] || m.Stats() != before || budget.Used() != used {
			t.Fatal(pair, got, err, "normalization changed the symbol owner")
		}
	}
	if got, err := NormParamSymbolName(m, Symbol{}); err != nil || got != "_" {
		t.Fatal(got, err)
	}
	for _, invalid := range []Symbol{NumericalSymbol(1), DummySymbol()} {
		if _, err := NormParamSymbolName(m, invalid); !errors.Is(err, ErrInvalidSymbol) {
			t.Fatal(invalid, err)
		}
	}
	foreign := internSymbol(t, other, "FOREIGN")
	if _, err := NormParamSymbolName(m, foreign); !errors.Is(err, ErrSymbolOwner) {
		t.Fatal(err)
	}
	if _, err := NormParamSymbolName(nil, Symbol{}); !errors.Is(err, ErrInvalidSymbol) {
		t.Fatal(err)
	}
	serviceSymbol := internSymbol(t, m, ":LIFETIME-NAME")
	output, err := NormParamSymbolName(m, serviceSymbol)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil || budget.Used() != 0 {
		t.Fatal(err, budget.Used())
	}
	if output != "lifetime_name" {
		t.Fatal("caller output depends on closed symbol owner")
	}
	if _, err := NormParamSymbolName(m, serviceSymbol); !errors.Is(err, arena.ErrClosed) {
		t.Fatal(err)
	}
}

func TestParamNameSymbolSingleOutputAllocation(t *testing.T) {
	m := symbolManagerTest(t, nil, false)
	symbol := internSymbol(t, m, ":"+strings.Repeat("UPPER-NAME", 40))
	var output string
	var err error
	allocations := testing.AllocsPerRun(100, func() { output, err = NormParamSymbolName(m, symbol) })
	if err != nil || output != strings.Repeat("upper_name", 40) || allocations != 1 {
		t.Fatal(err, allocations, len(output))
	}
}
