// Source: src/math/subpaving/subpaving_t_def.h:1809,1844; config_mpq.
package subpaving

import (
	"regexp"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestRationalSearchStatisticsTraceEntryExitAndNodeLimit(t *testing.T) {
	for _, maxNodes := range []uint32{0, 3} {
		c, _, _, _, _ := rationalClauseFixture(t)
		c.maxNodes, c.maxDepth = maxNodes, 1
		var output polynomialTraceFlushingBuffer
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		tags.Enable("subpaving_stats")
		if err := c.BindTrace(tags); err != nil {
			t.Fatal(err)
		}
		if err := c.Search(); err != nil {
			t.Fatal(err)
		}
		if tags.ShouldTrace("subpaving_stats") {
			got := output.String()
			if strings.Count(got, "statistics:\n") != 2 || strings.Count(got, "subpaving_t_def.h:1809") != 1 || strings.Count(got, "subpaving_t_def.h:1844") != 1 || !strings.Contains(got, "statistics:\n(:nodes 1)\n") || output.flushes != 2 {
				t.Fatalf("entry/exit source statistics: %q; flushes %d", got, output.flushes)
			}
			if maxNodes == 0 {
				if len(c.nodes) != 1 || strings.Count(got, "(:nodes 1)") != 2 {
					t.Fatal("node-limit break missed final statistics", len(c.nodes), got)
				}
			} else if len(c.nodes) != 3 || !regexp.MustCompile(`:nodes\s+3`).MatchString(got) || !regexp.MustCompile(`:splits\s+1`).MatchString(got) {
				t.Fatal("post-search statistics lost split/node counters", len(c.nodes), got)
			}
		} else if output.Len() != 0 || output.flushes != 0 {
			t.Fatal("erased statistics trace emitted", output.String(), output.flushes)
		}
	}
}

func TestRationalSearchStatisticsTraceStreamRefusalKeepsExit(t *testing.T) {
	c, _, _, _, _ := rationalClauseFixture(t)
	c.maxDepth = 1
	tags := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	tags.Enable("subpaving_stats")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.Search(); err != nil || len(c.nodes) != 3 || c.numSplits != 1 {
		t.Fatal("source statistics stream refusal changed search", err, len(c.nodes), c.numSplits)
	}
}
