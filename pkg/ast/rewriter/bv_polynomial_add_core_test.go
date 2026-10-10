// Source: src/ast/rewriter/poly_rewriter_def.h:527-660.
package rewriter

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	"testing"
)

func TestBVPolynomialNonflatAdditionCollectsAndCancelsCoefficients(t *testing.T) {
	m, _ := bvAddFixture(t)
	xs := bvMergeVars(t, m, 3)
	s, e := m.GetSort(xs[0])
	astTestMust(t, e)
	two := bvAddNum(t, m, 2, 8)
	minus := bvAddNum(t, m, 254, 8)
	terms := []ast.ASTHandle{bvMergeMul(t, m, two, xs[0]), xs[1], bvMergeMul(t, m, minus, xs[0]), xs[2], xs[2]}
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	core, e := NewBVPolynomialAdditionCore(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, core.Close()) }()
	st, e := core.Rewrite(s, ast.SliceArguments(terms), out, BVPolynomialAdditionOptions{})
	if e != nil {
		t.Fatalf("core: %v body=%v source=%v", e, core.bodyCause, core.sourceCause)
	}
	if st != RewriteDone {
		t.Fatal(st)
	}
	got, e := m.RefGet(out)
	astTestMust(t, e)
	n, e := m.GetNumArgs(got)
	astTestMust(t, e)
	if n != 2 {
		t.Fatal("cancelled coefficient retained", n)
	}
	first, e := m.GetArg(got, 0)
	astTestMust(t, e)
	if first != xs[1] {
		t.Fatal("first encounter order", first)
	}
	second, e := m.GetArg(got, 1)
	astTestMust(t, e)
	want := bvMergeMul(t, m, two, xs[2])
	if second != want {
		t.Fatal("repeated coefficient not collected")
	}
}

func TestBVPolynomialNonflatAdditionShortcutsConstantsAndOrdering(t *testing.T) {
	for _, row := range []struct {
		values  bool
		args    []int
		options BVPolynomialAdditionOptions
		status  RewriteStatus
		want    []int
	}{
		{false, []int{0, 1}, BVPolynomialAdditionOptions{}, RewriteFailed, nil},
		{false, []int{1, 0}, BVPolynomialAdditionOptions{SortSums: true}, RewriteDone, []int{0, 1}},
		{false, []int{1, 0}, BVPolynomialAdditionOptions{SortSums: true, ASTOrder: true}, RewriteDone, []int{0, 1}},
		{true, []int{255, 1}, BVPolynomialAdditionOptions{}, RewriteDone, nil},
		{true, []int{2, 3}, BVPolynomialAdditionOptions{}, RewriteDone, nil},
	} {
		m, _ := bvAddFixture(t)
		xs := bvMergeVars(t, m, 3)
		s, e := m.GetSort(xs[0])
		astTestMust(t, e)
		args := make([]ast.ASTHandle, len(row.args))
		for i, v := range row.args {
			if row.values {
				args[i] = bvAddNum(t, m, uint32(v), 8)
			} else {
				args[i] = xs[v]
			}
		}
		out, e := m.NewRef(xs[2])
		astTestMust(t, e)
		defer func() { astTestMust(t, m.DelRef(&out)) }()
		p, e := NewBVPolynomialAdditionCore(m)
		astTestMust(t, e)
		defer func() { astTestMust(t, p.Close()) }()
		st, e := p.Rewrite(s, ast.SliceArguments(args), out, row.options)
		astTestMust(t, e)
		if st != row.status {
			t.Fatal(row, st)
		}
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st == RewriteFailed {
			if got != xs[2] {
				t.Fatal("failed shortcut changed output")
			}
		} else if row.values {
			want := uint32(0)
			for _, v := range row.args {
				want += uint32(v)
			}
			want %= 256
			if got != bvAddNum(t, m, want, 8) {
				t.Fatal("constant sum")
			}
		} else {
			for i, v := range row.want {
				h, e := m.GetArg(got, uint32(i))
				astTestMust(t, e)
				if h != xs[v] {
					t.Fatal("monomial ordering")
				}
			}
		}
		astTestMust(t, p.SourceReady())
	}
}

