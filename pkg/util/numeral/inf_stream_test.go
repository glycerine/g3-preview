package numeral

import (
	"strings"
	"testing"

	utext "github.com/glycerine/g3/pkg/util/text"
)

func TestInfinitesimalSourceWholeStringStreamInsertion(t *testing.T) {
	s, budget := serviceNew(t)
	ordinary := infValue(t, s, 3, 2, "-2")
	integer := infIntValue(t, s, "3/2", -2)
	withInfinity := infEpsValue(t, s, "1", "3/2", "-2")
	defer func() {
		serviceMust(t, s.InfDel(&ordinary))
		serviceMust(t, s.InfIntDel(&integer))
		serviceMust(t, s.InfEpsInfDel(&withInfinity))
	}()

	for _, tc := range []struct {
		name, want string
		write      func(*utext.NumericTextStream) error
	}{
		{"inf_rational", "(3/2 -e*2)", func(out *utext.NumericTextStream) error { return s.InfDisplay(out, &ordinary) }},
		{"inf_int_rational", "(3/2 -e*2)", func(out *utext.NumericTextStream) error { return s.InfIntDisplay(out, &integer) }},
		{"inf_eps_rational", "(oo + (3/2 -e*2))", func(out *utext.NumericTextStream) error { return s.InfEpsInfDisplay(out, &withInfinity) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: budget})
			serviceMust(t, err)
			defer func() { serviceMust(t, out.Close()) }()
			serviceMust(t, out.SetWidth(24))
			serviceMust(t, out.SetFill('_'))
			serviceMust(t, out.SetShowPos(true))
			serviceMust(t, out.SetIntegerBase(16))
			serviceMust(t, tc.write(out))
			got, err := out.String()
			serviceMust(t, err)
			want := strings.Repeat("_", 24-len(tc.want)) + tc.want
			if got != want {
				t.Fatalf("whole-string insertion: got %q, want %q", got, want)
			}
			serviceMust(t, tc.write(out))
			got, err = out.String()
			serviceMust(t, err)
			if got != want+tc.want {
				t.Fatalf("width was not consumed once: got %q", got)
			}
		})
	}
}
