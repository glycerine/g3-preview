package smallint

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestInfinitesimalSourceStreamInsertionOrderAndWidth(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value Infinitesimal
		want  string
	}{
		{"finite", NewPair(3, 0), "__________+3"},
		{"positive epsilon", NewPair(3, 2), "___________(+3 +e*+2)"},
		{"negative epsilon", NewPair(-3, -2), "___________(-3 -e*-2)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			budget := arena.NewBudget(1 << 16)
			out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
			if err != nil {
				t.Fatal(err)
			}
			if err = out.SetWidth(12); err != nil {
				t.Fatal(err)
			}
			if err = out.SetFill('_'); err != nil {
				t.Fatal(err)
			}
			if err = out.SetShowPos(true); err != nil {
				t.Fatal(err)
			}
			if err = tc.value.AppendTo(out); err != nil {
				t.Fatal(err)
			}
			got, err := out.String()
			if err != nil || got != tc.want {
				t.Fatalf("source insertion: got %q, %v; want %q", got, err, tc.want)
			}
			if err = out.Close(); err != nil || budget.Used() != 0 {
				t.Fatalf("stream cleanup: %v, %d bytes", err, budget.Used())
			}
		})
	}
}
