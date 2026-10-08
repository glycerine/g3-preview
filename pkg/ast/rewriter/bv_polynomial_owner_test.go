// Source: src/ast/rewriter/poly_rewriter.h:28-41,95-164.
package rewriter

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

func TestBVPolynomialOwnerPersistentMapAndSharedSort(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	b := m.Budget()
	p, e := NewBVPolynomialRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 48)
	args := make([]ast.ASTHandle, 0, 96)
	for _, x := range xs {
		args = append(args, x, x)
	}
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	table := p.positions
	for i := 0; i < 2; i++ {
		st, e := p.RewriteAddition(ast.SliceArguments(args), out)
		astTestMust(t, e)
		if st != RewriteDone || p.positions != table || table.closed || table.size != 48 {
			t.Fatal("persistent map", st, table)
		}
		if p.addition != nil {
			t.Fatal("successful scope retained")
		}
	}
	backing := &table.entries[0]
	capacity := table.capacity
	st, e := p.RewriteAddition(ast.SliceArguments(args), out)
	astTestMust(t, e)
	if st != RewriteDone || backing != &table.entries[0] || capacity != table.capacity {
		t.Fatal("dense reset replaced retained backing")
	}
	sort, e := m.GetSort(xs[0])
	astTestMust(t, e)
	if p.currentSort != sort {
		t.Fatal("sort publication")
	}
	// Retire the large previous output before refusing reservations: its final
	// reference deletion can itself allocate unrelated numeric DT storage.
	astTestMust(t, m.SetRef(out, xs[0]))
	b.FailAfter(0)
	st, e = p.RewriteMultiplication(ast.SliceArguments(xs[:1]), out)
	b.FailAfter(-1)
	astTestMust(t, e)
	if st != RewriteDone || p.currentSort != sort || p.multiplication != nil {
		t.Fatal("singleton allocated or overwrote sort")
	}
}

func TestBVPolynomialOwnerSharedSortThroughHoistAndFailure(t *testing.T) {
	succeeded := false
	sawNestedFailure := false
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			c, g, _ := bvPolynomialParamsFixture(t)
			m, fid := bvAddFixture(t)
			local, e := c.NewValue()
			astTestMust(t, e)
			defer func() { astTestMust(t, c.Del(&local)) }()
			key := "hoist_mul"
			astTestMust(t, c.SetBoolName(&local, &key, true))
			p, e := NewBVPolynomialRewriter(m, c, g, &local)
			astTestMust(t, e)
			defer func() { astTestMust(t, p.Close()) }()
			xs := bvMergeVars(t, m, 3)
			s8, e := m.GetSort(xs[0])
			astTestMust(t, e)
			s16, e := m.BitVectorSort(16)
			astTestMust(t, e)
			info, e := m.NewFuncDeclInfo(fid, 6, nil, ast.ASTFuncFlags{}, false)
			astTestMust(t, e)
			d, e := m.MkFuncDecl(astTestSymbol(t, m, "owner-independent-result-sort"), []ast.ASTHandle{s8, s8}, s16, info)
			astTestMust(t, e)
			astTestMust(t, m.DelFuncDeclInfo(&info))
			left, e := m.MkApp(d, []ast.ASTHandle{xs[0], xs[1]})
			astTestMust(t, e)
			astTestPin(t, m, left)
			right, e := m.MkApp(d, []ast.ASTHandle{xs[0], xs[2]})
			astTestMust(t, e)
			astTestPin(t, m, right)
			out, e := m.NewRef(xs[0])
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			m.Budget().FailAfter(ordinal)
			st, e := p.RewriteAddition(ast.SliceArguments([]ast.ASTHandle{left, right}), out)
			m.Budget().FailAfter(-1)
			if e == nil {
				succeeded = true
				if st != RewriteFull || p.currentSort != s8 {
					t.Fatal("nested sort did not reach owner", st, p.currentSort, s8)
				}
				got, e := m.RefGet(out)
				astTestMust(t, e)
				zero, e := m.GetArg(got, 1)
				astTestMust(t, e)
				width, e := m.BitVectorSize(zero)
				astTestMust(t, e)
				if width != 8 {
					t.Fatal("zero factory used enclosing sort", width)
				}
			} else {
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal(e)
				}
				if p.currentSort != s16 && p.currentSort != s8 {
					t.Fatal("lost reached sort", p.currentSort)
				}
				if p.addition != nil && p.addition.mul != nil {
					h := p.addition.mul
					if h.sortSlot() != &p.currentSort {
						t.Fatal("hoist has disconnected sort")
					}
					if h.merge != nil {
						if h.merge.currentSort != &p.currentSort {
							t.Fatal("merger has disconnected sort")
						}
						if p.currentSort == s8 {
							sawNestedFailure = true
						}
					}
				}
				sort := p.currentSort
				if err := p.MakeMultiplicationPair(xs[1], xs[2], out); err != e {
					t.Fatal("stopped owner reused", err, e)
				}
				if p.currentSort != sort || !p.pair[0].IsNull() {
					t.Fatal("stopped call changed prefix")
				}
				if p.cause != e {
					t.Fatal("lost original cause")
				}
			}
		})
		if succeeded {
			break
		}
	}
	if !succeeded || !sawNestedFailure {
		t.Fatal("missing success or nested failure", succeeded, sawNestedFailure)
	}
}

