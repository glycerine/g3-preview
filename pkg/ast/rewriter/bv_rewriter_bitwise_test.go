// Source: src/ast/rewriter/bv_rewriter.cpp:1971-1998.
package rewriter

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

func TestBVRewriterOriginalNegatedBitwiseAlgorithms(t *testing.T) {
	for _, num := range []int{1, 3, 17, 40} {
		m := astTestManager(t)
		_, e := m.RegisterArithmeticPlugin()
		satOK(t, e)
		_, e = m.RegisterBitVectorPlugin()
		satOK(t, e)
		sort, e := m.BitVectorSort(8)
		satOK(t, e)
		var args []ast.ASTHandle
		values := map[ast.ASTHandle]uint64{}
		for i := 0; i < num; i++ {
			x, e := m.MkVar(uint32(i), sort)
			satOK(t, e)
			astTestPin(t, m, x)
			args = append(args, x)
			values[x] = uint64((i*47 + 19) & 255)
		}
		out := astTestPin(t, m, args[0])
		var eval func(ast.ASTHandle) uint64
		eval = func(h ast.ASTHandle) uint64 {
			if v, ok := values[h]; ok {
				return v
			}
			kind, e := m.GetDeclKind(h)
			satOK(t, e)
			n, e := m.GetNumArgs(h)
			satOK(t, e)
			value := uint64(0)
			for i := uint32(0); i < n; i++ {
				arg, e := m.GetArg(h, i)
				satOK(t, e)
				v := eval(arg)
				switch kind {
				case 32:
					value = (^v) & 255
				case 31:
					value |= v
				case 33:
					value ^= v
				default:
					t.Fatal("unexpected reducer operator", kind)
				}
			}
			return value
		}
		for op, rewrite := range []func(*ast.ASTManager, []ast.ASTHandle, ast.ASTRef) (RewriteStatus, error){RewriteBitVectorNAND, RewriteBitVectorNOR, RewriteBitVectorXNOR, RewriteBitVectorAND} {
			status, e := rewrite(m, args, out)
			satOK(t, e)
			got, e := m.RefGet(out)
			satOK(t, e)
			want := uint64(0)
			if op == 0 || op == 3 {
				want = 255
			}
			for _, arg := range args {
				switch op {
				case 0, 3:
					want &= values[arg]
				case 1:
					want |= values[arg]
				case 2:
					want ^= values[arg]
				}
			}
			expectedStatus := Rewrite2
			if op == 3 {
				expectedStatus = Rewrite3
			} else {
				want = (^want) & 255
			}
			if status != expectedStatus || eval(got) != want {
				t.Fatal("bitwise meaning/status", num, op, status, eval(got), want)
			}
			if op == 3 {
				for pattern := uint64(0); pattern < 256; pattern++ {
					for _, arg := range args {
						values[arg] = pattern
					}
					if eval(got) != pattern {
						t.Fatal("AND uniform truth table", num, pattern, eval(got))
					}
				}
			}
			if op == 0 || op == 3 {
				source := got
				if op == 3 {
					k, e := m.GetDeclKind(got)
					satOK(t, e)
					n, e := m.GetNumArgs(got)
					satOK(t, e)
					if k != 32 || n != 1 {
						t.Fatal("AND outer NOT", k, n)
					}
					source, e = m.GetArg(got, 0)
					satOK(t, e)
				}
				kind, e := m.GetDeclKind(source)
				satOK(t, e)
				n, e := m.GetNumArgs(source)
				satOK(t, e)
				if kind != 31 || n != uint32(num) {
					t.Fatal("De Morgan OR shape", op, kind, n)
				}
				for i, arg := range args {
					neg, e := m.GetArg(source, uint32(i))
					satOK(t, e)
					k, e := m.GetDeclKind(neg)
					satOK(t, e)
					base, e := m.GetArg(neg, 0)
					satOK(t, e)
					if k != 32 || base != arg {
						t.Fatal("De Morgan argument order", op, i)
					}
				}
			} else {
				kind, e := m.GetDeclKind(got)
				satOK(t, e)
				inner, e := m.GetArg(got, 0)
				satOK(t, e)
				k, e := m.GetDeclKind(inner)
				satOK(t, e)
				expected := int32(31)
				if op == 2 {
					expected = 33
				}
				if kind != 32 || k != expected {
					t.Fatal("negated source shape", kind, k)
				}
			}
		}
	}
}

