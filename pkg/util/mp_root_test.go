// Source active assertions: src/test/mpz.cpp tst_root. MPQ has no root tests;
// its expectations below derive from mpq.h/cpp short-circuit output algorithms.
package util

import (
	"errors"
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestIntegerSourceRoots(t *testing.T) {
	m, _ := integerManager(t)
	for _, f := range []struct {
		a     int32
		n     uint32
		want  int32
		exact bool
	}{{213, 5, 3, false}, {-213, 5, -2, false}, {0, 3, 0, true}, {8, 3, 2, true}, {-8, 3, -2, true}} {
		a := NewInt(f.a)
		exact, e := m.RootInPlace(&a, f.n)
		if e != nil || exact != f.exact || a.Val != f.want || !a.IsSmall() {
			t.Fatal(f, exact, e, integerString(t, m, &a))
		}
		// The two source diagnostic lines use the same asserted results;
		// their expected bytes are permanent, not read from a C++ process.
		if f.a == 213 && "213^{1/5}: "+integerString(t, m, &a)+"\n" != "213^{1/5}: 3\n" {
			t.Fatal("source positive diagnostic")
		}
		if f.a == -213 && "-213^{1/5}: "+integerString(t, m, &a)+"\n" != "-213^{1/5}: -2\n" {
			t.Fatal("source negative diagnostic")
		}
		a = NewInt(f.a)
		out := NewInt(29)
		exact, e = m.Root(&a, f.n, &out)
		if e != nil || exact != f.exact || out.Val != f.want || a.Val != f.a {
			t.Fatal(f, exact, e)
		}
	}
}

// Independent finite mathematical search uses different whole-integer bounds;
// it checks signed ceiling roots, without reproducing source log/power iteration.
func rootOracle(a *big.Int, n uint32) (*big.Int, bool) {
	magnitude := new(big.Int).Abs(new(big.Int).Set(a))
	lo, hi := new(big.Int), new(big.Int).Add(magnitude, big.NewInt(1))
	one := big.NewInt(1)
	for new(big.Int).Sub(hi, lo).Cmp(one) > 0 {
		mid := new(big.Int).Rsh(new(big.Int).Add(lo, hi), 1)
		p := new(big.Int).Exp(mid, new(big.Int).SetUint64(uint64(n)), nil)
		if p.Cmp(magnitude) <= 0 {
			lo = mid
		} else {
			hi = mid
		}
	}
	exact := new(big.Int).Exp(lo, new(big.Int).SetUint64(uint64(n)), nil).Cmp(magnitude) == 0
	if a.Sign() < 0 {
		return lo.Neg(lo), exact
	}
	if exact {
		return lo, true
	}
	return hi, false
}
func TestIntegerRootIndependentAndAliases(t *testing.T) {
	m, _ := integerManager(t)
	for a := int64(-64); a <= 64; a++ {
		for n := uint32(1); n <= 7; n++ {
			if a < 0 && n%2 == 0 {
				continue
			}
			want, precise := rootOracle(big.NewInt(a), n)
			for _, alias := range []bool{false, true} {
				in, out := NewInt(int32(a)), NewInt(27)
				dst := &out
				if alias {
					dst = &in
				}
				exact, e := m.Root(&in, n, dst)
				if e != nil || exact != precise || integerString(t, m, dst) != want.String() {
					t.Fatal(a, n, alias, exact, e)
				}
			}
		}
	}
	random := rand.New(rand.NewSource(0x726f6f74))
	for trial := 0; trial < 80; trial++ {
		a := new(big.Int).SetUint64(random.Uint64())
		a.Lsh(a, uint(random.Intn(65)))
		a.Add(a, new(big.Int).SetUint64(random.Uint64()))
		n := uint32(2 + random.Intn(7))
		if n%2 == 1 && trial%2 == 1 {
			a.Neg(a)
		}
		want, precise := rootOracle(a, n)
		for _, alias := range []bool{false, true} {
			var in, out Int
			setIntegerOracle(t, m, &in, a)
			dst := &out
			if alias {
				dst = &in
			}
			exact, e := m.Root(&in, n, dst)
			if e != nil || exact != precise || integerString(t, m, dst) != want.String() {
				t.Fatal(trial, n, alias, exact, e, want)
			}
		}
	}
}
func TestPerfectSquareIndependentAndSourceAlias(t *testing.T) {
	m, _ := integerManager(t)
	for a := int32(-2); a <= 512; a++ {
		in, out := NewInt(a), NewInt(37)
		exact, e := m.IsPerfectSquare(&in, &out)
		if e != nil {
			t.Fatal(e)
		}
		if a < 0 {
			if exact || out.Val != 37 {
				t.Fatal(a, exact, out)
			}
			continue
		}
		want, precise := rootOracle(big.NewInt(int64(a)), 2)
		if exact != precise || integerString(t, m, &out) != want.String() {
			t.Fatal(a, exact, want)
		}
		// Source resets output before rereading an aliased nonnegative input.
		exact, e = m.IsPerfectSquare(&in, &in)
		if e != nil || !exact || in.Val != 0 || !in.IsSmall() {
			t.Fatal("source alias reset", a, exact, e)
		}
	}
}
func TestRationalRootAndSquareSourcePublication(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		for _, f := range []struct {
			a     string
			n     uint32
			want  string
			exact bool
		}{{"8/27", 3, "2/3", true}, {"-8/27", 3, "-2/3", true}, {"213/32", 5, "3/13", false}, {"8/29", 3, "2/4", false}, {"1/2", 2, "1/2", false}, {"0", 3, "0", true}} {
			// Source rational root short-circuits: numerator false leaves prior den13.
			a, out := NewRational(), NewRational()
			if e := m.SetString(&a, f.a); e != nil {
				t.Fatal(e)
			}
			m.SetFraction(&out, 7, 13)
			exact, e := m.Root(&a, f.n, &out)
			if e != nil || exact != f.exact || rationalString(t, m, &out) != f.want {
				t.Fatal(synch, f, exact, e, rationalString(t, m, &out))
			}
		}
		for _, f := range []struct {
			a, want string
			exact   bool
		}{{"4/9", "2/3", true}, {"2/9", "2/13", false}, {"4/7", "2/3", false}, {"-4", "7", false}, {"-4/9", "7/13", false}, {"4", "2", true}} {
			a, out := NewRational(), NewRational()
			m.SetString(&a, f.a)
			m.SetFraction(&out, 7, 13)
			exact, e := m.IsPerfectSquare(&a, &out)
			if e != nil || exact != f.exact || rationalString(t, m, &out) != f.want {
				t.Fatal(synch, f, exact, e, rationalString(t, m, &out))
			}
		}
		for _, text := range []string{"4", "0"} {
			a := NewRational()
			m.SetString(&a, text)
			exact, e := m.IsPerfectSquare(&a, &a)
			if e != nil || !exact || rationalString(t, m, &a) != "0" {
				t.Fatal("source rational alias", text, e)
			}
		}
		for _, text := range []string{"4/9", "2/9"} {
			a := NewRational()
			m.SetString(&a, text)
			exact, e := m.IsPerfectSquare(&a, &a)
			if exact || !errors.Is(e, ErrDivideByZero) || rationalString(t, m, &a) != text {
				t.Fatal("source undefined 0/0 normalization", text, exact, e)
			}
		}

	}
}
func TestRootDomainsAndInitialization(t *testing.T) {
	m, b := integerManager(t)
	for _, f := range []struct {
		a        int32
		n        uint32
		sentinel error
	}{{1, 0, ErrRootDegreeZero}, {-1, 2, ErrRootNegativeEven}, {-1, 0, ErrRootNegativeEven}, {-2, math.MaxUint32, ErrNonterminatingPower}} {
		a, out := NewInt(f.a), NewInt(29)
		before := b.Used()
		precise, e := m.Root(&a, f.n, &out)
		// Source mpz.h:837 SET(out,a) precedes the in-place domain guard.
		if precise || !errors.Is(e, f.sentinel) || out != a || a.Val != f.a || b.Used() != before {
			t.Fatal(f, precise, e)
		}
		if f.sentinel == ErrNonterminatingPower {
			if !m.RootRaw.active || m.RootRaw.EndEntered || !m.PowerRaw.active || m.PowerRaw.endEntered || m.RootRaw.Values[0] != NewInt(-2) || m.RootRaw.Values[1] != NewInt(-1) || m.RootRaw.Values[2] != NewInt(-1) || m.PowerRaw.Value != NewInt(1) {
				t.Fatal("actual wrapped-mask RAW prefix", m.RootRaw, m.PowerRaw)
			}
			integerPhysicalAudit(t, m, &a, &out)
			serviceMust(t, m.CleanupRoot())
			integerPhysicalAudit(t, m, &a, &out)
		} else if m.RootRaw.active || m.PowerRaw.active {
			t.Fatal("prebody guarded domain constructed RAW locals")
		}
	}
	for _, f := range []struct {
		a     int32
		n     uint32
		want  int32
		exact bool
	}{{0, 0, 0, true}, {0, math.MaxUint32, 0, true}, {1, math.MaxUint32, 1, true}, {2, math.MaxUint32, 2, false}} {
		a := NewInt(f.a)
		exact, e := m.RootInPlace(&a, f.n)
		if e != nil || exact != f.exact || a.Val != f.want {
			t.Fatal(f, exact, e)
		}
	}
	rm, _ := rationalManager(t, false)
	a, out := NewRational(), NewRational()
	rm.SetFraction(&a, 1, 2)
	if _, e := rm.IsIntPerfectSquare(&a, &out); !errors.Is(e, ErrRootIntegerRequired) {
		t.Fatal(e)
	}
	var raw Rational
	if _, e := rm.Root(&raw, 2, &out); !errors.Is(e, ErrUninitializedRational) {
		t.Fatal(e)
	}
	if _, e := rm.Root(&out, 0, &out); !errors.Is(e, ErrRootDegreeZero) {
		t.Fatal(e)
	}
	m.Close()
	aInt := NewInt(1)
	if _, e := m.RootInPlace(&aInt, 2); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	rm.Close()
	if _, e := rm.IsPerfectSquare(&out, &out); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestRationalRootIndependentAndAliases(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		for _, n := range []uint32{1, 2, 3, 5} {
			for num := int64(-19); num <= 19; num++ {
				if num < 0 && n%2 == 0 {
					continue
				}
				for den := uint64(1); den <= 11; den++ {
					canonical := new(big.Rat).SetFrac(big.NewInt(num), new(big.Int).SetUint64(den))
					nr, preciseN := rootOracle(canonical.Num(), n)
					dr, preciseD := rootOracle(canonical.Denom(), n)
					for _, alias := range []bool{false, true} {
						a, out := NewRational(), NewRational()
						if e := m.SetFraction(&a, num, den); e != nil {
							t.Fatal(e)
						}
						m.SetFraction(&out, 7, 13)
						dst := &out
						wantDen := big.NewInt(13)
						if alias {
							dst = &a
							wantDen = new(big.Int).Set(canonical.Denom())
						}
						if preciseN {
							wantDen = dr
						}
						precise, e := m.Root(&a, n, dst)
						if e != nil || precise != (preciseN && preciseD) || integerString(t, m.Z, &dst.Num) != nr.String() || integerString(t, m.Z, &dst.Den) != wantDen.String() {
							t.Fatal(synch, num, den, n, alias, precise, e)
						}
						m.drop(&a)
						m.drop(&out)
					}
				}
			}
		}
	}
}
func TestBigPerfectSquaresAndIntegerWrapper(t *testing.T) {
	m, _ := integerManager(t)
	for _, shift := range []uint{31, 32, 63, 96, 127} {
		r := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), shift), big.NewInt(17))
		square := new(big.Int).Mul(r, r)
		for _, delta := range []int64{-1, 0, 1} {
			a := new(big.Int).Add(square, big.NewInt(delta))
			want, precise := rootOracle(a, 2)
			var in, out Int
			setIntegerOracle(t, m, &in, a)
			exact, e := m.IsPerfectSquare(&in, &out)
			if e != nil || exact != precise || integerString(t, m, &out) != want.String() {
				t.Fatal(shift, delta, e)
			}
			m.Del(&in)
			m.Del(&out)
		}
	}
	rm, _ := rationalManager(t, false)
	for _, v := range []int64{-9, 0, 1, 2, 9} {
		a, out := NewRational(), NewRational()
		rm.SetInt64(&a, v)
		rm.SetFraction(&out, 7, 13)
		exact, e := rm.IsIntPerfectSquare(&a, &out)
		if e != nil {
			t.Fatal(e)
		}
		if v < 0 {
			if exact || rationalString(t, rm, &out) != "7" {
				t.Fatal(v, exact)
			}
		} else {
			want, p := rootOracle(big.NewInt(v), 2)
			if exact != p || rationalString(t, rm, &out) != want.String() {
				t.Fatal(v, exact)
			}
		}
	}
}
func TestIntegerRootAndSquareFailureCleanup(t *testing.T) {
	large := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 160), big.NewInt(1))
	for operation := 0; operation < 3; operation++ {
		for _, bigOutput := range []bool{false, true} {
			for _, alias := range []bool{false, true} {
				failures, completed := 0, false
				for fault := int64(0); fault < 2048; fault++ {
					m, b := integerManager(t)
					var a, out Int
					setIntegerOracle(t, m, &a, large)
					out = NewInt(29)
					if bigOutput {
						setIntegerOracle(t, m, &out, new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 97), big.NewInt(29)))
					}
					if operation == 1 && !alias {
						if e := m.Copy(&out, &a); e != nil {
							t.Fatal(e)
						}
					}
					dst := &out
					if alias {
						dst = &a
					}
					inputBefore, outputBefore := integerString(t, m, &a), integerString(t, m, dst)
					initialDestination := *dst
					b.FailAfter(fault)
					var exact bool
					var e error
					switch operation {
					case 0:
						exact, e = m.Root(&a, 2, dst)
					case 1:
						exact, e = m.RootInPlace(dst, 2)
					case 2:
						exact, e = m.IsPerfectSquare(&a, dst)
					}
					if e != nil {
						// Inspect live ownership BEFORE String or Close can run.
						qopsAuditLiveOwnership(t, &RationalManager{Z: m}, []*Int{&a, &out})
						b.FailAfter(-1)
						failures++
						if !numericPhysicalReservationRefusal(e) {
							t.Fatal(operation, alias, fault, e)
						}
						if operation != 2 {
							scope := &m.RootRaw
							if scope.active {
								want := inputBefore
								if scope.EndEntered {
									root, _ := rootOracle(large, 2)
									want = root.String()
								}
								if integerString(t, m, dst) != want {
									t.Fatal("source Root caller SET/SWAP prefix", operation, alias, fault)
								}
							} else {
								if operation != 0 || alias || !m.IsFailureRef(dst.Cell) || dst.Val != 1 {
									t.Fatal("failed source caller SET route", operation, alias, fault, *dst)
								}
								want := m.FailedNullRef
								if initialDestination.Cell != 0 {
									want = m.FailedRef
								}
								if dst.Cell != want {
									t.Fatal("source caller SET exact pointer publication", fault, dst.Cell, want)
								}
								if x := m.check(dst); !errors.Is(x, arena.ErrInvalidSlot) {
									t.Fatal("actual failed caller pointer guard", x)
								}
							}
							if !alias && integerString(t, m, &a) != inputBefore {
								t.Fatal("source Root input changed", fault)
							}
							callerPending := !scope.active && m.PendingCell.Object != (umemory.ObjectRef{})
							serviceMust(t, m.CleanupRoot())
							if callerPending {
								serviceMust(t, m.CleanupPendingAllocation())
							}
						} else {
							scope := &m.PerfectSquare
							if scope.Active && !scope.outputStarted {
								if dst.Cell != 0 || dst.Val != 0 {
									t.Fatal("square source reset before raw body failure", alias, fault, *dst)
								}
							} else if scope.outputStarted {
								if m.IsFailureRef(dst.Cell) {
									if dst.Cell != m.FailedNullRef || dst.Val != 1 {
										t.Fatal("fresh caller final SET refusal", fault, *dst)
									}
									if x := m.check(dst); !errors.Is(x, arena.ErrInvalidSlot) {
										t.Fatal("caller source pointer guard", x)
									}
								} else {
									want, _ := rootOracle(large, 2)
									if integerString(t, m, dst) != want.String() {
										t.Fatal("source output published before raw end Del", fault)
									}
								}
							} else if x := m.check(dst); x == nil {
								// Reset itself stopped before scalar zero publication.
								if integerString(t, m, dst) != outputBefore {
									t.Fatal("square stopped Reset prefix", fault)
								}
							} else if !errors.Is(x, arena.ErrInvalidSlot) {
								t.Fatal("stopped Reset caller guard", x)
							}
							// Operation abort drains only its six actual RAW
							// locals. Final SET's unpublished caller block is
							// an explicitly separate caller cleanup obligation.
							serviceMust(t, m.CleanupPerfectSquare())
							if scope.outputStarted && m.PendingCell.Object != (umemory.ObjectRef{}) {
								serviceMust(t, m.CleanupPendingAllocation())
							}
							if !alias && integerString(t, m, &a) != inputBefore {
								t.Fatal("source square input changed", fault)
							}
						}
					} else {
						b.FailAfter(-1)
						completed = true
						want, p := rootOracle(large, 2)
						if operation == 2 && alias {
							want = big.NewInt(0)
							p = true
						}
						if exact != p || integerString(t, m, dst) != want.String() {
							t.Fatal("root success", operation, alias, fault, exact)
						}
					}
					m.Close()
					if b.Used() != 0 {
						t.Fatal("root close", operation, alias, fault, b.Used())
					}
					if completed {
						break
					}
				}
				// Source aliased square Reset is allocation-free; all other paths must
				// exhibit at least one failed preparation and a finite successful endpoint.
				if !completed || (!(operation == 2 && alias) && failures == 0) {
					t.Fatal(operation, alias, "sweep incomplete", failures)
				}
			}
		}
	}
}
func TestRationalRootSquareFailureCleanup(t *testing.T) {
	rootN := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 55), big.NewInt(3))
	rootD := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 37), big.NewInt(11))
	numerator := new(big.Int).Exp(rootN, big.NewInt(3), nil)
	denominator := new(big.Int).Add(new(big.Int).Exp(rootD, big.NewInt(3), nil), big.NewInt(1))
	for _, synch := range []bool{false, true} {
		for op := 0; op < 3; op++ {
			for _, bigOutput := range []bool{false, true} {
				for _, alias := range []bool{false, true} {
					failed, completed := false, false
					for fault := int64(0); fault < 4096; fault++ {
						m, b := rationalManager(t, synch)
						a, out := NewRational(), NewRational()
						setRationalOracle(t, m, &a, numerator, denominator)
						if op != 0 {
							m.resetDenominator(&a)
						} // integer query/square aliases have valid source domain
						m.SetFraction(&out, 7, 13)
						if bigOutput {
							setRationalOracle(t, m, &out, new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 103), big.NewInt(7)), new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 79), big.NewInt(13)))
						}
						dst := &out
						if alias {
							dst = &a
						}
						beforeA, beforeOut := rationalString(t, m, &a), rationalString(t, m, dst)
						b.FailAfter(fault)
						var precise bool
						var e error
						switch op {
						case 0:
							precise, e = m.Root(&a, 3, dst)
						case 1:
							precise, e = m.IsPerfectSquare(&a, dst)
						case 2:
							precise, e = m.IsIntPerfectSquare(&a, dst)
						}
						if e != nil {
							// These source branches do not normalize; no cached
							// rational GCD may legitimately add a live cell here.
							// Check BEFORE formatter or blanket owner Close.
							qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &out.Num, &out.Den})
							b.FailAfter(-1)
							failed = true
							if !numericPhysicalReservationRefusal(e) || rationalString(t, m, &a) != beforeA || rationalString(t, m, dst) != beforeOut {
								t.Fatal("rational root publication", synch, op, alias, fault, e)
							}
						} else {
							b.FailAfter(-1)
							completed = true
							if op != 0 && alias {
								if !precise || rationalString(t, m, dst) != "0" {
									t.Fatal("square alias", fault)
								}
							}
						}
						m.Close()
						if b.Used() != 0 {
							t.Fatal("rational root close", synch, op, alias, fault)
						}
						if completed {
							break
						}
					}
					if !completed || (!(op != 0 && alias) && !failed) {
						t.Fatal(synch, op, alias, "sweep incomplete")
					}
				}
			}
		}
	}
}
func TestRootWarmStorageAndClosedOutputs(t *testing.T) {
	m, b := integerManager(t)
	var a, root, square Int
	m.SetString(&a, "18446744073709551617")
	for i := 0; i < 3; i++ {
		if _, e := m.Root(&a, 3, &root); e != nil {
			t.Fatal(e)
		}
		if _, e := m.IsPerfectSquare(&a, &square); e != nil {
			t.Fatal(e)
		}
	}
	if !SourceExtendedGCDDebug {
		b.FailAfter(0)
	}
	before := numericWarmProfileSnapshot(t, m, b)
	var e error
	allocations := testing.AllocsPerRun(30, func() {
		_, e = m.Root(&a, 3, &root)
		if e == nil {
			_, e = m.IsPerfectSquare(&a, &square)
		}
	})
	b.FailAfter(-1)
	if e != nil {
		t.Fatal("warm roots", allocations, e)
	}
	numericWarmProfileAudit(t, m, b, before, allocations, SourceExtendedGCDDebug)
	m.Close()
	if _, e = m.IsPerfectSquare(&a, &root); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestRationalRootWarmStorage(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, b := rationalManager(t, synch)
		n := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 40), big.NewInt(3))
		d := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 31), big.NewInt(1))
		a, out, square := NewRational(), NewRational(), NewRational()
		setRationalOracle(t, m, &a, new(big.Int).Mul(n, n), new(big.Int).Mul(d, d))
		for i := 0; i < 3; i++ {
			if exact, e := m.Root(&a, 2, &out); e != nil || !exact {
				t.Fatal(exact, e)
			}
			if exact, e := m.IsPerfectSquare(&a, &square); e != nil || !exact {
				t.Fatal(exact, e)
			}
		}
		before := numericWarmProfileSnapshot(t, m.Z, b)
		var e error
		if !SourceExtendedGCDDebug {
			b.FailAfter(0)
		}
		allocations := testing.AllocsPerRun(20, func() {
			_, e = m.Root(&a, 2, &out)
			if e == nil {
				_, e = m.IsPerfectSquare(&a, &square)
			}
		})
		b.FailAfter(-1)
		if e != nil {
			t.Fatal("rational warm roots", synch, allocations, e)
		}
		numericWarmProfileAudit(t, m.Z, b, before, allocations, SourceExtendedGCDDebug)
		m.Close()
		if b.Used() != 0 {
			t.Fatal("rational warm close", synch, b.Used())
		}
	}
}
