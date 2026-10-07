// Source: src/ast/rewriter/bv_trailing.cpp:213-234,311-327.
package ast

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestASTBitVectorOriginalNumeralTrailingZeros(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	for _, width := range []uint32{1, 8, 65, 257} {
		sort, e := m.BitVectorSort(width)
		astTestMust(t, e)
		modulus := new(big.Int).Lsh(big.NewInt(1), uint(width))
		for _, value := range []*big.Int{new(big.Int), big.NewInt(1), big.NewInt(12), new(big.Int).Lsh(big.NewInt(3), uint(width-1)), new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), uint(width-1)))} {
			// Raw numerical metadata deliberately includes negative/oversized values.
			// Counting reads the stored value rather than normalizing the input first.
			input := bvRawOffsetTestNumeral(t, m, fid, sort, value.String(), width)
			out := astTestPin(t, m, input)
			wantCount := width
			if value.Sign() != 0 {
				wantCount = uint32(value.TrailingZeroBits())
				if wantCount > width {
					wantCount = width
				}
			}
			matched, count, e := m.CountBitVectorNumeralTrailingZeros(input)
			astTestMust(t, e)
			if !matched || count != wantCount {
				t.Fatal("stored numeral count", width, value, matched, count, wantCount)
			}
			for _, n := range []uint32{1, 2, width, ^uint32(0)} {
				astTestMust(t, m.SetRef(out, input))
				matched, removed, e := m.RemoveBitVectorNumeralTrailingZeros(input, n, out)
				astTestMust(t, e)
				want := wantCount
				if n < want {
					want = n
				}
				got, e := m.RefGet(out)
				astTestMust(t, e)
				if !matched || removed != want {
					t.Fatal("remove count", width, value, n, matched, removed, want)
				}
				if want == width {
					if !got.IsNull() {
						t.Fatal("whole width must publish null")
					}
					continue
				}
				if want == 0 {
					if got != input {
						t.Fatal("odd input should keep raw identity")
					}
					continue
				}
				reduced := new(big.Int).Quo(value, new(big.Int).Lsh(big.NewInt(1), uint(want)))
				reduced.Mod(reduced, new(big.Int).Rsh(new(big.Int).Set(modulus), uint(want)))
				q := m.numerals
				num, e := q.NewValue()
				astTestMust(t, e)
				var bits uint32
				yes, e := m.BitVectorNumeralValue(got, &num, &bits)
				astTestMust(t, e)
				text, e := q.String(&num)
				astTestMust(t, e)
				astTestMust(t, q.Del(&num))
				if !yes || bits != width-want || text != reduced.String() {
					t.Fatal("reduced numeral", width, value, n, bits, text, reduced)
				}
			}
		}
	}
	sort, e := m.BitVectorSort(8)
	astTestMust(t, e)
	x, e := m.MkVar(0, sort)
	astTestMust(t, e)
	astTestPin(t, m, x)
	before := m.Stats()
	m.Budget().FailAfter(0)
	matched, count, e := m.CountBitVectorNumeralTrailingZeros(x)
	astTestMust(t, e)
	rematched, removed, e := m.RemoveBitVectorNumeralTrailingZeros(x, 1, ASTRef{})
	astTestMust(t, e)
	zeroMatched, zeroRemoved, e := m.RemoveBitVectorNumeralTrailingZeros(ASTHandle{}, 0, ASTRef{})
	astTestMust(t, e)
	m.Budget().FailAfter(-1)
	if matched || rematched || zeroMatched || count != 0 || removed != 0 || zeroRemoved != 0 || before != m.Stats() {
		t.Fatal("erased nonnumeral/zero-limit work")
	}
}

