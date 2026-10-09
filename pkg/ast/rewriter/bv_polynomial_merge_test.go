// Source: src/ast/rewriter/poly_rewriter_def.h:886-901,947-977.
package rewriter

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func bvMergeVars(t *testing.T, m *ast.ASTManager, n int) []ast.ASTHandle {
	t.Helper()
	s, e := m.BitVectorSort(8)
	astTestMust(t, e)
	xs := make([]ast.ASTHandle, n)
	for i := range xs {
		xs[i], e = m.MkVar(uint32(i), s)
		astTestMust(t, e)
		astTestPin(t, m, xs[i])
	}
	return xs
}
func bvMergeMul(t *testing.T, m *ast.ASTManager, args ...ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	h, e := m.MkBitVectorPolynomialMul(ast.ASTHandle{}, ast.SliceArguments(args))
	astTestMust(t, e)
	astTestPin(t, m, h)
	return h
}

func TestBVPolynomialMergeOriginalFlattenAndMatchingSwaps(t *testing.T) {
	m, _ := bvAddFixture(t)
	xs := bvMergeVars(t, m, 4)
	left := bvMergeMul(t, m, bvMergeMul(t, m, xs[0], xs[1]), bvMergeMul(t, m, xs[2], xs[3]))
	right := bvMergeMul(t, m, xs[3], xs[2])
	var flat ast.ASTHandleBuffer
	defer flat.Close(m.Budget())
	astTestMust(t, bvPolynomialFlatten(m, 6, &flat, left))
	for i, want := range []ast.ASTHandle{xs[0], xs[2], xs[1], xs[3]} {
		if flat.Slice()[i] != want {
			t.Fatal("source flatten order", i)
		}
	}
	var sort ast.ASTHandle
	p, e := NewBVPolynomialMerger(m, &sort)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	got, e := p.Merge(left, right)
	astTestMust(t, e)
	astTestPin(t, m, got)
	s, e := m.GetSort(xs[3])
	astTestMust(t, e)
	if sort != s {
		t.Fatal("current sort not published from last left factor")
	}
	count, e := m.GetNumArgs(got)
	astTestMust(t, e)
	if count != 3 {
		t.Fatal("common-factor prefix", count)
	}
	for i, want := range []ast.ASTHandle{xs[2], xs[3]} {
		h, e := m.GetArg(got, uint32(i))
		astTestMust(t, e)
		if h != want {
			t.Fatal("matching swap order")
		}
	}
	sum, e := m.GetArg(got, 2)
	astTestMust(t, e)
	residual, e := m.GetArg(sum, 0)
	astTestMust(t, e)
	for i, want := range []ast.ASTHandle{xs[1], xs[0]} {
		h, e := m.GetArg(residual, uint32(i))
		astTestMust(t, e)
		if h != want {
			t.Fatal("residual swap order")
		}
	}
	one, e := m.GetArg(sum, 1)
	astTestMust(t, e)
	yes, e := m.IsBitVectorOperation(one, 0)
	astTestMust(t, e)
	if !yes {
		t.Fatal("empty residual did not construct numeral one")
	}
}

func TestBVPolynomialMergeRepeatedFactorsAndIdenticalProducts(t *testing.T) {
	m, _ := bvAddFixture(t)
	xs := bvMergeVars(t, m, 3)
	for _, row := range []struct {
		left, right ast.ASTHandle
		common      int
	}{{bvMergeMul(t, m, xs[0], xs[0], xs[1]), bvMergeMul(t, m, xs[0], xs[2]), 1}, {xs[0], xs[0], 1}, {bvMergeMul(t, m, xs[0], xs[1]), bvMergeMul(t, m, xs[1], xs[0]), 2}} {
		var sort ast.ASTHandle
		p, e := NewBVPolynomialMerger(m, &sort)
		astTestMust(t, e)
		got, e := p.Merge(row.left, row.right)
		astTestMust(t, e)
		astTestPin(t, m, got)
		n, e := m.GetNumArgs(got)
		astTestMust(t, e)
		if int(n) != row.common+1 {
			t.Fatal("factor multiplicity", n)
		}
		astTestMust(t, p.SourceReady())
		astTestMust(t, p.Close())
	}
}

func TestBVPolynomialMergeNoCommonSourceDebugGuardAndColdHeader(t *testing.T) {
	m, _ := bvAddFixture(t)
	xs := bvMergeVars(t, m, 2)
	used := m.Budget().Used()
	var sort ast.ASTHandle
	m.Budget().FailAfter(0)
	p, e := NewBVPolynomialMerger(m, &sort)
	m.Budget().FailAfter(-1)
	if p != nil || !errors.Is(e, arena.ErrInjectedFailure) || m.Budget().Used() != used {
		t.Fatal("header refusal", e)
	}
	p, e = NewBVPolynomialMerger(m, &sort)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	got, body := p.Merge(xs[0], xs[1])
	if ast.ASTSourceDebug {
		if !errors.Is(body, ast.ErrASTDomain) || !got.IsNull() {
			t.Fatal("source debug k>0 guard", body)
		}
	} else {
		astTestMust(t, body)
		astTestPin(t, m, got)
		kind, e := m.GetDeclKind(got)
		astTestMust(t, e)
		if kind != 4 {
			t.Fatal("source release no-common result")
		}
	}
	s, e := m.GetSort(xs[0])
	astTestMust(t, e)
	if sort != s {
		t.Fatal("sort publication must precede assertion")
	}
}

