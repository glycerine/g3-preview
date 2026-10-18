package polynomial

import (
	"errors"
	"fmt"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicCopyFromCoreReservationPrefixes(t *testing.T) {
	for sk := 1; sk <= 2; sk++ {
		for dk := 0; dk <= 2; dk++ {
			t.Run(fmt.Sprintf("%d-to-%d", sk, dk), func(t *testing.T) {
				success, failures := false, 0
				for prefix := int64(0); prefix < 128 && !success; prefix++ {
					t.Run(fmt.Sprint(prefix), func(t *testing.T) {
						source, _ := algebraicFixture(t)
						target, _ := algebraicFixture(t)
						src := algebraicForeignValue(t, source, sk)
						if sk == 2 {
							serviceMust(t, source.Del(&src))
							src = algebraicLargeDyadicPromotionRoot(t, source)
						}
						dst := algebraicForeignValue(t, target, dk)
						budget := target.config.Rationals.Budget()
						budget.FailAfter(prefix)
						body := target.CopyFromCore(&dst, source, &src)
						budget.FailAfter(-1)
						if body != nil {
							failures++
							if !numericPhysicalReservationRefusal(body) {
								t.Fatal("unexpected refusal", body)
							}
							if !target.foreignCopy.active || source.foreignCopyBorrowers != 1 {
								t.Fatal("failed invocation lost source lifetime")
							}
							for _, e := range []error{source.Del(&src), source.CleanupRetained(), source.Close(), target.CopyFromCore(&dst, source, &src)} {
								if !errors.Is(e, ErrAlgebraicPending) {
									t.Fatal("pending invocation lost exclusion", e)
								}
							}
						} else {
							success = true
						}
						serviceMust(t, target.CleanupRetained())
						if target.foreignCopy.active || source.foreignCopyBorrowers != 0 || source.config.Polynomials.algebraicCopyBorrowers != 0 || target.config.Polynomials.algebraicCopyBorrowers != 0 {
							t.Fatal("cleanup retained source borrow")
						}
						serviceMust(t, target.CleanupValue(&dst))
						serviceMust(t, source.Del(&src))
					})
				}
				if !success || failures == 0 && !(sk == 1 && dk == 1) {
					t.Fatal("missing refusal/success coverage", failures, success)
				}
			})
		}
	}
}

func TestAlgebraicCopyFromCoreAllocationCallback(t *testing.T) {
	for kind := 1; kind <= 2; kind++ {
		t.Run(fmt.Sprint(kind), func(t *testing.T) {
			source, _ := algebraicFixture(t)
			src := algebraicForeignValue(t, source, kind)
			var target *AlgebraicRootCore
			var dst Algebraic
			reached := false
			transport := func(f umemory.Failure) error {
				if target == nil || !target.foreignCopy.running {
					return f
				}
				reached = true
				for _, c := range []*AlgebraicRootCore{source, target} {
					for _, e := range []error{c.Close(), c.CleanupRetained()} {
						if !errors.Is(e, ErrAlgebraicPending) {
							t.Fatal("callback obtained core lifetime", e)
						}
					}
					for _, e := range []error{c.config.Polynomials.Close(), c.config.Polynomials.SourceFinalize()} {
						if !errors.Is(e, ErrUPolynomialCleanupPending) {
							t.Fatal("callback obtained univariate owner lifetime", e)
						}
					}
					z := c.config.Rationals.Integers()
					for _, e := range []error{z.CloseChecked(), z.SourceFinalize(), c.config.Rationals.CloseChecked(), c.config.Rationals.SourceFinalize()} {
						if e == nil {
							t.Fatal("callback obtained numeral owner lifetime")
						}
					}
					if z.Closed() {
						t.Fatal("callback retired numeral owner")
					}
				}
				if e := source.Del(&src); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("callback deleted source", e)
				}
				if e := target.Del(&dst); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("callback deleted destination", e)
				}
				return f
			}
			var rt *umemory.Runtime
			target, _, rt = algebraicNegTLSFixture(t, transport)
			pad := algebraicCellSourceTLSBoundary(t, target)
			body := target.CopyFromCore(&dst, source, &src)
			if body == nil || !reached || !target.foreignCopy.active || source.foreignCopyBorrowers != 1 {
				t.Fatal("allocation callback not reached with retained source", body)
			}
			if kind == 1 && !target.config.Rationals.Integers().MatchesCopyAllocationInvocation(&target.constQ.copy) {
				t.Fatal("stopped basic copy lost its accepted NUM receipt")
			}
			if kind == 2 && !target.config.Allocator.MatchesPendingAllocationOutcome(&target.cellAllocation) {
				t.Fatal("stopped root copy lost its accepted cell allocation outcome")
			}
			serviceMust(t, rt.SetMaxSize(0))
			contextTLSFinish(t, rt, func() error { return rt.Deallocate(&pad) })
			contextTLSFinish(t, rt, target.CleanupRetained)
			contextTLSFinish(t, rt, func() error { return target.CleanupValue(&dst) })
			serviceMust(t, source.Del(&src))
		})
	}
}

