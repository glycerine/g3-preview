// Source: src/ast/rewriter/bv_rewriter.cpp:1905-1969.
package rewriter

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"testing"
)

func bvNotTestOwner(t *testing.T, m *ast.ASTManager, arithmetic, simpl bool) *BVRewriter {
	t.Helper()
	c, g, _ := bvPolynomialParamsFixture(t)
	local, e := c.NewValue()
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, c.Del(&local)) })
	for _, v := range []struct {
		key   string
		value bool
	}{{"bvnot2arith", arithmetic}, {"bv_not_simpl", simpl}} {
		key := v.key
		astTestMust(t, c.SetBoolName(&local, &key, v.value))
	}
	p, e := NewBVRewriter(m, c, g, &local)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, p.Close()) })
	return p
}

func bvNotTestInput(t *testing.T, m *ast.ASTManager, mode string) (ast.ASTHandle, RewriteStatus) {
	t.Helper()
	xs := bvMergeVars(t, m, 2)
	x, y := xs[0], xs[1]
	one, minus := bvAddNum(t, m, 1, 8), bvAddNum(t, m, 255, 8)
	var h ast.ASTHandle
	var e error
	st := Rewrite1
	switch mode {
	case "not":
		h, e = m.MkBitVectorNot(x)
		st = RewriteDone
	case "numeric":
		h = bvAddNum(t, m, 127, 8)
		st = RewriteDone
	case "concat", "concatwide":
		args := []ast.ASTHandle{x, y}
		if mode == "concatwide" {
			args = make([]ast.ASTHandle, 40)
			for i := range args {
				args[i] = x
			}
		}
		h, e = m.MkBitVectorConcat(args)
		st = Rewrite2
	case "arith":
		h = x
	case "mulleft":
		h, e = m.MkBitVectorMul(minus, x)
	case "mulright":
		h, e = m.MkBitVectorMul(x, minus)
	case "mulfailed":
		h, e = m.MkBitVectorMul(one, x)
		st = RewriteFailed
	case "add":
		nx, q := m.MkBitVectorNot(x)
		astTestMust(t, q)
		astTestPin(t, m, nx)
		h, e = m.MkBitVectorAdd(nx, one)
	case "addnumeric":
		h, e = m.MkBitVectorAdd(bvAddNum(t, m, 67, 8), bvAddNum(t, m, 81, 8))
	case "addrightfailed":
		h, e = m.MkBitVectorAdd(one, x)
		st = RewriteFailed
	case "addleftfailed":
		h, e = m.MkBitVectorAdd(x, one)
		st = RewriteFailed
	case "addternary":
		h, e = m.MkFamilyApp(mustBVFamily(t, m), 4, nil, []ast.ASTHandle{one, one, one})
		st = RewriteFailed
	case "failed":
		h = x
		st = RewriteFailed
	default:
		t.Fatal(mode)
	}
	astTestMust(t, e)
	astTestPin(t, m, h)
	return h, st
}
func mustBVFamily(t *testing.T, m *ast.ASTManager) ast.FamilyID {
	t.Helper()
	f, e := m.BitVectorUtilityFamilyID()
	astTestMust(t, e)
	return f
}

