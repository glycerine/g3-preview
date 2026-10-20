package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestUPolynomialFactorsSourceSuccessThenManagerSource(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	fs, e := m.NewFactors()
	serviceMust(t, e)
	p, e := m.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, m.SetInts(p, []unumeral.Int{unumeral.NewInt(2), unumeral.NewInt(3)}))
	v := mustUPolyView(t, m, p)
	serviceMust(t, fs.PushCopy(v, 2))
	serviceMust(t, m.DelPolynomial(&p))
	header := fs.header
	serviceMust(t, fs.SourceFinalize())
	if !fs.sourceFinalized || fs.sourcePhase != 6 || fs.closed || fs.header != header || m.liveFactors != 1 || fs.count != 0 || fs.totalDegree != 0 || fs.factors != nil || fs.degrees != nil {
		t.Fatal("source/owning boundary", fs)
	}
	serviceMust(t, fs.SourceFinalize())
	serviceMust(t, m.SourceFinalize())
	serviceMust(t, fs.Close())
	if !fs.closed || fs.header != 0 || m.liveFactors != 0 || !fs.sourceFinalized || fs.sourcePhase != 6 {
		t.Fatal("owning cleanup lost source history")
	}
}

func TestUPolynomialFactorsSourceEveryStoppedCoefficient(t *testing.T) {
	for at := 0; at < 6; at++ {
		t.Run(fmt.Sprint(at), func(t *testing.T) {
			service, m, p, rt, _ := contextTLSOwner(t)
			z := service.Manager.Integers()
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() {
				if !fs.closed {
					contextTLSFinish(t, rt, fs.Close)
				}
			})
			digits := make([]uint32, 40000)
			digits[0], digits[len(digits)-1] = 1, 1
			var large unumeral.Int
			serviceMust(t, z.SetLimbs(&large, digits))
			for factor := 0; factor < 2; factor++ {
				nums := []unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(2), unumeral.NewInt(3)}
				for i := range nums {
					if factor*3+i >= at {
						nums[i] = large
					}
				}
				serviceMust(t, m.SetInts(p, nums))
				v := mustUPolyView(t, m, p)
				serviceMust(t, fs.PushCopy(v, 1))
			}
			constant := unumeral.NewInt(7)
			serviceMust(t, fs.SetConstant(&constant))
			serviceMust(t, m.DelPolynomial(&p))
			serviceMust(t, z.Del(&large))
			before := make([]unumeral.Int, 0, 6)
			for i := range fs.factors {
				before = append(before, m.values(&fs.factors[i])...)
			}
			header := fs.header
			constantBefore := fs.constant
			contextTLSNearMinimum(t, rt)
			blocks := rt.Stats().LiveBlocks
			stop := fs.SourceFinalize()
			if !errors.Is(stop, arena.ErrBounds) || fs.sourcePhase != 0 || fs.sourceFactor != uint32(at/3) || fs.sourceIndex != uint32(at%3) || fs.sourceFinalized || fs.closed || fs.header != header || rt.Stats().LiveBlocks >= blocks || fs.constant != constantBefore || fs.count != 2 {
				t.Fatal("actual ordered source stop", stop, fs.sourceFactor, fs.sourceIndex)
			}
			for i := at; i < 6; i++ {
				if m.values(&fs.factors[i/3])[i%3] != before[i] {
					t.Fatal("stopped/older coefficient changed", i)
				}
			}
			blocks = rt.Stats().LiveBlocks
			if fs.SourceFinalize() != stop || rt.Stats().LiveBlocks != blocks {
				t.Fatal("source release replayed")
			}
			contextTLSFinish(t, rt, fs.Close)
			if fs.sourceCause != stop || fs.sourceFactor != uint32(at/3) || fs.sourceIndex != uint32(at%3) || fs.sourceFinalized {
				t.Fatal("owning cleanup changed source history")
			}
		})
	}
}

func TestUPolynomialFactorsSourceStoppedConstantSet(t *testing.T) {
	service, m, p, rt, _ := contextTLSOwner(t)
	z := service.Manager.Integers()
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() {
		if !fs.closed {
			contextTLSFinish(t, rt, fs.Close)
		}
	})
	serviceMust(t, m.SetInts(p, []unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(2)}))
	v := mustUPolyView(t, m, p)
	serviceMust(t, fs.PushCopy(v, 3))
	serviceMust(t, m.DelPolynomial(&p))
	digits := make([]uint32, 40000)
	digits[0], digits[len(digits)-1] = 1, 1
	var large unumeral.Int
	serviceMust(t, z.SetLimbs(&large, digits))
	serviceMust(t, fs.SetConstant(&large))
	serviceMust(t, z.Del(&large))
	constant := fs.constant
	totalDegree, totalFactors := fs.totalDegree, fs.totalFactors
	contextTLSNearMinimum(t, rt)
	stop := fs.SourceFinalize()
	if !errors.Is(stop, arena.ErrBounds) || fs.sourcePhase != 2 || fs.constant != constant || fs.count != 0 || len(fs.factors) != 0 || len(fs.degrees) != 0 || fs.totalDegree != totalDegree || fs.totalFactors != totalFactors {
		t.Fatal("literal clear/set/stats ordering", stop, fs.sourcePhase)
	}
	contextTLSFinish(t, rt, fs.Close)
	if fs.sourceCause != stop || fs.sourcePhase != 2 || fs.sourceFinalized {
		t.Fatal("owning cleanup changed source stop")
	}
}

func TestUPolynomialFactorsOwningCloseDoesNotAdvanceSource(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	fs, e := m.NewFactors()
	serviceMust(t, e)
	serviceMust(t, fs.Close())
	if fs.sourceStarted || fs.sourceFinalized || fs.sourcePhase != 0 || fs.sourceCause != nil {
		t.Fatal("owning close relabeled as source destructor")
	}
}

func TestUPolynomialFactorsSourceResetReadsAssertionOwner(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	m, _, _ := upolySourceOwner(t)
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() {
		if !fs.closed {
			serviceMust(t, fs.Close())
		}
	})
	p, e := m.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, m.SetInts(p, []unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(2)}))
	serviceMust(t, fs.PushCopy(mustUPolyView(t, m, p), 1))
	serviceMust(t, m.DelPolynomial(&p))
	before := append([]unumeral.Int(nil), m.values(&fs.factors[0])...)
	polynomialClosedCellAssertionContext(t)
	stop := fs.SourceFinalize()
	if !errors.Is(stop, arena.ErrClosed) || fs.sourcePhase != 0 || fs.sourceFactor != 0 || fs.sourceIndex != 0 || fs.sourceResetAdmitted || fs.count != 1 {
		t.Fatal("true reset predicate omitted assertion owner", stop)
	}
	for i, a := range before {
		if m.values(&fs.factors[0])[i] != a {
			t.Fatal("source reset crossed closed assertion owner")
		}
	}
	serviceMust(t, fs.Close())
	if fs.sourceCause != stop || fs.sourcePhase != 0 {
		t.Fatal("owning cleanup changed source cause")
	}
}
