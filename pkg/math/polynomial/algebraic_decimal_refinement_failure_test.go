package polynomial

import (
	"bytes"
	"strconv"
	"testing"
)

// The checked precision20 sweep crosses thousands of real refusal positions.
// Run it with an explicit long timeout; keep every ordinal through success.
func TestAlgebraicDecimalRefinementReservationPrefixes(t *testing.T) {
	success, failures := false, 0
	for fault := int64(0); fault < 32768 && !success; fault++ {
		if !t.Run(strconv.FormatInt(fault, 10), func(t *testing.T) {
			c, _, _ := algebraicCompareFixture(t)
			root := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
			budget := c.config.Rationals.Budget()
			var text bytes.Buffer
			budget.FailAfter(fault)
			body := c.DisplayDecimal(&text, &root, 20)
			budget.FailAfter(-1)
			if body != nil {
				failures++
				if !numericStorageReservationRefusal(body) {
					t.Fatal("unexpected decimal refinement failure", body)
				}
			} else {
				success = true
				if text.String() != "1.41421356237309504880?" {
					t.Fatal("independent decimal expansion", text.String())
				}
			}
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, c.config.Binary.CleanupDisplay())
			serviceMust(t, c.Del(&root))
		}) {
			return
		}
	}
	if !success || failures == 0 {
		t.Fatal("decimal campaign did not reach both refusal and success", failures, success)
	}
}