func TestBVRewriterNotOriginalBranches(t *testing.T) {
	for _, mode := range []string{"not", "numeric", "concat", "concatwide", "arith", "mulleft", "mulright", "mulfailed", "add", "addnumeric", "addrightfailed", "addleftfailed", "addternary", "failed"} {
		t.Run(mode, func(t *testing.T) {
			m, _, _, _ := boolRewriterConfigFixture(t)
			_, e := m.RegisterArithmeticPlugin()
			astTestMust(t, e)
			_, e = m.RegisterBitVectorPlugin()
			astTestMust(t, e)
			p := bvNotTestOwner(t, m, mode == "arith", true)
			arg, wantStatus := bvNotTestInput(t, m, mode)
			out, e := m.NewRef(arg)
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelRef(&out)) }()
			st, e := p.RewriteNot(arg, out)
			astTestMust(t, e)
			got, e := m.RefGet(out)
			astTestMust(t, e)
			if st != wantStatus {
				t.Fatal("status", st, wantStatus)
			}
			if st == RewriteFailed {
				if got != arg {
					t.Fatal("failed changed output")
				}
				st, e = p.RewriteNot(arg, ast.ASTRef{})
				astTestMust(t, e)
				if st != RewriteFailed {
					t.Fatal("nil output read")
				}
				return
			}
			switch mode {
			case "not":
				x, e := m.GetArg(arg, 0)
				astTestMust(t, e)
				if got != x {
					t.Fatal("double not")
				}
			case "numeric":
				if got != bvAddNum(t, m, 128, 8) {
					t.Fatal("complement")
				}
			case "concat", "concatwide":
				count, e := m.GetNumArgs(arg)
				astTestMust(t, e)
				n, e := m.GetNumArgs(got)
				astTestMust(t, e)
				kind, e := m.GetDeclKind(got)
				astTestMust(t, e)
				if n != count || kind != 37 {
					t.Fatal("concat shape")
				}
				for i := uint32(0); i < n; i++ {
					old, e := m.GetArg(arg, i)
					astTestMust(t, e)
					child, e := m.GetArg(got, i)
					astTestMust(t, e)
					k, e := m.GetDeclKind(child)
					astTestMust(t, e)
					a, e := m.GetArg(child, 0)
					astTestMust(t, e)
					if k != 32 || a != old {
						t.Fatal("concat order", i)
					}
				}
			case "arith", "mulleft", "mulright":
				k, e := m.GetDeclKind(got)
				astTestMust(t, e)
				left, e := m.GetArg(got, 0)
				astTestMust(t, e)
				right, e := m.GetArg(got, 1)
				astTestMust(t, e)
				target := arg
				wantKind := int32(5)
				if mode != "arith" {
					wantKind = 4
					i := uint32(1)
					if mode == "mulright" {
						i = 0
					}
					target, e = m.GetArg(arg, i)
					astTestMust(t, e)
				}
				if k != wantKind || left != bvAddNum(t, m, 255, 8) || right != target {
					t.Fatal("minus-one shape", k)
				}
			case "addnumeric":
				count, e := m.GetNumArgs(got)
				astTestMust(t, e)
				if count != 3 {
					t.Fatal("numeric add lost raw source shape")
				}
				for i, want := range []uint32{1, 188, 174} {
					child, e := m.GetArg(got, uint32(i))
					astTestMust(t, e)
					if child != bvAddNum(t, m, want, 8) {
						t.Fatal("numeric add operand order", i)
					}
				}
			case "add":
				k, e := m.GetDeclKind(got)
				astTestMust(t, e)
				n, e := m.GetNumArgs(got)
				astTestMust(t, e)
				if k != 4 || n != 3 {
					t.Fatal("raw ternary add")
				}
				first, e := m.GetArg(got, 0)
				astTestMust(t, e)
				third, e := m.GetArg(got, 2)
				astTestMust(t, e)
				ns, e := m.GetArg(arg, 0)
				astTestMust(t, e)
				x, e := m.GetArg(ns, 0)
				astTestMust(t, e)
				second, e := m.GetArg(got, 1)
				astTestMust(t, e)
				if first != bvAddNum(t, m, 1, 8) || second != x || third != bvAddNum(t, m, 254, 8) {
					t.Fatal("ordered one/ns/nt")
				}
			}
			if p.bvnot != nil || p.negatable != nil {
				t.Fatal("source owners survived")
			}
		})
	}
}

