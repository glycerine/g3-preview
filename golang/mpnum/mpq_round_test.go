// Licensed under the MIT license.
// Source assertions: src/test/mpq.cpp tst_prev_power_2 and src/test/mpz.cpp tst_log2.
package mpnum

import (
	"errors"
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func TestRationalSourcePrevPowerOfTwo(t *testing.T) {
	fixtures := []struct {
		n    int64
		d    uint64
		want uint32
	}{{-10, 1, 0}, {0, 1, 0}, {1, 1, 0}, {2, 1, 1}, {3, 1, 1}, {4, 1, 2}, {5, 1, 2}, {8, 1, 3}, {9, 1, 3}, {9, 2, 2}, {9, 4, 1}, {9, 5, 0}, {(1 << 60) + 1, 1, 60}, {1 << 60, 1, 60}, {(1 << 60) - 1, 1, 59}, {1 << 60, 3, 58}}
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a := NewRational()
		for _, f := range fixtures {
			if e := m.SetFraction(&a, f.n, f.d); e != nil {
				t.Fatal(e)
			}
			got, e := m.PrevPowerOfTwo(&a)
			if e != nil || got != f.want {
				t.Fatal(synch, f, got, e)
			}
		}
	}
}

func checkSourceLogBounds(t *testing.T, m *IntegerManager, a *Int, n *big.Int) {
	t.Helper()
	k, e := m.Log2(a)
	if e != nil {
		t.Fatal(e)
	}
	// Source ENSURE bounds use independent mathematical powers until production Power exists.
	low := new(big.Int).Lsh(big.NewInt(1), uint(k))
	high := new(big.Int).Lsh(big.NewInt(1), uint(k+1))
	if (n.Sign() != 0 && low.Cmp(n) > 0) || n.Cmp(high) > 0 {
		t.Fatal("source positive log bounds", n, k)
	}
	var neg Int
	defer m.Del(&neg)
	if e = m.Copy(&neg, a); e != nil {
		t.Fatal(e)
	}
	if e = m.Neg(&neg); e != nil {
		t.Fatal(e)
	}
	k, e = m.MLog2(&neg)
	if e != nil {
		t.Fatal(e)
	}
	negN := new(big.Int).Neg(n)
	low.Neg(new(big.Int).Lsh(big.NewInt(1), uint(k)))
	high.Neg(new(big.Int).Lsh(big.NewInt(1), uint(k+1)))
	if (negN.Sign() != 0 && negN.Cmp(low) > 0) || high.Cmp(negN) > 0 {
		t.Fatal("source negative log bounds", n, k)
	}
}
func TestIntegerSourceLogBounds(t *testing.T) {
	m, _ := integerManager(t)
	var a Int
	for i := int64(0); i < 1000; i++ {
		m.SetInt64(&a, i)
		checkSourceLogBounds(t, m, &a, big.NewInt(i))
	}
	n, _ := new(big.Int).SetString("1029489380487098723984579237", 10)
	for i := 0; i < 1000; i++ {
		n.Add(n, big.NewInt(1))
		if e := m.SetString(&a, n.String()); e != nil {
			t.Fatal(e)
		}
		checkSourceLogBounds(t, m, &a, n)
	}
}
func TestIntegerLogSignsSizesAndOwnership(t *testing.T) {
	m, budget := integerManager(t)
	var a Int
	r := rand.New(rand.NewSource(81))
	values := []*big.Int{big.NewInt(0), big.NewInt(1), big.NewInt(-1), big.NewInt(math.MinInt32), big.NewInt(math.MinInt64)}
	for i := 0; i < 200; i++ {
		values = append(values, roundRandomInteger(r, 1+r.Intn(16)))
	}
	for _, n := range values {
		setIntegerOracle(t, m, &a, n)
		before := budget.Used()
		budget.FailAfter(0)
		l, e := m.Log2(&a)
		ml, e2 := m.MLog2(&a)
		sz, e3 := m.Bitsize(&a)
		prev, e4 := m.PrevPowerOfTwo(&a)
		budget.FailAfter(-1)
		want := uint32(0)
		if n.Sign() != 0 {
			want = uint32(n.BitLen() - 1)
		}
		if e != nil || e2 != nil || e3 != nil || e4 != nil || sz != want+1 || prev != l || budget.Used() != before {
			t.Fatal(n, l, ml, sz, e, e2, e3, e4)
		}
		if (n.Sign() > 0 && (l != want || ml != 0)) || (n.Sign() < 0 && (ml != want || l != 0)) || (n.Sign() == 0 && (l != 0 || ml != 0)) {
			t.Fatal(n, l, ml)
		}
	}
	m.Close()
	if _, e := m.Log2(&a); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func independentRound(n, d *big.Int, ceil bool) *big.Int {
	q, r := new(big.Int), new(big.Int)
	q.QuoRem(n, d, r)
	if r.Sign() != 0 {
		if ceil && n.Sign() > 0 {
			q.Add(q, big.NewInt(1))
		}
		if !ceil && n.Sign() < 0 {
			q.Sub(q, big.NewInt(1))
		}
	}
	return q
}
func TestRationalRoundIndependentAndAliases(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		r := rand.New(rand.NewSource(92))
		for i := 0; i < 500; i++ {
			n, d := big.NewInt(int64(i%61-30)), big.NewInt(int64(i%15+1))
			if i >= 300 {
				n = roundRandomInteger(r, 1+r.Intn(12))
				d.Abs(roundRandomInteger(r, 1+r.Intn(5)))
				if d.Sign() == 0 {
					d.SetInt64(1)
				}
			}
			for _, ceil := range []bool{false, true} {
				want := independentRound(n, d, ceil)
				for alias := 0; alias < 2; alias++ {
					a, out := NewRational(), NewRational()
					setRationalOracle(t, m, &a, n, d)
					dst := &out
					if alias == 1 {
						dst = &a
					}
					var err error
					if ceil {
						err = m.Ceil(&a, dst)
					} else {
						err = m.Floor(&a, dst)
					}
					if err != nil || rationalString(t, m, dst) != want.String() || dst.den.val != 1 || !dst.den.IsSmall() {
						t.Fatal(i, ceil, alias, err, want)
					}
					m.Del(&a)
					m.Del(&out)
				}
				a := NewRational()
				setRationalOracle(t, m, &a, n, d)
				var out Int
				var err error
				if ceil {
					err = m.CeilToInt(&a, &out)
				} else {
					err = m.FloorToInt(&a, &out)
				}
				if err != nil || integerString(t, m.z, &out) != want.String() {
					t.Fatal(i, ceil, err, want)
				}
				// Source integer output may alias the numerator; the sign must be captured before division.
				if ceil {
					err = m.CeilToInt(&a, &a.num)
				} else {
					err = m.FloorToInt(&a, &a.num)
				}
				if err != nil || integerString(t, m.z, &a.num) != want.String() {
					t.Fatal("numerator alias", i, ceil, err)
				}
				m.Del(&a)
				m.z.Del(&out)
			}
			a := NewRational()
			setRationalOracle(t, m, &a, n, d)
			got, e := m.PrevPowerOfTwo(&a)
			floor := independentRound(n, d, false)
			want := uint32(0)
			if floor.Sign() > 0 {
				want = uint32(floor.BitLen() - 1)
			}
			if e != nil || got != want {
				t.Fatal(i, got, want, e)
			}
			m.Del(&a)
		}
	}
}

