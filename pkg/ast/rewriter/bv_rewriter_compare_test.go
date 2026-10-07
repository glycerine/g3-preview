// Source: src/ast/rewriter/bv_rewriter.cpp:2040-2046,2058-2074.
package rewriter

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

func TestBVRewriterOriginalCompareAndReductionOR(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	satOK(t, e)
	_, e = m.RegisterBitVectorPlugin()
	satOK(t, e)
	sort, e := m.BitVectorSort(65)
	satOK(t, e)
	x, e := m.MkVar(0, sort)
	satOK(t, e)
	astTestPin(t, m, x)
	y, e := m.MkVar(1, sort)
	satOK(t, e)
	astTestPin(t, m, y)
	zero, e := m.BitVectorNumeralUnsigned(0, 65)
	satOK(t, e)
	astTestPin(t, m, zero)
	value, e := m.BitVectorNumeralUnsigned(17, 65)
	satOK(t, e)
	astTestPin(t, m, value)
	bit0, e := m.BitVectorNumeralUnsigned(0, 1)
	satOK(t, e)
	astTestPin(t, m, bit0)
	bit1, e := m.BitVectorNumeralUnsigned(1, 1)
	satOK(t, e)
	astTestPin(t, m, bit1)
	out := astTestPin(t, m, x)
	for _, test := range []struct {
		left, right, want ast.ASTHandle
		status            RewriteStatus
	}{
		{x, x, bit1, RewriteDone}, {value, value, bit1, RewriteDone}, {zero, value, bit0, RewriteDone},
	} {
		status, e := RewriteBitVectorCompare(m, test.left, test.right, out)
		satOK(t, e)
		got, e := m.RefGet(out)
		satOK(t, e)
		if status != test.status || got != test.want {
			t.Fatal("compare branch", status, got, test.want)
		}
	}
	eq, e := m.MkEq(x, y)
	satOK(t, e)
	want, e := m.MkIte(eq, bit1, bit0)
	satOK(t, e)
	astTestPin(t, m, want)
	status, e := RewriteBitVectorCompare(m, x, y, out)
	satOK(t, e)
	got, e := m.RefGet(out)
	satOK(t, e)
	if status != Rewrite2 || got != want {
		t.Fatal("compare ITE expansion", status, got, want)
	}
	for _, test := range []struct{ input, want ast.ASTHandle }{{zero, bit0}, {value, bit1}} {
		status, e := RewriteBitVectorReductionOR(m, test.input, out)
		satOK(t, e)
		got, e := m.RefGet(out)
		satOK(t, e)
		if status != RewriteDone || got != test.want {
			t.Fatal("reduction OR", status, got, test.want)
		}
	}
	satOK(t, m.SetRef(out, x))
	before := m.Stats()
	m.Budget().FailAfter(0)
	status, e = RewriteBitVectorReductionOR(m, x, out)
	satOK(t, e)
	emptyStatus, e := RewriteBitVectorReductionOR(m, y, ast.ASTRef{})
	satOK(t, e)
	m.Budget().FailAfter(-1)
	got, e = m.RefGet(out)
	satOK(t, e)
	if status != RewriteFailed || emptyStatus != RewriteFailed || got != x || m.Stats() != before {
		t.Fatal("unmatched reduction read output/allocated")
	}
}

func TestBVRewriterComparisonNumeralIdentityErasesParameters(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	satOK(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	satOK(t, e)
	sort, e := m.BitVectorSort(8)
	satOK(t, e)
	// Source comparison only inspects app/family/kind, not numeral metadata.
	// These distinct raw NUM apps deliberately have no numeric parameters.
	var literals []ast.ASTHandle
	for _, name := range []string{"raw-comp-numeral-left", "raw-comp-numeral-right"} {
		symbol, e := m.Symbols.Intern(name)
		satOK(t, e)
		info, e := m.NewFuncDeclInfo(fid, 0, nil, ast.ASTFuncFlags{}, false)
		satOK(t, e)
		d, e := m.MkFuncDecl(symbol, nil, sort, info)
		satOK(t, e)
		satOK(t, m.DelFuncDeclInfo(&info))
		h, e := m.MkApp(d, nil)
		satOK(t, e)
		astTestPin(t, m, h)
		literals = append(literals, h)
	}
	bit0, e := m.BitVectorNumeralUnsigned(0, 1)
	satOK(t, e)
	astTestPin(t, m, bit0)
	out := astTestPin(t, m, literals[0])
	status, e := RewriteBitVectorCompare(m, literals[0], literals[1], out)
	satOK(t, e)
	got, e := m.RefGet(out)
	satOK(t, e)
	if status != RewriteDone || got != bit0 {
		t.Fatal("compare read numeric metadata instead of source identity")
	}
}

func TestBVRewriterCompareAndReductionOREveryColdReservation(t *testing.T) {
	for op := 0; op < 2; op++ {
		success := false
		failures := 0
		for ordinal := int64(0); ordinal < 128; ordinal++ {
			t.Run(strconv.Itoa(op)+"/"+strconv.FormatInt(ordinal, 10), func(t *testing.T) {
				m := astTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				satOK(t, e)
				_, e = m.RegisterBitVectorPlugin()
				satOK(t, e)
				sort, e := m.BitVectorSort(65)
				satOK(t, e)
				x, e := m.MkVar(0, sort)
				satOK(t, e)
				astTestPin(t, m, x)
				y, e := m.MkVar(1, sort)
				satOK(t, e)
				astTestPin(t, m, y)
				value, e := m.BitVectorNumeralUnsigned(17, 65)
				satOK(t, e)
				astTestPin(t, m, value)
				out := astTestPin(t, m, x)
				m.Budget().FailAfter(ordinal)
				var body error
				var status RewriteStatus
				if op == 0 {
					status, body = RewriteBitVectorCompare(m, x, y, out)
				} else {
					status, body = RewriteBitVectorReductionOR(m, value, out)
				}
				m.Budget().FailAfter(-1)
				if body != nil {
					failures++
					if !errors.Is(body, arena.ErrInjectedFailure) {
						t.Fatal("unexpected refusal", body)
					}
					got, e := m.RefGet(out)
					satOK(t, e)
					if got != x {
						t.Fatal("factory failure published output")
					}
				} else {
					success = true
					want := RewriteDone
					if op == 0 {
						want = Rewrite2
					}
					if status != want {
						t.Fatal("rewrite status", status, want)
					}
				}
			})
			if success {
				break
			}
		}
		if !success || failures == 0 {
			t.Fatal("reservation horizon", op, success, failures)
		}
	}
}
