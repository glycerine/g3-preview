// Source: src/smt/params/qi_params.cpp:43-64; src/util/params.cpp:928-930.
// Selected libstdc++ numeric/C-string ostream profile; direct Go file emission
// is an explicit adapter. Source author: Leonardo de Moura (2012-12-02).
// Licensed under the MIT license; see LICENSE.txt.
package g3

func (out *NumericTextStream) AppendCStringView(v ParamsStringView) error {
	if e := out.lock(); e != nil {
		return e
	}
	defer out.mu.Unlock()
	if !v.present {
		out.fileBad = true
		return nil
	} // source null insertion sets badbit.
	n, e := v.length()
	if e != nil {
		return e
	} // strlen precedes sentry/badbit.
	if out.fileBad {
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
