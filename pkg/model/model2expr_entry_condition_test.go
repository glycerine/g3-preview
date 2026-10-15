package model

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestModel2ExprEntryConditionSourceArityAndReferenceScope(t *testing.T) {
	b := arena.NewBudget(1 << 30)
	sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	if e != nil {
		t.Fatal(e)
	}
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := m.Close(); e != nil {
			t.Fatal(e)
		}
		if e := q.Close(); e != nil {
			t.Fatal(e)
		}
		if e := sy.Close(); e != nil {
			t.Fatal(e)
		}
		if b.Used() != 0 {
			t.Fatal("retained entry-condition owner", b.Used())
		}
	})
	yes, e := m.MkTrue()
	if e != nil {
		t.Fatal(e)
	}
	no, e := m.MkFalse()
	if e != nil {
		t.Fatal(e)
	}
	sort, e := m.MkBoolSort()
	if e != nil {
		t.Fatal(e)
	}
	for _, args := range [][]ast.ASTHandle{nil, {yes}, {yes, no}} {
		fi, e := NewFunctionInterpretation(m, uint32(len(args)))
		if e != nil {
			t.Fatal(e)
		}
		if e = fi.InsertEntry(args, yes); e != nil {
			t.Fatal(e)
		}
		out, e := m.NewRef(ast.ASTHandle{})
		if e != nil {
			t.Fatal(e)
		}
		frame, e := newModelEntryCondition(m, uint32(len(args)), fi.entries[0], out)
		if e != nil {
			t.Fatal(e)
		}
		got, e := m.RefGet(out)
		if e != nil {
			t.Fatal(e)
		}
		var want ast.ASTHandle
		if len(args) == 0 {
			want = yes
		} else {
			terms := make([]ast.ASTHandle, len(args))
			for i, arg := range args {
				v, e := m.MkVar(uint32(i), sort)
				if e != nil {
					t.Fatal(e)
				}
				terms[i], e = m.MkEq(v, arg)
				if e != nil {
					t.Fatal(e)
				}
			}
			if len(args) == 1 {
				want = terms[0]
			} else {
				want, e = m.MkAndArgs(terms)
				if e != nil {
					t.Fatal(e)
				}
			}
		}
		if got != want {
			t.Fatal("entry condition", len(args), got, want)
		}
		if e = frame.Close(); e != nil {
			t.Fatal(e)
		}
		if e = m.DelRef(&out); e != nil {
			t.Fatal(e)
		}
		if e = fi.Close(); e != nil {
			t.Fatal(e)
		}
	}
}