func TestBVPolynomialOwnerEveryPersistentMapAndCoreReservation(t *testing.T) {
	succeeded := false
	for ordinal := int64(0); ordinal < 512; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			c, g, _ := bvPolynomialParamsFixture(t)
			m, _ := bvAddFixture(t)
			p, e := NewBVPolynomialRewriter(m, c, g, nil)
			astTestMust(t, e)
			xs := bvMergeVars(t, m, 24)
			args := make([]ast.ASTHandle, 0, 48)
			for _, x := range xs {
				args = append(args, x, x)
			}
			out, e := m.NewRef(xs[0])
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			table := p.positions
			m.Budget().FailAfter(ordinal)
			st, e := p.RewriteAddition(ast.SliceArguments(args), out)
			m.Budget().FailAfter(-1)
			if e == nil {
				succeeded = true
				if st != RewriteDone || p.addition != nil || table.closed {
					t.Fatal("successful invocation retained or destroyed map")
				}
			} else {
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal(e)
				}
				if p.addition != nil && (p.addition.positions != table || p.addition.ownsPositions || p.addition.sortSlot() != &p.currentSort) {
					t.Fatal("child ownership disconnected")
				}
				saved := p.currentSort
				if _, err := p.RewriteAddition(ast.SliceArguments(args), out); err != e {
					t.Fatal("retry", err)
				}
				if saved != p.currentSort {
					t.Fatal("retry changed sort")
				}
			}
			copy := *p
			if err := copy.Close(); !errors.Is(err, ast.ErrASTHandle) {
				t.Fatal("copied owner", err)
			}
			cause := p.cause
			astTestMust(t, p.Close())
			astTestMust(t, p.Close())
			if !table.closed || p.cause != cause {
				t.Fatal("cleanup history or map lifetime")
			}
		})
		if succeeded {
			break
		}
	}
	if !succeeded {
		t.Fatal("never reached success")
	}
}

func TestBVPolynomialOwnerConstructorEveryReservation(t *testing.T) {
	for _, domain := range []string{"ast", "params"} {
		succeeded := false
		for ordinal := int64(0); ordinal < 128; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", domain, ordinal), func(t *testing.T) {
				c, g, b := bvPolynomialParamsFixture(t)
				m, _ := bvAddFixture(t)
				charged := m.Budget()
				if domain == "params" {
					charged = b
				}
				charged.FailAfter(ordinal)
				p, e := NewBVPolynomialRewriter(m, c, g, nil)
				charged.FailAfter(-1)
				if e == nil {
					succeeded = true
					if p.config.passes != 2 || p.positions.capacity != 8 {
						t.Fatal("incomplete constructor")
					}
				} else {
					if !errors.Is(e, arena.ErrInjectedFailure) {
						t.Fatal(e)
					}
					if p != nil {
						if !p.stopped || p.cause != e {
							t.Fatal("lost constructor prefix")
						}
						if domain == "params" && p.positions == nil {
							t.Fatal("configuration preceded map construction")
						}
					}
				}
				if p != nil {
					cause := p.cause
					astTestMust(t, p.Close())
					if p.cause != cause {
						t.Fatal("cleanup changed history")
					}
				}
			})
			if succeeded {
				break
			}
		}
		if !succeeded {
			t.Fatal(domain, "never reached success")
		}
	}
}

func TestBVPolynomialOwnerConvenienceAndConfigurationAcrossCalls(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVPolynomialRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	xs := bvMergeVars(t, m, 3)
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	astTestMust(t, p.MakeMultiplicationPair(xs[0], xs[1], out))
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if got != bvMergeMul(t, m, xs[0], xs[1]) {
		t.Fatal("FAILED multiplication convenience fallback")
	}
	astTestMust(t, p.MakeAdditionPair(xs[0], xs[1], out))
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if got != bvMulHoistSum(t, m, xs[0], xs[1]) {
		t.Fatal("FAILED addition convenience fallback")
	}
	local, e := c.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, c.Del(&local)) }()
	key := "flat"
	astTestMust(t, c.SetBoolName(&local, &key, false))
	key = "som"
	astTestMust(t, c.SetBoolName(&local, &key, true))
	astTestMust(t, p.SetSortSums(true))
	astTestMust(t, p.Update(&local))
	s, e := p.State()
	astTestMust(t, e)
	if s.Flat || s.SumOfMonomials || !s.SortSums {
		t.Fatal("configuration not applied", s)
	}
	oldMap := p.positions
	astTestMust(t, p.MakeAdditionPair(xs[1], xs[0], out))
	got, e = m.RefGet(out)
	astTestMust(t, e)
	first, e := m.GetArg(got, 0)
	astTestMust(t, e)
	if first != xs[0] || p.positions != oldMap {
		t.Fatal("updated options not consumed")
	}
	st, e := p.RewriteMultiplication(ast.SliceArguments([]ast.ASTHandle{xs[1], xs[0]}), out)
	astTestMust(t, e)
	if st != RewriteDone {
		t.Fatal("updated nonflat multiplication", st)
	}
	got, e = m.RefGet(out)
	astTestMust(t, e)
	if got != bvMergeMul(t, m, xs[0], xs[1]) {
		t.Fatal("multiplication options not consumed")
	}
}
