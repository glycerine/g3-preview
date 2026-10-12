// Source: src/ast/ast.cpp:2265-2309; src/util/buffer.h.
package ast

import (
	"errors"
	"strconv"
	"testing"

	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTLabelConcreteSymbolBufferAndRecognition(t *testing.T) {
	m := astTestManager(t)
	b, e := m.MkBoolSort()
	astTestMust(t, e)
	arg, e := m.MkConst(astTestSymbol(t, m, "arg"), b)
	astTestMust(t, e)
	astTestPin(t, m, arg)
	first := astTestSymbol(t, m, "first")
	second := astTestSymbol(t, m, "second")
	prefix := astTestSymbol(t, m, "prior")
	base := m.Stats().LiveParameters
	label, e := m.MkLabel(true, []usymbol.Symbol{first, second}, arg)
	astTestMust(t, e)
	astTestPin(t, m, label)
	if got := m.Stats().LiveParameters; got != base+3 {
		t.Fatal("label construction parameter copies", got, base)
	}
	names := []usymbol.Symbol{prefix}
	pos := false
	yes, e := m.IsLabel(label, &pos, &names)
	astTestMust(t, e)
	if !yes || !pos || len(names) != 3 || names[0] != prefix || names[1] != first || names[2] != second {
		t.Fatal("label recognition and append order", yes, pos, names)
	}
	yes, e = m.IsLabel(arg, &pos, &names)
	astTestMust(t, e)
	if yes || !pos || len(names) != 3 {
		t.Fatal("nonlabel changed outputs", yes, pos, names)
	}
	lit, e := m.MkLabelLit([]usymbol.Symbol{second, first})
	astTestMust(t, e)
	astTestPin(t, m, lit)
	yes, e = m.IsLabelLit(lit, &names)
	astTestMust(t, e)
	if !yes || len(names) != 5 || names[3] != second || names[4] != first {
		t.Fatal("label literal append order", yes, names)
	}
	yes, e = m.IsLabelLit(label, &names)
	astTestMust(t, e)
	if yes || len(names) != 5 {
		t.Fatal("label matched literal branch", yes, names)
	}
	negative, e := m.MkLabel(false, []usymbol.Symbol{first}, arg)
	astTestMust(t, e)
	astTestPin(t, m, negative)
	pos = true
	yes, e = m.IsLabel(negative, &pos, &names)
	astTestMust(t, e)
	if !yes || pos || len(names) != 6 || names[5] != first {
		t.Fatal("negative label", yes, pos, names)
	}
	other := astTestManager(t)
	foreign := astTestSymbol(t, other, "foreign")
	before := m.Stats().LiveParameters
	if _, e = m.MkLabel(true, []usymbol.Symbol{first, foreign}, arg); !errors.Is(e, usymbol.ErrOwner) {
		t.Fatal("foreign label name", e)
	}
	if got := m.Stats().LiveParameters; got != before {
		t.Fatal("failed label retained temporary parameters", got, before)
	}
}

func TestASTLabelTemporaryParametersOnReservationFailure(t *testing.T) {
	for refusal := int64(0); refusal < 12; refusal++ {
		t.Run(strconv.FormatInt(refusal, 10), func(t *testing.T) {
			m := astTestManager(t)
			b, e := m.MkBoolSort()
			astTestMust(t, e)
			arg, e := m.MkConst(astTestSymbol(t, m, "arg"), b)
			astTestMust(t, e)
			astTestPin(t, m, arg)
			first := astTestSymbol(t, m, "first")
			second := astTestSymbol(t, m, "second")
			before := m.Stats()
			base := before.LiveParameters
			m.BudgetField.FailAfter(refusal)
			label, e := m.MkLabel(true, []usymbol.Symbol{first, second}, arg)
			m.BudgetField.FailAfter(-1)
			if e != nil {
				after := m.Stats()
				// mk_app constructs/registers its declaration first. An app
				// refusal may leave that declaration and its copied parameters
				// available for a later hash-consed retry.
				noDecl := after.LiveParameters == base && after.LiveNodes == before.LiveNodes && after.LiveInfos == before.LiveInfos
				withDecl := after.LiveParameters == base+3 && after.LiveNodes == before.LiveNodes+1 && after.LiveInfos == before.LiveInfos+1
				if !noDecl && !withDecl {
					t.Fatal("failed constructor retained a partial temporary", before, after, e)
				}
				label, e = m.MkLabel(true, []usymbol.Symbol{first, second}, arg)
				astTestMust(t, e)
				astTestPin(t, m, label)
				retried := m.Stats()
				if retried.LiveParameters != base+3 || retried.LiveNodes != before.LiveNodes+2 {
					t.Fatal("retry did not reuse the completed declaration", before, after, retried)
				}
				return
			}
			astTestPin(t, m, label)
			if got := m.Stats().LiveParameters; got != base+3 {
				t.Fatal("successful declaration parameter copies", got, base)
			}
		})
	}
}
