// Source: src/ast/rewriter/bv_rewriter.cpp:2000-2021.
package rewriter

import (
	"errors"
	"math/big"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
)

func TestBVRewriterOriginalFixedRotations(t *testing.T) {
	for _, width := range []uint32{1, 8, 65, 257} {
		m := astTestManager(t)
		_, e := m.RegisterArithmeticPlugin()
		satOK(t, e)
		_, e = m.RegisterBitVectorPlugin()
		satOK(t, e)
		p, e := NewMkExtractProc(m)
		satOK(t, e)
		t.Cleanup(func() { satOK(t, p.Close()) })
		sort, e := m.BitVectorSort(width)
		satOK(t, e)
		x, e := m.MkVar(0, sort)
		satOK(t, e)
		astTestPin(t, m, x)
		out := astTestPin(t, m, x)
		input := new(big.Int)
		for i := uint32(0); i < width; i++ {
			if (i*7+5)%11 < 6 {
				input.SetBit(input, int(i), 1)
			}
		}
		var eval func(ast.ASTHandle) *big.Int
		eval = func(h ast.ASTHandle) *big.Int {
			if h == x {
				return new(big.Int).Set(input)
			}
			kind, e := m.GetDeclKind(h)
			satOK(t, e)
			switch kind {
			case 40:
				var low, high uint32
				var base ast.ASTHandle
				yes, e := m.BitVectorExtractValue(h, &low, &high, &base)
				satOK(t, e)
				if !yes {
					t.Fatal("extract recognizer")
				}
				v := eval(base)
				v.Rsh(v, uint(low))
				mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(high-low+1)), big.NewInt(1))
				return v.And(v, mask)
			case 37:
				n, e := m.GetNumArgs(h)
				satOK(t, e)
				v := new(big.Int)
				for i := uint32(0); i < n; i++ {
					arg, e := m.GetArg(h, i)
					satOK(t, e)
					size, e := m.BitVectorSize(arg)
					satOK(t, e)
					v.Lsh(v, uint(size))
					v.Or(v, eval(arg))
				}
				return v
			default:
				t.Fatal("unexpected rotation operator", kind)
				return nil
			}
		}
		for _, n := range []uint32{0, 1, width - 1, width, width + 1, ^uint32(0)} {
			for direction, rewrite := range []func(*MkExtractProc, uint32, ast.ASTHandle, ast.ASTRef) (RewriteStatus, error){RewriteBitVectorRotateLeft, RewriteBitVectorRotateRight} {
				status, e := rewrite(p, n, x, out)
				satOK(t, e)
				got, e := m.RefGet(out)
				satOK(t, e)
				shift := n % width
				if shift == 0 || width == 1 {
					if status != RewriteDone || got != x {
						t.Fatal("identity rotation", width, n, direction, status)
					}
					continue
				}
				if status != Rewrite2 {
					t.Fatal("rotation rewrite depth", status)
				}
				want := new(big.Int)
				for i := uint32(0); i < width; i++ {
					destination := (i + shift) % width
					if direction == 1 {
						destination = (i + width - shift) % width
					}
					want.SetBit(want, int(destination), input.Bit(int(i)))
				}
				if eval(got).Cmp(want) != 0 {
					t.Fatal("independent rotation meaning", width, n, direction, eval(got), want)
				}
				// Source emits the low portion first, then the high portion.
				leftShift := shift
				if direction == 1 {
					leftShift = width - shift
				}
				first, e := m.GetArg(got, 0)
				satOK(t, e)
				second, e := m.GetArg(got, 1)
				satOK(t, e)
				for i, part := range []ast.ASTHandle{first, second} {
					var low, high uint32
					var base ast.ASTHandle
					yes, e := m.BitVectorExtractValue(part, &low, &high, &base)
					satOK(t, e)
					wantLow, wantHigh := uint32(0), width-leftShift-1
					if i == 1 {
						wantLow = width - leftShift
						wantHigh = width - 1
					}
					if !yes || low != wantLow || high != wantHigh || base != x {
						t.Fatal("source extract order", i, low, high, wantLow, wantHigh)
					}
				}
			}
		}
	}
}

func TestBVRewriterRotationIdentityErasesExtractionWork(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	satOK(t, e)
	_, e = m.RegisterBitVectorPlugin()
	satOK(t, e)
	p, e := NewMkExtractProc(m)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, p.Close()) })
	sort, e := m.BitVectorSort(8)
	satOK(t, e)
	x, e := m.MkVar(0, sort)
	satOK(t, e)
	astTestPin(t, m, x)
	out := astTestPin(t, m, x)
	before := m.Stats()
	m.Budget().FailAfter(0)
	for _, rewrite := range []func(*MkExtractProc, uint32, ast.ASTHandle, ast.ASTRef) (RewriteStatus, error){RewriteBitVectorRotateLeft, RewriteBitVectorRotateRight} {
		status, e := rewrite(p, 16, x, out)
		satOK(t, e)
		if status != RewriteDone {
			t.Fatal("identity status", status)
		}
	}
	m.Budget().FailAfter(-1)
	if m.Stats() != before || !p.cached.IsNull() {
		t.Fatal("identity reached extraction cache")
	}
}

func TestBVRewriterRotationEveryColdReservation(t *testing.T) {
	for direction, rewrite := range []func(*MkExtractProc, uint32, ast.ASTHandle, ast.ASTRef) (RewriteStatus, error){RewriteBitVectorRotateLeft, RewriteBitVectorRotateRight} {
		success := false
		failures := 0
		for ordinal := int64(0); ordinal < 128; ordinal++ {
			t.Run(strconv.Itoa(direction)+"/"+strconv.FormatInt(ordinal, 10), func(t *testing.T) {
				m := astTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				satOK(t, e)
				_, e = m.RegisterBitVectorPlugin()
				satOK(t, e)
				p, e := NewMkExtractProc(m)
				satOK(t, e)
				t.Cleanup(func() { satOK(t, p.Close()) })
				sort, e := m.BitVectorSort(65)
				satOK(t, e)
				x, e := m.MkVar(0, sort)
				satOK(t, e)
				astTestPin(t, m, x)
				out := astTestPin(t, m, x)
				m.Budget().FailAfter(ordinal)
				status, body := rewrite(p, 7, x, out)
				m.Budget().FailAfter(-1)
				got, e := m.RefGet(out)
				satOK(t, e)
				if body != nil {
					failures++
					if !errors.Is(body, arena.ErrInjectedFailure) || got != x {
						t.Fatal("refusal/output", body, got)
					}
				} else {
					success = true
					if status != Rewrite2 {
						t.Fatal("rotation status", status)
					}
				}
			})
			if success {
				break
			}
		}
		if !success || failures == 0 {
			t.Fatal("reservation horizon", direction, success, failures)
		}
	}
}