func TestBVPolynomialNonflatAdditionZeroAndConstantLeadingGuard(t *testing.T) {
	m, _ := bvAddFixture(t)
	xs := bvMergeVars(t, m, 2)
	s, e := m.GetSort(xs[0])
	astTestMust(t, e)
	for _, row := range []struct {
		args   []ast.ASTHandle
		status RewriteStatus
		want   ast.ASTHandle
	}{
		{[]ast.ASTHandle{bvAddNum(t, m, 7, 8), xs[0]}, RewriteFailed, xs[1]},
		{[]ast.ASTHandle{bvAddNum(t, m, 0, 8), xs[0]}, RewriteDone, xs[0]},
		{[]ast.ASTHandle{xs[0], bvAddNum(t, m, 7, 8)}, RewriteDone, bvMulHoistSum(t, m, bvAddNum(t, m, 7, 8), xs[0])},
		{[]ast.ASTHandle{xs[0], bvMergeMul(t, m, bvAddNum(t, m, 255, 8), xs[0])}, RewriteDone, bvAddNum(t, m, 0, 8)},
	} {
		astTestPin(t, m, row.want)
		out, e := m.NewRef(xs[1])
		astTestMust(t, e)
		p, e := NewBVPolynomialAdditionCore(m)
		astTestMust(t, e)
		st, e := p.Rewrite(s, ast.SliceArguments(row.args), out, BVPolynomialAdditionOptions{})
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != row.status || got != row.want {
			t.Fatal("guard/cancellation", st, got, row.status, row.want)
		}
		astTestMust(t, p.Close())
		astTestMust(t, m.DelRef(&out))
	}
}

func TestBVPolynomialNonflatAdditionBothHoistPaths(t *testing.T) {
	for _, cmul := range []bool{false, true} {
		m, _ := bvAddFixture(t)
		xs := bvMergeVars(t, m, 4)
		s, e := m.GetSort(xs[0])
		astTestMust(t, e)
		var args []ast.ASTHandle
		if cmul {
			two := bvAddNum(t, m, 2, 8)
			args = []ast.ASTHandle{bvMergeMul(t, m, two, xs[0]), bvMergeMul(t, m, two, xs[1])}
		} else {
			args = []ast.ASTHandle{bvMergeMul(t, m, xs[0], xs[1]), bvMergeMul(t, m, xs[0], xs[2])}
		}
		for _, h := range args {
			astTestPin(t, m, h)
		}
		out, e := m.NewRef(ast.ASTHandle{})
		astTestMust(t, e)
		defer func() { astTestMust(t, m.DelRef(&out)) }()
		p, e := NewBVPolynomialAdditionCore(m)
		astTestMust(t, e)
		defer func() { astTestMust(t, p.Close()) }()
		st, e := p.Rewrite(s, ast.SliceArguments(args), out, BVPolynomialAdditionOptions{HoistCoefficients: cmul, HoistMultiplication: !cmul})
		astTestMust(t, e)
		if cmul && st != RewriteDone || !cmul && st != RewriteFull {
			t.Fatal("hoist status", cmul, st)
		}
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if cmul {
			yes, e := m.IsBitVectorOperation(got, 6)
			astTestMust(t, e)
			if !yes {
				t.Fatal("coefficient hoist")
			}
		} else {
			n, e := m.GetNumArgs(got)
			astTestMust(t, e)
			if n != 2 {
				t.Fatal("multiplication hoist placeholders")
			}
		}
	}
}

