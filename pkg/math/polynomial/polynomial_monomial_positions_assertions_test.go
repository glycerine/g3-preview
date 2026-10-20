package polynomial

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMonomialPositionsSourceAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			choices := action + action
			if action == "disabled" {
				choices = ""
			}
			output := polynomialAssertionActionsFixture(t, choices)
			if polynomialSourceDebug && action == "disabled" {
				serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
			}
			m, p, xs := positionFixture(t)
			before := m.config.Budget.Used()
			reset := p.Reset(xs[0])
			want := arena.ErrBounds
			throwing := polynomialSourceDebug && action == "T"
			if throwing {
				want = ErrMonomialAssertion
			}
			if !errors.Is(reset, want) || len(p.values) != 0 {
				t.Fatal("reset source gate/native boundary", reset, len(p.values))
			}
			serviceMust(t, p.Set(xs[0], 3))
			duplicate := p.Set(xs[0], 5)
			if throwing {
				if !errors.Is(duplicate, ErrMonomialAssertion) {
					t.Fatal(duplicate)
				}
			} else {
				serviceMust(t, duplicate)
			}
			value, e := p.Get(xs[0])
			serviceMust(t, e)
			expected := uint32(5)
			if throwing {
				expected = 3
			}
			if value != expected || m.config.Budget.Used() != before {
				t.Fatal(value, expected, m.config.Budget.Used(), before)
			}
			if polynomialSourceDebug {
				_, e = output.Seek(0, 0)
				serviceMust(t, e)
				text, e := io.ReadAll(output)
				serviceMust(t, e)
				if action == "disabled" {
					if len(text) != 0 {
						t.Fatal(string(text))
					}
				} else {
					if !strings.Contains(string(text), "Line: 467\nid < m_m2pos.size()") || !strings.Contains(string(text), "Line: 474\nm_m2pos[id] == UINT_MAX") {
						t.Fatal(string(text))
					}
				}
			}
		})
	}
}

func TestMonomialPositionsClosedAssertionOwnerPaths(t *testing.T) {
	m, p, xs := positionFixture(t)
	sentinel, e := p.Get(xs[0])
	serviceMust(t, e)
	if sentinel != ^uint32(0) {
		t.Fatal(sentinel)
	}
	length := len(p.values)
	growing, e := NewMonomialPositions(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, growing.CloseChecked()) })
	id, e := growing.id(xs[7])
	serviceMust(t, e)
	before := m.config.Budget.Used()
	polynomialClosedCellAssertionContext(t)
	set := p.Set(xs[0], 9)
	reset := p.Reset(xs[0])
	if polynomialSourceDebug {
		if !errors.Is(set, arena.ErrClosed) || !errors.Is(reset, arena.ErrClosed) {
			t.Fatal(set, reset)
		}
	} else {
		serviceMust(t, set)
		serviceMust(t, reset)
	}
	value, e := p.Get(xs[0])
	serviceMust(t, e) // Source get has no process assertion.
	if value != sentinel || len(p.values) != length || m.config.Budget.Used() != before {
		t.Fatal("source gates changed map storage", value, len(p.values))
	}
	// Source reserve(id+1,UINT_MAX) precedes the assertion switch. A closed
	// switch owner must still leave the actual native slice prefix published.
	growth := growing.Set(xs[7], 11)
	if polynomialSourceDebug {
		if !errors.Is(growth, arena.ErrClosed) {
			t.Fatal(growth)
		}
	} else {
		serviceMust(t, growth)
	}
	if uint64(len(growing.values)) != uint64(id)+1 || m.config.Budget.Used() != before {
		t.Fatal("source reserve did not precede gate", len(growing.values), id)
	}
	for i, position := range growing.values {
		want := sentinel
		if !polynomialSourceDebug && uint32(i) == id {
			want = 11
		}
		if position != want {
			t.Fatal(i, position, want)
		}
	}
}
