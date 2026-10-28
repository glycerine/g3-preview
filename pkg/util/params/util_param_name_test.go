package params

// Source: src/util/params.cpp norm_param_name; independent finite fixtures.

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
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
	if got, err := NormParamSymbolName(m, usymbol.Symbol{}); err != nil || got != "_" {
		t.Fatal(got, err)
	}
	for _, numerical := range []usymbol.Symbol{usymbol.Numerical(0), usymbol.Numerical(1)} {
		got, err := NormParamSymbolName(m, numerical)
		if GlobalParamsSourceDebug {
			if !errors.Is(err, usymbol.ErrInvalid) {
				t.Fatal(numerical, err)
			}
		} else if err != nil || got != "_" {
			t.Fatal(numerical, got, err)
		}
	}
	if _, err := NormParamSymbolName(m, usymbol.Dummy()); !errors.Is(err, usymbol.ErrInvalid) {
		t.Fatal(err)
	}
	foreign := internSymbol(t, other, "FOREIGN")
	if _, err := NormParamSymbolName(m, foreign); !errors.Is(err, usymbol.ErrOwner) {
		t.Fatal(err)
	}
	if _, err := NormParamSymbolName(nil, usymbol.Symbol{}); !errors.Is(err, usymbol.ErrInvalid) {
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

func TestParamNameNumericalSymbolWithAssertionsDisabled(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	if err != nil {
		t.Fatal(err)
	}
	d, err := diagnostic.NewDebugContext(region, b)
	if err != nil {
		t.Fatal(err)
	}
	m, err := usymbol.NewManager(usymbol.Config{Budget: b, Debug: d})
	if err != nil {
		t.Fatal(err)
	}
	if err = d.EnableAssertions(false); err != nil {
		t.Fatal(err)
	}
	for _, s := range []usymbol.Symbol{usymbol.Numerical(0), usymbol.Numerical(7)} {
		got, err := NormParamSymbolName(m, s)
		if err != nil || got != "_" {
			t.Fatal(s, got, err)
		}
	}
	if err = m.Close(); err != nil {
		t.Fatal(err)
	}
	if err = d.Close(); err != nil {
		t.Fatal(err)
	}
	if err = region.Close(); err != nil || b.Used() != 0 {
		t.Fatal(err, b.Used())
	}
}
