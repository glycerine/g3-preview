package z3rand

import "testing"

func TestSourceRandomVectors(t *testing.T) {
	g := New(^uint32(0))
	want := [...]uint32{35, 29739, 3374, 11141, 31308, 7870, 5253, 2445, 26706, 3992}
	for i, expected := range want {
		if got := g.Next(); got != expected {
			t.Fatalf("step %d=%d want %d", i, got, expected)
		}
	}
	g.SetSeed(^uint32(0))
	if g.Bounded(7) != 0 || Max != 32767 {
		t.Fatal("bounded result or maximum changed")
	}
}

func TestShuffleTraversal(t *testing.T) {
	g := New(0)
	values := []int32{0, 1, 2, 3, 4}
	ShuffleInt32(values, g)
	want := [...]int32{0, 2, 1, 4, 3}
	for i := range want {
		if values[i] != want[i] {
			t.Fatalf("shuffle %v want %v", values, want)
		}
	}
}

func TestSourceZeroSeed(t *testing.T) {
	var g Generator
	want := [...]uint32{38, 7719, 21238, 2437, 8855, 11797, 8365, 32285, 10450, 30612}
	for i, expected := range want {
		if got := g.Next(); got != expected {
			t.Fatalf("step %d=%d want %d", i, got, expected)
		}
	}
}

func TestConcreteShuffleSourceDrawOrder(t *testing.T) {
	want := [...]int32{0, 2, 1, 4, 3}
	g := New(0)
	values := []uint32{0, 1, 2, 3, 4}
	ShuffleUint32(values, g)
	for i, expected := range want {
		if values[i] != uint32(expected) {
			t.Fatalf("uint32 shuffle %v want %v", values, want)
		}
	}
	if got := g.Next(); got != 8855 {
		t.Fatalf("shuffle consumed wrong source draws: next=%d want 8855", got)
	}
	g.SetSeed(0)
	signed := []int32{-2, -1, 0, 1, 2}
	ShuffleInt32(signed, g)
	for i, expected := range want {
		if signed[i] != expected-2 {
			t.Fatalf("int32 shuffle %v at %d", signed, i)
		}
	}
	if got := g.Next(); got != 8855 {
		t.Fatal("signed shuffle draw order", got)
	}
	for n := 0; n <= 1; n++ {
		g.SetSeed(0)
		ShuffleUint32(values[:n], g)
		if got := g.Next(); got != 38 {
			t.Fatal("empty/singleton uint32 shuffle consumed draw", n, got)
		}
		g.SetSeed(0)
		ShuffleInt32(signed[:n], g)
		if got := g.Next(); got != 38 {
			t.Fatal("empty/singleton int32 shuffle consumed draw", n, got)
		}
	}
}
