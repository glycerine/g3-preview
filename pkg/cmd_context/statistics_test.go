package cmd_context

import (
	"bytes"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/solver"
)

func TestCommandContextStatisticsResultPriorityAndProcessCounters(t *testing.T) {
	c := contextFixture(t)
	m, err := c.Manager()
	contextMust(t, err)
	runtime := m.NumeralService().RuntimeOwner()
	block, err := runtime.Allocate(2 << 20)
	contextMust(t, err)
	defer func() { contextMust(t, runtime.Deallocate(&block)) }()
	_, err = m.Canceled()
	contextMust(t, err)
	result, err := solver.NewSimpleCheckSatResult(m)
	contextMust(t, err)
	c.tacticResult = result
	stats, err := result.Statistics()
	contextMust(t, err)
	stats.UpdateUint("selected result", 17)
	stats.UpdateUint("dup", 3)
	// The live solver exists as well; the explicit result must take precedence.
	before := m.Limit.Count()
	maximum, err := runtime.MaximumUsed()
	contextMust(t, err)
	used, err := runtime.AllocationSize()
	contextMust(t, err)
	allocations, err := runtime.AllocationCount()
	contextMust(t, err)
	var out bytes.Buffer
	contextMust(t, c.WriteStatistics(&out))
	text := out.String()
	for key, want := range map[string]float64{"selected-result": 17, "dup": 3, "rlimit-count": float64(before), "memory": float64((100*used)/(1024*1024)) / 100, "max-memory": float64((100*maximum)/(1024*1024)) / 100, "num-allocs": float64(allocations)} {
		match := regexp.MustCompile(`:` + key + `\s+([0-9.]+)`).FindStringSubmatch(text)
		if want == 0 && len(match) == 0 {
			continue
		}
		if len(match) != 2 {
			t.Fatal("missing statistic", key, text)
		}
		got, err := strconv.ParseFloat(match[1], 64)
		contextMust(t, err)
		if got != want {
			t.Fatal("statistic value", key, got, want, text)
		}
	}
	if m.Limit.Count() != before {
		t.Fatal("statistics read consumed a resource step")
	}
	contextMust(t, c.clearCheckSatResult())
	out.Reset()
	contextMust(t, c.WriteStatistics(&out))
	if strings.Contains(out.String(), "selected-result") {
		t.Fatal("retired result still contributes")
	}
}

func TestCommandContextStatisticsWriterAndWatch(t *testing.T) {
	c := contextFixture(t)
	if err := c.WriteStatistics(nil); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal("nil statistics writer", err)
	}
	_, err := c.CheckSat(nil)
	contextMust(t, err)
	if c.watch.Running() || c.watch.ElapsedNanoseconds() == 0 {
		t.Fatal("check-sat stopwatch was not closed")
	}
	var out bytes.Buffer
	contextMust(t, c.WriteStatistics(&out))
	if !strings.Contains(out.String(), ":time") {
		t.Fatal("check time missing", out.String())
	}
	for _, limit := range []int{0, 1, 4, 12} {
		if err := c.WriteStatistics(&reasonUnknownWriter{remaining: limit}); !errors.Is(err, io.ErrClosedPipe) {
			t.Fatal("statistics writer refusal", limit, err)
		}
	}
	contextMust(t, c.Close())
	if err := c.WriteStatistics(&out); err == nil {
		t.Fatal("closed context statistics accepted")
	}
}
