package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"

	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func TestLPExactLeavingIndependentFeasibleModels(t *testing.T) {
	rng := rand.New(rand.NewPCG(971, 391))
	counters := map[string]int{}
	for world := 0; world < 480; world++ {
		h := newLPPairHarness(t)
		n := 1 + world%6
		s := lpNewLARLUPreparedModel(t, h, lpExactIdentity(n))
		_, e := s.AddVar(1000)
		h.must(e)
		m := s.Core.RPrimal
		entering := uint32(n)
		direction := int64(1)
		if world%2 == 1 {
			direction = -1
		}
		m.SignOfEnteringDelta = int32(direction)
		h.must(m.Ed.Resize(uint32(n)))
		m.RowsNZ = make([]uint32, n)
		type limit struct {
			j    uint32
			x, y *big.Rat
			nz   uint32
		}
		limits := map[uint32]limit{}
		order := rng.Perm(n)
		for _, i := range order {
			ed := big.NewRat(int64(1+rng.IntN(7)), int64(1+rng.IntN(5)))
			if rng.IntN(2) == 0 {
				ed.Neg(ed)
			}
			h.must(m.A.Set(uint32(i), entering, &h.pair(ed.RatString(), "0").X))
			h.must(m.Ed.SetValue(&h.pair(ed.RatString(), "0").X, uint32(i)))
			typ := ColumnType((world + i) % 5)
			if world%11 == 0 {
				typ = ColumnFree
			}
			h.must(s.Bounds.Types.Set(uint32(i), typ))
			x, y := big.NewRat(int64((world+i)%4), 3), big.NewRat(int64((world+i)%3), 7)
			if typ == ColumnFixed {
				x.SetInt64(0)
				y.SetInt64(0)
			}
			abs := new(big.Rat).Abs(ed)
			upx, upy := new(big.Rat).Mul(x, abs), new(big.Rat).Mul(y, abs)
			h.must((*m.UpperBounds)[i].Copy(h.m, h.pair(upx.RatString(), upy.RatString())))
			h.must((*m.LowBounds)[i].Copy(h.m, h.pair(new(big.Rat).Neg(upx).RatString(), new(big.Rat).Neg(upy).RatString())))
			m.RowsNZ[i] = uint32((world+i)%3 + 1)
			positive := ed.Sign()*int(direction) < 0
			eligible := typ == ColumnBoxed || typ == ColumnFixed || (typ == ColumnLowBound && !positive) || (typ == ColumnUpperBound && positive)
			if eligible {
				limits[uint32(i)] = limit{uint32(i), x, y, m.RowsNZ[i]}
			}
		}
		kind := ColumnType((world / 3) % 5)
		h.must(s.Bounds.Types.Set(entering, kind))
		lowx, lowy := big.NewRat(-int64(world%5+1), 3), big.NewRat(-int64(world%3), 7)
		upx, upy := big.NewRat(int64(world%5+1), 3), big.NewRat(int64(world%3), 7)
		h.must((*m.LowBounds)[entering].Copy(h.m, h.pair(lowx.RatString(), lowy.RatString())))
		h.must((*m.UpperBounds)[entering].Copy(h.m, h.pair(upx.RatString(), upy.RatString())))
		out := h.pair("97", "101")
		alias := world%13 == 0
		if alias {
			out = &(*m.X)[entering]
		}
		beforeX, beforeY := big.NewRat(97, 1), big.NewRat(101, 1)
		if alias {
			beforeX.SetInt64(0)
			beforeY.SetInt64(0)
		}
		seed := uint32(world + 17)
		s.Settings.RandomSeed(seed)
		random := z3rand.New(seed)
		start := int(random.Next() % uint32(n))
		var best *limit
		compare := func(ax, ay, bx, by *big.Rat) int {
			if c := ax.Cmp(bx); c != 0 {
				return c
			}
			return ay.Cmp(by)
		}
		for _, v := range limits {
			v := v
			if best == nil || compare(v.x, v.y, best.x, best.y) < 0 || (compare(v.x, v.y, best.x, best.y) == 0 && v.nz < best.nz) {
				best = &v
			}
		}
		var candidates []uint32
		if best != nil {
			for k := 0; k < n; k++ {
				i := uint32(order[(start+k)%n])
				if v, ok := limits[i]; ok && compare(v.x, v.y, best.x, best.y) == 0 && v.nz == best.nz {
					candidates = append(candidates, i)
				}
			}
		}
		want := -1
		wantX, wantY := new(big.Rat).Set(beforeX), new(big.Rat).Set(beforeY)
		if best != nil {
			wantX.Set(best.x)
			wantY.Set(best.y)
		}
		enteringX, enteringY := big.NewRat(0, 1), big.NewRat(0, 1)
		if alias {
			enteringX.Set(wantX)
			enteringY.Set(wantY)
		}
		stepX, stepY := new(big.Rat), new(big.Rat)
		if direction > 0 {
			stepX.Sub(upx, enteringX)
			stepY.Sub(upy, enteringY)
		} else {
			stepX.Sub(enteringX, lowx)
			stepY.Sub(enteringY, lowy)
		}
		canFlip := kind == ColumnBoxed
		if best != nil {
			canFlip = canFlip || (kind == ColumnUpperBound && direction > 0) || (kind == ColumnLowBound && direction < 0)
			canFlip = canFlip && compare(stepX, stepY, best.x, best.y) <= 0
		}
		// Debug rejects negative accepted bounded flips. Exclude those invalid-domain
		// aliases here; their native/debug behavior has a separate explicit test.
		if best != nil && canFlip && compare(stepX, stepY, big.NewRat(0, 1), big.NewRat(0, 1)) < 0 {
			out = h.pair("97", "101")
			alias = false
			enteringX.SetInt64(0)
			enteringY.SetInt64(0)
			if direction > 0 {
				stepX.Set(upx)
				stepY.Set(upy)
			} else {
				stepX.Neg(lowx)
				stepY.Neg(lowy)
			}
			canFlip = compare(stepX, stepY, best.x, best.y) <= 0
		}
		if canFlip {
			want = int(entering)
			wantX.Set(stepX)
			wantY.Set(stepY)
			counters["flip"]++
		} else if best != nil {
			want = int(candidates[random.Next()%uint32(len(candidates))])
			counters["basic"]++
			if len(candidates) > 1 {
				counters["ties"]++
			}
		} else {
			counters["unlimited"]++
		}
		got, err := m.FindLeavingAndTPrecise(entering, out)
		h.must(err)
		if got != want || !reflect.DeepEqual(m.LeavingCandidates, candidates) {
			t.Fatal(world, got, want, m.LeavingCandidates, candidates)
		}
		h.check(out, wantX, wantY)
		if s.Settings.RandomNext() != random.Next() {
			t.Fatal("random consumption", world)
		}
		if m.leavingMultiplier != nil || m.leavingRatio != nil {
			t.Fatal("successful scratch retained")
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
	for _, k := range []string{"flip", "basic", "ties", "unlimited"} {
		if counters[k] == 0 {
			t.Fatal("missing model profile", k, counters)
		}
	}
}

func TestLPExactLeavingGuardsAndBreakpointDispatch(t *testing.T) {
	for _, op := range []string{"nil", "empty", "zero", "ed-index", "basis-index", "row-nz", "unknown", "breakpoints"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpNewRemovalModel(t, h, "-1", "3", false)
			m := s.Core.RPrimal
			h.must(m.Ed.Resize(1))
			h.must(m.Ed.SetValue(&h.pair("1", "0").X, 0))
			m.RowsNZ = []uint32{2}
			m.SignOfEnteringDelta = 1
			h.must(s.Bounds.Types.Set(2, ColumnLowBound))
			h.must((*m.LowBounds)[2].Copy(h.m, h.pair("-1", "0")))
			out := h.pair("97", "101")
			want := ErrBounds
			m.LeavingCandidates = []uint32{77}
			seed := uint32(37)
			s.Settings.RandomSeed(seed)
			random := z3rand.New(seed)
			switch op {
			case "nil":
				out = nil
			case "empty":
				h.must(m.Ed.Clear())
				random.Next()
			case "zero":
				h.must(h.m.SetInt32(&m.Ed.Data[0], 0))
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "ed-index":
				m.Ed.Index[0] = 99
			case "basis-index":
				*m.Basis = nil
			case "row-nz":
				m.RowsNZ = nil
			case "unknown":
				h.must(s.Bounds.Types.Set(2, ColumnType(99)))
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "breakpoints":
				h.must(m.InfSet.Resize(3))
				h.must(m.InfSet.Insert(2))
				s.Settings.UseBreakpointsInFeasibilitySearch = true
				h.must(m.BreakpointQueue.Resize(4))
				h.must((*m.X)[2].Copy(h.m, h.pair("4", "0")))
				h.must((*m.LowBounds)[2].Copy(h.m, h.pair("0", "0")))
				h.must(h.m.SetInt32(&m.D.Data[0], -1))
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			}
			got, err := m.FindLeavingAndTPrecise(0, out)
			if err != want {
				t.Fatal(got, err, want)
			}
			if op == "empty" || op == "nil" || op == "breakpoints" {
				if !reflect.DeepEqual(m.LeavingCandidates, []uint32{77}) {
					t.Fatal("early prefix cleared candidates")
				}
				if s.Settings.RandomNext() != random.Next() {
					t.Fatal("early RNG prefix")
				}
			}
			if op == "row-nz" {
				if !reflect.DeepEqual(m.LeavingCandidates, []uint32{2}) {
					t.Fatal("row-nz check preceded candidate append")
				}
				h.check(out, big.NewRat(1, 1), big.NewRat(0, 1))
			}
			if op == "breakpoints" && !sourceDebugEnabled {
				if got != 2 {
					t.Fatal(got)
				}
				h.check(out, big.NewRat(4, 1), big.NewRat(0, 1))
			}
			if err != nil {
				if _, e := m.FindLeavingAndTPrecise(0, h.pair("0", "0")); e != ErrExactVectorPending {
					t.Fatal("missing stopped scan", e)
				}
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactLeavingPhysicalOwnership(t *testing.T) {
	for _, op := range []string{"ratio", "tie", "flip", "inline-minimum"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1600; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					s := lpNewRemovalModel(t, h, "-1", "3", true)
					m := s.Core.RPrimal
					h.must(m.Ed.Resize(2))
					m.RowsNZ = []uint32{2, 2}
					m.SignOfEnteringDelta = 1
					seed := uint32(1)
					for z3rand.New(seed).Next()%2 != 1 {
						seed++
					}
					s.Settings.RandomSeed(seed)
					huge := strings.Repeat("1234567890", 12)
					ed := h.pair(huge+"1/7", "0")
					for i := uint32(0); i < 2; i++ {
						h.must(m.Ed.SetValue(&ed.X, i))
						if op == "inline-minimum" {
							h.must(h.m.SetInt32(&m.Ed.Data[i], -2147483648))
						}
						h.must(s.Bounds.Types.Set(2+i, ColumnLowBound))
						h.must((*m.LowBounds)[2+i].Copy(h.m, h.pair("-"+huge+fmt.Sprint(i+3)+"/11", huge+"5/13")))
						if op == "inline-minimum" {
							h.must(s.Bounds.Types.Set(2+i, ColumnUpperBound))
							h.must((*m.UpperBounds)[2+i].Copy(h.m, h.pair(huge+fmt.Sprint(i+3)+"/11", huge+"5/13")))
						}
					}
					if op == "tie" {
						h.must((*m.LowBounds)[3].Copy(h.m, &(*m.LowBounds)[2]))
					}
					if op == "flip" {
						h.must(s.Bounds.Types.Set(0, ColumnBoxed))
						h.must((*m.UpperBounds)[0].Copy(h.m, h.pair("1/3", "1/17")))
					}
					out := h.pair("97", "101")
					h.budget.FailAfter(ordinal)
					_, err := m.FindLeavingAndTPrecise(0, out)
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.calls.pending() && m.PairCoreSolverBase.calls.pending() {
							t.Fatal("adopted delegated comparison")
						}
						if _, e := m.FindLeavingAndTPrecise(0, out); e != ErrExactVectorPending {
							t.Fatal(e)
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
				t.Fatal(op, direct, success, failures)
			}
		}
	}
}

func TestLPExactLeavingIndependentInfeasibleModels(t *testing.T) {
	for world := 0; world < 240; world++ {
		h := newLPPairHarness(t)
		const n = 4
		s := lpNewLARLUPreparedModel(t, h, lpExactIdentity(n))
		_, e := s.AddVar(1000)
		h.must(e)
		_, e = s.AddVar(1001)
		h.must(e)
		m := s.Core.RPrimal
		h.must(m.Ed.Resize(n))
		m.RowsNZ = []uint32{2, 4, 3, 2}
		m.SignOfEnteringDelta = 1
		if world%2 == 1 {
			m.SignOfEnteringDelta = -1
		}
		h.must((*m.X)[n+1].Copy(h.m, h.pair("1", "0")))
		h.must(m.InfSet.Resize(n + 2))
		type limit struct {
			x, y *big.Rat
			nz   uint32
		}
		limits := map[uint32]limit{}
		cmp := func(ax, ay, bx, by *big.Rat) int {
			if c := ax.Cmp(bx); c != 0 {
				return c
			}
			return ay.Cmp(by)
		}
		for offset := 0; offset < n; offset++ {
			i := uint32((world + offset) % n)
			kind := ColumnType((world + int(i)) % 5)
			if i == 0 {
				kind = ColumnBoxed
			}
			h.must(s.Bounds.Types.Set(i, kind))
			lowx, lowy, upx, upy := big.NewRat(-1, 1), big.NewRat(-1, 13), big.NewRat(2, 1), big.NewRat(2, 13)
			if kind == ColumnFixed {
				upx.Set(lowx)
				upy.Set(lowy)
			}
			x, y := big.NewRat(int64((world+int(i))%6-2), 1), big.NewRat(0, 1)
			if i == 0 {
				x.SetInt64(3)
			}
			h.must((*m.X)[i].Copy(h.m, h.pair(x.RatString(), "0")))
			// Companion nonbasic value1 keeps each actual row homogeneous: x_i-x_i*z=0.
			if x.Sign() != 0 {
				h.must(m.A.Set(i, n+1, &h.pair(new(big.Rat).Neg(x).RatString(), "0").X))
			}
			h.must((*m.LowBounds)[i].Copy(h.m, h.pair(lowx.RatString(), lowy.RatString())))
			h.must((*m.UpperBounds)[i].Copy(h.m, h.pair(upx.RatString(), upy.RatString())))
			below := cmp(x, y, lowx, lowy) < 0
			above := cmp(x, y, upx, upy) > 0
			if (kind == ColumnLowBound && below) || (kind == ColumnUpperBound && above) || ((kind == ColumnBoxed || kind == ColumnFixed) && (below || above)) {
				h.must(m.InfSet.Insert(i))
			}
			ed := big.NewRat(int64((world+int(i))%4+1), 3)
			if (world+int(i))%2 == 0 {
				ed.Neg(ed)
			}
			h.must(m.A.Set(i, n, &h.pair(ed.RatString(), "0").X))
			h.must(m.Ed.SetValue(&h.pair(ed.RatString(), "0").X, i))
			multiplier := new(big.Rat).Mul(new(big.Rat).Neg(ed), big.NewRat(int64(m.SignOfEnteringDelta), 1))
			positive := multiplier.Sign() > 0
			var tx, ty *big.Rat
			active, zero := false, false
			switch kind {
			case ColumnLowBound:
				tx, ty = lowx, lowy
				if positive {
					active = below
				} else {
					c := cmp(x, y, lowx, lowy)
					active = c > 0
					zero = c == 0
				}
			case ColumnUpperBound:
				tx, ty = upx, upy
				if positive {
					c := cmp(x, y, upx, upy)
					active = c < 0
					zero = c == 0
				} else {
					active = above
				}
			case ColumnBoxed, ColumnFixed:
				if positive {
					if below {
						active = true
						tx, ty = lowx, lowy
					} else {
						c := cmp(x, y, upx, upy)
						active = c < 0
						zero = c == 0
						tx, ty = upx, upy
					}
				} else {
					if above {
						active = true
						tx, ty = upx, upy
					} else {
						c := cmp(x, y, lowx, lowy)
						active = c > 0
						zero = c == 0
						tx, ty = lowx, lowy
					}
				}
			}
			if active || zero {
				qx, qy := big.NewRat(0, 1), big.NewRat(0, 1)
				if active {
					qx.Quo(new(big.Rat).Sub(tx, x), multiplier)
					qy.Quo(new(big.Rat).Sub(ty, y), multiplier)
				}
				if cmp(qx, qy, big.NewRat(0, 1), big.NewRat(0, 1)) < 0 {
					qx.SetInt64(0)
					qy.SetInt64(0)
				}
				limits[i] = limit{qx, qy, m.RowsNZ[i]}
			}
		}
		seed := uint32(world + 93)
		s.Settings.RandomSeed(seed)
		random := z3rand.New(seed)
		start := int(random.Next() % n)
		var best *limit
		for _, v := range limits {
			v := v
			if best == nil || cmp(v.x, v.y, best.x, best.y) < 0 || (cmp(v.x, v.y, best.x, best.y) == 0 && v.nz < best.nz) {
				best = &v
			}
		}
		var candidates []uint32
		if best != nil {
			for k := 0; k < n; k++ {
				i := m.Ed.Index[(start+k)%n]
				if v, ok := limits[i]; ok && cmp(v.x, v.y, best.x, best.y) == 0 && v.nz == best.nz {
					candidates = append(candidates, i)
				}
			}
		}
		want := -1
		out := h.pair("97", "101")
		if best != nil {
			want = int(candidates[random.Next()%uint32(len(candidates))])
		}
		got, err := m.FindLeavingAndTPrecise(n, out)
		h.must(err)
		if got != want || !reflect.DeepEqual(m.LeavingCandidates, candidates) {
			t.Fatal(world, got, want, m.LeavingCandidates, candidates)
		}
		if best != nil {
			h.check(out, best.x, best.y)
		} else {
			h.check(out, big.NewRat(97, 1), big.NewRat(101, 1))
		}
		if s.Settings.RandomNext() != random.Next() {
			t.Fatal("infeasible RNG", world)
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactLeavingPreservesOlderComparisonOwner(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpNewRemovalModel(t, h, "-1", "3", true)
			m := s.Core.RPrimal
			h.must(m.Ed.Resize(2))
			m.RowsNZ = []uint32{2, 2}
			m.SignOfEnteringDelta = 1
			for i := uint32(0); i < 2; i++ {
				h.must(m.Ed.SetValue(&h.pair("1", "0").X, i))
				h.must(s.Bounds.Types.Set(2+i, ColumnLowBound))
				h.must((*m.LowBounds)[2+i].Copy(h.m, h.pair("-1", "-1/7")))
			}
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			var caller exactEtaCalls
			before, e := caller.begin(h.m)
			h.must(e)
			h.budget.FailAfter(ordinal)
			_, err := h.m.Compare(&v.X, &v.Y)
			h.budget.FailAfter(-1)
			err = caller.after(h.m, before, err)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				matched = true
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				_, err = m.FindLeavingAndTPrecise(0, h.pair("97", "101"))
				if m.calls.comparison != 0 || m.PairCoreSolverBase.calls.comparison != 0 {
					t.Fatal("adopted older comparison", err)
				}
			}
			h.must(s.Close())
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("destroyed older caller")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no older comparison invocation")
	}
}

func TestLPExactLeavingLiveBasisAliases(t *testing.T) {
	for _, alias := range []string{"x", "low", "upper"} {
		h := newLPPairHarness(t)
		s := lpNewRemovalModel(t, h, "-1", "3", true)
		m := s.Core.RPrimal
		h.must(m.Ed.Resize(2))
		m.RowsNZ = []uint32{2, 2}
		m.SignOfEnteringDelta = 1
		for i := uint32(0); i < 2; i++ {
			h.must(m.Ed.SetValue(&h.pair("1", "0").X, i))
			h.must(s.Bounds.Types.Set(2+i, ColumnLowBound))
		}
		h.must((*m.LowBounds)[2].Copy(h.m, h.pair("-1", "0")))
		h.must((*m.LowBounds)[3].Copy(h.m, h.pair("-3", "0")))
		// Start with row1. Publishing its theta3 changes the row0 input before row0
		// is read when t aliases row0 X or its low bound.
		seed := uint32(1)
		for z3rand.New(seed).Next()%2 != 1 {
			seed++
		}
		s.Settings.RandomSeed(seed)
		out := &(*m.X)[2]
		wantJ, wantX := 3, int64(3)
		if alias == "low" {
			out = &(*m.LowBounds)[2]
			wantJ, wantX = 2, 0
		}
		if alias == "upper" {
			out = &(*m.UpperBounds)[2]
			wantJ, wantX = 2, 1
		}
		got, e := m.FindLeavingAndTPrecise(0, out)
		h.must(e)
		if got != wantJ {
			t.Fatal(alias, got, wantJ)
		}
		h.check(out, big.NewRat(wantX, 1), big.NewRat(0, 1))
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactLeavingEnteringAliasNegativeFlip(t *testing.T) {
	h := newLPPairHarness(t)
	s := lpNewRemovalModel(t, h, "-1", "3", false)
	m := s.Core.RPrimal
	h.must(m.Ed.Resize(1))
	h.must(m.Ed.SetValue(&h.pair("1", "0").X, 0))
	m.RowsNZ = []uint32{2}
	m.SignOfEnteringDelta = 1
	h.must(s.Bounds.Types.Set(2, ColumnLowBound))
	h.must((*m.LowBounds)[2].Copy(h.m, h.pair("-3", "0")))
	h.must(s.Bounds.Types.Set(0, ColumnBoxed))
	h.must((*m.UpperBounds)[0].Copy(h.m, h.pair("1", "0")))
	out := &(*m.X)[0]
	got, err := m.FindLeavingAndTPrecise(0, out)
	if sourceDebugEnabled {
		if err != ErrAssertion || got != -1 || m.leavingRatio == nil {
			t.Fatal(got, err)
		}
		h.check(out, big.NewRat(3, 1), big.NewRat(0, 1))
		h.check(&m.leavingRatio.Data[0], big.NewRat(-2, 1), big.NewRat(0, 1))
	} else {
		h.must(err)
		if got != 0 {
			t.Fatal(got)
		}
		h.check(out, big.NewRat(-2, 1), big.NewRat(0, 1))
	}
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactLeavingSecondRowMetadataAndSparseIndex(t *testing.T) {
	for _, op := range []string{"missing-nz", "only-row0", "bad-entering"} {
		h := newLPPairHarness(t)
		s := lpNewRemovalModel(t, h, "-1", "3", true)
		m := s.Core.RPrimal
		h.must(m.Ed.Resize(2))
		m.RowsNZ = []uint32{2}
		m.SignOfEnteringDelta = 1
		h.must(m.Ed.SetValue(&h.pair("1", "0").X, 0))
		if op != "only-row0" {
			h.must(m.Ed.SetValue(&h.pair("1", "0").X, 1))
		}
		for j := uint32(2); j < 4; j++ {
			h.must(s.Bounds.Types.Set(j, ColumnLowBound))
			h.must((*m.LowBounds)[j].Copy(h.m, h.pair("-1", "0")))
		}
		seed := uint32(1)
		for z3rand.New(seed).Next()%2 != 0 {
			seed++
		}
		s.Settings.RandomSeed(seed)
		entering := uint32(0)
		want := ErrBounds
		if op == "only-row0" {
			want = nil
		}
		if op == "bad-entering" {
			entering = 99
			h.must(s.Bounds.Types.Set(2, ColumnFree))
			h.must(s.Bounds.Types.Set(3, ColumnFree))
		}
		out := h.pair("97", "101")
		got, err := m.FindLeavingAndTPrecise(entering, out)
		if err != want {
			t.Fatal(op, got, err, want)
		}
		if op == "missing-nz" {
			if !reflect.DeepEqual(m.LeavingCandidates, []uint32{2}) || m.leavingRatio == nil {
				t.Fatal("lost first row prefix")
			}
			h.check(out, big.NewRat(1, 1), big.NewRat(0, 1))
		}
		if op == "only-row0" {
			if got != 2 {
				t.Fatal(got)
			}
			h.check(out, big.NewRat(1, 1), big.NewRat(0, 1))
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}
