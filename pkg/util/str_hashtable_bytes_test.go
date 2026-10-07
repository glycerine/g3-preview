// Licensed under the MIT license.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestStrHashtableBytesCStringFixturesAndBorrow(t *testing.T) {
	h, r := newSH(t)
	defer r.Close()
	defer h.Close()
	for _, s := range []string{"", "ascii", "\x80\xfftail", "prefix"} {
		k := strStore(t, h, s, 0)
		if _, _, e := h.InsertIfNotThereCore(k); e != nil {
			t.Fatal(e)
		}
		input := append([]byte(s), 0, 'x', 'y')
		entry, yes, e := h.FindCStringBytes(input)
		if e != nil || !yes {
			t.Fatal(s, yes, e)
		}
		got, e := entry.Key()
		if e != nil || got != k {
			t.Fatal(got, k, e)
		}
		hash, e := CStringBytesHash(input)
		if e != nil || hash != String(s, 17) {
			t.Fatal("existing frozen source byte hash", s, hash, e)
		}
		if len(s) != 0 {
			input[0] ^= 1
			_, yes, e = h.FindCStringBytes(input)
			if e != nil || yes {
				t.Fatal("caller buffer retained/mutated stored key", s, yes, e)
			}
		}
	}
	if _, yes, e := h.FindCStringBytes(nil); e != nil || !yes {
		t.Fatal("nil query is empty CString", yes, e)
	}
}

func TestStrHashtableBytesPrepareGrowthAndEpoch(t *testing.T) {
	r, e := arena.NewBytesRegion(arena.BytesRegionConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	b := arena.NewBudget(1 << 20)
	h, e := NewStrHashtable(r, StrHashtableConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	defer h.Close()
	var first string
	for i := 0; i < 7; i++ {
		s := string([]byte{'a' + byte(i)})
		if i == 0 {
			first = s
		}
		k := strStore(t, h, s, 0)
		if _, _, e = h.InsertIfNotThereCore(k); e != nil {
			t.Fatal(e)
		}
	}
	old := h.Stats()
	b.FailAfter(0)
	if _, _, e = h.PrepareFindCStringBytes([]byte(first)); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("source grows before duplicate", e, old, h.Stats())
	}
	if h.Stats() != old {
		t.Fatal("failed table reserve changed graph")
	}
	b.FailAfter(-1)
	if _, yes, e := h.PrepareFindCStringBytes([]byte(first)); e != nil || !yes {
		t.Fatal(yes, e)
	}
	h.epoch = ^uint64(0)
	if _, _, e = h.PrepareFindCStringBytes([]byte("missing")); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("miss must preflight publication before spelling allocation", e)
	}
	if _, yes, e := h.PrepareFindCStringBytes([]byte(first)); e != nil || !yes {
		t.Fatal("hit at exhausted epoch", yes, e)
	}
}

func TestStrHashtableBytesPermanentHashFixtures(t *testing.T) {
	var input [25]byte
	for i := range input {
		input[i] = byte(i*19 + 127)
	}
	for n, want := range sourceVectors(t).String {
		got := strQueryBytesHash(input[:n], 123)
		if got != want {
			t.Fatal("independent frozen C++ byte vector", n, got, want)
		}
	}
}

func TestStrHashtableBytesLongWarmQueryZeroAllocations(t *testing.T) {
	h, r := newSH(t)
	defer r.Close()
	defer h.Close()
	spelling := "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789"
	k := strStore(t, h, spelling, 0)
	if _, _, e := h.InsertIfNotThereCore(k); e != nil {
		t.Fatal(e)
	}
	input := []byte(spelling)
	var failure error
	var found bool
	n := testing.AllocsPerRun(100, func() { _, found, failure = h.PrepareFindCStringBytes(input) })
	if n != 0 || failure != nil || !found {
		t.Fatal("warmed borrowed long-name lookup", n, found, failure)
	}
}
