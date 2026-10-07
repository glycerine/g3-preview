package util

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestEscapedThirteenSourcePrintedFixtures(t *testing.T) {
	cases := []struct {
		in   *string
		trim bool
		want string
	}{}
	add := func(s string, trim bool, w string) {
		cases = append(cases, struct {
			in   *string
			trim bool
			want string
		}{&s, trim, w})
	}
	add("\"hello\"\"world\"\n\n", false, "\\\"hello\\\"\\\"world\\\"\n\n")
	add("\"hello\"\nworld\"\n\n\n", true, "\\\"hello\\\"\nworld\\\"")
	add("\"hello\"\nworld\"\n", true, "\\\"hello\\\"\nworld\\\"")
	add("\"hello\"\nworld\"", true, "\\\"hello\\\"\nworld\\\"")
	add("\"hello\"\n\"world\"\n\n", false, "\\\"hello\\\"\n\\\"world\\\"\n\n")
	add("\n\n\n", true, "")
	add("\n\n\n", false, "\n\n\n")
	add("\n", true, "")
	add("\n", false, "\n")
	add("", true, "")
	add("", false, "")
	cases = append(cases, struct {
		in   *string
		trim bool
		want string
	}{nil, true, ""}, struct {
		in   *string
		trim bool
		want string
	}{nil, false, ""})
	if len(cases) != 13 {
		t.Fatal(len(cases))
	}
	for i, c := range cases {
		got, e := EscapedCString(c.in, c.trim, 0, nil)
		if e != nil || got != c.want {
			t.Fatalf("case%d %q %v", i, got, e)
		}
	}
}
func TestEscapedIndependentByteIndentAndTrim(t *testing.T) {
	for _, trim := range []bool{false, true} {
		for indent := uint32(0); indent < 5; indent++ {
			for code := 0; code < 256; code++ {
				in := string([]byte{byte(code), '"', '\\', '\n', 'x', '\r', '\n', '\n'})
				prefix := in
				if k := strings.IndexByte(prefix, 0); k >= 0 {
					prefix = prefix[:k]
				}
				if trim {
					prefix = strings.TrimRight(prefix, "\n")
				}
				want := strings.ReplaceAll(prefix, "\"", "\\\"")
				want = strings.ReplaceAll(want, "\n", "\n"+strings.Repeat(" ", int(indent)))
				got, e := EscapedCString(&in, trim, indent, nil)
				if e != nil || got != want {
					t.Fatalf("byte%d trim%v indent%d %q/%q %v", code, trim, indent, got, want, e)
				}
			}
		}
	}
}
func TestEscapedPublishedFailurePrefixesAndBudgets(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	dst, _ := NewStringBuffer64(StringBufferConfig{Budget: b})
	defer dst.Close()
	dst.AppendCString(strings.Repeat("x", 63))
	header := b.Used()
	b.FailAfter(0)
	q := "\""
	e := AppendEscapedCString(dst, &q, false, 0)
	if !errors.Is(e, arena.ErrInjectedFailure) || dst.Position() != 64 || b.Used() != header {
		t.Fatal(e, dst.Stats())
	}
	v, _ := dst.Read(63)
	if v != '\\' {
		t.Fatal(v)
	}
	b.FailAfter(-1)
	dst.Reset()
	dst.AppendCString(strings.Repeat("x", 61))
	newline := "\n"
	b.FailAfter(0)
	e = AppendEscapedCString(dst, &newline, false, 4)
	if !errors.Is(e, arena.ErrInjectedFailure) || dst.Position() != 64 {
		t.Fatal(e, dst.Position())
	}
	for i := uint64(62); i < 64; i++ {
		v, _ := dst.Read(i)
		if v != ' ' {
			t.Fatal(v)
		}
	}
	bounded := arena.NewBudget(header)
	finite, _ := NewStringBuffer64(StringBufferConfig{Budget: bounded})
	e = AppendEscapedCString(finite, &newline, false, ^uint32(0))
	if !errors.Is(e, arena.ErrBudget) || finite.Position() != 64 || bounded.Used() != header {
		t.Fatal("full unsigned indent prefix", e, finite.Position(), bounded.Used())
	}
	first, _ := finite.Read(0)
	last, _ := finite.Read(63)
	if first != '\n' || last != ' ' {
		t.Fatal(first, last)
	}
	finite.Close()
	for fail := int64(0); fail < 7; fail++ {
		budget := arena.NewBudget(^uint64(0))
		budget.FailAfter(fail)
		in := strings.Repeat("\"", 100)
		got, e := EscapedCString(&in, false, 0, budget)
		if budget.Used() != 0 {
			t.Fatal("leak")
		}
		if e == nil {
			if got != strings.Repeat("\\\"", 100) {
				t.Fatal(got)
			}
			break
		}
		if got != "" || !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal(got, e)
		}
	}
	if e = AppendEscapedCString(nil, nil, false, 0); !errors.Is(e, ErrStringBufferOwner) {
		t.Fatal(e)
	}
	dst.Close()
	if e = AppendEscapedCString(dst, nil, false, 0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
