package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"
)

func TestLPPairCoreValuesIndependentModels(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	rng := rand.New(rand.NewPCG(7519, 331))
	for _, tableau := range []bool{false, true} {
		for world := 0; world < 160; world++ {
			h := newLPPairHarness(t)
			n := 1 + world%4
			f := lpNewPairCoreFixture(t, h, uint32(n), uint32(n+1))
			if !tableau {
				f.settings.SimplexStrategy = StrategyLU
			}
			b := lpExactIdentity(n)
			d := make([]*big.Rat, n)
			for i := 0; i < n; i++ {
				d[i] = big.NewRat(int64((world+i)%7-3), int64(i+2))
				if !tableau {
					for j := 0; j <= i; j++ {
						b[i][j].SetFrac64(int64((world+i+j)%5+1), int64(j+2))
					}
				}
				for j := range b[i] {
					if b[i][j].Sign() != 0 {
						h.must(f.a.Set(uint32(i), uint32(j), &h.pair(b[i][j].RatString(), "0").X))
					}
				}
			}
			// The entering column is B*d. Arbitrary insertion order exercises real
			// tableau column links; the LU branch solves the actual nonunit basis.
			for _, i := range rng.Perm(n) {
				v := new(big.Rat)
				for j := 0; j < n; j++ {
					v.Add(v, new(big.Rat).Mul(b[i][j], d[j]))
				}
				if v.Sign() != 0 {
					h.must(f.a.Set(uint32(i), uint32(n), &h.pair(v.RatString(), "0").X))
				}
			}
			m, e := NewPairPrimalCoreSolver(f.input())
			h.must(e)
			m.ColumnsNZ = make([]uint32, n+1)
			if !tableau {
				h.must(m.SolveBd(uint32(n)))
				rng.Shuffle(len(m.Ed.Index), func(i, j int) { m.Ed.Index[i], m.Ed.Index[j] = m.Ed.Index[j], m.Ed.Index[i] })
			} else {
				// UpdateX must ignore Ed in tableau mode, while RestoreX still uses it.
				for i := 0; i < n; i++ {
					if (world+i)%2 == 0 {
						h.must(m.Ed.SetValue(&h.pair("999", "0").X, uint32(i)))
					}
				}
			}
			for i := 0; i < n; i++ {
				h.must(m.CopyXB.Data[i].Copy(h.m, h.pair(fmt.Sprint(90+i), fmt.Sprint(100+i))))
			}
			dx, dy := big.NewRat(int64(world%13-6), 7), big.NewRat(int64(world%17-8), 11)
			if world%9 == 0 {
				dx.SetInt64(0)
				dy.SetInt64(0)
			}
			delta := h.pair(dx.RatString(), dy.RatString())
			h.must(m.UpdateX(uint32(n), delta))
			h.check(&f.x.Data[n], dx, dy)
			for i := 0; i < n; i++ {
				h.check(&f.x.Data[i], new(big.Rat).Neg(new(big.Rat).Mul(dx, d[i])), new(big.Rat).Neg(new(big.Rat).Mul(dy, d[i])))
				h.check(&m.CopyXB.Data[i], big.NewRat(int64(90+i), 1), big.NewRat(int64(100+i), 1))
			}
			// Independently multiply the updated model: both exact Ax coordinates are0.
			for i := 0; i < n; i++ {
				sx, sy := new(big.Rat), new(big.Rat)
				for j := 0; j < n; j++ {
					sx.Sub(sx, new(big.Rat).Mul(b[i][j], new(big.Rat).Mul(dx, d[j])))
					sy.Sub(sy, new(big.Rat).Mul(b[i][j], new(big.Rat).Mul(dy, d[j])))
				}
				v := new(big.Rat)
				for j := 0; j < n; j++ {
					v.Add(v, new(big.Rat).Mul(b[i][j], d[j]))
				}
				sx.Add(sx, new(big.Rat).Mul(v, dx))
				sy.Add(sy, new(big.Rat).Mul(v, dy))
				if sx.Sign() != 0 || sy.Sign() != 0 {
					t.Fatal("independent Ax model")
				}
			}
			h.must(m.RestoreX(uint32(n), delta))
			h.check(&f.x.Data[n], big.NewRat(0, 1), big.NewRat(0, 1))
			indexed := map[uint32]bool{}
			for _, i := range m.Ed.Index {
				indexed[i] = true
			}
			for i := 0; i < n; i++ {
				x, y := new(big.Rat).Neg(new(big.Rat).Mul(dx, d[i])), new(big.Rat).Neg(new(big.Rat).Mul(dy, d[i]))
				if (dx.Sign() != 0 || dy.Sign() != 0) && indexed[uint32(i)] {
					x.SetInt64(int64(90 + i))
					y.SetInt64(int64(100 + i))
				}
				h.check(&f.x.Data[i], x, y)
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPPairCoreValuesLiveAliases(t *testing.T) {
	for _, tableau := range []bool{false, true} {
		for _, alias := range []string{"entering", "first-basic", "last-basic"} {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 2, 3)
			if !tableau {
				f.settings.SimplexStrategy = StrategyLU
			}
			for i := uint32(0); i < 2; i++ {
				h.must(f.a.Set(i, i, &h.pair("1", "0").X))
			}
			// Actual column order is row1 then row0.
			h.must(f.a.Set(1, 2, &h.pair("3", "0").X))
			h.must(f.a.Set(0, 2, &h.pair("2", "0").X))
			m, e := NewPairPrimalCoreSolver(f.input())
			h.must(e)
			h.must(m.Ed.SetValue(&h.pair("3", "0").X, 1))
			h.must(m.Ed.SetValue(&h.pair("2", "0").X, 0))
			old := [][2]int64{{5, 7}, {11, 13}, {17, 19}}
			for i, v := range old {
				h.must(f.x.Data[i].Copy(h.m, h.pair(fmt.Sprint(v[0]), fmt.Sprint(v[1]))))
			}
			a := 2
			if alias == "first-basic" {
				a = 1
			}
			if alias == "last-basic" {
				a = 0
			}
			want := make([][2]*big.Rat, 3)
			for i, v := range old {
				want[i] = [2]*big.Rat{big.NewRat(v[0], 1), big.NewRat(v[1], 1)}
			}
			for c := 0; c < 2; c++ {
				want[2][c].Add(want[2][c], want[a][c])
			}
			for _, i := range []int{1, 0} {
				coef := big.NewRat(int64(i+2), 1)
				for c := 0; c < 2; c++ {
					product := new(big.Rat).Mul(coef, want[a][c])
					want[i][c].Sub(want[i][c], product)
				}
			}
			h.must(m.UpdateX(2, &f.x.Data[a]))
			for i := range want {
				h.check(&f.x.Data[i], want[i][0], want[i][1])
			}
			// An alias to the entering value is zeroed by restore before backup reads.
			h.must(m.RestoreX(2, &f.x.Data[2]))
			for i := range f.x.Data {
				h.check(&f.x.Data[i], big.NewRat(0, 1), big.NewRat(0, 1))
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPPairCoreValuesGuardsAndPreciseIndexEarlyReturn(t *testing.T) {
	for _, op := range []string{"nil-update", "nil-restore", "bad-entering", "bad-restore-entering", "basis-index", "ed-index", "ed-storage", "basic-column", "column-count", "column-link", "restore-index", "restore-backup", "restore-zero", "index-off", "index-size"} {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 1, 2)
		h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
		h.must(f.a.Set(0, 1, &h.pair("2", "0").X))
		m, e := NewPairPrimalCoreSolver(f.input())
		h.must(e)
		h.must(m.Ed.SetValue(&h.pair("2", "0").X, 0))
		h.must(f.x.Data[1].Copy(h.m, h.pair("5", "7")))
		step := h.pair("1", "3")
		want := ErrBounds
		var err error
		switch op {
		case "nil-update":
			err = m.UpdateX(1, nil)
		case "nil-restore":
			err = m.RestoreX(1, nil)
		case "bad-entering":
			err = m.UpdateX(99, step)
		case "bad-restore-entering":
			err = m.RestoreX(99, step)
		case "basis-index":
			f.basis = nil
			err = m.UpdateX(1, step)
			h.check(&f.x.Data[1], big.NewRat(6, 1), big.NewRat(10, 1))
		case "ed-index":
			f.settings.SimplexStrategy = StrategyLU
			m.Ed.Index[0] = 99
			err = m.UpdateX(1, step)
		case "ed-storage":
			f.settings.SimplexStrategy = StrategyLU
			h.must(m.Ed.resizeData(0))
			m.Ed.Index = []uint32{0}
			err = m.UpdateX(1, step)
		case "basic-column":
			f.basis[0] = 99
			err = m.UpdateX(1, step)
		case "column-count":
			h.must(f.x.resizeData(3))
			err = m.UpdateX(2, step)
			h.check(&f.x.Data[2], big.NewRat(1, 1), big.NewRat(3, 1))
		case "restore-backup":
			h.must(m.CopyXB.resizeData(0))
			err = m.RestoreX(1, step)
			h.check(&f.x.Data[1], big.NewRat(4, 1), big.NewRat(4, 1))
		case "column-link":
			f.a.Columns[1][0].Offset = 99
			err = m.UpdateX(1, step)
		case "restore-index":
			m.Ed.Index[0] = 99
			err = m.RestoreX(1, step)
			h.check(&f.x.Data[1], big.NewRat(4, 1), big.NewRat(4, 1))
		case "restore-zero":
			want = nil
			err = m.RestoreX(99, h.pair("0", "0"))
		case "index-off":
			want = nil
			off, e := m.AMultXIsOffOnIndex([]uint32{99})
			err = e
			if off {
				t.Fatal("precise branch scanned actual bad Ax")
			}
		case "index-size":
			want = nil
			h.must(f.x.resizeData(1))
			if sourceDebugEnabled {
				want = ErrAssertion
			}
			_, err = m.AMultXIsOffOnIndex([]uint32{99})
		}
		if err != want {
			t.Fatal(op, err, want)
		}
		if err != nil {
			if e := m.UpdateX(1, step); e != ErrExactVectorPending {
				t.Fatal("lost core stop", e)
			}
			if _, e := m.AMultXIsOffOnIndex(nil); e != ErrExactVectorPending {
				t.Fatal("lost indexed check stop", e)
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPPairCoreValuesPhysicalOwnershipAndPrefixes(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, op := range []string{"lu-update", "tableau-update", "restore"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1200; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewPairCoreFixture(t, h, 2, 3)
					if op == "lu-update" {
						f.settings.SimplexStrategy = StrategyLU
					}
					for i := uint32(0); i < 2; i++ {
						h.must(f.a.Set(i, i, &h.pair("1", "0").X))
					}
					huge := strings.Repeat("1234567890", 12)
					d := h.pair(huge+"1/7", huge+"3/11")
					coefficient := h.pair(huge+"5/13", "0")
					for _, i := range []uint32{1, 0} {
						h.must(f.a.Set(i, 2, &coefficient.X))
					}
					m, e := NewPairPrimalCoreSolver(f.input())
					h.must(e)
					for _, i := range []uint32{1, 0} {
						h.must(m.Ed.SetValue(&coefficient.X, i))
						h.must(m.CopyXB.Data[i].Copy(h.m, h.pair(huge+"7/17", huge+"9/19")))
					}
					old := [][2]*big.Rat{{big.NewRat(2, 1), big.NewRat(3, 1)}, {big.NewRat(5, 1), big.NewRat(7, 1)}, {big.NewRat(11, 1), big.NewRat(13, 1)}}
					for i, v := range old {
						h.must(f.x.Data[i].Copy(h.m, h.pair(v[0].RatString(), v[1].RatString())))
					}
					dx, dy := mustLPRat(t, huge+"1/7"), mustLPRat(t, huge+"3/11")
					coef := mustLPRat(t, huge+"5/13")
					next := make([][2]*big.Rat, 3)
					for i := range old {
						next[i] = [2]*big.Rat{new(big.Rat), new(big.Rat)}
					}
					if op == "restore" {
						next[2][0].Sub(old[2][0], dx)
						next[2][1].Sub(old[2][1], dy)
						for i := 0; i < 2; i++ {
							next[i][0].Set(mustLPRat(t, huge+"7/17"))
							next[i][1].Set(mustLPRat(t, huge+"9/19"))
						}
					} else {
						next[2][0].Add(old[2][0], dx)
						next[2][1].Add(old[2][1], dy)
						for i := 0; i < 2; i++ {
							next[i][0].Sub(old[i][0], new(big.Rat).Mul(coef, dx))
							next[i][1].Sub(old[i][1], new(big.Rat).Mul(coef, dy))
						}
					}
					h.budget.FailAfter(ordinal)
					var err error
					if op == "restore" {
						err = m.RestoreX(2, d)
					} else {
						err = m.UpdateX(2, d)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if e := m.RestoreX(2, d); e != ErrExactVectorPending {
							t.Fatal(e)
						}
					}
					// Completed scalars form the source prefix. The first stopped scalar
					// may retain a partial numerator/denominator or an unreadable raw
					// invocation; subsequent destinations must remain untouched.
					prefixEnded := false
					for _, i := range []int{2, 1, 0} {
						for c := 0; c < 2; c++ {
							v := &f.x.Data[i].X
							if c == 1 {
								v = &f.x.Data[i].Y
							}
							str, readErr := h.m.String(v)
							var value *big.Rat
							if readErr == nil {
								value = mustLPRat(t, str)
							}
							isOld := value != nil && value.Cmp(old[i][c]) == 0
							isNew := value != nil && value.Cmp(next[i][c]) == 0
							if err == nil && !isNew {
								t.Fatal("successful scalar", i, c, str, readErr)
							}
							if prefixEnded && !isOld {
								t.Fatal("mutation after stopped scalar", i, c, str, readErr)
							}
							if !isNew {
								prefixEnded = true
							}
						}
					}
					h.must(m.Close())
					f.close()
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

func TestLPPairCoreValuesPreservesOlderComparisonOwner(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"update", "restore", "precise-index"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				f := lpNewPairCoreFixture(t, h, 1, 2)
				h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
				h.must(f.a.Set(0, 1, &h.pair("2", "0").X))
				m, e := NewPairPrimalCoreSolver(f.input())
				h.must(e)
				h.must(m.Ed.SetValue(&h.pair("2", "0").X, 0))
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
					step := h.pair("3/7", "5/11")
					switch op {
					case "update":
						err = m.UpdateX(1, step)
					case "restore":
						err = m.RestoreX(1, step)
					default:
						_, err = m.AMultXIsOffOnIndex([]uint32{99})
					}
					h.must(err)
					if m.PairCoreSolverBase.calls.comparison != 0 || m.calls.comparison != 0 {
						t.Fatal("adopted older comparison")
					}
				}
				h.must(m.Close())
				f.close()
				if older.Active && h.m.ComparisonInvocationState() != older {
					t.Fatal("destroyed caller comparison")
				}
				h.must(caller.cleanupRaw(h.m))
				h.must(caller.cleanupRoots(h.m))
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !matched {
			t.Fatal(op, "no real older invocation")
		}
	}
}
