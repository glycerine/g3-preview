// Source: src/math/interval/interval_def.h:208-210; src/util/trace.h.
package interval

import (
	"io"
	"strconv"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func (r ArbitraryRoot) traceNthRoot(a *numeral.ArbitraryFloat, n uint32, lo, hi *numeral.ArbitraryFloat) error {
	tags := r.TraceTags
	if tags == nil {
		tags = diagnostic.BorrowProcessTraceTags()
	}
	if !tags.ShouldTrace("nth_root_trace") {
		return nil
	}
	var formatErr error
	_ = tags.EmitShort("nth_root_trace", func(w io.Writer) error {
		write := func(s string) { _, _ = io.WriteString(w, s) }
		display := func(v *numeral.ArbitraryFloat) bool {
			text, e := r.Numbers.ToString(v)
			if e != nil {
				formatErr = e
				return false
			}
			write(text)
			return true
		}
		for _, item := range [...]struct {
			bound *numeral.ArbitraryFloat
			op    string
		}{{lo, " >= "}, {hi, " <= "}} {
			write("[nth-root] (")
			if !display(a) {
				return formatErr
			}
			write(")^(1/")
			write(strconv.FormatUint(uint64(n), 10))
			write(")")
			write(item.op)
			if !display(item.bound) {
				return formatErr
			}
			_ = diagnostic.WriteTraceEndl(w)
		}
		return nil
	})
	return formatErr
}
