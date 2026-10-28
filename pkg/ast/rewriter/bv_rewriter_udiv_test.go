package rewriter

// Source: src/ast/rewriter/bv_rewriter.cpp:1034-1099; src/ast/rewriter/bv_rewriter.h:122,127.

import (
	"bytes"
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"testing"
)

func TestBVRewriterUnsignedDivisionOriginalBranchesAndOptionOrder(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	local, e := c.NewValue()
	astTestMust(t, e)
	defer func() { astTestMust(t, c.Del(&local)) }()
	p, e := NewBVRewriter(m, c, g, &local)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	var trace bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&trace)
	tags.Enable("bv_udiv")
	astTestMust(t, p.BindTrace(tags))
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	xs := bvMergeVars(t, m, 2)
	x, y := xs[0], xs[1]
	for _, tc := range []struct{ a, b, want uint32 }{{249, 3, 83}, {255, 2, 127}, {128, 255, 0}, {249, 1, 249}, {128, 0, 255}} {
		st, e := p.RewriteUnsignedDivision(bvAddNum(t, m, tc.a, 8), bvAddNum(t, m, tc.b, 8), out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != RewriteDone || got != bvAddNum(t, m, tc.want, 8) {
			t.Fatal(tc, st, got)
		}
	}
	for _, inv := range []bool{false, true} {
		key := "udiv2mul"
		astTestMust(t, c.SetBoolName(&local, &key, inv))
		astTestMust(t, p.Update(&local))
		for _, divisor := range []uint32{2, 3, 6} {
			rhs := bvAddNum(t, m, divisor, 8)
			st, e := p.RewriteUnsignedDivision(x, rhs, out)
			astTestMust(t, e)
			want, e := m.MkFamilyApp(p.bvFamily, 18, nil, []ast.ASTHandle{x, rhs})
			astTestMust(t, e)
			astTestPin(t, m, want)
			status := RewriteDone
			if divisor == 2 {
				want, e = m.MkBitVectorLSHR(x, bvAddNum(t, m, 1, 8))
				astTestMust(t, e)
				astTestPin(t, m, want)
				status = Rewrite1
			}
			if divisor == 3 && inv {
				want, e = m.MkBitVectorMul(bvAddNum(t, m, 171, 8), x)
				astTestMust(t, e)
				astTestPin(t, m, want)
				status = Rewrite1
			}
			got, e := m.RefGet(out)
			astTestMust(t, e)
			if st != status || got != want {
				t.Fatal("power before configured inverse", inv, divisor, st, got, want)
			}
		}
		// Numeral folding precedes both optional symbolic rewrites.
		st, e := p.RewriteUnsignedDivision(bvAddNum(t, m, 249, 8), bvAddNum(t, m, 3, 8), out)
		astTestMust(t, e)
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != RewriteDone || got != bvAddNum(t, m, 83, 8) {
			t.Fatal("numeric before inverse", inv, st)
		}
	}
	for _, hi := range []bool{true, false} {
		key := "hi_div0"
		astTestMust(t, c.SetBoolName(&local, &key, hi))
		astTestMust(t, p.Update(&local))
		zero := bvAddNum(t, m, 0, 8)
		special, e := m.MkFamilyApp(p.bvFamily, 13, nil, []ast.ASTHandle{x})
		astTestMust(t, e)
		astTestPin(t, m, special)
		trace.Reset()
		st, e := p.RewriteUnsignedDivision(x, zero, out)
		astTestMust(t, e)
		if tags.ShouldTrace("bv_udiv") {
			wantFlag := "1"
			if !hi {
				wantFlag = "0"
			}
			want := "-------- [bv_udiv] mk_bv_udiv_core src/ast/rewriter/bv_rewriter.cpp:1038 ---------\nhi_div0: " + wantFlag + "\n------------------------------------------------\n"
			if got := trace.String(); got != want {
				t.Fatalf("source unsigned division trace: %q", got)
			}
		} else if trace.Len() != 0 {
			t.Fatalf("erased unsigned division trace: %q", trace.String())
		}
		trace.Reset()
		want, status := bvAddNum(t, m, 255, 8), RewriteDone
		if !hi {
			want = special
			status = Rewrite1
		}
		got, e := m.RefGet(out)
		astTestMust(t, e)
		if st != status || got != want {
			t.Fatal("zero", hi, st, got, want)
		}
		st, e = p.RewriteUnsignedDivision(x, y, out)
		astTestMust(t, e)
		internal, e := m.MkFamilyApp(p.bvFamily, 18, nil, []ast.ASTHandle{x, y})
		astTestMust(t, e)
		astTestPin(t, m, internal)
		want, status = internal, RewriteDone
		if !hi {
			eq, e := m.MkEq(y, zero)
			astTestMust(t, e)
			astTestPin(t, m, eq)
			want, e = m.MkIte(eq, special, internal)
			astTestMust(t, e)
			astTestPin(t, m, want)
			status = Rewrite2
		}
		got, e = m.RefGet(out)
		astTestMust(t, e)
		if st != status || got != want {
			t.Fatal("symbolic", hi, st, got, want)
		}
		st, e = p.RewriteUnsignedDivisionInternal(x, zero, out)
		astTestMust(t, e)
		got, e = m.RefGet(out)
		astTestMust(t, e)
		if st != RewriteDone || got != bvAddNum(t, m, 255, 8) {
			t.Fatal("internal hardware zero", hi, st)
		}
	}
}

