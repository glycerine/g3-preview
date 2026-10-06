// Licensed under the MIT license; see LICENSE.txt.
// Source: src/util/mpz.cpp:1385-1410; src/util/mpq.cpp:130-141.
// Source author: Leonardo de Moura.
package g3

import "github.com/glycerine/g3/golang/mpnum"

func (out *NumericTextStream) AppendNumeralCString(v mpnum.NumeralCStringView) error {
	if e := out.lock(); e != nil {
		return e
	}
	defer out.mu.Unlock()
	var n uint64
	for {
		ch, e := v.Byte(n)
		if e != nil {
			return e
		}
		if ch == 0 {
			break
		}
		n++
	}
	if out.fileBad || out.fileFail {
		return nil
	}
	pad, e := out.fieldPadding(n)
	if e != nil {
		return e
	}
	if out.state.Alignment != NumericLeft {
		if e = out.padding(pad); e != nil {
			return e
		}
	}
	for i := uint64(0); i < n; i++ {
		ch, e := v.Byte(i)
		if e != nil {
			return e
		}
		if e = out.emitByte(ch); e != nil {
			return e
		}
	}
	if out.state.Alignment == NumericLeft {
		return out.padding(pad)
	}
	return nil
}