func TestBVRewriterNotEveryColdReservation(t *testing.T) {
	for _, mode := range []string{"not", "numeric", "concatwide", "arith", "mulleft", "mulright", "add", "addnumeric", "addrightfailed", "addleftfailed", "failed"} {
		complete, published, rightBeforeLeft := false, false, false
		for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m, _, _, _ := boolRewriterConfigFixture(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				_, e = m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				p := bvNotTestOwner(t, m, mode == "arith", true)
				arg, wantStatus := bvNotTestInput(t, m, mode)
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
				st, body := p.RewriteNot(arg, out)
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
					if st != wantStatus {
						t.Fatal("status", st, wantStatus)
					}
					if st == RewriteFailed && got != old {
						t.Fatal("failed publication")
					}
				} else {
					if !numericHoistRefusal(body) {
						t.Fatal(ordinal, body)
					}
					if got != old {
						published = true
						if st == RewriteFailed {
							t.Fatal("publication status")
						}
					}
					_, replay := p.RewriteNot(arg, out)
					if replay != body {
						t.Fatal("stopped replay", replay, body)
					}
				}
				if mode == "add" && body != nil && p.bvnot != nil && p.bvnot.nt != (ast.ASTRef{}) && p.bvnot.ns != (ast.ASTRef{}) {
					nt, x := m.RefGet(p.bvnot.nt)
					astTestMust(t, x)
					ns, x := m.RefGet(p.bvnot.ns)
					astTestMust(t, x)
					if !nt.IsNull() && ns.IsNull() {
						rightBeforeLeft = true
						if nt != bvAddNum(t, m, 254, 8) {
							t.Fatal("right helper result before left", nt)
						}
					}
				}
				cause := p.cause
				astTestMust(t, p.Close())
				astTestMust(t, p.Close())
				if p.cause != cause {
					t.Fatal("history changed")
				}
			})
		}
		if mode == "add" && !rightBeforeLeft {
			t.Fatal("right-before-left failure frontier absent")
		}
		if !complete {
			t.Fatal("campaign incomplete", mode)
		}
		if mode != "failed" && mode != "addrightfailed" && mode != "addleftfailed" && !published {
			t.Fatal("native replacement was not published", mode)
		}
	}
}

func TestBVRewriterNotFlagPrecedenceAndErasedIdentity(t *testing.T) {
	m, _, _, _ := boolRewriterConfigFixture(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	p := bvNotTestOwner(t, m, true, true)
	arg, _ := bvNotTestInput(t, m, "add")
	out, e := m.NewRef(arg)
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	st, e := p.RewriteNot(arg, out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	kind, e := m.GetDeclKind(got)
	astTestMust(t, e)
	if st != Rewrite1 || kind != 5 {
		t.Fatal("arithmetic flag precedes simplification")
	}
	concat, _ := bvNotTestInput(t, m, "concat")
	st, e = p.RewriteNot(concat, out)
	astTestMust(t, e)
	if st != Rewrite2 {
		t.Fatal("concat precedes flags")
	}
	numeric, _ := bvNotTestInput(t, m, "numeric")
	st, e = p.RewriteNot(numeric, out)
	astTestMust(t, e)
	if st != RewriteDone {
		t.Fatal("numeric precedes flags")
	}
	not, _ := bvNotTestInput(t, m, "not")
	x, e := m.GetArg(not, 0)
	astTestMust(t, e)
	astTestMust(t, m.SetRef(out, x))
	before := m.Stats()
	m.Budget().FailAfter(0)
	st, e = p.RewriteNot(not, out)
	m.Budget().FailAfter(-1)
	astTestMust(t, e)
	if st != RewriteDone || m.Stats() != before || p.bvnot != nil {
		t.Fatal("double negation constructed default val")
	}
	disabled := bvNotTestOwner(t, m, false, false)
	st, e = disabled.RewriteNot(arg, ast.ASTRef{})
	astTestMust(t, e)
	if st != RewriteFailed {
		t.Fatal("disabled simplification")
	}
}

func TestBVRewriterNotOriginalNonunaryGuard(t *testing.T) {
	m, _, _, _ := boolRewriterConfigFixture(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	p := bvNotTestOwner(t, m, false, false)
	xs := bvMergeVars(t, m, 2)
	sort, e := m.BitVectorSort(8)
	astTestMust(t, e)
	info, e := m.NewFuncDeclInfo(fid, 32, nil, ast.ASTFuncFlags{}, false)
	astTestMust(t, e)
	decl, e := m.MkFuncDecl(astTestSymbol(t, m, "bvnot-raw-two"), []ast.ASTHandle{sort, sort}, sort, info)
	astTestMust(t, e)
	astTestMust(t, m.DelFuncDeclInfo(&info))
	arg, e := m.MkApp(decl, xs)
	astTestMust(t, e)
	astTestPin(t, m, arg)
	out, e := m.NewRef(xs[1])
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	st, e := p.RewriteNot(arg, out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteDone || got != xs[0] {
		t.Fatal("original top guard has no unary assertion", st, got)
	}
}
