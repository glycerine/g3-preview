// Source: src/ast/rewriter/bv_rewriter.cpp:1513-1523,1555-1566.
package rewriter

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestBVRewriterOriginalRepeatAndZeroExtend(t *testing.T) {
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
	out := astTestPin(t, m, x)
	for _, n := range []uint32{1, 2, 17, 40} {
		status, e := RewriteBitVectorRepeat(m, n, x, out)
		satOK(t, e)
		got, e := m.RefGet(out)
		satOK(t, e)
		if n == 1 {
			if status != RewriteDone || got != x {
				t.Fatal("repeat identity")
			}
			continue
		}
		if status != Rewrite1 {
			t.Fatal("repeat rewrite depth", status)
		}
		size, e := m.BitVectorSize(got)
		satOK(t, e)
		num, e := m.GetNumArgs(got)
		satOK(t, e)
		kind, e := m.GetDeclKind(got)
		satOK(t, e)
		if size != n*8 || num != n || kind != 37 {
			t.Fatal("repeat shape", size, num, kind)
		}
		for i := uint32(0); i < n; i++ {
			arg, e := m.GetArg(got, i)
			satOK(t, e)
			if arg != x {
				t.Fatal("repeat operand", i)
			}
		}
	}
	for _, n := range []uint32{0, 1, 8, 65, 257} {
		status, e := RewriteBitVectorZeroExtend(m, n, x, out)
		satOK(t, e)
		got, e := m.RefGet(out)
		satOK(t, e)
		if n == 0 {
			if status != RewriteDone || got != x {
				t.Fatal("zero extension identity")
			}
			continue
		}
		if status != Rewrite1 {
			t.Fatal("zero extension rewrite depth", status)
		}
		first, e := m.GetArg(got, 0)
		satOK(t, e)
		second, e := m.GetArg(got, 1)
		satOK(t, e)
		zero, e := m.IsBitVectorZero(first)
		satOK(t, e)
		width, e := m.BitVectorSize(first)
		satOK(t, e)
		size, e := m.BitVectorSize(got)
		satOK(t, e)
		if !zero || width != n || size != 8+n || second != x {
			t.Fatal("zero extension source shape", zero, width, size, second)
		}
	}
	// Both identity paths erase all buffer/numerical/factory work.
	satOK(t, m.SetRef(out, x))
	before := m.Stats()
	m.Budget().FailAfter(0)
	_, e = RewriteBitVectorRepeat(m, 1, x, out)
	satOK(t, e)
	_, e = RewriteBitVectorZeroExtend(m, 0, x, out)
	satOK(t, e)
	m.Budget().FailAfter(-1)
	if m.Stats() != before {
		t.Fatal("identity allocated")
	}
}

func TestBVRewriterRepeatAndZeroExtendEveryColdReservation(t *testing.T) {
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
				sort, e := m.BitVectorSort(8)
				satOK(t, e)
				x, e := m.MkVar(0, sort)
				satOK(t, e)
				astTestPin(t, m, x)
				out := astTestPin(t, m, x)
				m.Budget().FailAfter(ordinal)
				var body error
				var status RewriteStatus
				if op == 0 {
					status, body = RewriteBitVectorRepeat(m, 17, x, out)
				} else {
					status, body = RewriteBitVectorZeroExtend(m, 65, x, out)
				}
				m.Budget().FailAfter(-1)
				if body != nil {
					failures++
					if !errors.Is(body, arena.ErrInjectedFailure) {
						var f umemory.Failure
						if !errors.As(body, &f) || f.Kind != umemory.FailureMemout {
							t.Fatal("unexpected refusal", body)
						}
					}
					got, e := m.RefGet(out)
					satOK(t, e)
					if got != x {
						t.Fatal("factory failure published output")
					}
				} else {
					success = true
					if status != Rewrite1 {
						t.Fatal("rewrite depth", status)
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
