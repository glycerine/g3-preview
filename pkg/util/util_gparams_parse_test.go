// Licensed under the MIT license.
package util

import (
	"math"
	"math/big"
	"strconv"
	"strings"
	"testing"
)

func TestGlobalParamsCLongFilteredSaturationAndNarrow(t *testing.T) {
	for _, x := range []struct {
		s        string
		u64, u32 uint32
	}{
		{"", 0, 0}, {"000000000000000000000001", 1, 1}, {"2147483647", 2147483647, 2147483647},
		{"2147483648", 2147483648, 2147483647}, {"4294967295", 4294967295, 2147483647},
		{"4294967296", 0, 2147483647}, {"9223372036854775807", 4294967295, 2147483647},
		{"9223372036854775808", 4294967295, 2147483647}, {"999999999999999999999999999999999999", 4294967295, 2147483647},
	} {
		for _, bits := range []uint8{32, 64} {
			got := gparamsParseUint(x.s, bits)
			want := x.u64
			if bits == 32 {
				want = x.u32
			}
			if got != want {
				t.Fatal(x.s, bits, got, want)
			}
		}
	}
}
func TestGlobalParamsCDoubleFilteredPrefixAndIEEEBoundary(t *testing.T) {
	for _, x := range []struct {
		s    string
		want float64
	}{
		{"", 0}, {"-", 0}, {"/", 0}, {"--1", 0}, {".", 0}, {"-.", 0}, {"1/2", 1}, {"2.3.4", 2.3},
		{"12-3", 12}, {".125/5", .125}, {"-.5", -.5}, {"00001.50", 1.5}, {"-0", math.Copysign(0, -1)},
	} {
		if got := gparamsParseDouble(x.s); math.Float64bits(got) != math.Float64bits(x.want) {
			t.Fatal(x.s, got, x.want)
		}
	}
	// Exact overflow-to-infinity threshold in nearest ties-to-even. The oracle
	// derives it independently; ordinary production has no math/big dependency.
	threshold := new(big.Int).Lsh(big.NewInt(1), 1024)
	threshold.Sub(threshold, new(big.Int).Lsh(big.NewInt(1), 970))
	for _, delta := range []int64{-1000, -1, 0, 1, 1000} {
		z := new(big.Int).Add(threshold, big.NewInt(delta))
		text := z.String()
		want, _ := strconv.ParseFloat(text, 64)
		got := gparamsParseDouble(text)
		if math.Float64bits(got) != math.Float64bits(want) {
			t.Fatal(delta, got, want)
		}
		neg := gparamsParseDouble("-" + text)
		if math.Float64bits(neg) != math.Float64bits(-want) {
			t.Fatal("negative", delta)
		}
	}
}

var globalParamsParseSink float64

func TestGlobalParamsCDoubleNoHiddenOverflowErrorAllocation(t *testing.T) {
	huge := strings.Repeat("9", 400)
	for _, text := range []string{"1.5", "-0.000000000000000000000000000000000000000000000000000001", huge} {
		if n := testing.AllocsPerRun(100, func() { globalParamsParseSink = gparamsParseDouble(text) }); n != 0 {
			t.Fatal("parse allocation", n)
		}
	}
	if !math.IsInf(globalParamsParseSink, 1) {
		t.Fatal(globalParamsParseSink)
	}
}
