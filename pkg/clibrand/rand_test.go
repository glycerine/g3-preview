package clibrand

// Source dependency: src/math/polynomial/polynomial.cpp:4053-4068;
// src/test/lp.cpp:1459.

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

//go:embed test_vectors/glibc31.json
var reference embed.FS

type vector struct {
	Seed   uint32   `json:"seed"`
	Values []uint32 `json:"values"`
}
type fixtures struct {
	Profile string   `json:"profile"`
	RandMax uint32   `json:"rand_max"`
	Default []uint32 `json:"default"`
	Seeds   []vector `json:"seeds"`
}

func load(t *testing.T) fixtures {
	t.Helper()
	b, e := reference.ReadFile("test_vectors/glibc31.json")
	if e != nil {
		t.Fatal(e)
	}
	var f fixtures
	if e = json.Unmarshal(b, &f); e != nil {
		t.Fatal(e)
	}
	if f.Profile != "glibc31" || f.RandMax != 2147483647 || len(f.Seeds) != 262 || len(f.Default) != 128 {
		t.Fatal("reference profile", f.Profile, f.RandMax, len(f.Seeds), len(f.Default))
	}
	return f
}
func TestGLibc31CapturedLibrarySeeds(t *testing.T) {
	f := load(t)
	for _, v := range f.Seeds {
		t.Run(fmt.Sprint(v.Seed), func(t *testing.T) {
			var g GLibc31
			g.Seed(v.Seed)
			for i, want := range v.Values {
				got := g.Next()
				if got != want {
					t.Fatalf("seed %d draw %d: got %d want %d", v.Seed, i, got, want)
				}
				if got > RandMax {
					t.Fatal("negative C int profile", got)
				}
			}
		})
	}
}
func TestGLibc31ZeroValueAndSeedZero(t *testing.T) {
	f := load(t)
	var unseeded, zero, one GLibc31
	zero.Seed(0)
	one.Seed(1)
	for i, want := range f.Default {
		for label, g := range map[string]*GLibc31{"unseeded": &unseeded, "zero": &zero, "one": &one} {
			if got := g.Next(); got != want {
				t.Fatal(label, i, got, want)
			}
		}
	}
}
func TestGLibc31ReseedAndIndependentState(t *testing.T) {
	f := load(t)
	var a, b GLibc31
	a.Seed(17)
	b.Seed(17)
	for i := 0; i < 83; i++ {
		if a.Next() != b.Next() {
			t.Fatal("independent identical seed", i)
		}
	}
	a.Seed(0x80000000)
	for _, v := range f.Seeds {
		if v.Seed == 0x80000000 {
			for i, want := range v.Values {
				if got := a.Next(); got != want {
					t.Fatal("full reseed", i, got, want)
				}
			}
		}
	}
	for i := 83; i < 128; i++ {
		if got := b.Next(); got != f.Seeds[17].Values[i] {
			t.Fatal("reseed mutated peer", i, got)
		}
	}
}
func TestProcessRandDefaultWithoutSrand(t *testing.T) {
	if os.Getenv("G3_CRAND_DEFAULT_CHILD") == "1" {
		for i, want := range load(t).Default {
			if got := Rand(); got != want {
				t.Fatal("fresh process", i, got, want)
			}
		}
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestProcessRandDefaultWithoutSrand$")
	cmd.Env = append(os.Environ(), "G3_CRAND_DEFAULT_CHILD=1")
	if out, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("fresh compiled Go process: %v\n%s", e, out)
	}
}
func TestProcessRandBorrowedDrawsShareSrand(t *testing.T) {
	f := load(t)
	drawA, drawB := Rand, Rand
	Srand(17)
	for i, want := range f.Seeds[17].Values {
		draw := drawA
		if i%3 == 0 {
			draw = drawB
		}
		if got := draw(); got != want {
			t.Fatal("borrowed global stream", i, got, want)
		}
	}
	Srand(2)
	for i, want := range f.Seeds[2].Values {
		if got := drawB(); got != want {
			t.Fatal("borrowed stream sees reseed", i, got, want)
		}
	}
	t.Cleanup(func() { Srand(1) })
}
func TestGLibc31DrawsAllocateNoStorage(t *testing.T) {
	var g GLibc31
	g.Seed(3)
	if n := testing.AllocsPerRun(1000, func() { g.Next() }); n != 0 {
		t.Fatal("fixed-state draw allocations", n)
	}
	Srand(3)
	if n := testing.AllocsPerRun(1000, func() { Rand() }); n != 0 {
		t.Fatal("shared-state draw allocations", n)
	}
	t.Cleanup(func() { Srand(1) })
}
