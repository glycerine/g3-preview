package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func lpNewLARLUPreparedModel(t *testing.T, h *lpPairHarness, b [][]*big.Rat) *LARSolver {
	t.Helper()
	s, e := NewLARSolver(h.m)
	h.must(e)
	s.Settings.SimplexStrategy = StrategyLU
	s.Settings.DensityThreshold = 1e100
	for i := range b {
		_, e = s.AddVar(uint32(31 + i))
		h.must(e)
	}
	c := s.Core
	for i := range b {
		h.must(c.RA.AddRow())
		c.DA.AddRow()
		c.RBasis = append(c.RBasis, uint32(i))
		c.DBasis = append(c.DBasis, uint32(i))
		for j, v := range b[i] {
			p := h.pair(v.RatString(), "0")
			h.must(c.RA.Set(uint32(i), uint32(j), &p.X))
			f, _ := v.Float64()
			h.must(c.DA.Set(uint32(i), uint32(j), f))
		}
	}
	h.must(c.RPrimal.InitBasisHeadingAndNonBasicColumns())
	h.must(c.DPrimal.InitBasisHeadingAndNonBasicColumns())
	return s
}

func TestLARLUExtendIndependentBasisAndPolicy(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	h := newLPPairHarness(t)
	for world := 0; world < 36; world++ {
		n := 1 + world%4
		b := lpExactIdentity(n)
		for i := range b {
			for j := range b[i] {
				b[i][j].SetFrac64(int64((world+i+j)%5-2), 7)
				if i == j {
					b[i][j].SetInt64(int64(13 + n + i))
				}
			}
		}
		s := lpNewLARLUPreparedModel(t, h, b)
		c := s.Core
		h.must(s.AddLastRowsToPairLU(c.RPrimal))
		h.must(s.AddLastRowsToFloatLU(c.DPrimal))
		oldP, oldF := c.RPrimal.Factorization, c.DPrimal.Factorization
		if oldP == nil || oldF == nil {
			t.Fatal("missing initial factorization")
		}
		full := lpExactIdentity(n + 1)
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				full[i][j].Set(b[i][j])
			}
		}
		for j := 0; j < n; j++ {
			full[n][j].SetFrac64(int64(j+1), 11)
		}
		full[n][n].SetInt64(17)
		_, e := s.AddVar(uint32(31 + n))
		h.must(e)
		h.must(c.RA.AddRow())
		c.DA.AddRow()
		c.RBasis = append(c.RBasis, uint32(n))
		c.DBasis = append(c.DBasis, uint32(n))
		for j, v := range full[n] {
			p := h.pair(v.RatString(), "0")
			h.must(c.RA.Set(uint32(n), uint32(j), &p.X))
			f, _ := v.Float64()
			h.must(c.DA.Set(uint32(n), uint32(j), f))
		}
		h.must(c.RPrimal.InitBasisHeadingAndNonBasicColumns())
		h.must(c.DPrimal.InitBasisHeadingAndNonBasicColumns())
		refactor := world%3 != 0
		if world%3 == 1 {
			oldP.RefactorCounter = 199
			oldF.RefactorCounter = 199
		}
		if world%3 == 2 {
			oldP.RefactorCounter = ^uint32(0)
			oldF.RefactorCounter = ^uint32(0)
		}
		h.must(s.AddLastRowsToPairLU(c.RPrimal))
		h.must(s.AddLastRowsToFloatLU(c.DPrimal))
		if (oldP != c.RPrimal.Factorization) != refactor || (oldF != c.DPrimal.Factorization) != refactor {
			t.Fatal("refactor threshold policy", world)
		}
		x, y := make([]*big.Rat, n+1), make([]*big.Rat, n+1)
		for i := range x {
			x[i] = big.NewRat(int64(i-world), 13)
			y[i] = big.NewRat(int64(world+i+1), 17)
		}
		lpExactLUUpdateCheckSolutions(t, h, nil, c.RPrimal.Factorization, full, x, y)
		lpFloatLUCheckSolutions(t, c.DPrimal.Factorization, full, x)
		lpFloatLUCheckFactorization(t, c.DPrimal.Factorization, full)
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARLUExtendDenseAndDegeneratedLifecycle(t *testing.T) {
	for _, op := range []string{"dense", "initial-singular", "extension-singular"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			b := lpExactIdentity(6)
			for i := range b {
				for j := range b[i] {
					if i == j {
						b[i][j].SetInt64(80)
					} else {
						b[i][j].SetInt64(1)
					}
				}
			}
			if op == "initial-singular" {
				for j := range b[5] {
					b[5][j].SetInt64(0)
				}
			}
			s := lpNewLARLUPreparedModel(t, h, b)
			c := s.Core
			s.Settings.DensityThreshold = 0
			h.must(s.AddLastRowsToPairLU(c.RPrimal))
			h.must(s.AddLastRowsToFloatLU(c.DPrimal))
			if op == "initial-singular" {
				if c.RPrimal.Factorization != nil || c.DPrimal.Factorization != nil {
					t.Fatal("new degenerated owners must be deleted")
				}
			} else {
				oldP, oldF := c.RPrimal.Factorization, c.DPrimal.Factorization
				if !oldP.HasDenseSubmatrix() || !oldF.HasDenseSubmatrix() {
					t.Fatal("missing actual dense tail")
				}
				if op == "extension-singular" {
					// A zero new basis column makes the newly refactored matrix singular.
					_, e := s.AddVar(97)
					h.must(e)
					h.must(c.RA.AddRow())
					c.DA.AddRow()
					c.RBasis = append(c.RBasis, 6)
					c.DBasis = append(c.DBasis, 6)
					h.must(c.RPrimal.InitBasisHeadingAndNonBasicColumns())
					h.must(c.DPrimal.InitBasisHeadingAndNonBasicColumns())
				}
				h.must(s.AddLastRowsToPairLU(c.RPrimal))
				h.must(s.AddLastRowsToFloatLU(c.DPrimal))
				if op == "extension-singular" {
					if c.RPrimal.Factorization != nil || c.DPrimal.Factorization != nil {
						t.Fatal("singular replacement kept")
					}
				} else {
					if c.RPrimal.Factorization == oldP || c.DPrimal.Factorization == oldF {
						t.Fatal("dense must refactor")
					}
				}
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARLUExtendPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, direct := range []bool{false, true} {
		for _, op := range []string{"initial", "refactor", "extend"} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 300; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%t/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					bigval := mustLPRat(t, "1"+strings.Repeat("0", 127)+"3/7")
					b := lpExactIdentity(1)
					b[0][0].Set(bigval)
					s := lpNewLARLUPreparedModel(t, h, b)
					c := s.Core
					if op != "initial" {
						h.must(s.AddLastRowsToPairLU(c.RPrimal))
					}
					if op == "refactor" {
						c.RPrimal.Factorization.RefactorCounter = 200
					}
					if op == "extend" {
						_, e := s.AddVar(97)
						h.must(e)
						h.must(c.RA.AddRow())
						c.RBasis = append(c.RBasis, 1)
						h.must(c.RA.Set(1, 0, &h.pair(bigval.RatString(), "0").X))
						h.must(c.RA.Set(1, 1, &h.pair("1", "0").X))
						h.must(c.RPrimal.InitBasisHeadingAndNonBasicColumns())
					}
					h.budget.FailAfter(ordinal)
					err := s.AddLastRowsToPairLU(c.RPrimal)
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						done = true
						if c.RPrimal.Factorization == nil {
							t.Fatal("lost nonsingular factorization")
						}
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if s.calls.pending() {
							t.Fatal("outer adopted numeric child")
						}
						if s.AddLastRowsToPairLU(c.RPrimal) != ErrExactVectorPending {
							t.Fatal("outer not stopped")
						}
						if c.RPrimal.Factorization == nil {
							t.Fatal("lost partial numeric owner")
						}
					}
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if !success || failures == 0 {
				t.Fatal("incomplete campaign", direct, op, success, failures)
			}
		}
	}
}