func TestBVRewriterNANDWarmNativeBufferIgnoresBudgetFailure(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	satOK(t, e)
	_, e = m.RegisterBitVectorPlugin()
	satOK(t, e)
	sort, e := m.BitVectorSort(8)
	satOK(t, e)
	x, e := m.MkVar(0, sort)
	satOK(t, e)
	astTestPin(t, m, x)
	args := make([]ast.ASTHandle, 17)
	for i := range args {
		args[i] = x
	}
	out := astTestPin(t, m, x)
	_, e = RewriteBitVectorNAND(m, args, out)
	satOK(t, e)
	satOK(t, m.SetRef(out, x))
	before := m.Budget().Used()
	m.Budget().FailAfter(0)
	_, body := RewriteBitVectorNAND(m, args, out)
	m.Budget().FailAfter(-1)
	got, e := m.RefGet(out)
	satOK(t, e)
	if ast.ASTSourceDebug {
		if !errors.Is(body, arena.ErrInjectedFailure) || got != x || m.Budget().Used() != before {
			t.Fatal("debug direct AST allocation refusal must keep the NAND output", body, got, before, m.Budget().Used())
		}
		return
	}
	if body != nil || got.IsNull() || m.Budget().Used() != before {
		t.Fatal("native buffer output/charge", body, got, before, m.Budget().Used())
	}
}

func TestBVRewriterBitwiseEveryColdReservation(t *testing.T) {
	for op, rewrite := range []func(*ast.ASTManager, []ast.ASTHandle, ast.ASTRef) (RewriteStatus, error){RewriteBitVectorNAND, RewriteBitVectorNOR, RewriteBitVectorXNOR, RewriteBitVectorAND} {
		success := false
		failures := 0
		for ordinal := int64(0); ordinal < 128; ordinal++ {
			t.Run(strconv.Itoa(op)+"/"+strconv.FormatInt(ordinal, 10), func(t *testing.T) {
				m := astTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				satOK(t, e)
				_, e = m.RegisterBitVectorPlugin()
				satOK(t, e)
				sort, e := m.BitVectorSort(8)
				satOK(t, e)
				x, e := m.MkVar(0, sort)
				satOK(t, e)
				astTestPin(t, m, x)
				y, e := m.MkVar(1, sort)
				satOK(t, e)
				astTestPin(t, m, y)
				out := astTestPin(t, m, x)
				m.Budget().FailAfter(ordinal)
				args := []ast.ASTHandle{x, y, x}
				if op == 3 {
					args = make([]ast.ASTHandle, 17)
					for i := range args {
						args[i] = x
						if i%2 != 0 {
							args[i] = y
						}
					}
				}
				status, body := rewrite(m, args, out)
				m.Budget().FailAfter(-1)
				if body != nil {
					failures++
					if !errors.Is(body, arena.ErrInjectedFailure) {
						t.Fatal("unexpected refusal", body)
					}
					got, e := m.RefGet(out)
					satOK(t, e)
					if got != x {
						t.Fatal("failed factory published output")
					}
				} else {
					success = true
					expected := Rewrite2
					if op == 3 {
						expected = Rewrite3
					}
					if status != expected {
						t.Fatal("rewrite depth", status)
					}
				}
			})
			if success {
				break
			}
		}
		if !success {
			t.Fatal("actual reservation horizon", op, success, failures)
		}
	}
}

func TestBVRewriterANDWarmNativeBufferIgnoresBudgetFailure(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	satOK(t, e)
	_, e = m.RegisterBitVectorPlugin()
	satOK(t, e)
	sort, e := m.BitVectorSort(8)
	satOK(t, e)
	x, e := m.MkVar(0, sort)
	satOK(t, e)
	astTestPin(t, m, x)
	args := make([]ast.ASTHandle, 17)
	for i := range args {
		args[i] = x
	}
	out := astTestPin(t, m, x)
	_, e = RewriteBitVectorAND(m, args, out)
	satOK(t, e)
	satOK(t, m.SetRef(out, x))
	before := m.Budget().Used()
	m.Budget().FailAfter(0)
	_, body := RewriteBitVectorAND(m, args, out)
	m.Budget().FailAfter(-1)
	got, e := m.RefGet(out)
	satOK(t, e)
	if ast.ASTSourceDebug {
		if !errors.Is(body, arena.ErrInjectedFailure) || got != x || m.Budget().Used() != before {
			t.Fatal("debug direct AST allocation refusal must keep the AND output", body, got, before, m.Budget().Used())
		}
		return
	}
	if body != nil || got.IsNull() || m.Budget().Used() != before {
		t.Fatal("native buffer output/charge", body, got, before, m.Budget().Used())
	}
}