func TestAlgebraicCopyFromCoreOwningManagerLifetime(t *testing.T) {
	cfg, _, _ := algebraicManagerFixture(t)
	source, e := NewAlgebraicManager(cfg)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, source.Close()) })
	target, _ := algebraicFixture(t)
	src := algebraicForeignValue(t, source.AlgebraicRootCore, 1)
	var dst Algebraic
	budget := target.config.Rationals.Budget()
	budget.FailAfter(0)
	body := target.CopyFromCore(&dst, source.AlgebraicRootCore, &src)
	budget.FailAfter(-1)
	if body == nil || !target.foreignCopy.active {
		t.Fatal("expected stopped cross-owner invocation", body)
	}
	header := source.header
	if e := source.Close(); !errors.Is(e, ErrAlgebraicPending) {
		t.Fatal("owning manager closed borrowed source", e)
	}
	if source.header != header || source.privateFinalized || !source.completed || source.finishing || source.AlgebraicRootCore.closing {
		t.Fatal("owning manager advanced destruction while borrowed")
	}
	serviceMust(t, target.CleanupRetained())
	serviceMust(t, target.CleanupValue(&dst))
	serviceMust(t, target.CopyFromCore(&dst, source.AlgebraicRootCore, &src))
	serviceMust(t, source.Del(&src))
	serviceMust(t, source.Close())
	l, e := target.location(&dst)
	serviceMust(t, e)
	r, e := target.load(l)
	serviceMust(t, e)
	text, e := target.config.Rationals.Integers().String(r.value.BorrowNumerator())
	serviceMust(t, e)
	if text != "-1267650600228229401496703205377" {
		t.Fatal("target depended on closed source manager", text)
	}
	serviceMust(t, target.Del(&dst))
}

func algebraicForeignValue(t *testing.T, c *AlgebraicRootCore, kind int) Algebraic {
	t.Helper()
	var a Algebraic
	switch kind {
	case 1:
		q := unumeral.NewRational()
		serviceMust(t, c.config.Rationals.Integers().SetString(q.BorrowNumerator(), "-1267650600228229401496703205377"))
		serviceMust(t, c.config.Rationals.Integers().SetString(q.BorrowDenominator(), "1361129467683753853853498429727072845825"))
		serviceMust(t, c.SetRational(&a, &q))
		serviceMust(t, c.config.Rationals.Del(&q))
	case 2:
		a = algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 3, 1, true)
		l, e := c.location(&a)
		serviceMust(t, e)
		r, e := c.load(l)
		serviceMust(t, e)
		v, e := c.sourcePolynomial(&r)
		serviceMust(t, e)
		_, e = c.displayRootIndex(l, v, 0)
		serviceMust(t, e)
	}
	return a
}

