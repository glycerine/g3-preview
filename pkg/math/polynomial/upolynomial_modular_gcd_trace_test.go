package polynomial

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestUPolynomialModularGCDLiteralTrace(t *testing.T) {
	for _, f := range []struct {
		a, b  []int32
		want  []int64
		lines []int
	}{
		{[]int32{1, 1}, []int32{2, 1}, []int64{1}, []int{752, 774}},
		{[]int32{-200000, -99998, 1}, []int32{-300000, -99997, 1}, []int64{-100000, 1}, []int{752, 752, 802}},
		{[]int32{-2, 1, 1}, []int32{-39105, 39104, 1}, []int64{-1, 1}, []int{752, 752, 786, 802}},
		{[]int32{-2, 78205, 39103}, []int32{-3, 117308, 39103}, []int64{-1, 39103}, []int{752, 758, 752, 786, 752, 802}},
		{[]int32{-1, 0, 1}, []int32{-2, -39101, 39103}, []int64{-1, 1}, []int{752, 762, 752, 786, 802}},
	} {
		for _, refused := range []bool{false, true} {
			t.Run(fmt.Sprint(f.lines, refused), func(t *testing.T) {
				m, _, _ := upolySourceOwner(t)
				a, b, g := upolySourceNew(t, m, f.a), upolySourceNew(t, m, f.b), upolySourceNew(t, m, []int32{99})
				var text strings.Builder
				bad := factorDegreeRefusedTrace{cause: errors.New("mgcd output refused")}
				tags := diagnostic.NewTraceTagsWithWriter(&text)
				if refused {
					tags = diagnostic.NewTraceTagsWithWriter(&bad)
				}
				tags.Enable("mgcd")
				serviceMust(t, m.modular.BindTrace(tags))
				defer m.modular.BindTrace(nil)
				serviceMust(t, m.GCD(a, b, g))
				upolyGCDExpect(t, m, g, f.want)
				if !tags.ShouldTrace("mgcd") {
					if text.Len() != 0 || bad.calls != 0 {
						t.Fatal("erased trace executed")
					}
					return
				}
				if refused {
					if bad.calls != 1 || bad.flushes != len(f.lines)+1 {
						t.Fatal("sticky writes/footer flushes", bad.calls, bad.flushes, f.lines)
					}
					return
				}
				var want strings.Builder
				primeIndex := 0
				for _, line := range f.lines {
					body := "bad prime, leading coefficient vanished\n"
					switch line {
					case 752:
						body = fmt.Sprintf("trying prime: %d\n", upolyBigPrimes[primeIndex])
						primeIndex++
					case 774:
						body = "done, modular gcd is one\n"
					case 786:
						body = "discarding image\n"
					case 802:
						body = "found GCD\n"
					}
					fmt.Fprintf(&want, "-------- [mgcd] mod_gcd src/math/polynomial/upolynomial.cpp:%d ---------\n%s------------------------------------------------\n", line, body)
				}
				entryEnd := strings.Index(text.String(), "------------------------------------------------\n")
				if entryEnd < 0 || !strings.HasPrefix(text.String(), "-------- [mgcd] mod_gcd src/math/polynomial/upolynomial.cpp:726 ") {
					t.Fatal("missing entry record", text.String())
				}
				got := text.String()[entryEnd+len("------------------------------------------------\n"):]
				if got != want.String() {
					t.Fatalf("records:\n%s\nwant:\n%s", text.String(), want.String())
				}
			})
		}
	}
}

// A bad-prime record is inside scoped_set_zp, before restoration to Z.
type upolyModGCDTraceBorrowProbe struct {
	t    *testing.T
	m    *UPolynomialManager
	seen int
}

func (w *upolyModGCDTraceBorrowProbe) Write(p []byte) (int, error) {
	if string(p) == "trying prime: " {
		finite, e := w.m.modular.Modular()
		serviceMust(w.t, e)
		if finite {
			w.t.Fatal("prime record after entering finite field")
		}
		if w.m.Close() == nil || w.m.SourceFinalize() == nil {
			w.t.Fatal("prime callback retired manager")
		}
	}
	if string(p) == "bad prime, leading coefficient vanished\n" {
		w.seen++
		finite, e := w.m.modular.Modular()
		serviceMust(w.t, e)
		if !finite {
			w.t.Fatal("bad-prime record after field restoration")
		}
		if w.m.Close() == nil || w.m.SourceFinalize() == nil {
			w.t.Fatal("trace callback retired receiving manager")
		}
	}
	return len(p), nil
}
func TestUPolynomialModularGCDBadPrimeTraceBorrow(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	a := upolySourceNew(t, m, []int32{-2, 78205, 39103})
	b := upolySourceNew(t, m, []int32{-3, 117308, 39103})
	g := upolySourceNew(t, m, nil)
	probe := upolyModGCDTraceBorrowProbe{t: t, m: m}
	tags := diagnostic.NewTraceTagsWithWriter(&probe)
	tags.Enable("mgcd")
	serviceMust(t, m.modular.BindTrace(tags))
	defer m.modular.BindTrace(nil)
	serviceMust(t, m.GCD(a, b, g))
	upolyGCDExpect(t, m, g, []int64{-1, 39103})
	if tags.ShouldTrace("mgcd") && probe.seen != 1 {
		t.Fatal("bad-prime scope not observed", probe.seen)
	}
}

func TestUPolynomialModularGCDPrimeTableSmallInsertion(t *testing.T) {
	for _, p := range upolyBigPrimes {
		if p == 0 || p > 2147483647 {
			t.Fatalf("source prime requires retained BIG insertion child: %d", p)
		}
	}
}
