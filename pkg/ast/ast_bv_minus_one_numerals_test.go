package ast

// Source: src/ast/rewriter/bv_rewriter.cpp:1146-1153.

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func TestASTBitVectorMinusOneEveryActualReservation(t *testing.T) {
	for _, mode := range []string{"allones", "noncanonical", "negative", "other", "symbolic"} {
		complete := false
		for ordinal := int64(0); ordinal < 512 && !complete; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", mode, ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				fid, e := m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				sort, e := m.BitVectorSort(65)
				astTestMust(t, e)
				text := "36893488147419103231"
				switch mode {
				case "noncanonical":
					text = "73786976294838206463"
				case "negative":
					text = "-1"
				case "other":
					text = "36893488147419103230"
				}
				h := bvRawOffsetTestNumeral(t, m, fid, sort, text, 65)
				if mode == "symbolic" {
					h, e = m.MkVar(0, sort)
					astTestMust(t, e)
					astTestPin(t, m, h)
				}
				m.Budget().FailAfter(ordinal)
				n, body := NewBitVectorMinusOneNumerals(m)
				var width uint32
				matched, numeral := false, false
				if body == nil {
					numeral, body = n.Read(h, &width)
				}
				if body == nil && numeral {
					matched, body = n.Compare(width)
				}
				m.Budget().FailAfter(-1)
				if body == nil {
					complete = true
					if matched != (mode == "allones") {
						t.Fatal("stored-value comparison", mode, matched)
					}
					if numeral {
						if width != 65 {
							t.Fatal("stored width", width)
						}
						got, e := m.numerals.String(&n.values[0])
						astTestMust(t, e)
						if got != text {
							t.Fatal("read normalized input", got, text)
						}
						for i := 1; i < 4; i++ {
							if !n.placed[i] {
								t.Fatal("comparison full expression ended early", i)
							}
						}
						got, e = m.numerals.String(&n.values[3])
						astTestMust(t, e)
						if got != "36893488147419103231" {
							t.Fatal("comparison bound", got)
						}
					} else {
						if width != 0 {
							t.Fatal("unmatched wrote width", width)
						}
						for i := 1; i < 4; i++ {
							if n.placed[i] {
								t.Fatal("symbolic constructed bound", i)
							}
						}
					}
					astTestMust(t, n.SourceEnd())
					for _, placed := range n.placed {
						if placed {
							t.Fatal("source local survived return")
						}
					}
				} else {
					var f umemory.Failure
					if !errors.Is(body, arena.ErrInjectedFailure) && (!errors.As(body, &f) || f.Kind != umemory.FailureMemout) {
						t.Fatal(body)
					}
					if n != nil {
						if _, e = n.Compare(65); e != ErrASTPending {
							t.Fatal("stopped replay", e)
						}
					}
				}
				if n != nil {
					copy := *n
					if e = copy.Close(); e != ErrASTHandle {
						t.Fatal("copied owner", e)
					}
					b, s, _ := n.scalar.Causes()
					source := n.sourceCause
					astTestMust(t, n.Close())
					astTestMust(t, n.Close())
					ab, as, _ := n.scalar.Causes()
					if ab != b || as != s || n.sourceCause != source {
						t.Fatal("cleanup changed source history")
					}
				}
			})
		}
		if !complete {
			t.Fatal("actual campaign incomplete", mode)
		}
	}
}