func TestAlgebraicCopyFromCoreBranches(t *testing.T) {
	for sk := 0; sk < 3; sk++ {
		for dk := 0; dk < 3; dk++ {
			t.Run(fmt.Sprintf("%d-to-%d", sk, dk), func(t *testing.T) {
				source, _ := algebraicFixture(t)
				target, _ := algebraicFixture(t)
				src := algebraicForeignValue(t, source, sk)
				dst := algebraicForeignValue(t, target, dk)
				t.Cleanup(func() {
					serviceMust(t, target.CleanupRetained())
					serviceMust(t, target.Del(&dst))
					serviceMust(t, source.Del(&src))
				})
				var before algebraicRecord
				if sk != 0 {
					l, e := source.location(&src)
					serviceMust(t, e)
					before, e = source.load(l)
					serviceMust(t, e)
				}
				serviceMust(t, target.CopyFromCore(&dst, source, &src))
				if source.foreignCopyBorrowers != 0 || target.foreignCopy.active {
					t.Fatal("successful copy retained source borrow")
				}
				if sk == 0 {
					if dst != (Algebraic{}) {
						t.Fatal("zero did not erase destination")
					}
					return
				}
				l, e := target.location(&dst)
				serviceMust(t, e)
				out, e := target.load(l)
				serviceMust(t, e)
				if sk == 1 {
					for i, a := range []*unumeral.Int{out.value.BorrowNumerator(), out.value.BorrowDenominator()} {
						b := before.value.BorrowNumerator()
						if i == 1 {
							b = before.value.BorrowDenominator()
						}
						cmp, e := target.config.Rationals.Integers().CompareFromOwners(target.config.Rationals.Integers(), a, source.config.Rationals.Integers(), b)
						serviceMust(t, e)
						if cmp != 0 {
							t.Fatal("basic field changed", i)
						}
					}
				} else if out.size != before.size || out.flags != before.flags || out.interval.BorrowLower().KField != before.interval.BorrowLower().KField || out.interval.BorrowUpper().KField != before.interval.BorrowUpper().KField {
					t.Fatal("root fields changed")
				}
				serviceMust(t, source.Del(&src))
				if sk == 2 {
					one := unumeral.NewRational()
					serviceMust(t, target.config.Rationals.SetFraction(&one, 1, 1))
					cmp, e := target.CompareRational(&dst, &one)
					serviceMust(t, e)
					serviceMust(t, target.config.Rationals.Del(&one))
					if cmp <= 0 {
						t.Fatal("copied positive sqrt2 lost after source deletion")
					}
				} else {
					text, e := target.config.Rationals.Integers().String(out.value.BorrowNumerator())
					serviceMust(t, e)
					if text != "-1267650600228229401496703205377" {
						t.Fatal(text)
					}
				}
			})
		}
	}
}

// Exercise copy_poly with independently owned big numeral cells. A small-only
// fixture would not distinguish a foreign owner from the receiving manager.
func TestAlgebraicCopyPolynomialFromOwner(t *testing.T) {
	source, sp := algebraicFixture(t)
	target, tp := algebraicFixture(t)
	want := []string{"-1267650600228229401496703205377", "7", "1361129467683753853853498429727072845825"}
	values := make([]unumeral.Int, len(want))
	for i := range values {
		serviceMust(t, source.config.Rationals.Integers().SetString(&values[i], want[i]))
	}
	poly, e := sp.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, sp.SetInts(poly, values))
	for i := range values {
		serviceMust(t, source.config.Rationals.Integers().Del(&values[i]))
	}
	v, e := sp.wholeView(poly)
	serviceMust(t, e)
	a, e := target.allocate(1)
	serviceMust(t, e)
	var dst Algebraic
	target.publish(&dst, a)
	t.Cleanup(func() { serviceMust(t, target.Del(&dst)) })
	l, e := target.location(&dst)
	serviceMust(t, e)
	r, e := target.load(l)
	serviceMust(t, e)
	serviceMust(t, target.copyPolynomialFromOwner(l, &r, sp, v))
	if r.size != uint32(len(want)) || l.coefficientInitialized != r.size {
		t.Fatalf("copy prefix size=%d initialized=%d", r.size, l.coefficientInitialized)
	}
	serviceMust(t, sp.DelPolynomial(&poly))
	out, e := tp.wholeView(r.poly)
	serviceMust(t, e)
	for i, text := range want {
		n, e := tp.read(out, uint32(i))
		serviceMust(t, e)
		got, e := target.config.Rationals.Integers().String(n)
		serviceMust(t, e)
		if got != text {
			t.Fatalf("coefficient %d: got %s want %s", i, got, text)
		}
	}
}

func TestAlgebraicCopyUsesRecordedPolynomialSize(t *testing.T) {
	c, p := algebraicFixture(t)
	src := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	var dst Algebraic
	t.Cleanup(func() {
		serviceMust(t, c.Del(&dst))
		serviceMust(t, c.Del(&src))
	})
	l, e := c.location(&src)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	// The algebraic cell captures its source extent independently of the Go
	// descriptor's backing extent. Extra backing must not extend copy_poly.
	serviceMust(t, p.SetInts(r.poly, []unumeral.Int{
		unumeral.NewInt(-2), unumeral.NewInt(0), unumeral.NewInt(1), unumeral.NewInt(99),
	}))
	serviceMust(t, c.Copy(&dst, &src))
	dl, e := c.location(&dst)
	serviceMust(t, e)
	out, e := c.load(dl)
	serviceMust(t, e)
	v, e := p.wholeView(out.poly)
	serviceMust(t, e)
	if out.size != r.size || v.count != r.size || dl.coefficientInitialized != r.size {
		t.Fatalf("recorded=%d copied=%d backing=%d initialized=%d", r.size, out.size, v.count, dl.coefficientInitialized)
	}
	if out.flags != r.flags {
		t.Fatalf("flags changed: %x -> %x", r.flags, out.flags)
	}
}
