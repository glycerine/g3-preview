//go:build g3trace

package interval

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRationalRootRetainedTraceSitesSourceOrder(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	m, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	a, p, lo, hi := numeral.NewRational(), numeral.NewRational(), numeral.NewRational(), numeral.NewRational()
	t.Cleanup(func() {
		for _, err := range []error{m.Del(&hi), m.Del(&lo), m.Del(&p), m.Del(&a), m.CloseChecked()} {
			if err != nil {
				t.Error(err)
			}
		}
		if budget.Used() != 0 {
			t.Error("rational root trace owners", budget.Used())
		}
	})
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	show := func(v *numeral.Rational) string {
		t.Helper()
		var b bytes.Buffer
		must(m.DisplayTo(&b, v))
		return b.String()
	}
	var writer rootTraceWriter
	tags := diagnostic.NewTraceTagsWithWriter(&writer)
	root := RationalRoot{Numbers: m, Limit: &control.Reslimit{Limit: 1000}, TraceTags: tags}
	must(m.SetFractionSigned(&p, 1, 8192))
	must(m.SetInt32(&a, 2))
	must(root.Bounds(&a, 3, &p, &lo, &hi))
	if writer.Len() != 0 || writer.flushes != 0 {
		t.Fatalf("disabled rational root trace: %q / %d", writer.String(), writer.flushes)
	}
	for _, tag := range []string{"nth_root", "nth_root_bug", "nth_root_trace"} {
		tags.Enable(tag)
	}
	must(m.SetInt32(&a, 1))
	must(root.Bounds(&a, 3, &p, &lo, &hi))
	if writer.Len() != 0 || writer.flushes != 0 {
		t.Fatalf("easy rational root traced: %q / %d", writer.String(), writer.flushes)
	}
	must(m.SetInt32(&a, 2))
	must(root.Bounds(&a, 3, &p, &lo, &hi))
	short := "[nth-root] (" + show(&a) + ")^(1/3) >= " + show(&lo) + "\n" +
		"[nth-root] (" + show(&a) + ")^(1/3) <= " + show(&hi) + "\n"
	got := writer.String()
	if !strings.HasSuffix(got, short) || strings.Contains(got, "[nth_root_bug]") {
		t.Fatalf("precise root short/bound-check trace: %q, want tail %q and no bug record", got, short)
	}
	iterations := strings.TrimSuffix(got, short)
	header := "-------- [nth_root] approx_nth_root src/math/interval/interval_def.h:315 ---------\n"
	parts := strings.Split(iterations, header)
	if len(parts) < 3 || parts[0] != "" || writer.flushes != len(parts) {
		t.Fatalf("rational iteration count/flushes: %d records, %d flushes", len(parts)-1, writer.flushes)
	}
	for i, part := range parts[1:] {
		if !strings.HasSuffix(part, "------------------------------------------------\n") {
			t.Fatalf("iteration %d footer: %q", i, part)
		}
		body := strings.TrimSuffix(part, "------------------------------------------------\n")
		fields := strings.Split(body, "\n")
		for j, prefix := range []string{"A:       " + show(&a), "x:       ", "x_prime: ", "d:       "} {
			if len(fields) != 5 || !strings.HasPrefix(fields[j], prefix) || j != 0 && len(fields[j]) == len(prefix) {
				t.Fatalf("iteration %d field %d: %q", i, j, body)
			}
		}
	}
	writer.Reset()
	writer.flushes = 0
	must(m.SetInt32(&a, 4))
	must(root.Bounds(&a, 2, &p, &lo, &hi))
	if writer.flushes != 1 || strings.Contains(writer.String(), "-------- [") {
		t.Fatalf("degree-two precise root emitted iteration/bound record: %q / %d", writer.String(), writer.flushes)
	}
}
