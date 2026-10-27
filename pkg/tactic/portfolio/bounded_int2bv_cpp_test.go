package portfolio_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic/portfolio"
	"github.com/glycerine/g3/pkg/util/lbool"
)

type boundedPinnedBackend struct {
	*pbRecordingBackend
	t *testing.T
	m *ast.ASTManager
}

func (b *boundedPinnedBackend) Assert(h ast.ASTHandle) error {
	enumPin(b.t, b.m, h)
	return b.pbRecordingBackend.Assert(h)
}
func boundedHelperWidth(t *testing.T, m *ast.ASTManager, root ast.ASTHandle) uint32 {
	t.Helper()
	todo := []ast.ASTHandle{root}
	seen := map[ast.ASTHandle]bool{}
	for len(todo) > 0 {
		h := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		if seen[h] {
			continue
		}
		seen[h] = true
		kind, err := m.GetKind(h)
		enumMust(t, err)
		if kind != ast.ASTApp {
			continue
		}
		n, err := m.GetNumArgs(h)
		enumMust(t, err)
		if n == 0 {
			d, err := m.GetDecl(h)
			enumMust(t, err)
			fam, err := m.GetFamilyID(d)
			enumMust(t, err)
			if fam == ast.NullFamilyID {
				sort, err := m.GetSort(h)
				enumMust(t, err)
				yes, err := m.IsBitVectorSort(sort)
				enumMust(t, err)
				if yes {
					width, err := m.BitVectorSize(h)
					enumMust(t, err)
					return width
				}
			}
		}
		for i := uint32(0); i < n; i++ {
			a, err := m.GetArg(h, i)
			enumMust(t, err)
			todo = append(todo, a)
		}
	}
	t.Fatal("converted formula contains no helper BV constant")
	return 0
}
func TestBoundedInt2BVSolverSourceWidthsAndRangeGuards(t *testing.T) {
	for _, lo := range []int64{-5, 0, 7} {
		for _, n := range []int64{1, 2, 3, 4, 7, 8, 9} {
			t.Run(fmt.Sprint(lo, "/", n), func(t *testing.T) {
				f := enumSolverFixture(t)
				m := f.manager
				b := &boundedPinnedBackend{&pbRecordingBackend{Solver: f.s}, t, m}
				s, err := portfolio.NewBoundedInt2BVSolver(m, f.services.Context, nil, b, f.services)
				enumMust(t, err)
				t.Cleanup(func() { enumMust(t, s.Close()) })
				x := boundedConst(t, m, "width-x")
				boundedBounds(t, m, s, x, lo, lo+n-1)
				_, err = s.NumAssertions()
				enumMust(t, err)
				count := 2
				if lo != 0 && n&(n-1) != 0 {
					count++
				}
				if len(b.asserted) != count {
					t.Fatal("source range guard condition", len(b.asserted), count)
				}
				want := uint32(1)
				for bound := int64(1); bound <= n; bound *= 2 {
					want++
				}
				width := boundedHelperWidth(t, m, b.asserted[0])
				if width != want {
					t.Fatal("source get_num_bits width", width, want)
				}
			})
		}
	}
}
func TestBoundedInt2BVSolverNonlinearPrebuiltCPPAndModels(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	for _, bounds := range [][2]int64{{-7, -1}, {0, 6}, {4, 10}} {
		for _, target := range []int64{0, 1, 4, 9, 16, 25, 49, 100} {
			t.Run(fmt.Sprint(bounds, "/", target), func(t *testing.T) {
				f, s := boundedFixture(t)
				m := f.manager
				x := boundedConst(t, m, "nonlinear-x")
				boundedBounds(t, m, s, x, bounds[0], bounds[1])
				square := boundedOp(t, m, 9, x, x)
				eq, err := m.MkEq(square, boundedNumber(t, m, target))
				enumMust(t, err)
				enumPin(t, m, eq)
				enumMust(t, s.Assert(eq))
				expected := lbool.False
				value := int64(0)
				for n := bounds[0]; n <= bounds[1]; n++ {
					if n*n == target {
						expected = lbool.True
						value = n
					}
				}
				status, err := s.CheckSat(nil)
				enumMust(t, err)
				if status != expected {
					t.Fatal("independent finite-domain oracle", status, expected)
				}
				if status == lbool.True {
					md, err := s.AcquireModel()
					enumMust(t, err)
					if md == nil {
						t.Fatal("missing model")
					}
					got := boundedValue(t, m, md, f.services, x)
					enumMust(t, md.DecRef())
					actual, err := strconv.ParseInt(got, 10, 64)
					enumMust(t, err)
					if actual != value {
						t.Fatal("reconstructed nonlinear integer", actual, value)
					}
				}
				input := fmt.Sprintf("(set-logic QF_FD)(declare-const x Int)(assert (>= x %d))(assert (<= x %d))(assert (= (* x x) %d))(check-sat)\n", bounds[0], bounds[1], target)
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, reference, "-in")
				cmd.Stdin = strings.NewReader(input)
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatal(err, string(out))
				}
				want := "unsat"
				if expected == lbool.True {
					want = "sat"
				}
				if strings.TrimSpace(string(out)) != want {
					t.Fatal("prebuilt C++ mismatch or surplus output", string(out), want)
				}
			})
		}
	}
}