func TestBVPolynomialMergeLastFactorSortBeforeResidualFactories(t *testing.T) {
	m, fid := bvAddFixture(t)
	xs := bvMergeVars(t, m, 2)
	s8, e := m.GetSort(xs[0])
	astTestMust(t, e)
	s16, e := m.BitVectorSort(16)
	astTestMust(t, e)
	// A manually declared MUL has an independent result sort. The selected
	// body must consult the final visited factor, not this enclosing metadata.
	info, e := m.NewFuncDeclInfo(fid, 6, nil, ast.ASTFuncFlags{}, false)
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "merge-independent-result-sort-args"), []ast.ASTHandle{s8, s8}, s16, info)
	astTestMust(t, e)
	astTestMust(t, m.DelFuncDeclInfo(&info))
	left, e := m.MkApp(d, xs)
	astTestMust(t, e)
	astTestPin(t, m, left)
	currentSort := s16
	p, e := NewBVPolynomialMerger(m, &currentSort)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	got, e := p.Merge(left, xs[0])
	astTestMust(t, e)
	astTestPin(t, m, got)
	if currentSort != s8 {
		t.Fatal("read enclosing product sort instead of last factor")
	}
	sum, e := m.GetArg(got, 1)
	astTestMust(t, e)
	one, e := m.GetArg(sum, 1)
	astTestMust(t, e)
	width, e := m.BitVectorSize(one)
	astTestMust(t, e)
	if width != 8 {
		t.Fatal("empty product constructed with wrong current sort", width)
	}
}

func TestBVPolynomialMergeEveryFactoryAndSpillReservation(t *testing.T) {
	complete, failures := false, 0
	for ordinal := int64(0); ordinal < 100 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, _ := bvAddFixture(t)
			xs := bvMergeVars(t, m, 40)
			left := bvMergeMul(t, m, xs...)
			right := left
			var sort ast.ASTHandle
			p, e := NewBVPolynomialMerger(m, &sort)
			astTestMust(t, e)
			defer func() { astTestMust(t, p.Close()) }()
			m.Budget().FailAfter(ordinal)
			got, body := p.Merge(left, right)
			m.Budget().FailAfter(-1)
			if body != nil {
				failures++
				if !errors.Is(body, arena.ErrInjectedFailure) {
					var f util.Failure
					if !errors.As(body, &f) || f.Kind != util.FailureMemout {
						t.Fatal("unexpected refusal", body)
					}
				}
				if _, e = p.Merge(left, right); !errors.Is(e, ast.ErrASTPending) {
					t.Fatal("failed source call reused", e)
				}
			} else {
				complete = true
				astTestPin(t, m, got)
				n, e := m.GetNumArgs(got)
				astTestMust(t, e)
				if n != 41 {
					t.Fatal("common multiplicity lost", n)
				}
			}
			ready := p.SourceReady()
			astTestMust(t, p.Close())
			if p.SourceReady() != ready {
				t.Fatal("cleanup changed source history")
			}
		})
	}
	if !complete || failures == 0 {
		t.Fatal("merge horizon", complete, failures)
	}
}

func TestBVPolynomialMergeEveryActualEmptyProductReservation(t *testing.T) {
	complete, failures := false, 0
	for ordinal := int64(0); ordinal < 200 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			budget := arena.NewBudget(1 << 28)
			rt, e := util.NewRuntime(util.RuntimeConfig{Budget: budget, SizeTBits: 64, Transport: func(f util.Failure) error { return f }})
			astTestMust(t, e)
			q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Memory: rt, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
			astTestMust(t, e)
			sy, e := usymbol.NewManager(usymbol.Config{Budget: budget})
			astTestMust(t, e)
			m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: budget})
			astTestMust(t, e)
			t.Cleanup(func() {
				astTestMust(t, m.Close())
				astTestMust(t, q.Close())
				astTestMust(t, sy.Close())
				astTestMust(t, rt.Close())
				if budget.Used() != 0 {
					t.Fatal("actual Runtime graph retained", budget.Used())
				}
			})
			_, e = m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			_, e = m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			s, e := m.BitVectorSort(65)
			astTestMust(t, e)
			x, e := m.MkVar(0, s)
			astTestMust(t, e)
			astTestPin(t, m, x)
			var currentSort ast.ASTHandle
			p, e := NewBVPolynomialMerger(m, &currentSort)
			astTestMust(t, e)
			defer func() { astTestMust(t, p.Close()) }()
			budget.FailAfter(ordinal)
			got, body := p.Merge(x, x)
			budget.FailAfter(-1)
			if currentSort != s {
				t.Fatal("sort publication after numerical failure")
			}
			if body != nil {
				failures++
				if !errors.Is(body, arena.ErrInjectedFailure) {
					var f util.Failure
					if !errors.As(body, &f) || f.Kind != util.FailureMemout {
						t.Fatal("unexpected actual refusal", body)
					}
				}
				if !got.IsNull() {
					t.Fatal("failed merge returned result")
				}
				if _, e = p.Merge(x, x); !errors.Is(e, ast.ErrASTPending) {
					t.Fatal("failed call retried", e)
				}
			} else {
				complete = true
				astTestPin(t, m, got)
				sum, e := m.GetArg(got, 1)
				astTestMust(t, e)
				for i := uint32(0); i < 2; i++ {
					one, e := m.GetArg(sum, i)
					astTestMust(t, e)
					out, e := q.NewValue()
					astTestMust(t, e)
					var width uint32
					yes, e := m.BitVectorNumeralValue(one, &out, &width)
					astTestMust(t, e)
					text, e := q.String(&out)
					astTestMust(t, e)
					astTestMust(t, q.Del(&out))
					if !yes || text != "1" || width != 65 {
						t.Fatal("empty residual identity", text, width)
					}
				}
			}
			ready := p.SourceReady()
			astTestMust(t, p.Close())
			if p.SourceReady() != ready {
				t.Fatal("owning close replaced source history")
			}
		})
	}
	if !complete || failures == 0 {
		t.Fatal("actual empty-product horizon", complete, failures)
	}
}
