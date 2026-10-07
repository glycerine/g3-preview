// Source: src/util/hashtable.h:581-600; src/util/str_hashtable.h:24-31.
// Source author: Leonardo de Moura (2006-09-11; 2006-09-13).
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestStrHashtableLegacyStatisticsDumpIntoActualFieldFormattingAndCallerWidth(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	region := traceRegion(t, b)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Statistics: true})
	paramsMust(t, e)
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	defer func() {
		paramsMust(t, out.Close())
		paramsMust(t, h.CloseChecked())
		paramsMust(t, region.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	var names []string
	for n := 0; len(names) < 3; n++ {
		text := fmt.Sprintf("dump-%d", n)
		if String(text, 17)&7 != 0 {
			continue
		}
		names = append(names, text)
		paramsMust(t, h.Insert(debugTag(t, region, text)))
	}
	paramsMust(t, out.SetWidth(4))
	paramsMust(t, out.SetFill('.'))
	paramsMust(t, h.DumpInto(out))
	got, e := out.String()
	paramsMust(t, e)
	want := "...[" + strings.Join(names, " ") + "]"
	if got != want || out.State().Width != 0 {
		t.Fatal(got, want, out.State())
	}
}
func TestStrHashtableLegacyStatisticsDumpIntoClosedFileStillReachesActualBorrowedCString(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	region := traceRegion(t, b)
	foreign := traceRegion(t, b)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Statistics: true})
	paramsMust(t, e)
	paramsMust(t, h.InsertFrom(foreign, debugTag(t, foreign, "late-borrow")))
	f, e := os.CreateTemp(".", "str-table-output-*")
	paramsMust(t, e)
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b, BorrowedFile: f})
	paramsMust(t, e)
	defer func() {
		paramsMust(t, out.Close())
		paramsMust(t, h.CloseChecked())
		paramsMust(t, region.Close())
		paramsMust(t, foreign.Close())
		os.Remove(f.Name())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	paramsMust(t, f.Close())
	paramsMust(t, foreign.Close())
	e = h.DumpInto(out)
	if e == nil {
		t.Fatal("badbit suppressed reached strlen lifetime")
	}
	bad, re := out.Bad()
	paramsMust(t, re)
	if !bad {
		t.Fatal("opening insertion did not reach actual file failure")
	}
}

func strHashtableLegacyStatisticsOutputPhysical(t *testing.T, b *arena.Budget, region *arena.BytesRegion, h *StrHashtable, out *NumericTextStream) {
	t.Helper()
	hs := h.Stats()
	s := region.Stats()
	d := s.Descriptors
	ss := out.Stats()
	want := hs.OwnerBytes + hs.TableBytes + hs.DirectoryBytes + hs.RetainedBytes + s.OwnerBytes + s.PayloadBytes + s.PageHeaderBytes + s.DirectoryBytes + s.LogBytes + s.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes + ss.OwnerBytes + ss.ScratchBytes + ss.OutputOwnerBytes + ss.OutputOverflowBytes
	if b.Used() != want {
		t.Fatal("named real legacy field/table and output graph", b.Used(), want)
	}
}
func TestStrHashtableLegacyStatisticsDumpIntoEveryOutputReservationAndExactPrefix(t *testing.T) {
	success := -1
	for ordinal := int64(0); ordinal < 8; ordinal++ {
		b := arena.NewBudget(1 << 24)
		region := traceRegion(t, b)
		h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Statistics: true})
		paramsMust(t, e)
		text := strings.Repeat("x", 300)
		paramsMust(t, h.Insert(debugTag(t, region, text)))
		out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
		paramsMust(t, e)
		b.FailAfter(ordinal)
		beforeCount := h.CollisionCount()
		e = h.DumpInto(out)
		if h.CollisionCount() != beforeCount {
			t.Fatal("display must not change source collision field")
		}
		strHashtableLegacyStatisticsOutputPhysical(t, b, region, h, out)
		n := out.Buffer.Position()
		var got strings.Builder
		for i := uint64(0); i < n; i++ {
			c, re := out.Buffer.Read(i)
			paramsMust(t, re)
			got.WriteByte(c)
		}
		want := "[" + text + "]"
		if e == nil {
			success = int(ordinal)
			if got.String() != want {
				t.Fatal(got.String(), want)
			}
		} else {
			length := uint64(64) << uint(ordinal)
			if n != length || got.String() != want[:length] {
				t.Fatal("literal byte publication before expansion refusal", ordinal, n, got.String())
			}
		}
		b.FailAfter(-1)
		paramsMust(t, out.Close())
		paramsMust(t, h.CloseChecked())
		paramsMust(t, region.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
		if e == nil {
			break
		}
	}
	if success != 3 {
		t.Fatal("all actual output growth frontiers", success)
	}
}
func TestStrHashtableLegacyStatisticsDumpIntoActualFileAndEmptyBorrowErasure(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	region := traceRegion(t, b)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Statistics: true})
	paramsMust(t, e)
	f, e := os.CreateTemp(".", "str-table-output-*")
	paramsMust(t, e)
	defer os.Remove(f.Name())
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b, BorrowedFile: f})
	paramsMust(t, e)
	paramsMust(t, region.Close())
	b.FailAfter(0)
	paramsMust(t, h.DumpInto(out))
	strHashtableLegacyStatisticsOutputPhysical(t, b, region, h, out)
	got, e := os.ReadFile(f.Name())
	paramsMust(t, e)
	if string(got) != "[]" {
		t.Fatal(string(got))
	}
	b.FailAfter(-1)
	paramsMust(t, out.Close())
	paramsMust(t, f.Close())
	paramsMust(t, h.CloseChecked())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestStrHashtableLegacyStatisticsDumpIntoBorrowedSpellingRereadAndFileBytes(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	region := traceRegion(t, b)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Statistics: true})
	paramsMust(t, e)
	key := debugTag(t, region, "abcdef")
	paramsMust(t, h.Insert(key))
	paramsMust(t, region.Write(key.Allocation(), key.Offset()+2, 0))
	f, e := os.CreateTemp(".", "str-table-output-*")
	paramsMust(t, e)
	defer os.Remove(f.Name())
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b, BorrowedFile: f})
	paramsMust(t, e)
	paramsMust(t, out.SetAlignment(NumericLeft))
	paramsMust(t, out.SetWidth(3))
	paramsMust(t, out.SetFill('_'))
	b.FailAfter(0)
	paramsMust(t, h.DumpInto(out))
	strHashtableLegacyStatisticsOutputPhysical(t, b, region, h, out)
	got, e := os.ReadFile(f.Name())
	paramsMust(t, e)
	if string(got) != "[__ab]" {
		t.Fatal("strlen ignores cached key length and retains separate field insertion", string(got))
	}
	b.FailAfter(-1)
	paramsMust(t, out.Close())
	paramsMust(t, f.Close())
	paramsMust(t, h.CloseChecked())
	paramsMust(t, region.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
