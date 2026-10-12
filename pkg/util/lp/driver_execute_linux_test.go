//go:build linux

package lp

import (
	"errors"
	"strings"
	"testing"

	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func TestLPExecutionLinuxBatchStopsBeforeOpeningCatalogueFile(t *testing.T) {
	h := newLPPairHarness(t)
	failure := errors.New("batch first solver report")
	w := &lpMPSDriverActionWriter{action: func(text string) error {
		if strings.HasPrefix(text, "solving ") {
			return failure
		}
		return nil
	}}
	d := NewLPTestDriver(h.m, z3rand.New(0), MPSDriverConfig{Output: w, Clock: lpBenchmarkClock()})
	// Exact mode bypasses the known-solution floating prelude. Refuse the
	// solver's first report before it can open any real HOME catalogue file.
	err := d.Run([]string{"--solve_some_mps", "--mpq", "--rep_frq", "0"})
	if !errors.Is(err, failure) || d.Finished || d.closed || d.Batch == nil || d.Batch.Completed != 0 || d.Batch.Current == nil || d.Batch.Current.rationalReader != nil {
		t.Fatal(d, err)
	}
	h.must(d.Close())
	lpExactVectorNamedCensus(t, h, nil)
}
