// Source: src/ast/rewriter/bv_rewriter.cpp:1425-1446.
package ast

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func TestASTBitVectorLeadingZeroEveryActualReservation(t *testing.T) {
	for _, tc := range []struct {
		text           string
		steps, leading uint32
	}{{"36893488147419103231", 65, 0}, {"18446744073709551616", 65, 0}, {"0", 0, 65}, {"-1", 0, 65}} {
		complete := false
		for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", tc.text, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				fid, e := m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				sort, e := m.BitVectorSort(65)
				astTestMust(t, e)
				h := bvRawOffsetTestNumeral(t, m, fid, sort, tc.text, 65)
				m.Budget().FailAfter(ordinal)
				n, body := NewBitVectorLeadingZeroNumerals(m, nil)
				width, ignored, steps := uint32(65), uint32(0), uint32(0)
				if body == nil {
					width, body = m.BitVectorSize(h)
				}
				if body == nil {
					_, body = n.Read(h, &ignored)
				}
				for body == nil {
					var positive bool
					positive, body = n.IsPositive()
					if body != nil || !positive {
						break
					}
					if width == 0 {
						t.Fatal("unexpected source assertion")
					}
					width--
					body = n.Step()
					if body == nil {
						steps++
					}
				}
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					if ignored != 65 || width != tc.leading || steps != tc.steps {
						t.Fatal("literal division loop", ignored, width, steps, tc)
					}
					if n.placed[1] || n.placed[2] {
						t.Fatal("division expression survived iteration")
					}
					want := "0"
					if tc.text == "-1" {
						want = "-1"
					}
					got, e := m.numerals.String(&n.values[0])
					astTestMust(t, e)
					if got != want {
						t.Fatal("physical v", got, want)
					}
					astTestMust(t, n.SourceEnd())
					if n.placed[0] {
						t.Fatal("v survived source end")
					}
				} else {
					var f umemory.Failure
					if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &f) || f.Kind != umemory.FailureMemout) {
						t.Fatal(body)
					}
					if n != nil {
						if _, e = n.IsPositive(); e != ErrASTPending {
							t.Fatal("stopped replay", e)
						}
					}
				}
				if n != nil {
					copy := *n
					if e = copy.Close(); e != ErrASTHandle {
						t.Fatal("copied frame", e)
					}
					b, s, _ := n.scalar.Causes()
					astTestMust(t, n.Close())
					astTestMust(t, n.Close())
					ab, as, _ := n.scalar.Causes()
					if b != ab || s != as {
						t.Fatal("cleanup changed history")
					}
				}
			})
		}
		if !complete {
			t.Fatal("actual leading campaign incomplete", tc.text)
		}
	}
}

func TestASTBitVectorLeadingZeroRecursiveParentRetained(t *testing.T) {
	m := bvOffsetRuntimeTestManager(t)
	parent, e := NewBitVectorLeadingZeroNumerals(m, nil)
	astTestMust(t, e)
	m.Budget().FailAfter(0)
	child, e := NewBitVectorLeadingZeroNumerals(m, parent)
	m.Budget().FailAfter(-1)
	if child != nil || !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal("refused child header", child, e)
	}
	if !parent.placed[0] || parent.closed {
		t.Fatal("older frame lost")
	}
	child, e = NewBitVectorLeadingZeroNumerals(m, parent)
	astTestMust(t, e)
	if child.Parent() != parent {
		t.Fatal("wrong recursive parent")
	}
	astTestMust(t, child.SourceEnd())
	astTestMust(t, child.Close())
	if !parent.placed[0] {
		t.Fatal("child ended older v")
	}
	astTestMust(t, parent.SourceEnd())
	astTestMust(t, parent.Close())
}
