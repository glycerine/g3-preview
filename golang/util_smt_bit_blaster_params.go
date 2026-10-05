// Source: src/ast/rewriter/bit_blaster/bit_blaster_params.h:22-40.
// Source author: Leonardo de Moura (2008-10-02).
// Licensed under the MIT license; see LICENSE.txt.
package g3

// Actual inline scalar base. Disabled registration has no active update/getter.
type SMTBitBlasterParams struct{ ExtGates, Quantifiers bool }

func NewSMTBitBlasterParams() SMTBitBlasterParams { return SMTBitBlasterParams{false, false} }
func (p SMTBitBlasterParams) Display(out *NumericTextStream) error {
	if e := out.AppendCString("m_bb_ext_gates="); e != nil {
		return e
	}
	bExtGates := int64(0)
	if p.ExtGates {
		bExtGates = 1
	}
	if e := out.AppendInt64(bExtGates); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_bb_quantifiers="); e != nil {
		return e
	}
	bQuantifiers := int64(0)
	if p.Quantifiers {
		bQuantifiers = 1
	}
	if e := out.AppendInt64(bQuantifiers); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	return nil
}