func TestLARLUExtendEntryAndChildGuards(t *testing.T) {
	for _, op := range []string{"nil-pair", "nil-float", "pair-heading", "float-heading", "closed-float", "pair-ready", "pair-constructor", "float-constructor", "pair-extension", "float-extension"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpNewLARLUPreparedModel(t, h, lpExactIdentity(1))
			c := s.Core
			var err error
			want := ErrBounds
			switch op {
			case "nil-pair":
				err = s.AddLastRowsToPairLU(nil)
			case "nil-float":
				err = s.AddLastRowsToFloatLU(nil)
			case "pair-heading":
				h.must(s.AddLastRowsToPairLU(c.RPrimal))
				h.must(c.RA.AddRow())
				h.must(c.RA.Set(1, 0, &h.pair("1", "0").X))
				c.RHeading = nil
				err = s.AddLastRowsToPairLU(c.RPrimal)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "float-heading":
				h.must(s.AddLastRowsToFloatLU(c.DPrimal))
				c.DA.AddRow()
				h.must(c.DA.Set(1, 0, 1))
				c.DHeading = nil
				err = s.AddLastRowsToFloatLU(c.DPrimal)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "closed-float":
				h.must(s.AddLastRowsToFloatLU(c.DPrimal))
				c.DPrimal.Factorization.Close()
				err = s.AddLastRowsToFloatLU(c.DPrimal)
				want = ErrLUClosed
			case "pair-ready":
				if c.RPrimal.UpdateColumnInInfSet(99) != ErrBounds {
					t.Fatal("missing child guard")
				}
				err = s.AddLastRowsToPairLU(c.RPrimal)
				want = ErrExactVectorPending
			case "pair-extension":
				h.must(s.AddLastRowsToPairLU(c.RPrimal))
				h.must(c.RA.AddRow())
				h.must(c.RA.Set(1, 0, &h.pair("1", "0").X))
				c.RHeading[0] = 99
				err = s.AddLastRowsToPairLU(c.RPrimal)
			case "float-extension":
				h.must(s.AddLastRowsToFloatLU(c.DPrimal))
				c.DA.AddRow()
				h.must(c.DA.Set(1, 0, 1))
				c.DHeading[0] = 99
				err = s.AddLastRowsToFloatLU(c.DPrimal)
			case "pair-constructor":
				c.RBasis[0] = 99
				err = s.AddLastRowsToPairLU(c.RPrimal)
				if c.RPrimal.Factorization == nil {
					t.Fatal("lost partial constructor")
				}
			case "float-constructor":
				c.DBasis[0] = 99
				err = s.AddLastRowsToFloatLU(c.DPrimal)
				if c.DPrimal.Factorization == nil {
					t.Fatal("lost partial constructor")
				}
			}
			if err != want {
				t.Fatal(op, err, want)
			}
			if err != nil {
				if s.AddLastRowsToPairLU(c.RPrimal) != ErrExactVectorPending || s.AddLastRowsToFloatLU(c.DPrimal) != ErrExactVectorPending {
					t.Fatal("lost stopped owner")
				}
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}
