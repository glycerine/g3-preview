package rewriter

// Source: src/ast/rewriter/bv_rewriter.cpp:1971-1998.

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func bvDerivedBitwiseCall(p *BVRewriter, op string, args ast.AppArguments, out ast.ASTRef) (RewriteStatus, error) {
	switch op {
	case "and":
		return p.RewriteAND(args, out)
	case "nand":
		return p.RewriteNAND(args, out)
	case "nor":
		return p.RewriteNOR(args, out)
	case "xnor":
		return p.RewriteXNOR(args, out)
	default:
		panic(op)
	}
}
func TestBVRewriterDerivedBitwiseOriginalShape(t *testing.T) {
	for _, op := range []string{"and", "nand", "nor", "xnor"} {
		for _, count := range []int{1, 3, 17, 40} {
			t.Run(fmt.Sprintf("%s/%d", op, count), func(t *testing.T) {
				m, _, _, _ := boolRewriterConfigFixture(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				p := bvNotTestOwner(t, m, false, false)
				xs := bvMergeVars(t, m, 2)
				args := make([]ast.ASTHandle, count)
				for i := range args {
					args[i] = xs[i%2]
				}
				out, e := m.NewRef(xs[0])
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				st, e := bvDerivedBitwiseCall(p, op, ast.SliceArguments(args), out)
				astTestMust(t, e)
				wantStatus := Rewrite2
				if op == "and" {
					wantStatus = Rewrite3
				}
				if st != wantStatus {
					t.Fatal("depth", st, wantStatus)
				}
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if op != "nand" {
					kind, e := m.GetDeclKind(got)
					astTestMust(t, e)
					n, e := m.GetNumArgs(got)
					astTestMust(t, e)
					if kind != 32 || n != 1 {
						t.Fatal("outer NOT shape", kind, n)
					}
					got, e = m.GetArg(got, 0)
					astTestMust(t, e)
				}
				kind, e := m.GetDeclKind(got)
				astTestMust(t, e)
				n, e := m.GetNumArgs(got)
				astTestMust(t, e)
				wantKind := int32(31)
				if op == "xnor" {
					wantKind = 33
				}
				if kind != wantKind || n != uint32(count) {
					t.Fatal("inner shape", kind, n)
				}
				for i, a := range args {
					h, e := m.GetArg(got, uint32(i))
					astTestMust(t, e)
					if op == "and" || op == "nand" {
						k, e := m.GetDeclKind(h)
						astTestMust(t, e)
						if k != 32 {
							t.Fatal("De Morgan NOT")
						}
						h, e = m.GetArg(h, 0)
						astTestMust(t, e)
					}
					if h != a {
						t.Fatal("operand order", i)
					}
				}
				if len(p.bitwiseArgs) != 0 {
					t.Fatal("buffer survived source return")
				}
			})
		}
	}
}

func TestBVRewriterDerivedBitwiseEveryColdReservation(t *testing.T) {
	for _, op := range []string{"and", "nand", "nor", "xnor"} {
		complete, published, retained := false, false, false
		for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				m, _, _, _ := boolRewriterConfigFixture(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				p := bvNotTestOwner(t, m, false, false)
				xs := bvMergeVars(t, m, 2)
				args := make([]ast.ASTHandle, 40)
				for i := range args {
					args[i] = xs[i%2]
				}
				sort, e := m.BitVectorSort(8)
				astTestMust(t, e)
				var children [32]ast.ASTHandle
				for i := range children {
					children[i], e = m.MkVar(uint32(i+100), sort)
					astTestMust(t, e)
				}
				old, e := m.MkBitVectorOr(children[:])
				astTestMust(t, e)
				out, e := m.NewRef(old)
				astTestMust(t, e)
				defer func() { astTestMust(t, m.DelRef(&out)) }()
				m.Budget().FailAfter(ordinal)
				st, body := bvDerivedBitwiseCall(p, op, ast.SliceArguments(args), out)
				m.Budget().FailAfter(-1)
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if body == nil {
					complete = true
					nativeResult, nativeErr := m.RefGet(out)
					astTestMust(t, nativeErr)
					if nativeResult != old {
						published = true
					}
					want := Rewrite2
					if op == "and" {
						want = Rewrite3
					}
					if st != want {
						t.Fatal("depth", st, want)
					}
					if len(p.bitwiseArgs) != 0 {
						t.Fatal("source buffer leaked")
					}
				} else {
					if !numericHoistRefusal(body) {
						t.Fatal(body)
					}
					if len(p.bitwiseArgs) > 0 {
						retained = true
					}
					if got != old {
						published = true
						if st == RewriteFailed {
							t.Fatal("publication status")
						}
					}
					_, replay := bvDerivedBitwiseCall(p, op, ast.SliceArguments(args), out)
					if replay != body {
						t.Fatal("stopped replay", replay, body)
					}
				}
				cause := p.cause
				astTestMust(t, p.Close())
				astTestMust(t, p.Close())
				if p.cause != cause || len(p.bitwiseArgs) != 0 {
					t.Fatal("own cleanup/history")
				}
			})
		}
		if !complete || !published {
			t.Fatal("native publication campaign incomplete", op, complete, published)
		}
		if (op == "and" || op == "nand") && !retained {
			t.Fatal("stopped borrowed buffer not retained", op)
		}
	}
}

func TestBVRewriterDerivedBitwiseBorrowsAppArguments(t *testing.T) {
	m, _, _, _ := boolRewriterConfigFixture(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	p := bvNotTestOwner(t, m, false, false)
	xs := bvMergeVars(t, m, 2)
	source, e := m.MkBitVectorOr([]ast.ASTHandle{xs[0], xs[1], xs[0]})
	astTestMust(t, e)
	astTestPin(t, m, source)
	args, e := m.AppArgumentView(source)
	astTestMust(t, e)
	out, e := m.NewRef(xs[0])
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	for _, op := range []string{"and", "nand", "nor", "xnor"} {
		_, e = bvDerivedBitwiseCall(p, op, args, out)
		astTestMust(t, e)
		h, e := m.RefGet(out)
		astTestMust(t, e)
		astTestPin(t, m, h)
	}
	// NOR/XNOR have no ptr_buffer. The original warm small factory fits its
	// existing release storage, while DEBUG still reserves a candidate.
	for _, op := range []string{"nor", "xnor"} {
		warmOwner := bvNotTestOwner(t, m, false, false)
		before := m.Budget().Used()
		m.Budget().FailAfter(0)
		_, e = bvDerivedBitwiseCall(warmOwner, op, args, out)
		m.Budget().FailAfter(-1)
		if ast.ASTSourceDebug {
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal("DEBUG candidate reservation", op, e)
			}
		} else {
			astTestMust(t, e)
		}
		if len(warmOwner.bitwiseArgs) != 0 {
			t.Fatal("NOR/XNOR created a borrowed buffer")
		}
		if m.Budget().Used() != before {
			t.Fatal("borrowed arguments allocated", op)
		}
	}
}