func TestBVRewriterUnsignedDivisionEveryColdReservationAndPublication(t *testing.T) {
	for _, hi := range []bool{true, false} {
		for _, mode := range []string{"numeric", "zero", "one", "power", "inverse", "even", "symbolic"} {
			complete, published := false, false
			for ordinal := int64(0); ordinal < 1024 && !complete; ordinal++ {
				t.Run(fmt.Sprintf("%t/%s/%d", hi, mode, ordinal), func(t *testing.T) {
					c, g, _ := bvPolynomialParamsFixture(t)
					m, _ := bvAddFixture(t)
					local, e := c.NewValue()
					astTestMust(t, e)
					key := "udiv2mul"
					astTestMust(t, c.SetBoolName(&local, &key, true))
					key = "hi_div0"
					astTestMust(t, c.SetBoolName(&local, &key, hi))
					p, e := NewBVRewriter(m, c, g, &local)
					astTestMust(t, e)
					astTestMust(t, c.Del(&local))
					xs := bvMergeVars(t, m, 2)
					lhs, rhs := bvAddNum(t, m, 249, 8), bvAddNum(t, m, 3, 8)
					switch mode {
					case "zero":
						lhs = xs[0]
						rhs = bvAddNum(t, m, 0, 8)
					case "one":
						lhs = xs[0]
						rhs = bvAddNum(t, m, 1, 8)
					case "power":
						lhs = xs[0]
						rhs = bvAddNum(t, m, 2, 8)
					case "inverse":
						lhs = xs[0]
					case "even":
						lhs = xs[0]
						rhs = bvAddNum(t, m, 6, 8)
					case "symbolic":
						lhs = xs[0]
						rhs = xs[1]
					}
					sort, e := m.BitVectorSort(8)
					astTestMust(t, e)
					args := make([]ast.ASTHandle, 32)
					for i := range args {
						args[i], e = m.MkVar(uint32(i+100), sort)
						astTestMust(t, e)
					}
					old, e := m.MkBitVectorOr(args)
					astTestMust(t, e)
					out, e := m.NewRef(old)
					astTestMust(t, e)
					defer func() { astTestMust(t, m.DelRef(&out)) }()
					m.Budget().FailAfter(ordinal)
					st, body := p.RewriteUnsignedDivision(lhs, rhs, out)
					m.Budget().FailAfter(-1)
					if body == nil {
						complete = true
						nativeResult, nativeErr := m.RefGet(out)
						astTestMust(t, nativeErr)
						if nativeResult != old {
							published = true
						}
						want := RewriteDone
						if mode == "power" || mode == "inverse" || mode == "zero" && !hi {
							want = Rewrite1
						}
						if mode == "symbolic" && !hi {
							want = Rewrite2
						}
						if st != want {
							t.Fatal("source branch status", hi, mode, st, want)
						}
					} else {
						if !numericHoistRefusal(body) {
							t.Fatal(body)
						}
						got, e := m.RefGet(out)
						astTestMust(t, e)
						if got != old {
							published = true
							if st == RewriteFailed {
								t.Fatal("published status", st)
							}
						}
						if _, e = p.RewriteUnsignedDivision(lhs, rhs, out); e != body {
							t.Fatal("stopped replay", e, body)
						}
					}
					cause := p.cause
					astTestMust(t, p.Close())
					astTestMust(t, p.Close())
					if p.cause != cause {
						t.Fatal("cleanup changed history")
					}
				})
			}
			if !complete || !published {
				t.Fatal("incomplete unsigned division campaign", hi, mode, complete, published)
			}
		}
	}
}

func TestBVRewriterUnsignedDivisionStoredWidths(t *testing.T) {
	c, g, _ := bvPolynomialParamsFixture(t)
	m, _ := bvAddFixture(t)
	p, e := NewBVRewriter(m, c, g, nil)
	astTestMust(t, e)
	defer func() { astTestMust(t, p.Close()) }()
	out, e := m.NewRef(ast.ASTHandle{})
	astTestMust(t, e)
	defer func() { astTestMust(t, m.DelRef(&out)) }()
	st, e := p.RewriteUnsignedDivision(bvAddNum(t, m, 249, 8), bvAddNum(t, m, 13, 4), out)
	astTestMust(t, e)
	got, e := m.RefGet(out)
	astTestMust(t, e)
	if st != RewriteDone || got != bvAddNum(t, m, 19, 8) {
		t.Fatal("independent unsigned widths", st, got)
	}
}