func TestBVPolynomialNonflatAdditionEveryColdReservation(t *testing.T) {
	complete, failures := false, 0
	for ordinal := int64(0); ordinal < 300 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, _ := bvAddFixture(t)
			xs := bvMergeVars(t, m, 24)
			s, e := m.GetSort(xs[0])
			astTestMust(t, e)
			args := make([]ast.ASTHandle, 0, 48)
			two := bvAddNum(t, m, 2, 8)
			for _, x := range xs {
				h := bvMergeMul(t, m, two, x)
				astTestPin(t, m, h)
				args = append(args, h, x)
			}
			out, e := m.NewRef(ast.ASTHandle{})
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			p, e := NewBVPolynomialAdditionCore(m)
			astTestMust(t, e)
			defer func() { astTestMust(t, p.Close()) }()
			m.Budget().FailAfter(ordinal)
			st, body := p.Rewrite(s, ast.SliceArguments(args), out, BVPolynomialAdditionOptions{SortSums: true, HoistCoefficients: true})
			m.Budget().FailAfter(-1)
			if body != nil {
				failures++
				if !numericHoistRefusal(body) {
					t.Fatal("unexpected refusal", ordinal, body)
				}
				if _, e = p.Rewrite(s, ast.SliceArguments(args), out, BVPolynomialAdditionOptions{}); !errors.Is(e, ast.ErrASTPending) {
					t.Fatal("stopped call reused", e)
				}
			} else {
				complete = true
				if st != RewriteDone {
					t.Fatal(st)
				}
				astTestMust(t, p.SourceReady())
			}
			for _, root := range p.vector {
				refs, e := m.GetRefCount(root)
				astTestMust(t, e)
				if refs == 0 {
					t.Fatal("native addition root lost its reference")
				}
			}
			history := p.SourceReady()
			astTestMust(t, p.Close())
			if p.vector != nil || p.visited != nil || p.multiple != nil || p.flatArgs != nil {
				t.Fatal("completed cleanup retained native collections")
			}
			one, e := ast.NewASTFastMark1(m)
			astTestMust(t, e)
			twoMark, e := ast.NewASTFastMark2(m)
			astTestMust(t, e)
			for _, x := range xs {
				a, e := one.IsMarked(x)
				astTestMust(t, e)
				b, e := twoMark.IsMarked(x)
				astTestMust(t, e)
				if a || b {
					t.Fatal("failed call retained mark planes")
				}
			}
			astTestMust(t, twoMark.Close())
			astTestMust(t, one.Close())
			if p.SourceReady() != history {
				t.Fatal("cleanup changed history")
			}
		})
	}
	if !complete || failures == 0 {
		t.Fatal("reservation horizon", complete, failures)
	}
}

func numericHoistRefusal(e error) bool {
	if errors.Is(e, arena.ErrInjectedFailure) {
		return true
	}
	var f umemory.Failure
	return errors.As(e, &f) && f.Kind == umemory.FailureMemout
}

func TestBVPolynomialNonflatAdditionEveryActualNumericReservation(t *testing.T) {
	complete, failures := false, 0
	for ordinal := int64(0); ordinal < 200 && !complete; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			budget := arena.NewBudget(1 << 27)
			rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64, Transport: func(f umemory.Failure) error { return f }})
			astTestMust(t, e)
			q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}}})
			astTestMust(t, e)
			sy, e := usymbol.NewManager(usymbol.Config{Budget: budget})
			astTestMust(t, e)
			m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: sy, Numerals: q, Budget: budget})
			astTestMust(t, e)
			t.Cleanup(func() {
				astTestMust(t, m.Close())
				astTestMust(t, sy.Close())
				astTestMust(t, q.CloseChecked())
				astTestMust(t, rt.Close())
				if budget.Used() != 0 {
					t.Fatal("actual owner graph retained", budget.Used())
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
			v, e := q.NewValue()
			astTestMust(t, e)
			astTestMust(t, q.SetString(&v, "18446744073709551615"))
			c, e := m.BitVectorNumeral(&v, 65)
			astTestMust(t, e)
			astTestPin(t, m, c)
			astTestMust(t, q.Del(&v))
			term := bvMergeMul(t, m, c, x)
			astTestPin(t, m, term)
			args := ast.SliceArguments([]ast.ASTHandle{term, term, x})
			out, e := m.NewRef(ast.ASTHandle{})
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			p, e := NewBVPolynomialAdditionCore(m)
			astTestMust(t, e)
			defer func() { astTestMust(t, p.Close()) }()
			budget.FailAfter(ordinal)
			st, body := p.Rewrite(s, args, out, BVPolynomialAdditionOptions{})
			budget.FailAfter(-1)
			if body != nil {
				failures++
				if !numericHoistRefusal(body) {
					t.Fatal("actual refusal", ordinal, body)
				}
				if _, e = p.Rewrite(s, args, out, BVPolynomialAdditionOptions{}); !errors.Is(e, ast.ErrASTPending) {
					t.Fatal("failed call retried", e)
				}
			} else {
				complete = true
				if st != RewriteDone {
					t.Fatal(st)
				}
				got, e := m.RefGet(out)
				astTestMust(t, e)
				literal, e := m.GetArg(got, 0)
				astTestMust(t, e)
				value, e := q.NewValue()
				astTestMust(t, e)
				var width uint32
				yes, e := m.BitVectorNumeralValue(literal, &value, &width)
				astTestMust(t, e)
				text, e := q.String(&value)
				astTestMust(t, e)
				astTestMust(t, q.Del(&value))
				if !yes || width != 65 || text != "36893488147419103231" {
					t.Fatal("collected coefficient", yes, width, text)
				}
			}
			ready := p.SourceReady()
			astTestMust(t, p.Close())
			if p.SourceReady() != ready {
				t.Fatal("owning cleanup changed source history")
			}
		})
	}
	if !complete || failures == 0 {
		t.Fatal("actual numeric horizon", complete, failures)
	}
}