func roundRandomInteger(r *rand.Rand, limbs int) *big.Int {
	b := make([]byte, limbs*4)
	r.Read(b)
	n := new(big.Int).SetBytes(b)
	if r.Intn(2) == 0 {
		n.Neg(n)
	}
	return n
}

func TestRationalRoundFailureCleanup(t *testing.T) {
	n := new(big.Int).Lsh(big.NewInt(1), 400)
	n.Add(n, big.NewInt(17))
	n.Neg(n)
	d := new(big.Int).Lsh(big.NewInt(1), 80)
	d.Add(d, big.NewInt(3))
	failures, successes := 0, 0
	for _, synch := range []bool{false, true} {
		for op := 0; op < 5; op++ {
			for alias := 0; alias < 2; alias++ {
				for after := int64(0); after < 48; after++ {
					m, budget := rationalManager(t, synch)
					a, out := NewRational(), NewRational()
					setRationalOracle(t, m, &a, n, d)
					m.SetInt64(&out, 17)
					var integer Int
					m.z.SetInt64(&integer, 17)
					original := rationalString(t, m, &a)
					beforeA, beforeOut, beforeInteger := a, out, integer
					sourceNum := roundPublicationDecode(t, m.z, &a.num)
					sourceDen := roundPublicationDecode(t, m.z, &a.den)
					dst := &out
					intDst := &integer
					if alias == 1 {
						dst = &a
						intDst = &a.num
					}
					var power uint32
					call := func() error {
						switch op {
						case 0:
							return m.FloorToInt(&a, intDst)
						case 1:
							return m.CeilToInt(&a, intDst)
						case 2:
							return m.Floor(&a, dst)
						case 3:
							return m.Ceil(&a, dst)
						default:
							var e error
							power, e = m.PrevPowerOfTwo(&a)
							return e
						}
					}
					budget.FailAfter(after)
					err := call()
					qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &out.num, &out.den, &integer})
					if err != nil {
						if !numericPhysicalReservationRefusal(err) {
							t.Fatal("round source error", synch, op, alias, after, err)
						}
						if op == 4 {
							if a != beforeA || out != beforeOut || integer != beforeInteger {
								t.Fatal("source PrevPower local changed caller", synch, alias, after)
							}
						} else {
							// Decode completed source actions BEFORE disabling faults or
							// rendering. This wide quotient cannot publish failedRef:
							// source TMP transfer may precede its replacement refusal.
							target := intDst
							if op >= 2 {
								target = &dst.num
							}
							actual := roundPublicationDecode(t, m.z, target)
							old := big.NewInt(17)
							if alias == 1 {
								old = sourceNum
							}
							quotient := new(big.Int).Quo(sourceNum, sourceDen)
							corrected := independentRound(sourceNum, sourceDen, op == 1 || op == 3)
							if actual.Cmp(old) != 0 && actual.Cmp(quotient) != 0 && actual.Cmp(corrected) != 0 {
								t.Fatal("not a completed source round action", synch, op, alias, after, actual)
							}
							if a.den != beforeA.den || out.den != beforeOut.den {
								t.Fatal("denominator reset before successful integer round", synch, op, alias, after)
							}
							if alias == 0 && a != beforeA {
								t.Fatal("nonoutput input changed", synch, op, after)
							}
							if (op < 2 || alias == 1) && out != beforeOut {
								t.Fatal("nonoutput rational changed", synch, op, alias, after)
							}
							if (op >= 2 || alias == 1) && integer != beforeInteger {
								t.Fatal("nonoutput integer changed", synch, op, alias, after)
							}
						}
					}
					budget.FailAfter(-1)
					if err != nil {
						failures++
					} else {
						successes++
					}
					want := independentRound(n, d, op == 1 || op == 3)
					if err == nil {
						switch op {
						case 0, 1:
							if integerString(t, m.z, intDst) != want.String() {
								t.Fatal(op, alias, after)
							}
						case 2, 3:
							if rationalString(t, m, dst) != want.String() {
								t.Fatal(op, alias, after)
							}
						case 4:
							if power != 0 || rationalString(t, m, &a) != original {
								t.Fatal(power)
							}
						}
					}
					m.Close()
					if budget.Used() != 0 {
						t.Fatal("round leaked", budget.Used())
					}
				}
			}
		}
	}
	if failures == 0 || successes == 0 {
		t.Fatal("round failure sweep boundary", failures, successes)
	}
}
func TestRationalRoundInitializationAndClose(t *testing.T) {
	m, _ := rationalManager(t, true)
	var raw Rational
	out := NewRational()
	var integer Int
	if e := m.Floor(&raw, &out); !errors.Is(e, ErrUninitializedRational) {
		t.Fatal(e)
	}
	if e := m.CeilToInt(&raw, &integer); !errors.Is(e, ErrUninitializedRational) {
		t.Fatal(e)
	}
	if _, e := m.PrevPowerOfTwo(&raw); !errors.Is(e, ErrUninitializedRational) {
		t.Fatal(e)
	}
	m.Close()
	if e := m.FloorToInt(&out, &integer); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestRationalRoundWarmZeroAllocations(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, budget := rationalManager(t, synch)
		a, out := NewRational(), NewRational()
		n := new(big.Int).Lsh(big.NewInt(1), 300)
		n.Add(n, big.NewInt(7))
		setRationalOracle(t, m, &a, n, big.NewInt(13))
		var integer Int
		call := func() error {
			if e := m.FloorToInt(&a, &integer); e != nil {
				return e
			}
			if e := m.CeilToInt(&a, &integer); e != nil {
				return e
			}
			if e := m.Floor(&a, &out); e != nil {
				return e
			}
			if e := m.Ceil(&a, &out); e != nil {
				return e
			}
			_, e := m.PrevPowerOfTwo(&a)
			return e
		}
		for i := 0; i < 3; i++ {
			if e := call(); e != nil {
				t.Fatal(e)
			}
		}
		beforeWarm := numericWarmProfileSnapshot(t, m.z, budget)
		var err error
		got := testing.AllocsPerRun(20, func() { err = call() })
		if err != nil {
			t.Fatal(err)
		}
		numericWarmProfileAudit(t, m.z, budget, beforeWarm, got, m.z.allocator.Stats().DirectObjects > 0)
	}
}
