package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func TestLARSignatureExactPositionIndependentModels(t *testing.T) {
	for world := 0; world < 500; world++ {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 0, 1)
		f.settings.SimplexStrategy = StrategyTableauRows
		kinds := []ColumnType{ColumnFree, ColumnLowBound, ColumnUpperBound, ColumnFixed, ColumnBoxed}
		kind := kinds[world%5]
		f.types[0] = kind
		lo, up := big.NewRat(int64(world%17-8), 7), big.NewRat(int64(world%17-8), 7)
		if kind != ColumnFixed {
			up.Add(up, big.NewRat(2, 1))
		}
		x := new(big.Rat).Set(lo)
		epsilon := new(big.Rat)
		switch world / 5 % 5 {
		case 1:
			x.Set(up)
		case 2:
			x.Add(x, big.NewRat(1, 1))
		case 3:
			epsilon.SetFrac64(1, 13)
		case 4:
			epsilon.SetFrac64(-1, 13)
		}
		h.must(f.low.Data[0].Copy(h.m, h.pair(lo.RatString(), "0")))
		h.must(f.up.Data[0].Copy(h.m, h.pair(up.RatString(), "0")))
		h.must(f.x.Data[0].Copy(h.m, h.pair(x.RatString(), epsilon.RatString())))
		m, err := NewPairPrimalCoreSolver(f.input())
		h.must(err)
		want := PositionNotAtBound
		if kind == ColumnFree {
			want = PositionFreeOfBounds
		} else if x.Cmp(lo) == 0 && epsilon.Sign() == 0 && kind != ColumnUpperBound {
			want = PositionAtLowBound
			if kind == ColumnFixed {
				want = PositionAtFixed
			}
		} else if x.Cmp(up) == 0 && epsilon.Sign() == 0 && (kind == ColumnUpperBound || kind == ColumnBoxed) {
			want = PositionAtUpperBound
		}
		got, err := m.GetNonBasicColumnValuePosition(0)
		h.must(err)
		if got != want {
			t.Fatal(world, kind, got, want)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARSignatureExtractionIndependentModels(t *testing.T) {
	for world := 0; world < 120; world++ {
		h := newLPPairHarness(t)
		c := lpNewCoreScopeFixture(t, h)
		c.Bounds.Types.values[1], c.Bounds.Types.values[2] = ColumnBoxed, ColumnUpperBound
		h.must(c.Bounds.Upper.Set(1, h.pair("2", "0")))
		h.must(c.Bounds.Upper.Set(2, h.pair("3", "0")))
		c.DUpper[1], c.DUpper[2] = 2, 3
		v := int64(world % 3)
		h.must(c.RX.Data[1].Copy(h.m, h.pair(fmt.Sprint(v), "0")))
		h.must(c.RX.Data[2].Copy(h.m, h.pair("3", "0")))
		c.DX[1], c.DX[2] = float64(v), 3
		want := PositionNotAtBound
		if v == 0 {
			want = PositionAtLowBound
		} else if v == 2 {
			want = PositionAtUpperBound
		}
		signature := LARSolutionSignature{99: PositionFreeOfBounds, 0: PositionAtFixed}
		h.must(c.ExtractSignatureFromPairSolver(&signature, c.RPrimal))
		if len(signature) != 2 || signature[1] != want || signature[2] != PositionAtUpperBound {
			t.Fatal("pair extraction", signature)
		}
		clear(signature)
		signature[88] = PositionAtFixed
		h.must(c.ExtractSignatureFromFloatSolver(&signature, c.DPrimal))
		if len(signature) != 2 || signature[1] != want || signature[2] != PositionAtUpperBound {
			t.Fatal("float extraction", signature)
		}
		h.must(c.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARSignaturePreparationIndependentSystems(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for world := 0; world < 160; world++ {
		for _, floating := range []bool{false, true} {
			h := newLPPairHarness(t)
			c := lpNewCoreScopeFixture(t, h)
			h.must(c.PrefixR())
			h.must(c.PrefixD())
			c.settings.RandomSeed(uint32(world + 1))
			rng := z3rand.New(uint32(world + 1))
			kinds := []ColumnType{ColumnLowBound, ColumnUpperBound, ColumnFixed, ColumnBoxed}
			positions := []NonBasicColumnValuePosition{PositionAtLowBound, PositionAtUpperBound, PositionAtFixed, PositionFreeOfBounds, PositionNotAtBound}
			signature := make(LARSolutionSignature)
			realValues, epsValues := make([]*big.Rat, 3), make([]*big.Rat, 3)
			for j := uint32(1); j < 3; j++ {
				kind := kinds[(world+int(j))%4]
				c.Bounds.Types.values[j] = kind
				low, up := big.NewRat(int64(world%13-6), 1), big.NewRat(int64(world%13+1), 1)
				ly, uy := big.NewRat(int64(world%7-3), 11), big.NewRat(int64(world%5-2), 13)
				h.must(c.Bounds.Low.Set(j, h.pair(low.RatString(), ly.RatString())))
				h.must(c.Bounds.Upper.Set(j, h.pair(up.RatString(), uy.RatString())))
				c.DLow[j], c.DUpper[j] = float64(world%13-6), float64(world%13+1)
				pos := positions[(world+int(j))%5]
				signature[j] = pos
				side := 1
				switch pos {
				case PositionAtUpperBound, PositionAtFixed:
					side = 2
				case PositionFreeOfBounds:
					side = 0
				case PositionNotAtBound:
					if kind == ColumnUpperBound {
						side = 2
					} else if kind == ColumnBoxed {
						if rng.Next()%2 == 0 {
							side = 2
						}
					}
				}
				if side == 0 {
					realValues[j], epsValues[j] = new(big.Rat), new(big.Rat)
				} else if side == 1 {
					realValues[j], epsValues[j] = low, ly
				} else {
					realValues[j], epsValues[j] = up, uy
				}
			}
			var err error
			if floating {
				err = c.PrepareFloatSolverXWithSignature(signature, c.DPrimal)
			} else {
				err = c.PreparePairSolverXWithSignature(signature, c.RPrimal)
			}
			h.must(err)
			if c.settings.RandomNext() != rng.Next() {
				t.Fatal("RNG", world, floating)
			}
			basic := new(big.Rat).Neg(new(big.Rat).Add(new(big.Rat).Mul(big.NewRat(2, 1), realValues[1]), new(big.Rat).Mul(big.NewRat(3, 1), realValues[2])))
			basicEps := new(big.Rat).Neg(new(big.Rat).Add(new(big.Rat).Mul(big.NewRat(2, 1), epsValues[1]), new(big.Rat).Mul(big.NewRat(3, 1), epsValues[2])))
			if floating {
				if c.DX[0] != float64(-2*realValues[1].Num().Int64()-3*realValues[2].Num().Int64()) {
					t.Fatal("floating reconstruction", world, c.DX)
				}
			} else {
				h.check(&c.RX.Data[0], basic, basicEps)
				for j := 1; j < 3; j++ {
					h.check(&c.RX.Data[j], realValues[j], epsValues[j])
				}
			}
			h.must(c.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLARSignatureReachedPrefixes(t *testing.T) {
	for _, floating := range []bool{false, true} {
		for _, phase := range []string{"nil-signature", "nil-solver", "stopped", "extract-short-types", "prepare-short-x", "prepare-short-low", "prepare-short-upper", "prepare-nonzero-rhs", "prepare-basic-heading", "prepare-invalid-position", "prepare-free-interior", "prepare-no-lu"} {
			t.Run(fmt.Sprintf("%t/%s", floating, phase), func(t *testing.T) {
				h := newLPPairHarness(t)
				c := lpNewCoreScopeFixture(t, h)
				h.must(c.PrefixR())
				h.must(c.PrefixD())
				signature := LARSolutionSignature{1: PositionAtLowBound, 2: PositionAtUpperBound}
				h.must(c.Bounds.Low.Set(1, h.pair("7", "0")))
				h.must(c.Bounds.Upper.Set(2, h.pair("11", "0")))
				c.DLow[1], c.DUpper[2] = 7, 11
				var err error
				savedTypes, savedX, savedLow, savedUp := c.Bounds.Types.values, c.RX.Data, c.Bounds.Low.values, c.Bounds.Upper.values
				savedDX, savedDLow, savedDUp := c.DX, c.DLow, c.DUpper
				extract := false
				succeeds := false
				switch phase {
				case "nil-signature":
					extract = true
					if floating {
						err = c.ExtractSignatureFromFloatSolver(nil, c.DPrimal)
					} else {
						err = c.ExtractSignatureFromPairSolver(nil, c.RPrimal)
					}
				case "nil-solver":
					extract = true
					if floating {
						err = c.ExtractSignatureFromFloatSolver(&signature, nil)
					} else {
						err = c.ExtractSignatureFromPairSolver(&signature, nil)
					}
				case "stopped":
					c.stopped = true
				case "extract-short-types":
					extract = true
					c.Bounds.Types.values = c.Bounds.Types.values[:2]
					signature[99] = PositionAtFixed
					if floating {
						err = c.ExtractSignatureFromFloatSolver(&signature, c.DPrimal)
					} else {
						err = c.ExtractSignatureFromPairSolver(&signature, c.RPrimal)
					}
				case "prepare-short-x":
					if floating {
						c.DX = c.DX[:2]
					} else {
						c.RX.Data = c.RX.Data[:2]
					}
				case "prepare-short-low":
					if floating {
						c.DLow = nil
					} else {
						c.Bounds.Low.values = nil
					}
				case "prepare-short-upper":
					if floating {
						c.DUpper = nil
					} else {
						c.Bounds.Upper.values = nil
					}
				case "prepare-nonzero-rhs":
					h.must(c.RightSides.Data[0].Copy(h.m, h.pair("3", "0")))
					c.DRightSides[0] = 3
					succeeds = !sourceDebugEnabled
				case "prepare-basic-heading":
					c.RHeading[1] = 0
					succeeds = !sourceDebugEnabled
				case "prepare-invalid-position":
					signature[2] = NonBasicColumnValuePosition(99)
					succeeds = !sourceDebugEnabled
				case "prepare-free-interior":
					c.Bounds.Types.values[2] = ColumnFree
					signature[2] = PositionNotAtBound
					succeeds = !sourceDebugEnabled
				case "prepare-no-lu":
					if floating {
						h.must(c.DPrimal.Factorization.Close())
						c.DPrimal.Factorization = nil
					} else {
						h.must(c.RPrimal.Factorization.Close())
						c.RPrimal.Factorization = nil
					}
				}
				if !extract {
					if floating {
						err = c.PrepareFloatSolverXWithSignature(signature, c.DPrimal)
					} else {
						err = c.PreparePairSolverXWithSignature(signature, c.RPrimal)
					}
				}
				if (err == nil) != succeeds {
					t.Fatal("result", err, succeeds)
				}
				if phase == "extract-short-types" {
					if len(signature) != 1 || signature[1] != PositionFreeOfBounds {
						t.Fatal("clear and extraction prefix", signature)
					}
				}
				if phase == "prepare-short-x" || phase == "prepare-short-upper" || phase == "prepare-no-lu" || phase == "prepare-nonzero-rhs" || phase == "prepare-invalid-position" || phase == "prepare-free-interior" {
					if floating {
						if c.DX[1] != 7 {
							t.Fatal("floating prefix", c.DX)
						}
					} else {
						h.check(&c.RX.Data[1], big.NewRat(7, 1), new(big.Rat))
					}
				}
				if !succeeds {
					if c.PrepareFloatSolverXWithSignature(nil, c.DPrimal) != ErrExactVectorPending {
						t.Fatal("stopped owner retry")
					}
				}
				c.Bounds.Types.values, c.RX.Data, c.Bounds.Low.values, c.Bounds.Upper.values = savedTypes, savedX, savedLow, savedUp
				c.DX, c.DLow, c.DUpper = savedDX, savedDLow, savedDUp
				h.must(c.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

func TestLARSignaturePhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, direct := range []bool{false, true} {
		success, failures := false, 0
		for ordinal := int64(0); ordinal < 1000; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("%t/%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				c := lpNewCoreScopeFixture(t, h)
				h.must(c.PrefixR())
				huge := strings.Repeat("1234567890", 20) + "7/13"
				h.must(c.Bounds.Low.Set(1, h.pair(huge, "0")))
				h.must(c.Bounds.Upper.Set(2, h.pair(huge, "0")))
				signature := LARSolutionSignature{1: PositionAtLowBound, 2: PositionAtUpperBound}
				h.budget.FailAfter(ordinal)
				err := c.PreparePairSolverXWithSignature(signature, c.RPrimal)
				h.budget.FailAfter(-1)
				if err == nil {
					success, done = true, true
				} else {
					failures++
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					if c.PreparePairSolverXWithSignature(signature, c.RPrimal) != ErrExactVectorPending {
						t.Fatal("stopped retry")
					}
				}
				h.must(c.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
			if done {
				break
			}
		}
		if !success || (direct && failures == 0) {
			t.Fatal("campaign", direct, success, failures)
		}
	}
}