func TestBVPolynomialNonflatAdditionConstructorRefusalAndCopyGuards(t *testing.T) {
	// Only the actual addition and shared-binding headers are charged.
	for ordinal := int64(0); ordinal < 2; ordinal++ {
		m, _ := bvAddFixture(t)
		before := m.Budget().Used()
		m.Budget().FailAfter(ordinal)
		p, e := NewBVPolynomialAdditionCore(m)
		m.Budget().FailAfter(-1)
		if p != nil || !errors.Is(e, arena.ErrInjectedFailure) || m.Budget().Used() != before {
			t.Fatal("constructor prefix", ordinal, e)
		}
	}
	m, _ := bvAddFixture(t)
	p, e := NewBVPolynomialAdditionCore(m)
	astTestMust(t, e)
	copied := *p
	if _, e = copied.Rewrite(ast.ASTHandle{}, ast.AppArguments{}, ast.ASTRef{}, BVPolynomialAdditionOptions{}); !errors.Is(e, ast.ErrASTHandle) {
		t.Fatal("copied owner accepted", e)
	}
	astTestMust(t, p.Close())
	astTestMust(t, p.Close())
}

func TestBVPolynomialAdditionNativeMembershipPreservesForeignMarkPlanes(t *testing.T) {
	m, _ := bvAddFixture(t)
	xs := bvMergeVars(t, m, 48)
	currentSort, e := m.GetSort(xs[0])
	astTestMust(t, e)
	want := bvMergeMul(t, m, bvAddNum(t, m, 2, 8), bvMulHoistSum(t, m, xs...))
	one, e := ast.NewASTFastMark1(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, one.Close()) }()
	two, e := ast.NewASTFastMark2(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, two.Close()) }()
	var args []ast.ASTHandle
	for i := len(xs) - 1; i >= 0; i-- {
		astTestMust(t, one.Mark(xs[i], true))
		astTestMust(t, two.Mark(xs[i], true))
		args = append(args, xs[i], xs[i])
	}
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	p, e := NewBVPolynomialAdditionCore(m)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	st, e := p.Rewrite(currentSort, ast.SliceArguments(args), out, BVPolynomialAdditionOptions{SortSums: true, HoistCoefficients: true})
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteDone || got != want {
		t.Fatal("native duplicate/coefficient grouping", st, got, want)
	}
	astTestMust(t, p.Close())
	for _, x := range xs {
		marked1, e := one.IsMarked(x)
		astTestMust(t, e)
		marked2, e := two.IsMarked(x)
		astTestMust(t, e)
		if !marked1 || !marked2 {
			t.Fatal("addition changed another caller's marks")
		}
	}
}
