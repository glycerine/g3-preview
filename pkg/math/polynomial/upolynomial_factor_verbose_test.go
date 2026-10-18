package polynomial

import (
	"fmt"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	utext "github.com/glycerine/g3/pkg/util/text"
	"os"
	"testing"
)

func factorVerboseFixture(t *testing.T, level uint32, out *utext.NumericTextStream) {
	t.Helper()
	oldLevel, e := diagnostic.ProcessVerbosityLevel()
	serviceMust(t, e)
	oldOut, e := diagnostic.BorrowProcessVerboseStream()
	serviceMust(t, e)
	serviceMust(t, diagnostic.SetProcessVerboseStream(out))
	serviceMust(t, diagnostic.SetProcessVerbosityLevel(level))
	t.Cleanup(func() {
		serviceMust(t, diagnostic.SetProcessVerbosityLevel(oldLevel))
		serviceMust(t, diagnostic.SetProcessVerboseStream(oldOut))
	})
}
func TestUPolynomialFactorVerboseThresholdAndResult(t *testing.T) {
	for _, f := range []struct {
		name  string
		coeff []int32
		count uint32
	}{{"zero", nil, 0}, {"constant", []int32{7}, 0}, {"quadratic", []int32{-1, 0, 1}, 2}} {
		for _, level := range []uint32{999, 1000, 1001} {
			t.Run(f.name+fmt.Sprint(level), func(t *testing.T) {
				q, s := preparationPhysicalFixture(t)
				m := s.manager
				p := upolySourceNew(t, m, f.coeff)
				fs, e := m.NewFactors()
				serviceMust(t, e)
				t.Cleanup(func() { serviceMust(t, s.CleanupRetained()); serviceMust(t, fs.Close()) })
				stream, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: q.Budget()})
				serviceMust(t, e)
				t.Cleanup(func() { serviceMust(t, stream.Close()) })
				factorVerboseFixture(t, level, stream)
				ok, e := s.Factor(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams())
				serviceMust(t, e)
				if !ok || fs.count != f.count {
					t.Fatal("verbose changed factorization", ok, fs.count)
				}
				text, e := stream.String()
				serviceMust(t, e)
				want := ""
				if level >= 1000 {
					want = fmt.Sprintf("(polynomial-factorization :distinct-factors %d)\n", f.count)
				}
				if text != want {
					t.Fatal(text, want)
				}
			})
		}
	}
}
func TestUPolynomialFactorVerboseUsesNumericFlags(t *testing.T) {
	q, s := preparationPhysicalFixture(t)
	m := s.manager
	p := upolySourceNew(t, m, []int32{1, 1})
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	for i := 0; i < 10; i++ {
		serviceMust(t, fs.PushCopy(mustUPolyView(t, m, p), uint32(i+1)))
	}
	stream, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: q.Budget()})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, stream.Close()) })
	serviceMust(t, stream.SetIntegerBase(16))
	serviceMust(t, stream.SetShowBase(true))
	factorVerboseFixture(t, 1000, stream)
	serviceMust(t, s.verboseFactorCount(fs))
	text, e := stream.String()
	serviceMust(t, e)
	if text != "(polynomial-factorization :distinct-factors 0xa)\n" {
		t.Fatal("source insertion flags/count versus multiplicity", text)
	}
}
func TestUPolynomialFactorVerboseClosedFileAndFailedBody(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		t.Run(fmt.Sprint(cancel), func(t *testing.T) {
			q, s := preparationPhysicalFixture(t)
			m := s.manager
			p := upolySourceNew(t, m, []int32{-1, 0, 1})
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, s.CleanupRetained()); serviceMust(t, fs.Close()) })
			file, e := os.CreateTemp(t.TempDir(), "factor-verbose-*")
			serviceMust(t, e)
			stream, e := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: q.Budget(), BorrowedFile: file})
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, stream.Close()) })
			serviceMust(t, file.Close())
			factorVerboseFixture(t, 1000, stream)
			if cancel {
				m.config.Checkpoint = func() bool { return false }
			}
			_, e = s.Factor(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams())
			if cancel {
				if e == nil || stream.FileBad {
					t.Fatal("abnormal body reached verbose stream", e, stream.FileBad)
				}
			} else {
				serviceMust(t, e)
				if !stream.FileBad || fs.count != 2 {
					t.Fatal("closed physical file changed result/failbit", stream.FileBad, fs.count)
				}
			}
		})
	}
}
