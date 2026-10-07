// Source: src/util/hashtable.h:581-600; src/util/str_hashtable.h:24-31.
package util

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestStrHashtableDumpIntoActualFieldFormattingAndCallerWidth(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	paramsMust(t, e)
	region := traceRegion(t, b)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
	paramsMust(t, e)
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	paramsMust(t, e)
	defer func() {
		paramsMust(t, out.Close())
		paramsMust(t, h.CloseChecked())
		paramsMust(t, region.Close())
		paramsMust(t, r.Close())
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
func TestStrHashtableDumpIntoClosedFileStillReachesActualBorrowedCString(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	paramsMust(t, e)
	region := traceRegion(t, b)
	foreign := traceRegion(t, b)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
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
		paramsMust(t, r.Close())
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

func strHashtableOutputPhysical(t *testing.T, b *arena.Budget, r *Runtime, region *arena.BytesRegion, h *StrHashtable, out *NumericTextStream) {
	t.Helper()
	rr, e := h.RuntimeRoots()
	paramsMust(t, e)
	roots := map[Block]bool{}
	for _, token := range append([]Block{rr.Object, rr.Table}, rr.Acquisitions...) {
		if token == (Block{}) {
			continue
		}
		if roots[token] {
			t.Fatal("duplicate root", token)
		}
		roots[token] = true
		_, e = r.Size(token)
		paramsMust(t, e)
	}
	rs := r.Stats()
	if uint32(len(roots)) != rs.LiveBlocks {
		t.Fatal("actual raw bijection", roots, rs)
	}
	hs := h.Stats()
	s := region.Stats()
	d := s.Descriptors
	ss := out.Stats()
	want := rs.TotalBytes + hs.OwnerBytes + hs.DirectoryBytes + rr.AcquisitionBytes + s.OwnerBytes + s.PayloadBytes + s.PageHeaderBytes + s.DirectoryBytes + s.LogBytes + s.ScopeBytes + d.OwnerBytes + d.RecordBytes + d.MetadataBytes + d.DirectoryBytes + ss.OwnerBytes + ss.ScratchBytes + ss.OutputOwnerBytes + ss.OutputOverflowBytes
	if b.Used() != want {
		t.Fatal("exact named cold and physical graph", b.Used(), want)
	}
}
func TestStrHashtableDumpIntoEveryOutputReservationAndExactPrefix(t *testing.T) {
	success := -1
	for ordinal := int64(0); ordinal < 8; ordinal++ {
		b := arena.NewBudget(1 << 24)
		r, e := NewRuntime(RuntimeConfig{Budget: b})
		paramsMust(t, e)
		region := traceRegion(t, b)
		h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
		paramsMust(t, e)
		text := strings.Repeat("x", 300)
		paramsMust(t, h.Insert(debugTag(t, region, text)))
		out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
		paramsMust(t, e)
		b.FailAfter(ordinal)
		e = h.DumpInto(out)
		strHashtableOutputPhysical(t, b, r, region, h, out)
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
		paramsMust(t, r.Close())
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
func TestStrHashtableDumpIntoActualFileAndEmptyBorrowErasure(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	paramsMust(t, e)
	region := traceRegion(t, b)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
	paramsMust(t, e)
	f, e := os.CreateTemp(".", "str-table-output-*")
	paramsMust(t, e)
	defer os.Remove(f.Name())
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b, BorrowedFile: f})
	paramsMust(t, e)
	paramsMust(t, region.Close())
	b.FailAfter(0)
	paramsMust(t, h.DumpInto(out))
	strHashtableOutputPhysical(t, b, r, region, h, out)
	got, e := os.ReadFile(f.Name())
	paramsMust(t, e)
	if string(got) != "[]" {
		t.Fatal(string(got))
	}
	b.FailAfter(-1)
	paramsMust(t, out.Close())
	paramsMust(t, f.Close())
	paramsMust(t, h.CloseChecked())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestStrHashtableDumpIntoBorrowedSpellingRereadAndFileBytes(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	paramsMust(t, e)
	region := traceRegion(t, b)
	h, e := NewStrHashtable(region, StrHashtableConfig{Budget: b, Runtime: r})
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
	strHashtableOutputPhysical(t, b, r, region, h, out)
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
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