func TestASTBitVectorTrailingNumeralEveryActualReservation(t *testing.T) {
	for _, remove := range []bool{false, true} {
		success, failures, division, nested := false, 0, false, false
		for ordinal := int64(0); ordinal < 256; ordinal++ {
			t.Run(fmt.Sprint(remove, "/", ordinal), func(t *testing.T) {
				m := bvOffsetRuntimeTestManager(t)
				_, e := m.RegisterArithmeticPlugin()
				astTestMust(t, e)
				fid, e := m.RegisterBitVectorPlugin()
				astTestMust(t, e)
				sort, e := m.BitVectorSort(8)
				astTestMust(t, e)
				input := bvTrailingColdTestNumeral(t, m, fid, sort)
				out := astTestPin(t, m, input)
				m.Budget().FailAfter(ordinal)
				var matched bool
				var count uint32
				var body error
				if remove {
					matched, count, body = m.RemoveBitVectorNumeralTrailingZeros(input, 2, out)
				} else {
					matched, count, body = m.CountBitVectorNumeralTrailingZeros(input)
				}
				m.Budget().FailAfter(-1)
				if body != nil {
					failures++
					if !errors.Is(body, arena.ErrInjectedFailure) {
						var f util.Failure
						if !errors.As(body, &f) || f.Kind != util.FailureMemout {
							t.Fatal("unexpected refusal", body)
						}
					}
					got, e := m.RefGet(out)
					astTestMust(t, e)
					if got != input {
						t.Fatal("failed numerical body published output")
					}
					for f := m.bitVectorPlugin.numeralFrames; f != nil; f = f.next {
						if !f.trailingNumeral {
							continue
						}
						childBody, _, _ := f.children[1].Causes()
						if !errors.Is(childBody, arena.ErrInvalidSlot) {
							if m.numerals.Manager.Z.DivisionRaw.Invocation == 0 {
								t.Fatal("negative division child not reached")
							}
							division = true
							// The count loop increments before the failed division; removal
							// increments only after it. The stopped child retains its call site.
							if !remove && count == 0 {
								t.Fatal("count not incremented before division refusal")
							}
							if remove && childBody != nil && count >= 2 {
								t.Fatal("removal incremented before failed division")
							}
						}
						if f.normalizationChild.frame != nil {
							nested = true
							if !f.consumed[1] {
								t.Fatal("numerical helper's two survived into reduced constructor")
							}
							if f.normalizationChild.sourceCause != nil && f.consumed[0] {
								t.Fatal("older numeral ended before nested source stop")
							}
						}
					}
				} else {
					success = true
					want := uint32(8)
					if remove {
						want = 2
					}
					if !matched || count != want || m.bitVectorPlugin.numeralFrames != nil {
						t.Fatal("completed numerical branch", matched, count, want)
					}
					if remove {
						got, e := m.RefGet(out)
						astTestMust(t, e)
						q := m.numerals
						num, e := q.NewValue()
						astTestMust(t, e)
						var width uint32
						yes, e := m.BitVectorNumeralValue(got, &num, &width)
						astTestMust(t, e)
						text, e := q.String(&num)
						astTestMust(t, e)
						astTestMust(t, q.Del(&num))
						if !yes || width != 6 || text != "0" {
							t.Fatal("reduced raw negative value", yes, width, text)
						}
					}
				}
			})
			if success {
				break
			}
		}
		if !success || failures == 0 || !division || remove && !nested {
			t.Fatal("actual numerical horizon", remove, success, failures, division, nested)
		}
	}
}

func TestASTBitVectorTrailingNumeralPublicationScope(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	input, e := m.BitVectorNumeralUnsigned(12, 8)
	astTestMust(t, e)
	astTestPin(t, m, input)
	matched, removed, body := m.RemoveBitVectorNumeralTrailingZeros(input, 1, ASTRef{})
	if !matched || removed != 1 || !errors.Is(body, ErrASTHandle) {
		t.Fatal("reached publication", matched, removed, body)
	}
	f := m.bitVectorPlugin.numeralFrames
	if f == nil || !f.trailingNumeral || !f.sourceDone || !f.normalizationChild.retired || f.retired {
		t.Fatal("missing reached parent owner")
	}
	for _, i := range []int{0, 1} {
		if !f.placed[i] || !f.complete[i] || !f.consumed[i] {
			t.Fatal("local numeral/two DT", i)
		}
	}
}

// Construct -3*2^1024 without decimal parsing/normalization warming Tmp[0].
// The recognizer intentionally reads oversized raw metadata of width eight.
func bvTrailingColdTestNumeral(t *testing.T, m *ASTManager, fid FamilyID, sort ASTHandle) ASTHandle {
	t.Helper()
	q := m.numerals
	n, e := q.NewValue()
	astTestMust(t, e)
	var words [33]uint32
	words[32] = 3
	astTestMust(t, q.Manager.Z.SetLimbs(&n.Raw.Num, words[:]))
	astTestMust(t, q.Manager.Z.Neg(&n.Raw.Num))
	value, e := m.NewParameterRational(&n)
	astTestMust(t, e)
	bits, e := m.NewParameterUint(8)
	astTestMust(t, e)
	info, e := m.NewFuncDeclInfo(fid, 0, []ASTParameter{value, bits}, ASTFuncFlags{}, false)
	astTestMust(t, e)
	d, e := m.MkFuncDecl(astTestSymbol(t, m, "bv-trailing-cold"), nil, sort, info)
	astTestMust(t, e)
	astTestMust(t, m.DelFuncDeclInfo(&info))
	astTestMust(t, m.DelParameter(&bits))
	astTestMust(t, m.DelParameter(&value))
	astTestMust(t, q.Del(&n))
	h, e := m.MkApp(d, nil)
	astTestMust(t, e)
	astTestPin(t, m, h)
	return h
}
